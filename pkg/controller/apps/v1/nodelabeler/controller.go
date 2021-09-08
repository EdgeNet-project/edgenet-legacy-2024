package nodelabeler

import (
	"fmt"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/node"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

// The main structure of controller
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

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
}

// NewController returns a new controller
func NewController(
	kubeclientset kubernetes.Interface,
	informer coreinformers.NodeInformer) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset: kubeclientset,
		lister:        informer.Lister(),
		synced:        informer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NodeContributions"),
		recorder:      recorder,
	}

	klog.Info("Setting up event handlers")

	// Event handlers deal with events of resources. In here, we take into consideration of adding and updating nodes.
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Put the resource object into a key
			key, err := cache.MetaNamespaceKeyFunc(obj)
			log.Infof("Add node detected: %s", key)
			if err == nil {
				// Add the key to the queue
				controller.workqueue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			updated := node.CompareIPAddresses(oldObj.(*corev1.Node), newObj.(*corev1.Node))
			if updated {
				key, err := cache.MetaNamespaceKeyFunc(newObj)
				log.Infof("Update node detected: %s", key)
				if err == nil {
					controller.workqueue.Add(key)
				}
			}
		},
	})

	node.Clientset = kubeclientset

	return controller
}

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
			utilruntime.HandleError(fmt.Errorf("nodecontribution '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	klog.V(4).Infof("processNextItem: object created/updated detected: %s", key)
	c.setNodeGeolocation(item)

	return nil
}

func (c *Controller) setNodeGeolocation(obj interface{}) {
	log.Info("Handler.ObjectCreated")
	// Get internal and external IP addresses of the node
	internalIP, externalIP := node.GetNodeIPAddresses(obj.(*corev1.Node))
	result := false
	// Check if the external IP exists to use it in the first place
	if externalIP != "" {
		log.Infof("External IP: %s", externalIP)
		result = node.GetGeolocationByIP(obj.(*corev1.Node).Name, externalIP)
	}
	// Check if the internal IP exists and
	// the result of detecting geolocation by external IP is false
	if internalIP != "" && !result {
		log.Infof("Internal IP: %s", internalIP)
		node.GetGeolocationByIP(obj.(*corev1.Node).Name, internalIP)
	}
}
