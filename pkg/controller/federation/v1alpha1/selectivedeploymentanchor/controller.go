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
	messageDelegationComplete   = "Selective deployment delegated to the responsible clusters"
	messageResourceDelegated    = "Selective deployment delegated to the responsible cluster"
	messagePending              = "Selective deployment anchor awaits for the scheduling decision to be made"
	messageFedManagerMissing    = "Federation manager is not assigned yet"
	messagePathEmpty            = "Next federation manager to be appointed is missing"
	messageDeploymentFailed     = "Selective deployment cannot be made in the workload cluster(s)"
	messageDelegationFailed     = "Anchor cannot be delegated to the next federation manager"
)

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
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events(metav1.NamespaceAll)})
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
	// Crashloop backoff limit to avoid endless loop
	if exceedsBackoffLimit := selectivedeploymentanchorCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
		// TODO: If it exceeds the limit, run a cleanup function
		// c.cleanup(selectivedeploymentanchorCopy)
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
			// In this state, at least a federation manager should already be assigned
			if selectivedeploymentanchorCopy.Spec.FederationManager == nil {
				c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageFedManagerMissing)
				c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusFailed, messageFedManagerMissing)
				return
			}
			// If the assigned federation manager is the current cluster, then make the selective deployment
			if selectivedeploymentanchorCopy.Spec.FederationManager.Name == namespaceLabels["edge-net.io/cluster-uid"] {
				if ok := c.makeSelectiveDeployment(selectivedeploymentanchorCopy); !ok {
					c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusFailed, messageDeploymentFailed)
				} else {
					c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusDelegated, messageDelegationComplete)
				}
				return
			}
			// If the assigned federation manager is not the current cluster, then delegate the job to the following federation manager.
			// FedScheduler forms a path to follow and puts this path in anchor in such a case.
			// Path being empty at this point means that the selective deployment is failed.
			sdaLabels := selectivedeploymentanchorCopy.GetLabels()
			if len(selectivedeploymentanchorCopy.Spec.FederationManager.Path) == 0 && (selectivedeploymentanchorCopy.Spec.FederationUID != nil && sdaLabels["edge-net.io/origin-federation-uid"] == *selectivedeploymentanchorCopy.Spec.FederationUID) {
				c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusFailed, messagePathEmpty)
				return
			}
			if ok := c.conveySelectiveDeploymentAnchor(selectivedeploymentanchorCopy); !ok {
				c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageDelegationFailed)
				c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusFailed, messageDelegationFailed)
				return
			}
			c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeNormal, federationv1alpha1.StatusDelegated, messageDelegationComplete)
			c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusDelegated, messageDelegationComplete)
		default:
			if selectivedeploymentanchorCopy.Spec.FederationManager != nil {
				c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeNormal, federationv1alpha1.StatusAssigned, messageFedManagerAssigned)
				c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusAssigned, messageFedManagerAssigned)
				return
			}
			// This goroutine is here to grant a privilege to the federation scheduler to manipulate the object.
			// The goal is to avoid concurrency issues that delay scheduling decisions.
			go func() {
				time.Sleep(5 * time.Second)
				klog.Infoln(selectivedeploymentanchorCopy.Status.State)
				c.updateStatus(context.TODO(), selectivedeploymentanchorCopy, federationv1alpha1.StatusPendingScheduler, messagePending)
			}()
		}
	} else {
		c.edgenetclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchorCopy.GetNamespace()).Delete(context.TODO(), selectivedeploymentanchorCopy.GetName(), metav1.DeleteOptions{})
	}
}

func (c *Controller) makeSelectiveDeployment(selectivedeploymentanchorCopy *federationv1alpha1.SelectiveDeploymentAnchor) bool {
	// Since the assigned federation manager is the current one, check if the workload clusters are also selected
	if len(selectivedeploymentanchorCopy.Spec.WorkloadClusters) == 0 {
		return false
	}
	// Anchor hold UID information of selected cluster. We need to get the clusters from this UID info.
	// Thus, we list the clusters and compare cluster UIDs with selected ones.
	// We then get the secret of the cluster to create a client in order to access the remote cluster.
	// Finally, it creates the selective deployment in the remote cluster.
	clusterRaw, err := c.edgenetclientset.FederationV1alpha1().Clusters(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false
	}
	for _, clusterRow := range clusterRaw.Items {
		for _, workloadCluster := range selectivedeploymentanchorCopy.Spec.WorkloadClusters {
			if clusterRow.Spec.Role == federationv1alpha1.WorkloadRole && workloadCluster == clusterRow.Spec.UID {
				// Get the secret containing the creds of the remote cluster
				clusterSecret, err := c.kubeclientset.CoreV1().Secrets(clusterRow.GetNamespace()).Get(context.TODO(), clusterRow.Spec.SecretName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				// Create a client for the remote cluster by using the cluster's secret
				remoteClusterConfig := bootstrap.PrepareRestConfig(clusterRow.Spec.Server, string(clusterSecret.Data["token"]), clusterSecret.Data["ca.crt"])
				remotekubeclientset, err := bootstrap.CreateKubeClientset(remoteClusterConfig)
				if err != nil {
					return false
				}
				// Create a namespace with the name of the namespace in which the original selective deployment lives in the remote cluster if it does not exist
				remoteNamespace := new(corev1.Namespace)
				remoteNamespace.SetName(selectivedeploymentanchorCopy.Spec.OriginRef.Namespace)
				annotations := map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/access-scope=federation"}
				remoteNamespace.SetAnnotations(annotations)
				if _, err := remotekubeclientset.CoreV1().Namespaces().Create(context.TODO(), remoteNamespace, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					klog.Infoln(err)
					return false
				}
				// A selective deployment creates a secret containing the creds of its origin cluster.
				// The goal is to put the remote selective deployment in direct contact with the originating one, thus avoiding unnecessary traffic in the federation.
				selectiveDeploymentSecret, err := c.kubeclientset.CoreV1().Secrets(selectivedeploymentanchorCopy.GetNamespace()).Get(context.TODO(), selectivedeploymentanchorCopy.Spec.OriginRef.UID, metav1.GetOptions{})
				if err != nil {
					return false
				}
				// Create an EdgeNet client for the originating cluster by using the selective deployment's secret
				originatingClusterConfig := bootstrap.PrepareRestConfig(string(selectiveDeploymentSecret.Data["server"]), string(selectiveDeploymentSecret.Data["token"]), selectiveDeploymentSecret.Data["ca.crt"])
				originedgeclientset, err := bootstrap.CreateEdgeNetClientset(originatingClusterConfig)
				if err != nil {
					return false
				}
				originSelectiveDeployment, err := originedgeclientset.AppsV1alpha2().SelectiveDeployments(selectivedeploymentanchorCopy.Spec.OriginRef.Namespace).Get(context.TODO(), selectivedeploymentanchorCopy.Spec.OriginRef.Name, metav1.GetOptions{})
				if err != nil {
					klog.Infoln(err)
					return false
				}
				// Create the secret containing the creds of the originating cluster in the remote cluster
				remoteSelectiveDeploymentSecret := new(corev1.Secret)
				remoteSelectiveDeploymentSecret.SetName(selectiveDeploymentSecret.GetName())
				remoteSelectiveDeploymentSecret.SetNamespace(originSelectiveDeployment.GetNamespace())
				remoteSelectiveDeploymentSecret.Data = selectiveDeploymentSecret.Data
				if _, err = remotekubeclientset.CoreV1().Secrets(remoteSelectiveDeploymentSecret.GetNamespace()).Create(context.TODO(), remoteSelectiveDeploymentSecret, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					klog.Infoln(err)
					return false
				}
				// TODO: Copy configmaps, secrets, etc. from the originating cluster before making the selective deployment in the remote cluster
				// Create the selective deployment in the remote cluster
				remoteSelectiveDeployment := new(appsv1alpha2.SelectiveDeployment)
				remoteSelectiveDeployment.SetName(originSelectiveDeployment.GetName())
				remoteSelectiveDeployment.SetNamespace(originSelectiveDeployment.GetNamespace())
				remoteSelectiveDeployment.SetAnnotations(map[string]string{"edge-net.io/selective-deployment": "follower", "edge-net.io/origin-selective-deployment-uid": selectivedeploymentanchorCopy.Spec.OriginRef.UID})
				remoteSelectiveDeployment.Spec = originSelectiveDeployment.Spec
				remoteedgeclientset, err := bootstrap.CreateEdgeNetClientset(remoteClusterConfig)
				if err != nil {
					return false
				}
				if _, err = remoteedgeclientset.AppsV1alpha2().SelectiveDeployments(remoteSelectiveDeployment.GetNamespace()).Create(context.TODO(), remoteSelectiveDeployment, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
					klog.Infoln(err)
					return false
				}
				c.recorder.Event(selectivedeploymentanchorCopy, corev1.EventTypeNormal, federationv1alpha1.StatusDelegated, messageResourceDelegated)
			}
		}
	}
	return true
}

// conveySelectiveDeploymentAnchor moves the job to the next federation manager in the path
func (c *Controller) conveySelectiveDeploymentAnchor(selectivedeploymentanchorCopy *federationv1alpha1.SelectiveDeploymentAnchor) bool {
	// The next federation manager can be parent or child of the current one.
	// Therefore, we declare remote clientset pointers here.
	var remotekubeclientset *kubernetes.Clientset
	var remoteedgeclientset *clientset.Clientset
	// The next federation manager is the first one in the path
	var nextFederationManager string
	if len(selectivedeploymentanchorCopy.Spec.FederationManager.Path) > 0 {
		nextFederationManager = selectivedeploymentanchorCopy.Spec.FederationManager.Path[0]
	}

	// Get the secret containing the creds of the parent federation manager cluster
	parentFedmanagerSecret, err := c.kubeclientset.CoreV1().Secrets("edgenet").Get(context.TODO(), "federation", metav1.GetOptions{})
	klog.Infoln(err)
	klog.Infoln(errors.IsNotFound(err))
	// Compare the parent's UID with the next federation manager's UID. If they are the same, then the next federation manager is the parent.
	// Otherwise, the next federation manager is a child of the current one. The remote clientsets will be created depending on this information.
	if err == nil && (nextFederationManager == string(parentFedmanagerSecret.Data["remote-cluster-uid"]) || *selectivedeploymentanchorCopy.Spec.FederationUID != string(parentFedmanagerSecret.Data["federation-uid"])) {
		parentConfig := bootstrap.PrepareRestConfig(string(parentFedmanagerSecret.Data["server"]), string(parentFedmanagerSecret.Data["token"]), parentFedmanagerSecret.Data["ca.crt"])
		remotekubeclientset, err = bootstrap.CreateKubeClientset(parentConfig)
		if err != nil {
			klog.Infoln(err)
			return false
		}
		remoteedgeclientset, err = bootstrap.CreateEdgeNetClientset(parentConfig)
		if err != nil {
			klog.Infoln(err)
			return false
		}
	} else {
		kubesystemNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), metav1.NamespaceSystem, metav1.GetOptions{})
		if err != nil {
			klog.Infoln(err)
			return false
		}
		sdaLabels := selectivedeploymentanchorCopy.GetLabels()
		if string(kubesystemNamespace.GetUID()) == sdaLabels["edge-net.io/origin-federation-uid"] && string(kubesystemNamespace.GetUID()) != *selectivedeploymentanchorCopy.Spec.FederationUID {
			peerFedmanagerSecret, err := c.kubeclientset.CoreV1().Secrets("edgenet").Get(context.TODO(), *selectivedeploymentanchorCopy.Spec.FederationUID, metav1.GetOptions{})
			if err != nil {
				klog.Infoln(err)
				return false
			}
			peerConfig := bootstrap.PrepareRestConfig(string(peerFedmanagerSecret.Data["server"]), string(peerFedmanagerSecret.Data["token"]), peerFedmanagerSecret.Data["ca.crt"])
			remotekubeclientset, err = bootstrap.CreateKubeClientset(peerConfig)
			if err != nil {
				klog.Infoln(err)
				return false
			}
			remoteedgeclientset, err = bootstrap.CreateEdgeNetClientset(peerConfig)
			if err != nil {
				klog.Infoln(err)
				return false
			}
		} else {
			// List all clusters to retrieve the next federation manager's secret containing auth creds
			clusterRaw, err := c.edgenetclientset.FederationV1alpha1().Clusters(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				klog.Infoln(err)
				return false
			}
			match := false
		clusterLoop:
			for _, clusterRow := range clusterRaw.Items {
				klog.Infoln(clusterRow.Spec.UID)
				klog.Infoln(clusterRow.Spec.Role)
				klog.Infoln(nextFederationManager)
				if clusterRow.Spec.Role == federationv1alpha1.FederationManagerRole && nextFederationManager == clusterRow.Spec.UID {
					match = true
					childFedmanagerSecret, err := c.kubeclientset.CoreV1().Secrets(clusterRow.GetNamespace()).Get(context.TODO(), clusterRow.Spec.SecretName, metav1.GetOptions{})
					if err != nil {
						klog.Infoln(err)
						return false
					}
					config := bootstrap.PrepareRestConfig(clusterRow.Spec.Server, string(childFedmanagerSecret.Data["token"]), childFedmanagerSecret.Data["ca.crt"])
					remotekubeclientset, err = bootstrap.CreateKubeClientset(config)
					if err != nil {
						klog.Infoln(err)
						return false
					}
					remoteedgeclientset, err = bootstrap.CreateEdgeNetClientset(config)
					if err != nil {
						klog.Infoln(err)
						return false
					}
					break clusterLoop
				}
			}
			// If the next federation manager is not found, then return false.
			// It indicates the FedScheduler wrongly generates the path or one of the anchors carried false information.
			if !match {
				klog.Infoln("The next federation manager is not found")
				return false
			}
		}
	}
	// Create the propagation namespace if it does not exist
	/*remoteNamespace := new(corev1.Namespace)
	remoteNamespace.SetName(selectivedeploymentanchorCopy.GetNamespace())
	if _, err := remotekubeclientset.CoreV1().Namespaces().Create(context.TODO(), remoteNamespace, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		klog.Infoln(err)
		return false
	}*/
	// Create the selective deployment anchor with the originating selective deployment's secret in the next federation manager
	selectiveDeploymentSecret, err := c.kubeclientset.CoreV1().Secrets(selectivedeploymentanchorCopy.GetNamespace()).Get(context.TODO(), selectivedeploymentanchorCopy.Spec.OriginRef.UID, metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return false
	}
	remoteSelectiveDeploymentSecret := new(corev1.Secret)
	remoteSelectiveDeploymentSecret.SetName(selectiveDeploymentSecret.GetName())
	remoteSelectiveDeploymentSecret.SetNamespace(selectiveDeploymentSecret.GetNamespace())
	remoteSelectiveDeploymentSecret.Data = selectiveDeploymentSecret.Data
	if _, err := remotekubeclientset.CoreV1().Secrets(remoteSelectiveDeploymentSecret.GetNamespace()).Create(context.TODO(), remoteSelectiveDeploymentSecret, metav1.CreateOptions{}); err != nil {
		klog.Infoln(err)
		return false
	}
	remoteSelectiveDeploymentAnchor := new(federationv1alpha1.SelectiveDeploymentAnchor)
	remoteSelectiveDeploymentAnchor.SetName(selectivedeploymentanchorCopy.GetName())
	remoteSelectiveDeploymentAnchor.SetNamespace(selectivedeploymentanchorCopy.GetNamespace())
	remoteSelectiveDeploymentAnchor.Spec = selectivedeploymentanchorCopy.Spec
	// Remove the next federation manager from the path before creating the anchor in it
	remoteSelectiveDeploymentAnchor.Spec.FederationManager.Path = nil
	if len(selectivedeploymentanchorCopy.Spec.FederationManager.Path) > 1 {
		remoteSelectiveDeploymentAnchor.Spec.FederationManager.Path = selectivedeploymentanchorCopy.Spec.FederationManager.Path[1:]
	}
	if _, err := remoteedgeclientset.FederationV1alpha1().SelectiveDeploymentAnchors(remoteSelectiveDeploymentAnchor.GetNamespace()).Create(context.TODO(), remoteSelectiveDeploymentAnchor, metav1.CreateOptions{}); err != nil {
		klog.Infoln(err)
		return false
	}
	return true
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
