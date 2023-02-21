package nodelabeler

import (
	"context"
	"fmt"
	"time"

	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/EdgeNet-project/edgenet/pkg/multiprovider"
	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "nodelabeler-controller"

// Controller is the controller implementation
type Controller struct {
	kubeclientset    kubernetes.Interface
	edgenetclientset clientset.Interface

	lister corelisters.NodeLister
	synced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	maxmindURL        string
	maxmindAccountID  string
	maxmindLicenseKey string
}

// NewController returns a new controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	informer coreinformers.NodeInformer,
	maxmindURL string,
	maxmindAccountID string,
	maxmindLicenseKey string,
) *Controller {
	// Create event broadcaster
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	klog.V(4).Infoln("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:     kubeclientset,
		edgenetclientset:  edgenetclientset,
		lister:            informer.Lister(),
		synced:            informer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NodeLabeler"),
		recorder:          recorder,
		maxmindURL:        maxmindURL,
		maxmindAccountID:  maxmindAccountID,
		maxmindLicenseKey: maxmindLicenseKey,
	}

	klog.Infoln("Setting up event handlers")

	// Event handlers deal with events of resources. In here, we take into consideration of adding and updating nodes.
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNodelabeler,
		UpdateFunc: func(oldObj, newObj interface{}) {
			updated := multiprovider.CompareIPAddresses(oldObj.(*corev1.Node), newObj.(*corev1.Node))
			if updated {
				controller.enqueueNodelabeler(newObj)
			}
		},
	})

	return controller
}

// Run will set up the event handlers for the types of node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting Node Labeler Controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh,
		c.synced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.V(4).Infoln("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.V(4).Infoln("Started workers")
	<-stopCh
	klog.V(4).Infoln("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

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
		klog.V(4).Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	item, err := c.lister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("nodelabeler '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	klog.V(4).Infof("processNextItem: object created/updated detected: %s", key)
	c.setNodeGeolocation(item)

	return nil
}

func (c *Controller) setNodeGeolocation(obj interface{}) {
	klog.V(4).Infoln("Handler.ObjectCreated")
	nodeObj := obj.(*corev1.Node)

	internalIP, externalIP := multiprovider.GetNodeIPAddresses(nodeObj)
	result := false

	multiproviderManager := multiprovider.NewManager(c.kubeclientset, nil, nil, nil)

	// 1. Use the VPNPeer endpoint address if available.
	peer, err := c.edgenetclientset.NetworkingV1alpha1().VPNPeers().Get(context.TODO(), nodeObj.Name, v1.GetOptions{})
	if err != nil {
		klog.V(4).Infof(
			"Failed to find a matching VPNPeer object for %s: %s. The node IP will be used instead.",
			nodeObj.Name,
			err,
		)
	} else {
		klog.V(4).Infof("VPNPeer endpoint IP: %s", *peer.Spec.EndpointAddress)
		result = multiproviderManager.GetGeolocationByIP(
			c.maxmindURL,
			c.maxmindAccountID,
			c.maxmindLicenseKey,
			nodeObj.Name,
			*peer.Spec.EndpointAddress,
		)
	}

	// 2. Otherwise use the node external IP if available.
	if externalIP != "" && !result {
		klog.V(4).Infof("External IP: %s", externalIP)
		result = multiproviderManager.GetGeolocationByIP(
			c.maxmindURL,
			c.maxmindAccountID,
			c.maxmindLicenseKey,
			nodeObj.Name,
			externalIP,
		)
	}

	// 3. Otherwise use the node internal IP if available.
	if internalIP != "" && !result {
		klog.V(4).Infof("Internal IP: %s", internalIP)
		multiproviderManager.GetGeolocationByIP(
			c.maxmindURL,
			c.maxmindAccountID,
			c.maxmindLicenseKey,
			nodeObj.Name,
			internalIP,
		)
	}
}

func (c *Controller) enqueueNodelabeler(obj interface{}) {
	// Put the resource object into a key
	var key string
	var err error

	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}
