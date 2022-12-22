/*
Copyright 2022 Contributors to the EdgeNet project.

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

package managercache

import (
	"context"
	"fmt"
	"reflect"
	"time"

	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/federation/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/federation/v1alpha1"
	multitenancy "github.com/EdgeNet-project/edgenet/pkg/multitenancy"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "managercache-controller"

// Definitions of the state of the managercache resource
const (
	successSynced = "Synced"

	messageResourceSynced = "ManagerCache synced successfully"
	messageReady          = "ManagerCache is ready"
	messagePending        = "ManagerCache is pending a workload cluster"
	messageParentUpdated  = "ManagerCache updated at the parent federation manager"
	messageChildUpdated   = "ManagerCache updated at the child federation manager(s)"
)

// Controller is the controller implementation for ManagerCache resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	managercachesLister listers.ManagerCacheLister
	managercachesSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	managercacheInformer informers.ManagerCacheInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:       kubeclientset,
		edgenetclientset:    edgenetclientset,
		managercachesLister: managercacheInformer.Lister(),
		managercachesSynced: managercacheInformer.Informer().HasSynced,
		workqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ManagerCaches"),
		recorder:            recorder,
	}

	klog.Infoln("Setting up event handlers")
	// Set up an event handler for when ManagerCache resources change
	managercacheInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueManagerCache,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueManagerCache(new)
		},
	})

	return controller
}

// Run will set up the event handlers for the types of managercache and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting ManagerCache controller")

	klog.Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.managercachesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Infoln("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Infoln("Started workers")
	<-stopCh
	klog.Infoln("Shutting down workers")

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
// converge the two. It then updates the Status block of the ManagerCache
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	managercache, err := c.managercachesLister.Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("managercache '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processManagerCache(managercache.DeepCopy())
	c.recorder.Event(managercache, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueManagerCache takes a ManagerCache resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than ManagerCache.
func (c *Controller) enqueueManagerCache(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueManagerCacheAfter takes a ManagerCache resource and converts it into a namespace/name
// string which is then put onto the work queue after the expiry date to be deleted. This method should *not* be
// passed resources of any type other than ManagerCache.
func (c *Controller) enqueueManagerCacheAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

func (c *Controller) processManagerCache(managercacheCopy *federationv1alpha1.ManagerCache) {
	multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
	permitted, _, _ := multitenancyManager.EligibilityCheck(managercacheCopy.GetNamespace())
	if permitted {

		switch managercacheCopy.Status.State {
		case federationv1alpha1.StatusReady:
			// c.recorder.Event(managercacheCopy, corev1.EventTypeNormal, federationv1alpha1.StatusBound, messageRoleBound)
			remoteManagerCache := new(federationv1alpha1.ManagerCache)
			remoteManagerCache.SetName(managercacheCopy.GetName())
			remoteManagerCache.Spec = managercacheCopy.Spec
			if secretFMAuth, err := c.kubeclientset.CoreV1().Secrets("edgenet").Get(context.TODO(), "federation", metav1.GetOptions{}); err == nil {
				if managercacheCopy.Spec.Hierarchy.Parent == string(secretFMAuth.Data["cluster-uid"]) {
					remoteedgeclientset, _ := c.createRemoteEdgeNetClientset(string(secretFMAuth.Data["server"]), string(secretFMAuth.Data["serviceaccount"]), string(secretFMAuth.Data["token"]))
					if _, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Create(context.TODO(), remoteManagerCache, metav1.CreateOptions{}); err != nil && errors.IsAlreadyExists(err) {
						remoteManagerCacheCopy, _ := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), managercacheCopy.GetName(), metav1.GetOptions{})
						if !reflect.DeepEqual(managercacheCopy.Spec, remoteManagerCacheCopy.Spec) {
							remoteManagerCacheCopy.Spec = managercacheCopy.Spec
							remoteedgeclientset.FederationV1alpha1().ManagerCaches().Update(context.TODO(), remoteManagerCacheCopy, metav1.UpdateOptions{})
							c.recorder.Event(managercacheCopy, corev1.EventTypeNormal, federationv1alpha1.StatusUpdated, messageParentUpdated)
						}
					}
				}
			}
			clusterRaw, _ := c.edgenetclientset.FederationV1alpha1().Clusters("").List(context.TODO(), metav1.ListOptions{})
			for _, clusterRow := range clusterRaw.Items {
				if clusterRow.Spec.Role == "federation" {
					remoteAuthSecret, _ := c.kubeclientset.CoreV1().Secrets(clusterRow.GetNamespace()).Get(context.TODO(), clusterRow.Spec.SecretName, metav1.GetOptions{})
					remoteedgeclientset, _ := c.createRemoteEdgeNetClientset(string(remoteAuthSecret.Data["server"]), string(remoteAuthSecret.Data["serviceaccount"]), string(remoteAuthSecret.Data["token"]))
					if _, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Create(context.TODO(), remoteManagerCache, metav1.CreateOptions{}); err != nil && errors.IsAlreadyExists(err) {
						remoteManagerCacheCopy, _ := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), managercacheCopy.GetName(), metav1.GetOptions{})
						if !reflect.DeepEqual(managercacheCopy.Spec, remoteManagerCacheCopy.Spec) {
							remoteManagerCacheCopy.Spec = managercacheCopy.Spec
							remoteedgeclientset.FederationV1alpha1().ManagerCaches().Update(context.TODO(), remoteManagerCacheCopy, metav1.UpdateOptions{})
							c.recorder.Event(managercacheCopy, corev1.EventTypeNormal, federationv1alpha1.StatusUpdated, messageParentUpdated)
						}
					}
				}
			}
		default:
			if len(managercacheCopy.Spec.Clusters) == 0 {
				c.recorder.Event(managercacheCopy, corev1.EventTypeNormal, federationv1alpha1.StatusPending, messagePending)
				managercacheCopy.Status.State = federationv1alpha1.StatusPending
				managercacheCopy.Status.Message = messagePending
				c.updateStatus(context.TODO(), managercacheCopy)
				return
			}
			c.recorder.Event(managercacheCopy, corev1.EventTypeNormal, federationv1alpha1.StatusReady, messageReady)
			managercacheCopy.Status.State = federationv1alpha1.StatusReady
			managercacheCopy.Status.Message = messageReady
			c.updateStatus(context.TODO(), managercacheCopy)
		}
	} else {
		c.edgenetclientset.FederationV1alpha1().ManagerCaches().Delete(context.TODO(), managercacheCopy.GetName(), metav1.DeleteOptions{})
	}
}

// updateStatus calls the API to update the managercache status.
func (c *Controller) updateStatus(ctx context.Context, managercacheCopy *federationv1alpha1.ManagerCache) {
	if managercacheCopy.Status.State == federationv1alpha1.StatusFailed {
		managercacheCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().UpdateStatus(ctx, managercacheCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}

func (c *Controller) createRemoteEdgeNetClientset(server, username, token string) (*clientset.Clientset, error) {
	remoteConfig := new(rest.Config)
	remoteConfig.Host = server
	remoteConfig.Username = username
	remoteConfig.BearerToken = username
	// Create the clientset
	remoteedgeclientset, err := clientset.NewForConfig(remoteConfig)
	if err != nil {
		klog.Infoln(err)
	}
	return remoteedgeclientset, nil
}
