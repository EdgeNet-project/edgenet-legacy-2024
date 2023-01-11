package scheduler

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/federation/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/federation/v1alpha1"
	multitenancy "github.com/EdgeNet-project/edgenet/pkg/multitenancy"
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

const controllerAgentName = "scheduler-controller"

// Definitions of the state of the cluster resource
const (
	backoffLimit = 3

	successSynced = "Synced"

	messageResourceSynced = "Cluster synced successfully"
	messageUpdateFailed   = "Failed to update following the scheduling decision"
)

type Node struct {
	Name     string
	Children []*Node
}

// Controller is the controller implementation
type Controller struct {
	kubeclientset    kubernetes.Interface
	edgenetclientset clientset.Interface

	sdasLister listers.SelectiveDeploymentAnchorLister
	sdasSynced cache.InformerSynced

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
	sdaInformer informers.SelectiveDeploymentAnchorInformer,
) *Controller {
	// Create event broadcaster
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	klog.V(4).Infoln("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events(metav1.NamespaceAll)})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:    kubeclientset,
		edgenetclientset: edgenetclientset,
		sdasLister:       sdaInformer.Lister(),
		sdasSynced:       sdaInformer.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "scheduler"),
		recorder:         recorder,
	}

	klog.Infoln("Setting up event handlers")

	// Event handlers deal with events of resources. In here, we take into consideration of added and updated nodes.
	sdaInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueSelectiveDeploymentAnchor,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueSelectiveDeploymentAnchor(new)
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

	klog.V(4).Infoln("Starting Scheduler Controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh,
		c.sdasSynced); !ok {
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
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	selectivedeploymentanchor, err := c.sdasLister.SelectiveDeploymentAnchors(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("scheduler '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	klog.V(4).Infof("processNextItem: object created/updated detected: %s", key)

	c.processSelectiveDeploymentAnchor(selectivedeploymentanchor.DeepCopy())
	c.recorder.Event(selectivedeploymentanchor, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

func (c *Controller) enqueueSelectiveDeploymentAnchor(obj interface{}) {
	// Put the resource object into a key
	var key string
	var err error

	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}

// processSelectiveDeploymentAnchor is the main logic of the scheduler controller that runs at the federation level.
func (c *Controller) processSelectiveDeploymentAnchor(selectivedeploymentanchorCopy *federationv1alpha1.SelectiveDeploymentAnchor) { // Crashloop backoff limit to avoid endless loop
	if exceedsBackoffLimit := selectivedeploymentanchorCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
		return
	}
	if selectivedeploymentanchorCopy.Spec.FederationManager == nil || selectivedeploymentanchorCopy.Spec.WorkloadClusters == nil || len(selectivedeploymentanchorCopy.Spec.WorkloadClusters) == 0 {
		multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
		permitted, _, namespaceLabels := multitenancyManager.EligibilityCheck(selectivedeploymentanchorCopy.GetNamespace())
		if permitted {
			selector, _ := metav1.LabelSelectorAsSelector(selectivedeploymentanchorCopy.Spec.ClusterAffinity)
			// First of all, to make a faster scheduling decision, we check if the cluster affinity is satisfied by one of the workload clusters owned by the federation manager
			if selectivedeploymentanchorCopy.Spec.FederationManager == nil || selectivedeploymentanchorCopy.Spec.FederationManager.Name == namespaceLabels["edge-net.io/cluster-uid"] {
				if feasibleWorkloadClusters, err := c.getFeasibleChildWorkloadClusters(selector.String(), selectivedeploymentanchorCopy.Spec.ClusterReplicas); err == nil || len(feasibleWorkloadClusters) > 0 {
					// If the cluster affinity is satisfied by one of the workload clusters owned by the federation manager,
					// we assign the current federation manager and the workload cluster(s) to the SDA.
					selectivedeploymentanchorCopy.Spec.FederationManager = &federationv1alpha1.SelectedFederationManager{Name: namespaceLabels["edge-net.io/cluster-uid"]}
					selectivedeploymentanchorCopy.Spec.WorkloadClusters = append(selectivedeploymentanchorCopy.Spec.WorkloadClusters, feasibleWorkloadClusters...)
					if _, err := c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).Update(context.TODO(), selectivedeploymentanchorCopy, metav1.UpdateOptions{}); err != nil {
						c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeWarning, federationv1alpha1.StatusAssigned, messageUpdateFailed)
						c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusAssigned, messageUpdateFailed)
					}
					return
				}
			}
			// If no workload cluster satisfies the cluster affinity, we scan all manager clusters in the federation hierarchy to see if any of them can satisfy the affinity
			if selectivedeploymentanchorCopy.Spec.FederationManager == nil {
				feasibleFederationManager, path, ok := c.scanFederationManagers(namespaceLabels["edge-net.io/cluster-uid"], selector.String())
				if !ok {
					c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageUpdateFailed)
					c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusFailed, messageUpdateFailed)
					return
				}
				selectivedeploymentanchorCopy.Spec.FederationManager.Name = feasibleFederationManager
				selectivedeploymentanchorCopy.Spec.FederationManager.Path = path
				c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).Update(context.TODO(), selectivedeploymentanchorCopy, metav1.UpdateOptions{})
				if _, err := c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).Update(context.TODO(), selectivedeploymentanchorCopy, metav1.UpdateOptions{}); err != nil {
					c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeWarning, federationv1alpha1.StatusAssigned, messageUpdateFailed)
					c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusAssigned, messageUpdateFailed)
				}
			}
		}
	}
}

// getFeasibleChildWorkloadClusters returns the list of feasible child workload clusters that are managed by the current federation manager
func (c *Controller) getFeasibleChildWorkloadClusters(labelSelector string, clusterReplicaCount int) ([]string, error) {
	clusterRaw, err := c.edgenetclientset.FederationV1alpha1().Clusters(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		klog.Infoln(err)
		return nil, err
	}
	var feasibleWorkloadClusters []string
	if len(clusterRaw.Items) != 0 {
		for _, clusterRow := range clusterRaw.Items {
			if strings.ToLower(clusterRow.Spec.Role) == "workload" {
				feasibleWorkloadClusters = append(feasibleWorkloadClusters, clusterRow.Spec.UID)
			}
		}
	}
	var pickedClusterList []string
	if len(feasibleWorkloadClusters) != 0 {
		for i := 0; i < clusterReplicaCount; i++ {
			rand.Seed(time.Now().UnixNano())
			randomSelect := rand.Intn(len(feasibleWorkloadClusters))
			pickedClusterList = append(pickedClusterList, feasibleWorkloadClusters[randomSelect])
			feasibleWorkloadClusters[randomSelect] = feasibleWorkloadClusters[len(feasibleWorkloadClusters)-1]
			feasibleWorkloadClusters = feasibleWorkloadClusters[:len(feasibleWorkloadClusters)-1]
		}
	}
	return pickedClusterList, nil
}

// scanFederationManagers returns the federation manager with the highest score and the shortest path to it
func (c *Controller) scanFederationManagers(currentFederationManager string, labelSelector string) (string, []string, bool) {
	// managerList is a struct to store the parent and children of a manager and its score
	type managerList struct {
		parent   string
		children []string
		score    int
	}
	federationManagers := make(map[string]managerList)
	// keys is a list of manager names to be used while sorting federation managers by their scores
	var keys []string
	// Get the list of manager caches that match the cluster affinity provided in the spec
	// Thus, we have a list of managers that are eligible to be selected
	managerCachesRaw, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil || len(managerCachesRaw.Items) == 0 {
		return "", nil, false
	}
	// Iterate over the list of manager caches and calculate the score of each manager
	for _, managerCacheRow := range managerCachesRaw.Items {
		score := 0
		for _, managedCluster := range managerCacheRow.Spec.Clusters {
			switch managedCluster.RelativeResourceAvailability {
			case "Abundance":
				score += 3
			case "Normal":
				score += 2
			case "Limited":
				score += 1
			default:
				continue
			}
		}
		keys = append(keys, managerCacheRow.GetName())
		federationManagers[managerCacheRow.GetName()] = managerList{parent: managerCacheRow.Spec.Hierarchy.Parent, children: managerCacheRow.Spec.Hierarchy.Children, score: score}
	}
	// Sort the federation managers by their scores
	sort.SliceStable(keys, func(i, j int) bool {
		return federationManagers[keys[i]].score > federationManagers[keys[j]].score
	})
	// Get the tree structure of the federation managers
	tree, rootNode := c.createTree()
	if tree == nil || rootNode == nil {
		return "", nil, false
	}
	sourceNode := tree[currentFederationManager]
	// Pick the federation manager with the highest score
	destinationNode := tree[keys[0]]
	// Get the shortest path from the current federation manager to the selected federation manager
	var path []string
	// Look for the lowest common ancestor of the current federation manager and the selected federation manager
	lca := lowestCommonAncestor(rootNode, sourceNode, destinationNode)
	if lca == sourceNode {
		shortestPath := shortestPathFromParentToChild(sourceNode, destinationNode)
		for _, node := range shortestPath {
			path = append(path, node.Name)
		}
	} else if lca == destinationNode {
		shortestPath := shortestPathFromParentToChild(sourceNode, destinationNode)
		for _, node := range shortestPath {
			path = append(path, node.Name)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(path)))
	} else {
		shortestPath := shortestPathFromParentToChild(sourceNode, lca)
		shortestPath = append(shortestPath, shortestPathFromParentToChild(lca, destinationNode)...)
		for _, node := range shortestPath {
			path = append(path, node.Name)
		}
	}
	return destinationNode.Name, path, true
}

// createTree creates a tree structure of the federation managers
func (c *Controller) createTree() (map[string]*Node, *Node) {
	managerCachesRaw, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil
	}
	var rootFederationManager *Node
	federationManagerTree := make(map[string]*Node)
	for _, managerCacheRow := range managerCachesRaw.Items {
		node, ok := federationManagerTree[managerCacheRow.GetName()]
		if !ok {
			node = newNode(managerCacheRow.GetName())
		}

		if managerCacheRow.Spec.Hierarchy.Level == 0 {
			rootFederationManager = node
		} else {
			if parentNode, ok := federationManagerTree[managerCacheRow.Spec.Hierarchy.Parent]; ok {
				parentNode.Children = append(parentNode.Children, node)
			} else {
				parentNode = newNode(managerCacheRow.Spec.Hierarchy.Parent)
				federationManagerTree[managerCacheRow.Spec.Hierarchy.Parent] = parentNode
			}
		}
		federationManagerTree[managerCacheRow.GetName()] = node
	}
	return federationManagerTree, rootFederationManager
}

// newNode returns a new node
func newNode(name string) *Node {
	return &Node{Name: name, Children: []*Node{}}
}

// shortestPathFromParentToChild returns the shortest path from the parent to the child
func shortestPathFromParentToChild(start, end *Node) []*Node {
	if start == end {
		return []*Node{start}
	}
	var path []*Node
	for _, child := range start.Children {
		result := shortestPathFromParentToChild(child, end)
		if result != nil {
			path = append(path, result...)
		}
	}
	if len(path) > 0 {
		return append([]*Node{start}, path...)
	}
	return nil
}

// lowestCommonAncestor returns the lowest common ancestor of the two nodes
func lowestCommonAncestor(root *Node, p, q *Node) *Node {
	if root == nil || root == p || root == q {
		return root
	}
	var lca *Node
	for _, child := range root.Children {
		result := lowestCommonAncestor(child, p, q)
		if result == nil {
			continue
		}
		if lca == nil {
			lca = result
		} else {
			return root
		}
	}
	return lca
}

// updateStatus calls the API to update the selectivedeploymentanchor status.
func (c *Controller) updateStatus(ctx context.Context, selectivedeploymentanchorCopy *federationv1alpha1.SelectiveDeploymentAnchor, state, message string) {
	selectivedeploymentanchorCopy.Status.State = state
	selectivedeploymentanchorCopy.Status.Message = message
	if selectivedeploymentanchorCopy.Status.State == federationv1alpha1.StatusFailed {
		selectivedeploymentanchorCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).UpdateStatus(ctx, selectivedeploymentanchorCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}