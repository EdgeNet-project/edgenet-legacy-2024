/*
Copyright 2021 Contributors to the EdgeNet project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vpnpeer

import (
	"fmt"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/EdgeNet-project/edgenet/pkg/apis/networking/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/networking/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/networking/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const controllerAgentName = "vpnpeer-controller"

const (
	SuccessSynced         = "Synced"
	MessageResourceSynced = "VPNPeer synced successfully"
)

// Controller is the controller implementation for VPNPeer resources
type Controller struct {
	kubeclientset    kubernetes.Interface
	edgenetclientset clientset.Interface
	vpnpeersLister   listers.VPNPeerLister
	vpnpeersSynced   cache.InformerSynced
	workqueue        workqueue.RateLimitingInterface
	recorder         record.EventRecorder
	linkname         string
}

// NewController returns a new VPNPeer controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	vpnpeerInformer informers.VPNPeerInformer,
	linkname string) *Controller {
	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:    kubeclientset,
		edgenetclientset: edgenetclientset,
		vpnpeersLister:   vpnpeerInformer.Lister(),
		vpnpeersSynced:   vpnpeerInformer.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "VPNPeers"),
		recorder:         recorder,
		linkname:         linkname,
	}

	klog.Info("Setting up event handlers")
	vpnpeerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.enqueueVPNPeer,
		DeleteFunc: controller.enqueueVPNPeer,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueVPNPeer(old)
			controller.enqueueVPNPeer(new)
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Starting VPNPeer controller")

	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.vpnpeersSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the VPNPeer resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	peers, err := c.vpnpeersLister.List(labels.Everything())
	if err != nil {
		return err
	}

	peer, err := findPeer(peers, key)
	if err != nil {
		return err
	}

	if peer == nil {
		// A. Deletion
		err = removePeer(c.linkname, key)
		if err != nil {
			return err
		}
		klog.Infof("Peer with public key %s removed", key)
	} else {
		// B. Creation/Update
		err = addPeer(c.linkname, *peer)
		if err != nil {
			return err
		}
		klog.Infof("Peer with public key %s synced", key)
		c.recorder.Event(peer, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	}

	return nil
}

// enqueueVPNPeer takes a VPNPeer resource.
// This method should *not* be passed resources of any type other than VPNPeer.
func (c *Controller) enqueueVPNPeer(obj interface{}) {
	// We enqueue the peer public key instead of the object name.
	// This allows us to handle deletion (or public key update) easily,
	// since a WireGuard peer on a given link is uniquely identified by its public key.
	peer := obj.(*v1alpha1.VPNPeer)
	c.workqueue.Add(peer.Spec.PublicKey)
}

func findPeer(peers []*v1alpha1.VPNPeer, publicKey string) (*v1alpha1.VPNPeer, error) {
	var found *v1alpha1.VPNPeer
	for _, peer := range peers {
		if peer.Spec.PublicKey == publicKey {
			if found != nil {
				return nil, fmt.Errorf("multiple peers found with public key %s", publicKey)
			}
			found = peer
		}
	}
	return found, nil
}

func addPeer(linkname string, peer v1alpha1.VPNPeer) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("error while creating WG client: %s", err.Error())
	}

	publicKey, err := wgtypes.ParseKey(peer.Spec.PublicKey)
	if err != nil {
		return fmt.Errorf("error while parsing WG public key: %s", err.Error())
	}

	allowedIPs := []net.IPNet{
		{
			IP:   net.ParseIP(peer.Spec.AddressV4),
			Mask: net.CIDRMask(32, 32),
		},
		{
			IP:   net.ParseIP(peer.Spec.AddressV6),
			Mask: net.CIDRMask(128, 128),
		},
	}

	var endpoint *net.UDPAddr
	if peer.Spec.EndpointAddress != nil && peer.Spec.EndpointPort != nil {
		endpoint = &net.UDPAddr{
			IP:   net.ParseIP(*peer.Spec.EndpointAddress),
			Port: *peer.Spec.EndpointPort,
		}
	}

	keepaliveInterval := 5 * time.Second

	peerConfig := wgtypes.PeerConfig{
		AllowedIPs:                  allowedIPs,
		Endpoint:                    endpoint,
		PublicKey:                   publicKey,
		PersistentKeepaliveInterval: &keepaliveInterval,
		Remove:                      false,
		ReplaceAllowedIPs:           true,
		UpdateOnly:                  false,
	}

	deviceConfig := wgtypes.Config{
		Peers:        []wgtypes.PeerConfig{peerConfig},
		ReplacePeers: false,
	}

	err = client.ConfigureDevice(linkname, deviceConfig)
	if err != nil {
		return fmt.Errorf("error while configure WG device %s: %s", linkname, err.Error())
	}

	return nil
}

func removePeer(linkname string, publicKey string) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("error while creating WG client: %s", err.Error())
	}

	pk, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("error while parsing WG public key: %s", err.Error())
	}

	peerConfig := wgtypes.PeerConfig{
		PublicKey: pk,
		Remove:    true,
	}

	deviceConfig := wgtypes.Config{
		Peers:        []wgtypes.PeerConfig{peerConfig},
		ReplacePeers: false,
	}

	err = client.ConfigureDevice(linkname, deviceConfig)
	if err != nil {
		return fmt.Errorf("error while configure WG device %s: %s", linkname, err.Error())
	}

	return nil
}
