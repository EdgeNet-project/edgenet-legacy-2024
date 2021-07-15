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
	"github.com/EdgeNet-project/edgenet/pkg/apis/networking/v1alpha"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net"
	"time"

	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/networking/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/networking/v1alpha"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
		AddFunc: controller.enqueueVPNPeer,
		UpdateFunc: func(old, new interface{}) {
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
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	peer, err := c.vpnpeersLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("vpnpeer '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	err = addPeer(c.linkname, *peer)
	if err != nil {
		return err
	}

	c.recorder.Event(peer, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// enqueueVPNPeer takes a VPNPeer resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than VPNPeer.
func (c *Controller) enqueueVPNPeer(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func addPeer(linkname string, peer v1alpha.VPNPeer) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("error while creating WG client: %s", err.Error())
	}

	publicKey, err := wgtypes.ParseKey(peer.Spec.PublicKey)
	if err != nil {
		return fmt.Errorf("error while parsing WG public key: %s", err.Error())
	}

	allowedIPs := make([]net.IPNet, 0)
	if peer.Spec.AddressV4 != nil {
		ip := net.ParseIP(*peer.Spec.AddressV4)
		mask := net.CIDRMask(32, 32)
		allowedIPs = append(allowedIPs, net.IPNet{IP: ip, Mask: mask})
	}
	if peer.Spec.AddressV6 != nil {
		ip := net.ParseIP(*peer.Spec.AddressV6)
		mask := net.CIDRMask(128, 128)
		allowedIPs = append(allowedIPs, net.IPNet{IP: ip, Mask: mask})
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

//func removePeer(linkname string, nc v1alpha.NodeContribution) error {
//	if nc.Spec.VPN == nil {
//		klog.Infof("No VPN configuration specified for nodecontribution object %s", nc.Name)
//		return nil
//	}
//
//	client, err := wgctrl.New()
//	if err != nil {
//		return fmt.Errorf("error while creating WG client: %s", err.Error())
//	}
//
//	publicKey, err := wgtypes.ParseKey(nc.Spec.VPN.PublicKey)
//	if err != nil {
//		return fmt.Errorf("error while parsing WG public key: %s", err.Error())
//	}
//
//	peerConfig := wgtypes.PeerConfig{
//		PublicKey:                   publicKey,
//		Remove:                      true,
//	}
//
//	deviceConfig := wgtypes.Config{
//		Peers:        []wgtypes.PeerConfig{peerConfig},
//		ReplacePeers: false,
//	}
//
//	err = client.ConfigureDevice(linkname, deviceConfig)
//	if err != nil {
//		return fmt.Errorf("error while configure WG device %s: %s", linkname, err.Error())
//	}
//
//	return nil
//}
