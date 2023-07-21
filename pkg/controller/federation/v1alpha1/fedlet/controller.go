package fedlet

import (
	"context"
	"fmt"
	"reflect"
	"time"

	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/multiprovider"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const controllerAgentName = "fedlet-controller"

// Controller is the controller implementation
type Controller struct {
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

	maxmindURL        string
	maxmindAccountID  string
	maxmindLicenseKey string
}

// NewController returns a new controller
func NewController(
	kubeclientset kubernetes.Interface,
	informer coreinformers.NodeInformer,
) *Controller {
	// Create event broadcaster
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	klog.V(4).Infoln("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events(metav1.NamespaceAll)})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset: kubeclientset,
		lister:        informer.Lister(),
		synced:        informer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "fedlet"),
		recorder:      recorder,
	}

	klog.Infoln("Setting up event handlers")

	// Event handlers deal with events of resources. In here, we take into consideration of added and updated nodes.
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNode,
		UpdateFunc: func(oldObj, newObj interface{}) {
			updated := multiprovider.CompareAvailableResources(oldObj.(*corev1.Node), newObj.(*corev1.Node))
			if updated {
				controller.enqueueNode(newObj)
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

	klog.V(4).Infoln("Starting FedLet Controller")

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

	_, err = c.lister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("fedlet '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	klog.V(4).Infof("processNextItem: object created/updated detected: %s", key)
	c.updateClusterResourceStatus()

	return nil
}

func (c *Controller) enqueueNode(obj interface{}) {
	// Put the resource object into a key
	var key string
	var err error

	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}

// updateClusterResourceStatus checks the status of all nodes in the cluster and updates the cluster resource status
func (c *Controller) updateClusterResourceStatus() {
	// This secret provides necessary info for the cluster to access its federation manager.
	// If it is missing or a remote EdgeNet clientset cannot be created using it, no need to move onto the code below.
	fedmanagerSecret, err := c.kubeclientset.CoreV1().Secrets("edgenet").Get(context.TODO(), "federation", metav1.GetOptions{})
	if err != nil {
		return
	}
	config := bootstrap.PrepareRestConfig(string(fedmanagerSecret.Data["server"]), string(fedmanagerSecret.Data["token"]), fedmanagerSecret.Data["ca.crt"])
	remoteedgeclientset, err := bootstrap.CreateEdgeNetClientset(config)
	if err != nil {
		return
	}
	// bundledAllocatableResourcesList is a list to store the allocatable resources of each node that is open to federated workloads.
	// It groups the nodes that have the same allocatable resources together.
	var bundledAllocatableResourcesList []federationv1alpha1.BundledAllocatableResources
	// overallAllocatableResources calculates the total allocatable resources of all nodes in the cluster
	overallAllocatableResources := make(corev1.ResourceList)
	// overallCapacityResources calculates the total capacity resources of all nodes in the cluster
	overallCapacityResources := make(corev1.ResourceList)
	// Below is the logic to calculate the total allocatable resources and total capacity resources of all nodes in the cluster.
	// It also prepares the bundledAllocatableResourcesList.
	nodeRaw, _ := c.kubeclientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/access=public,edge-net.io/access-scope=federation"})
	if len(nodeRaw.Items) == 0 {
		return
	}
	for _, nodeRow := range nodeRaw.Items {
		for key, value := range nodeRow.Status.Allocatable {
			if allocatableQuantity, ok := overallAllocatableResources[key]; ok {
				allocatableQuantity.Add(value)
				overallAllocatableResources[key] = *resource.NewQuantity(allocatableQuantity.Value(), value.Format)
			} else {
				overallAllocatableResources[key] = value
			}
		}
		if len(bundledAllocatableResourcesList) > 0 {
			for key, bundledAllocatableResources := range bundledAllocatableResourcesList {
				if reflect.DeepEqual(bundledAllocatableResources, nodeRow.Status.Allocatable) {
					bundledAllocatableResourcesList[key].Count++
					break
				} else {
					bundledAllocatableResourcesList = append(bundledAllocatableResourcesList, federationv1alpha1.BundledAllocatableResources{Count: 1, ResourceList: nodeRow.Status.Allocatable})
				}
			}
		} else {
			bundledAllocatableResourcesList = append(bundledAllocatableResourcesList, federationv1alpha1.BundledAllocatableResources{Count: 1, ResourceList: nodeRow.Status.Allocatable})
		}
		for key, value := range nodeRow.Status.Capacity {
			if capacityQuantity, ok := overallCapacityResources[key]; ok {
				capacityQuantity.Add(value)
				overallCapacityResources[key] = *resource.NewQuantity(capacityQuantity.Value(), value.Format)
			} else {
				overallCapacityResources[key] = value
			}
		}
	}
	// There are four levels of relative resource availability: abundant, normal, limited, scarce.
	// The relative resource availability is calculated based on the ratio of consumed resources to capacity resources.
	relativeResourceAvailability := []string{federationv1alpha1.AbundantResources, federationv1alpha1.NormalResources, federationv1alpha1.LimitedResources, federationv1alpha1.ScarceResources}
	key := 0
	if len(overallAllocatableResources) > 0 && len(overallCapacityResources) > 0 {
		for resourceName, capacityQuantity := range overallCapacityResources {
			if allocatableQuantity, ok := overallAllocatableResources[resourceName]; ok {
				if capacityQuantity.Value() == 0 {
					continue
				}
				// The ratio of consumed resources to capacity resources
				ratio := float64((capacityQuantity.Value() - allocatableQuantity.Value()) / capacityQuantity.Value())
				if ratio > 0.5 && key < 1 {
					key = 0
				} else if ratio > 0.35 && key < 2 {
					key = 1
				} else if ratio > 0.15 && key < 3 {
					key = 2
				} else {
					key = 3
				}
			}
		}
	} else {
		key = 3
	}
	clusterStatus := relativeResourceAvailability[key]
	// Update the status of associated object in the federation manager
	if cluster, err := remoteedgeclientset.FederationV1alpha1().Clusters(string(fedmanagerSecret.Data["assigned-cluster-namespace"])).Get(context.TODO(), string(fedmanagerSecret.Data["assigned-cluster-name"]), metav1.GetOptions{}); err == nil {
		cluster.Status.RelativeResourceAvailability = clusterStatus
		cluster.Status.AllocatableResources = bundledAllocatableResourcesList
		if _, err := remoteedgeclientset.FederationV1alpha1().Clusters(string(fedmanagerSecret.Data["assigned-cluster-namespace"])).UpdateStatus(context.TODO(), cluster, metav1.UpdateOptions{}); err != nil {
			klog.Infoln(err)
		}
	} else {
		klog.Infoln(err)
	}
}
