package clusterlabeler

import (
	"context"
	"fmt"
	"time"

	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/federation/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/multiprovider"
	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "clusterlabeler-controller"

const (
	messageLabelUpdateFailed  = "Failed to update cluster labels"
	messageFetchGeoInfoFailed = "Failed to fetch geolocation information"
	messageGeoLabelsAttached  = "Geolocation labels attached"
)

// Controller is the controller implementation
type Controller struct {
	kubeclientset    kubernetes.Interface
	edgenetclientset clientset.Interface

	clustersLister listers.ClusterLister
	clustersSynced cache.InformerSynced

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
	informer informers.ClusterInformer,
	maxmindURL string,
	maxmindAccountID string,
	maxmindLicenseKey string,
) *Controller {
	// Create event broadcaster
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	klog.Infoln("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:     kubeclientset,
		edgenetclientset:  edgenetclientset,
		clustersLister:    informer.Lister(),
		clustersSynced:    informer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "clusterlabeler"),
		recorder:          recorder,
		maxmindURL:        maxmindURL,
		maxmindAccountID:  maxmindAccountID,
		maxmindLicenseKey: maxmindLicenseKey,
	}

	klog.Infoln("Setting up event handlers")

	// Event handlers deal with events of resources. In here, we take into consideration of adding and updating clusters.
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueclusterlabeler,
		UpdateFunc: func(oldObj, newObj interface{}) {
			if oldObj.(*federationv1alpha1.Cluster).Spec.Server != newObj.(*federationv1alpha1.Cluster).Spec.Server {
				controller.enqueueclusterlabeler(newObj)
			}
		},
	})

	return controller
}

// Run will set up the event handlers for the types of clusters, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting Cluster Labeler Controller")

	klog.Infoln("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh,
		c.clustersSynced); !ok {
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
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	cluster, err := c.clustersLister.Clusters(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("clusterlabeler '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	klog.Infof("processNextItem: object created/updated detected: %s", key)
	c.setClusterGeolocation(cluster.DeepCopy())

	return nil
}

func (c *Controller) enqueueclusterlabeler(obj interface{}) {
	// Put the resource object into a key
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) setClusterGeolocation(clusterCopy *federationv1alpha1.Cluster) {
	klog.Infoln("Handler.ObjectCreated")
	if clusterCopy.Status.State != federationv1alpha1.StatusReady {
		return
	}
	multiproviderManager := multiprovider.NewManager(c.kubeclientset, nil, c.edgenetclientset, nil)
	klog.Infof("IP: %s", clusterCopy.Spec.Server)
	if geoLabels, ok := multiproviderManager.GetGeolocationLabelsByIP(
		c.maxmindURL,
		c.maxmindAccountID,
		c.maxmindLicenseKey,
		clusterCopy.Spec.Server,
		false); ok {
		clusterLabels := clusterCopy.GetLabels()
		if clusterLabels == nil {
			clusterLabels = make(map[string]string)
		}
		for key, value := range geoLabels {
			clusterLabels[key] = value
		}
		clusterCopy.SetLabels(clusterLabels)
		if _, err := c.edgenetclientset.FederationV1alpha1().Clusters(clusterCopy.GetNamespace()).Update(context.TODO(), clusterCopy, metav1.UpdateOptions{}); err != nil {
			c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageLabelUpdateFailed)
			c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageLabelUpdateFailed)
			return
		}
	} else {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageFetchGeoInfoFailed)
		c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageFetchGeoInfoFailed)
		return
	}
	c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusReady, messageGeoLabelsAttached)
}

// updateStatus calls the API to update the cluster status.
func (c *Controller) updateStatus(ctx context.Context, clusterCopy *federationv1alpha1.Cluster, state, message string) {
	clusterCopy.Status.State = state
	clusterCopy.Status.Message = message
	if clusterCopy.Status.State == federationv1alpha1.StatusFailed {
		clusterCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.FederationV1alpha1().Clusters(clusterCopy.GetNamespace()).UpdateStatus(ctx, clusterCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
