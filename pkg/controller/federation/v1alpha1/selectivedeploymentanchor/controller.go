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

package selectivedeploymentanchor

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	appsv1alpha2 "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha2"
	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "selectivedeploymentanchor-controller"

// Definitions of the state of the selectivedeploymentanchor resource
const (
	backoffLimit = 3

	successSynced = "Synced"

	messageResourceSynced       = "SelectiveDeploymentAnchor synced successfully"
	messageFedManagerAssigned   = "Federation manager assigned for selective deployment"
	messageNoFeasibleFedManager = "No feasible federation manager found for selective deployment"
	messageResourceDelegated    = "Selective deployment delegated to the responsible cluster(s)"
)

type Node struct {
	Name     string
	Children []*Node
}

// Controller is the controller implementation for SelectiveDeploymentAnchor resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	selectivedeploymentanchorsLister listers.SelectiveDeploymentAnchorLister
	selectivedeploymentanchorsSynced cache.InformerSynced

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
	selectivedeploymentanchorInformer informers.SelectiveDeploymentAnchorInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:                    kubeclientset,
		edgenetclientset:                 edgenetclientset,
		selectivedeploymentanchorsLister: selectivedeploymentanchorInformer.Lister(),
		selectivedeploymentanchorsSynced: selectivedeploymentanchorInformer.Informer().HasSynced,
		workqueue:                        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "SelectiveDeploymentAnchors"),
		recorder:                         recorder,
	}

	klog.Infoln("Setting up event handlers")
	// Set up an event handler for when SelectiveDeploymentAnchor resources change
	selectivedeploymentanchorInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueSelectiveDeploymentAnchor,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueSelectiveDeploymentAnchor(new)
		},
	})

	return controller
}

// Run will set up the event handlers for the types of selectivedeploymentanchor and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting SelectiveDeploymentAnchor controller")

	klog.Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.selectivedeploymentanchorsSynced); !ok {
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
// converge the two. It then updates the Status block of the SelectiveDeploymentAnchor
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	selectivedeploymentanchor, err := c.selectivedeploymentanchorsLister.SelectiveDeploymentAnchors(namespace).Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("selectivedeploymentanchor '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processSelectiveDeploymentAnchor(selectivedeploymentanchor.DeepCopy())
	c.recorder.Event(selectivedeploymentanchor, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueSelectiveDeploymentAnchor takes a SelectiveDeploymentAnchor resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than SelectiveDeploymentAnchor.
func (c *Controller) enqueueSelectiveDeploymentAnchor(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueSelectiveDeploymentAnchorAfter takes a SelectiveDeploymentAnchor resource and converts it into a namespace/name
// string which is then put onto the work queue after the expiry date to be deleted. This method should *not* be
// passed resources of any type other than SelectiveDeploymentAnchor.
func (c *Controller) enqueueSelectiveDeploymentAnchorAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

func (c *Controller) processSelectiveDeploymentAnchor(selectivedeploymentanchorCopy *federationv1alpha1.SelectiveDeploymentAnchor) {
	if exceedsBackoffLimit := selectivedeploymentanchorCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
		return
	}
	multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
	permitted, _, namespaceLabels := multitenancyManager.EligibilityCheck(selectivedeploymentanchorCopy.GetNamespace())
	if permitted {

		switch selectivedeploymentanchorCopy.Status.State {
		case federationv1alpha1.StatusDelegated:
			// There is no reconcile for delegated anchors
			// Watch the delegated anchor
		case federationv1alpha1.StatusAssigned:
			if selectivedeploymentanchorCopy.Spec.FederationManager == nil {
				return
			}
			if selectivedeploymentanchorCopy.Spec.FederationManager.Name != namespaceLabels["edge-net.io/cluster-uid"] {
				var remotekubeclientset *kubernetes.Clientset
				var remoteedgeclientset *clientset.Clientset

				nextFederationManager := selectivedeploymentanchorCopy.Spec.FederationManager.Name
				if len(selectivedeploymentanchorCopy.Spec.FederationManager.Path) != 0 {
					nextFederationManager = selectivedeploymentanchorCopy.Spec.FederationManager.Path[0]
				}
				secretFMAuth, _ := c.kubeclientset.CoreV1().Secrets("edgenet").Get(context.TODO(), "federation", metav1.GetOptions{})
				if nextFederationManager == string(secretFMAuth.Data["cluster-uid"]) {
					config := bootstrap.PrepareRestConfig(string(secretFMAuth.Data["server"]), string(secretFMAuth.Data["token"]), secretFMAuth.Data["ca.crt"])
					remotekubeclientset, _ = bootstrap.CreateKubeClientset(config)
					remoteedgeclientset, _ = bootstrap.CreateEdgeNetClientset(config)
				} else {
					clusterRaw, err := c.edgenetclientset.FederationV1alpha1().Clusters("").List(context.TODO(), metav1.ListOptions{})
					if err != nil {
						return
					}
					match := false
					for _, clusterRow := range clusterRaw.Items {
						if strings.ToLower(clusterRow.Spec.Role) == "federation" && nextFederationManager == clusterRow.Spec.UID {
							match = true
							remoteAuthSecret, _ := c.kubeclientset.CoreV1().Secrets(clusterRow.GetNamespace()).Get(context.TODO(), clusterRow.Spec.SecretName, metav1.GetOptions{})
							config := bootstrap.PrepareRestConfig(string(remoteAuthSecret.Data["server"]), string(remoteAuthSecret.Data["token"]), remoteAuthSecret.Data["ca.crt"])
							remotekubeclientset, _ = bootstrap.CreateKubeClientset(config)
							remoteedgeclientset, _ = bootstrap.CreateEdgeNetClientset(config)
						}
					}
					if !match {
						return
					}
				}

				remoteNamespace := new(corev1.Namespace)
				remoteNamespace.SetName(selectivedeploymentanchorCopy.GetNamespace())
				remotekubeclientset.CoreV1().Namespaces().Create(context.TODO(), remoteNamespace, metav1.CreateOptions{})

				selectiveDeploymentSecret, _ := c.kubeclientset.CoreV1().Secrets(selectivedeploymentanchorCopy.GetNamespace()).Get(context.TODO(), selectivedeploymentanchorCopy.Spec.OriginRef.UID, metav1.GetOptions{})
				remoteSelectiveDeploymentSecret := new(corev1.Secret)
				remoteSelectiveDeploymentSecret.SetName(selectiveDeploymentSecret.GetName())
				remoteSelectiveDeploymentSecret.SetNamespace(selectiveDeploymentSecret.GetNamespace())
				remoteSelectiveDeploymentSecret.Data = selectiveDeploymentSecret.Data
				remotekubeclientset.CoreV1().Secrets(remoteSelectiveDeploymentSecret.GetNamespace()).Create(context.TODO(), remoteSelectiveDeploymentSecret, metav1.CreateOptions{})

				remoteSelectiveDeploymentAnchor := new(federationv1alpha1.SelectiveDeploymentAnchor)
				remoteSelectiveDeploymentAnchor.SetName(selectivedeploymentanchorCopy.GetName())
				remoteSelectiveDeploymentAnchor.SetNamespace(selectivedeploymentanchorCopy.GetNamespace())
				remoteSelectiveDeploymentAnchor.Spec = selectivedeploymentanchorCopy.Spec
				remoteSelectiveDeploymentAnchor.Spec.FederationManager.Path = remoteSelectiveDeploymentAnchor.Spec.FederationManager.Path[1:]
				remoteedgeclientset.FederationV1alpha1().SelectiveDeploymentAnchors(remoteSelectiveDeploymentAnchor.GetNamespace()).Create(context.TODO(), remoteSelectiveDeploymentAnchor, metav1.CreateOptions{})

				c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeNormal, federationv1alpha1.StatusDelegated, messageResourceDelegated)
				selectivedeploymentanchorCopy.Status.State = federationv1alpha1.StatusDelegated
				selectivedeploymentanchorCopy.Status.Message = messageResourceDelegated
				c.updateStatus(context.TODO(), selectivedeploymentanchorCopy)
			} else {
				if len(selectivedeploymentanchorCopy.Spec.WorkloadClusters) == 0 {
					return
				}
				clusterRaw, err := c.edgenetclientset.FederationV1alpha1().Clusters("").List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					return
				}
				for _, clusterRow := range clusterRaw.Items {
					for _, workloadCluster := range selectivedeploymentanchorCopy.Spec.WorkloadClusters {
						if strings.ToLower(clusterRow.Spec.Role) == "workload" && workloadCluster == clusterRow.Spec.UID {
							remoteAuthSecret, _ := c.kubeclientset.CoreV1().Secrets(clusterRow.GetNamespace()).Get(context.TODO(), clusterRow.Spec.SecretName, metav1.GetOptions{})
							config := bootstrap.PrepareRestConfig(clusterRow.Spec.Server, string(remoteAuthSecret.Data["token"]), remoteAuthSecret.Data["ca.crt"])
							remotekubeclientset, _ := bootstrap.CreateKubeClientset(config)
							remoteNamespace := new(corev1.Namespace)
							remoteNamespace.SetName(selectivedeploymentanchorCopy.Spec.OriginRef.Namespace)
							_, err := remotekubeclientset.CoreV1().Namespaces().Create(context.TODO(), remoteNamespace, metav1.CreateOptions{})
							klog.Infoln(err)
							selectiveDeploymentSecret, _ := c.kubeclientset.CoreV1().Secrets(selectivedeploymentanchorCopy.GetNamespace()).Get(context.TODO(), selectivedeploymentanchorCopy.Spec.OriginRef.UID, metav1.GetOptions{})

							config = bootstrap.PrepareRestConfig(string(selectiveDeploymentSecret.Data["server"]), string(selectiveDeploymentSecret.Data["token"]), selectiveDeploymentSecret.Data["ca.crt"])
							originedgeclientset, _ := bootstrap.CreateEdgeNetClientset(config)
							originSelectiveDeployment, err := originedgeclientset.AppsV1alpha2().SelectiveDeployments(selectivedeploymentanchorCopy.Spec.OriginRef.Namespace).Get(context.TODO(), selectivedeploymentanchorCopy.Spec.OriginRef.Name, metav1.GetOptions{})
							klog.Infoln(err)
							remoteSelectiveDeploymentSecret := new(corev1.Secret)
							remoteSelectiveDeploymentSecret.SetName(selectiveDeploymentSecret.GetName())
							remoteSelectiveDeploymentSecret.SetNamespace(originSelectiveDeployment.GetNamespace())
							remoteSelectiveDeploymentSecret.Data = selectiveDeploymentSecret.Data
							_, err = remotekubeclientset.CoreV1().Secrets(remoteSelectiveDeploymentSecret.GetNamespace()).Create(context.TODO(), remoteSelectiveDeploymentSecret, metav1.CreateOptions{})
							klog.Infoln(err)

							remoteSelectiveDeployment := new(appsv1alpha2.SelectiveDeployment)
							remoteSelectiveDeployment.SetName(originSelectiveDeployment.GetName())
							remoteSelectiveDeployment.SetNamespace(originSelectiveDeployment.GetNamespace())
							remoteSelectiveDeployment.SetAnnotations(map[string]string{"edge-net.io/selective-deployment": "follower", "edge-net.io/origin-selective-deployment-uid": selectivedeploymentanchorCopy.Spec.OriginRef.UID})
							remoteSelectiveDeployment.Spec = originSelectiveDeployment.Spec
							// remoteSelectiveDeployment.Spec = selectivedeploymentanchorCopy.Spec.OriginRef.
							config = bootstrap.PrepareRestConfig(clusterRow.Spec.Server, string(remoteAuthSecret.Data["token"]), remoteAuthSecret.Data["ca.crt"])
							remoteedgeclientset, _ := bootstrap.CreateEdgeNetClientset(config)

							_, err = remoteedgeclientset.AppsV1alpha2().SelectiveDeployments(remoteSelectiveDeployment.GetNamespace()).Create(context.TODO(), remoteSelectiveDeployment, metav1.CreateOptions{})
							klog.Infoln(err)
							c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeNormal, federationv1alpha1.StatusDelegated, messageResourceDelegated)
							selectivedeploymentanchorCopy.Status.State = federationv1alpha1.StatusDelegated
							selectivedeploymentanchorCopy.Status.Message = messageResourceDelegated
							c.updateStatus(context.TODO(), selectivedeploymentanchorCopy)
						}
					}
				}
			}
		default:
			if selectivedeploymentanchorCopy.Spec.ClusterAffinity == nil {
				return
			}

			selector, _ := metav1.LabelSelectorAsSelector(selectivedeploymentanchorCopy.Spec.ClusterAffinity)
			if feasibleWorkloadClusters, err := c.getFeasibleChildWorkloadClusters(selector.String(), selectivedeploymentanchorCopy.Spec.ClusterReplicas); err == nil || len(feasibleWorkloadClusters) > 0 {
				selectivedeploymentanchorCopy.Spec.FederationManager = &federationv1alpha1.SelectedFederationManager{Name: namespaceLabels["edge-net.io/cluster-uid"]}
				selectivedeploymentanchorCopy.Spec.WorkloadClusters = append(selectivedeploymentanchorCopy.Spec.WorkloadClusters, feasibleWorkloadClusters...)
				if updatedSDA, err := c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).Update(context.TODO(), selectivedeploymentanchorCopy, metav1.UpdateOptions{}); err == nil {
					c.recorder.Event(updatedSDA, corev1.EventTypeNormal, federationv1alpha1.StatusAssigned, messageFedManagerAssigned)
					updatedSDA.Status.State = federationv1alpha1.StatusAssigned
					updatedSDA.Status.Message = messageFedManagerAssigned
					c.updateStatus(context.TODO(), updatedSDA)
				}
				return
			}
			if selectivedeploymentanchorCopy.Spec.FederationManager == nil {
				feasibleFederationManager, path, ok := c.scanFederationManagers(namespaceLabels["edge-net.io/cluster-uid"], selector.String())
				if !ok {
					c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageNoFeasibleFedManager)
					selectivedeploymentanchorCopy.Status.State = federationv1alpha1.StatusFailed
					selectivedeploymentanchorCopy.Status.Message = messageNoFeasibleFedManager
					c.updateStatus(context.TODO(), selectivedeploymentanchorCopy)
					return
				}

				selectivedeploymentanchorCopy.Spec.FederationManager.Name = feasibleFederationManager
				selectivedeploymentanchorCopy.Spec.FederationManager.Path = path
				c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).Update(context.TODO(), selectivedeploymentanchorCopy, metav1.UpdateOptions{})
				if updatedSDA, err := c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).Update(context.TODO(), selectivedeploymentanchorCopy, metav1.UpdateOptions{}); err == nil {
					c.recorder.Event(updatedSDA, corev1.EventTypeNormal, federationv1alpha1.StatusAssigned, messageFedManagerAssigned)
					updatedSDA.Status.State = federationv1alpha1.StatusAssigned
					updatedSDA.Status.Message = messageFedManagerAssigned
					c.updateStatus(context.TODO(), updatedSDA)
				}
				return
			}

			c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeNormal, federationv1alpha1.StatusAssigned, messageFedManagerAssigned)
			selectivedeploymentanchorCopy.Status.State = federationv1alpha1.StatusAssigned
			selectivedeploymentanchorCopy.Status.Message = messageFedManagerAssigned
			c.updateStatus(context.TODO(), selectivedeploymentanchorCopy)
		}
	} else {
		c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).Delete(context.TODO(), selectivedeploymentanchorCopy.GetName(), metav1.DeleteOptions{})
	}
}

// updateStatus calls the API to update the selectivedeploymentanchor status.
func (c *Controller) updateStatus(ctx context.Context, selectivedeploymentanchorCopy *federationv1alpha1.SelectiveDeploymentAnchor) {
	if selectivedeploymentanchorCopy.Status.State == federationv1alpha1.StatusFailed {
		selectivedeploymentanchorCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).UpdateStatus(ctx, selectivedeploymentanchorCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}

// getFeasibleChildWorkloadClusters returns the list of feasible child workload clusters that are managed by the current federation manager
func (c *Controller) getFeasibleChildWorkloadClusters(labelSelector string, clusterReplicaCount int) ([]string, error) {
	clusterRaw, err := c.edgenetclientset.FederationV1alpha1().Clusters("").List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
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
	// keys is a list of manager names to be used while sort federation managers by their scores
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
			switch managedCluster.ResourceAvailability {
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
	lca := LowestCommonAncestor(rootNode, sourceNode, destinationNode)
	if lca == sourceNode {
		shortestPath := ShortestPathFromParentToChild(sourceNode, destinationNode)
		for _, node := range shortestPath {
			path = append(path, node.Name)
		}
	} else if lca == destinationNode {
		shortestPath := ShortestPathFromParentToChild(sourceNode, destinationNode)
		for _, node := range shortestPath {
			path = append(path, node.Name)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(path)))
	} else {
		shortestPath := ShortestPathFromParentToChild(sourceNode, lca)
		shortestPath = append(shortestPath, ShortestPathFromParentToChild(lca, destinationNode)...)
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

// ShortestPathFromParentToChild returns the shortest path from the parent to the child
func ShortestPathFromParentToChild(start, end *Node) []*Node {
	if start == end {
		return []*Node{start}
	}
	var path []*Node
	for _, child := range start.Children {
		result := ShortestPathFromParentToChild(child, end)
		if result != nil {
			path = append(path, result...)
		}
	}
	if len(path) > 0 {
		return append([]*Node{start}, path...)
	}
	return nil
}

// LowestCommonAncestor returns the lowest common ancestor of the two nodes
func LowestCommonAncestor(root *Node, p, q *Node) *Node {
	if root == nil || root == p || root == q {
		return root
	}

	var lca *Node
	for _, child := range root.Children {
		result := LowestCommonAncestor(child, p, q)
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
