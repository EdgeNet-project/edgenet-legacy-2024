package nodelabeler

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/node"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "nodelabeler-controller"

const (
	// successSynced         = "Synced"
	// messageResourceSynced = "Node Contribution synced successfully"
	// setupProcedure        = "Setup"
	// messageSetupPhase     = "Setup process commenced"
	// messageDoneDNS        = "DNS record configured"
	// messageDoneSSH        = "SSH connection established"
	// messageDoneKubeadm    = "Bootstrap token created and join command has been invoked"
	// messageDonePatch      = "Node scheduling updated"
	// messageTimeout        = "Procedure terminated due to timeout"
	// messageEnd            = "Procedure finished"
	inqueue = "In Queue"
	// inprogress            = "In Progress"
	// failure               = "Failure"
	// incomplete            = "Halting"
	// success               = "Successful"
	// create                = "create"
	// update                = "update"
	// delete                = "delete"
	// trueStr               = "True"
	// falseStr              = "False"
	// unknownStr            = "Unknown"
)

// The main structure of controller

type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	nodesLister corelisters.NodeLister
	nodesSynced cache.InformerSynced

	// See how to create this using lister-go
	nodelabelerLister corelisters.NodeLabelerLister
	nodelabelerSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder  record.EventRecorder
	publicKey ssh.Signer
}

// type controller struct {
// 	logger    *log.Entry
// 	clientset kubernetes.Interface
// 	queue     workqueue.RateLimitingInterface // YES
// 	informer  cache.SharedIndexInformer
// 	handler   HandlerInterface
// }

// NewController returns a new controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	nodeInformer coreinformers.NodeInformer,
	nodeLabelerInformer informers.NodeLabelerInformer) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	// Get the SSH Private Key of the control plane node
	key, err := ioutil.ReadFile("../../.ssh/id_rsa")
	if err != nil {
		klog.V(4).Info(err.Error())
		panic(err.Error())
	}

	publicKey, err := ssh.ParsePrivateKey(key)
	if err != nil {
		klog.V(4).Info(err.Error())
		panic(err.Error())
	}

	controller := &Controller{
		kubeclientset:     kubeclientset,
		edgenetclientset:  edgenetclientset,
		nodesLister:       nodeInformer.Lister(),
		nodesSynced:       nodeInformer.Informer().HasSynced,
		nodelabelerLister: nodeLabelerInformer.Lister(),
		nodelabelerSynced: nodeLabelerInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NodeContributions"),
		recorder:          recorder,
		publicKey:         publicKey,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	nodeLabelerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNodeContribution,
		UpdateFunc: func(old, new interface{}) {
			newNodeContribution := new.(*corev1alpha.NodeContribution)
			oldNodeContribution := old.(*corev1alpha.NodeContribution)
			if reflect.DeepEqual(newNodeContribution.Spec, oldNodeContribution.Spec) && (newNodeContribution.Status.State != inqueue) {
				return
			}
			controller.enqueueNodeContribution(new)
		},
	})

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newNode := new.(*corev1.Node)
			oldNode := old.(*corev1.Node)
			if newNode.ResourceVersion == oldNode.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
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
		c.nodelabelerSynced,
		c.nodesSynced); !ok {
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
	// I AM NOT SURE HOW TO FILL THIS
	return nil
}

func (c *Controller) handleObject(obj interface{}) {
	// I AM NOT SURE HOW TO FILL THIS
}

func (c *Controller) enqueueNodeContribution(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// Start function is entry point of the controller
// func Start(kubernetes kubernetes.Interface) {
// 	clientset := kubernetes

// 	// Create the shared informer to list and watch node resources
// 	informer := cache.NewSharedIndexInformer(
// 		&cache.ListWatch{
// 			// The main purpose of listing is to attach geo labels to whole nodes at the beginning
// 			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
// 				return clientset.CoreV1().Nodes().List(context.TODO(), options)
// 			},
// 			// This function watches all changes/updates of nodes
// 			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
// 				return clientset.CoreV1().Nodes().Watch(context.TODO(), options)
// 			},
// 		},
// 		&core_v1.Node{},
// 		0,
// 		cache.Indexers{},
// 	)
// 	// Create a work queue which contains a key of the resource to be handled by the handler
// 	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
// 	// Event handlers deal with events of resources. In here, we take into consideration of adding and updating nodes.
// 	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		AddFunc: func(obj interface{}) {
// 			// Put the resource object into a key
// 			key, err := cache.MetaNamespaceKeyFunc(obj)
// 			log.Infof("Add node detected: %s", key)
// 			if err == nil {
// 				// Add the key to the queue
// 				queue.Add(key)
// 			}
// 		},
// 		UpdateFunc: func(oldObj, newObj interface{}) {
// 			updated := node.CompareIPAddresses(oldObj.(*core_v1.Node), newObj.(*core_v1.Node))
// 			if updated {
// 				key, err := cache.MetaNamespaceKeyFunc(newObj)
// 				log.Infof("Update node detected: %s", key)
// 				if err == nil {
// 					queue.Add(key)
// 				}
// 			}
// 		},
// 	})
// 	controller := controller{
// 		logger:    log.NewEntry(log.New()),
// 		clientset: clientset,
// 		informer:  informer,
// 		queue:     queue,
// 		handler:   &Handler{},
// 	}

// 	// A channel to terminate elegantly
// 	stopCh := make(chan struct{})
// 	defer close(stopCh)
// 	// Run the controller loop as a background task to start processing resources
// 	go controller.run(stopCh, clientset)
// 	// A channel to observe OS signals for smooth shut down
// 	sigTerm := make(chan os.Signal, 1)
// 	signal.Notify(sigTerm, syscall.SIGTERM)
// 	signal.Notify(sigTerm, syscall.SIGINT)
// 	<-sigTerm
// }

// Run starts the controller loop
// func (c *controller) run(stopCh <-chan struct{}, clientset kubernetes.Interface) {
// 	// A Go panic which includes logging and terminating
// 	defer utilruntime.HandleCrash()
// 	// Shutdown after all goroutines have done
// 	defer c.queue.ShutDown()
// 	c.logger.Info("run: initiating")
// 	c.handler.Init(clientset)
// 	// Run the informer to list and watch resources
// 	go c.informer.Run(stopCh)

// 	// Synchronization to settle resources one
// 	if !cache.WaitForCacheSync(stopCh, c.hasSynced) {
// 		utilruntime.HandleError(fmt.Errorf("Error syncing cache"))
// 		return
// 	}
// 	c.logger.Info("run: cache sync complete")
// 	// Operate the runWorker
// 	wait.Until(c.runWorker, time.Second, stopCh)
// }

// To link the informer's HasSynced method to the Controller interface
func (c *controller) hasSynced() bool {
	return c.informer.HasSynced()
}

// To process new objects added to the queue
func (c *controller) runWorker() {
	log.Info("runWorker: starting")
	// Run processNextItem for all the changes
	for c.processNextItem() {
		log.Info("runWorker: processing next item")
	}

	log.Info("runWorker: completed")
}

// This function deals with the queue and sends each item in it to the specified handler to be processed.
func (c *controller) processNextItem() bool {
	log.Info("processNextItem: start")
	// Fetch the next item of the queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	// Get the key string
	keyRaw := key.(string)
	// Use the string key to get the object from the indexer
	item, exists, err := c.informer.GetIndexer().GetByKey(keyRaw)
	if err != nil {
		if c.queue.NumRequeues(key) < 3 {
			c.logger.Errorf("processNextItem: Failed fetching item with key %s, error is %v, retrying...", key, err)
			c.queue.AddRateLimited(key)
		} else {
			c.logger.Errorf("processNextItem: Failed fetching item with key %s, error is %v, no more retries", key, err)
			c.queue.Forget(key)
			utilruntime.HandleError(err)
		}
	}

	if exists {
		c.logger.Infof("processNextItem: object created/updated detected: %s", keyRaw)
		c.handler.SetNodeGeolocation(item)
		c.queue.Forget(key)
	}
	return true
}
