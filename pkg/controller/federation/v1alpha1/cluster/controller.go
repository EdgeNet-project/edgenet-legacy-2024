package cluster

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

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"reflect"
	"time"

	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/federation/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/multiprovider"
	multitenancy "github.com/EdgeNet-project/edgenet/pkg/multitenancy"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

const controllerAgentName = "cluster-controller"

// Definitions of the state of the cluster resource
const (
	backoffLimit = 3

	successSynced = "Synced"

	messageResourceSynced                   = "Cluster synced successfully"
	messageCredsPrepared                    = "Credentials for federation manager access prepared successfully"
	messageReady                            = "Inter-cluster communication is established"
	messageMissingSecretAtRemote            = "Secret storing federation managers's token is missing in the remote cluster"
	messageWrongSecretAtRemote              = "Secret storing federation manager's token is wrong in the remote cluster"
	messageWrongManagerCacheAtRemote        = "Manager cache is wrong in the remote cluster"
	messageMissingSecretFMAuth              = "Secret storing federation manager's token is missing in the federation manager"
	messageMissingSecretRemoteAuth          = "Secret storing remote cluster's token is missing in the federation manager"
	messageMissingServiceAccount            = "Remote cluster's service account is missing in the federation manager"
	messageRemoteClientFailed               = "Clientset for remote cluster cannot be created"
	messageCredsFailed                      = "Credentials for federation manager access cannot be prepared"
	messageRemoteSecretFailed               = "Remote secret cannot be created from the secret of credentials"
	messageRemoteSecretDeploymentFailed     = "Remote secret cannot be deployed to the remote cluster"
	messageManagerCacheUpdateFailed         = "Manager cache cannot be updated to include the new manager cluster as a child"
	messageRemoteManagerCacheCreationFailed = "Manager cache cannot be created in the remote cluster"
	messageInvalidHost                      = "Server field must be an IP Address"
	messageChildrenManagerDisableFailed     = "Children managers cannot be disabled"
	messageManagerCacheMissing              = "Manager cache is missing"
	messageRemoteManagerCacheListFailed     = "Peering federation's caches cannot be listed in the remote cluster"
)

// Controller is the controller implementation for Cluster resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
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
}

// NewController returns a new controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	clusterInformer informers.ClusterInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events(metav1.NamespaceAll)})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:    kubeclientset,
		edgenetclientset: edgenetclientset,
		clustersLister:   clusterInformer.Lister(),
		clustersSynced:   clusterInformer.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Clusters"),
		recorder:         recorder,
	}

	klog.Infoln("Setting up event handlers")
	// Set up an event handler for when Cluster resources change
	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueCluster,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueCluster(new)
		},
	})

	return controller
}

// Run will set up the event handlers for the types of cluster and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting Cluster controller")

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
// converge the two. It then updates the Status block of the Cluster
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	cluster, err := c.clustersLister.Clusters(namespace).Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("cluster '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processCluster(cluster.DeepCopy())
	c.recorder.Event(cluster, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueCluster takes a Cluster resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Cluster.
func (c *Controller) enqueueCluster(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueClusterAfter takes a Cluster resource and converts it into a namespace/name
// string which is then put onto the work queue after the expiry date to be deleted. This method should *not* be
// passed resources of any type other than Cluster.
func (c *Controller) enqueueClusterAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

func (c *Controller) processCluster(clusterCopy *federationv1alpha1.Cluster) {
	// Crashloop backoff limit to avoid endless loop
	if clusterCopy.Status.UpdateTimestamp != nil && clusterCopy.Status.UpdateTimestamp.Add(24*time.Hour).After(time.Now()) {
		if exceedsBackoffLimit := clusterCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
			return
		}
	}

	multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
	permitted, _, namespaceLabels := multitenancyManager.EligibilityCheck(clusterCopy.GetNamespace())
	if permitted {
		propagationNamespace := fmt.Sprintf(federationv1alpha1.FederationManagerNamespace, namespaceLabels["edge-net.io/cluster-uid"])
		switch clusterCopy.Status.State {
		case federationv1alpha1.StatusReady:
			// As the cluster is ready, we need to reconcile with the remote cluster
			c.reconcile(clusterCopy, propagationNamespace, namespaceLabels["edge-net.io/cluster-uid"])
		case federationv1alpha1.StatusCredsPrepared:
			// Make the config file from the cluster spec and create the remote kube clientset
			config, ok := c.prepareConfig(clusterCopy)
			if !ok {
				return
			}
			remotekubeclientset, err := bootstrap.CreateKubeClientset(config)
			if err != nil {
				c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageRemoteClientFailed)
				c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageRemoteClientFailed)
				return
			}
			remoteedgeclientset, err := bootstrap.CreateEdgeNetClientset(config)
			if err != nil {
				c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageRemoteClientFailed)
				c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageRemoteClientFailed)
				return
			}
			multiproviderManager := multiprovider.NewManager(c.kubeclientset, remotekubeclientset, c.edgenetclientset, remoteedgeclientset)
			// The federation framework uses a manager cache to keep track of the hierarchy of the manager clusters along with their location information.
			// In addition to that, a manager cache contains information regarding the resources and the locations of their workload clusters.
			// Thus, the manager cache is the source of truth for the federation framework to make scheduling decisions at the federation scale.
			// Below checks if the cluster's role is being a federation manager. If so, it adds the cluster as a child to its federation manager cache and then creates a manager cache of the cluster in the remote cluster.
			if clusterCopy.Spec.Role == federationv1alpha1.PeerRole {
				peeringFedCacheRaw, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/federation-uid=" + clusterCopy.Spec.UID})
				if err != nil {
					c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageRemoteManagerCacheListFailed)
					c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageRemoteManagerCacheListFailed)
					return
				}
				for _, peeringFedCacheRow := range peeringFedCacheRaw.Items {
					if peeringFedCacheRow.Status.State == federationv1alpha1.StatusReady {
						newManagerCache := new(federationv1alpha1.ManagerCache)
						newManagerCache.SetName(peeringFedCacheRow.GetName())
						newManagerCache.Spec = peeringFedCacheRow.Spec
						newManagerCache.SetLabels(peeringFedCacheRow.GetLabels())
						_, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().Create(context.TODO(), newManagerCache, metav1.CreateOptions{})
						if err != nil {
							c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageManagerCacheMissing)
							c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageManagerCacheMissing)
							return
						}
						createdPeerCache, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), peeringFedCacheRow.GetName(), metav1.GetOptions{})
						if err != nil {
							c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageManagerCacheMissing)
							c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageManagerCacheMissing)
							return
						}
						createdPeerCache.Status = peeringFedCacheRow.Status
						c.edgenetclientset.FederationV1alpha1().ManagerCaches().UpdateStatus(context.TODO(), createdPeerCache, metav1.UpdateOptions{})
					}
				}
			} else {
				updateTimestamp := metav1.Now()
				managerCache, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), namespaceLabels["edge-net.io/cluster-uid"], metav1.GetOptions{})
				if err != nil {
					// Create a manager cache for the root-level node
					newManagerCache := new(federationv1alpha1.ManagerCache)
					newManagerCache.SetName(namespaceLabels["edge-net.io/cluster-uid"])
					newManagerCache.Spec.FederationUID = namespaceLabels["edge-net.io/cluster-uid"]
					newManagerCache.Spec.Enabled = true
					newManagerCache.Spec.LatestUpdateTimestamp = &updateTimestamp
					newManagerCache.SetLabels(map[string]string{"edge-net.io/federation-uid": namespaceLabels["edge-net.io/cluster-uid"]})
					_, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().Create(context.TODO(), newManagerCache, metav1.CreateOptions{})
					if err != nil {
						c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageManagerCacheMissing)
						c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageManagerCacheMissing)
						return
					}
					c.enqueueCluster(clusterCopy)
					return
				}
				localCacheLabels := managerCache.GetLabels()
				clusterLabels := clusterCopy.GetLabels()
				if clusterCopy.Spec.Role == federationv1alpha1.FederationManagerRole {
					if managerCache.Spec.Hierarchy.Children == nil {
						managerCache.Spec.Hierarchy.Children = []federationv1alpha1.AssociatedManager{}
					}
					// Update the manager cache of the federation manager to include the cluster as a child
					child := federationv1alpha1.AssociatedManager{}
					child.Name = clusterCopy.Spec.UID
					child.Enabled = clusterCopy.Spec.Enabled
					isExists := false
					for key, value := range managerCache.Spec.Hierarchy.Children {
						if value.Name == clusterCopy.Spec.UID {
							managerCache.Spec.Hierarchy.Children[key] = child
							isExists = true
						}
					}
					if !isExists {
						managerCache.Spec.Hierarchy.Children = append(managerCache.Spec.Hierarchy.Children, child)
						managerCache.Spec.LatestUpdateTimestamp = &updateTimestamp
					}

					// Create a manager cache in the remote cluster
					remoteManagerCache := new(federationv1alpha1.ManagerCache)
					remoteManagerCache.SetName(clusterCopy.Spec.UID)
					parent := federationv1alpha1.AssociatedManager{}
					parent.Name = namespaceLabels["edge-net.io/cluster-uid"]
					parent.Enabled = managerCache.Spec.Enabled
					remoteManagerCache.Spec.Hierarchy.Parent = &parent
					remoteManagerCache.Spec.Hierarchy.Level = managerCache.Spec.Hierarchy.Level + 1 // The level of this manager cluster is one level higher than the federation manager
					remoteCacheLabels := clusterLabels
					remoteCacheLabels["edge-net.io/federation-uid"] = localCacheLabels["edge-net.io/federation-uid"]
					remoteManagerCache.SetLabels(remoteCacheLabels)
					remoteManagerCache.Spec.LatestUpdateTimestamp = &updateTimestamp
					remoteManagerCache.Spec.Enabled = clusterCopy.Spec.Enabled
					if err := multiproviderManager.CreateManagerCache(remoteManagerCache); err != nil && !errors.IsAlreadyExists(err) {
						c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageRemoteManagerCacheCreationFailed)
						c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageRemoteManagerCacheCreationFailed)
						return
					}
					if err := multiproviderManager.DisableChildrenManagers(); err != nil {
						c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageChildrenManagerDisableFailed)
						c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageChildrenManagerDisableFailed)
						return
					}
				} else {
					clusterCache := federationv1alpha1.ClusterCache{}
					clusterCache.Enabled = clusterCopy.Spec.Enabled
					clusterCache.AllocatableResources = clusterCopy.Status.AllocatableResources
					clusterCache.RelativeResourceAvailability = clusterCopy.Status.RelativeResourceAvailability
					if clusterCache.Characteristics == nil {
						clusterCache.Characteristics = make(map[string]string)
					}
					for key, value := range clusterLabels {
						clusterCache.Characteristics[key] = value
					}
					if managerCache.Spec.Clusters == nil {
						managerCache.Spec.Clusters = make(map[string]federationv1alpha1.ClusterCache)
					}
					managerCache.Spec.Clusters[clusterCopy.Spec.UID] = clusterCache
					managerCache.Spec.LatestUpdateTimestamp = &updateTimestamp
				}
				if _, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().Update(context.TODO(), managerCache, metav1.UpdateOptions{}); err != nil {
					c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageManagerCacheUpdateFailed)
					c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageManagerCacheUpdateFailed)
					return
				}

				// Here we prepare a secret to be deployed to the remote cluster by using the secret that is created while setting up the access credentials.
				// This secret has additional information about the API server, namespace, the cluster UID of federation manager.
				// The remote cluster will be consuming these information to access the federation manager and drive necessary operations.
				remoteSecret, enqueue, err := multiproviderManager.PrepareSecretForRemoteCluster(clusterCopy.Spec.UID, propagationNamespace, namespaceLabels["edge-net.io/cluster-uid"], "federation", "edgenet")
				remoteSecret.Data["assigned-cluster-namespace"] = []byte(clusterCopy.GetNamespace())
				remoteSecret.Data["assigned-cluster-name"] = []byte(clusterCopy.GetName())
				remoteSecret.Data["federation-uid"] = []byte(localCacheLabels["edge-net.io/federation-uid"])
				if err != nil {
					if enqueue {
						c.enqueueClusterAfter(clusterCopy, 1*time.Minute)
						return
					}
					c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageRemoteSecretFailed)
					c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageRemoteSecretFailed)
					return
				}
				if err := multiproviderManager.DeploySecret(remoteSecret); err != nil {
					c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageRemoteSecretDeploymentFailed)
					c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageRemoteSecretDeploymentFailed)
					return
				}
			}
			c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusReady, messageReady)
			c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusReady, messageReady)
		default:
			// Below creates a secret tied to a service account along with a role binding for the remote cluster.
			// The remote cluster will use this secret to communicate with its federation manager, thus gaining access to the federated resources.
			multiproviderManager := multiprovider.NewManager(c.kubeclientset, nil, c.edgenetclientset, nil)
			// TODO: We should support both using the IP address of the cluster or do a DNS lookup if it is a hostname]
			host, _, err := net.SplitHostPort(clusterCopy.Spec.Server)
			if err != nil {
				c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageInvalidHost)
				c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageInvalidHost)
				return
			}
			recordType := multiprovider.GetRecordType(host)
			if recordType == "" {
				c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageInvalidHost)
				c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageInvalidHost)
				return
			}
			if err := multiproviderManager.SetupRemoteAccessCredentials(clusterCopy.Spec.UID, propagationNamespace, federationv1alpha1.RemoteClusterRole); err != nil {
				c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageCredsFailed)
				c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageCredsFailed)
				return
			}
			// This part binds a ClusterRole to the service account to grant the predefined permissions to the serviceaccount in the provider's namespace
			roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: federationv1alpha1.RemoteClusterRole}
			rbSubjects := []rbacv1.Subject{{Kind: "ServiceAccount", Name: clusterCopy.Spec.UID, Namespace: propagationNamespace}}
			roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", federationv1alpha1.RemoteClusterRole, clusterCopy.Spec.UID), Namespace: clusterCopy.GetNamespace()},
				Subjects: rbSubjects, RoleRef: roleRef}
			roleBindLabels := map[string]string{"edge-net.io/generated": "true"}
			roleBind.SetLabels(roleBindLabels)
			if _, err := c.kubeclientset.RbacV1().RoleBindings(clusterCopy.GetNamespace()).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil {
				if !errors.IsAlreadyExists(err) {
					c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageCredsFailed)
					c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageCredsFailed)
					return
				}
				if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(clusterCopy.GetNamespace()).Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{}); err == nil {
					roleBinding.RoleRef = roleBind.RoleRef
					roleBinding.Subjects = roleBind.Subjects
					roleBinding.SetLabels(roleBind.GetLabels())
					if _, err := c.kubeclientset.RbacV1().RoleBindings(clusterCopy.GetNamespace()).Update(context.TODO(), roleBinding, metav1.UpdateOptions{}); err != nil {
						c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageCredsFailed)
						c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageCredsFailed)
						return
					}
				}
			}
			c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusCredsPrepared, messageCredsPrepared)
			c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusCredsPrepared, messageCredsPrepared)
		}
	} else {
		c.edgenetclientset.FederationV1alpha1().Clusters(clusterCopy.GetNamespace()).Delete(context.TODO(), clusterCopy.GetName(), metav1.DeleteOptions{})
	}
}

func (c *Controller) reconcile(clusterCopy *federationv1alpha1.Cluster, propagationNamespace, fedmanagerUID string) {
	var state, message string = clusterCopy.Status.State, clusterCopy.Status.Message
	// Check if the remote cluster's service account exists
	if _, err := c.kubeclientset.CoreV1().ServiceAccounts(propagationNamespace).Get(context.TODO(), clusterCopy.Spec.UID, metav1.GetOptions{}); err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageMissingServiceAccount)
		state = federationv1alpha1.StatusReconciliation
		message = messageMissingServiceAccount
	}
	// Check if the secret that is tied to the remote cluster's service account exists
	authSecret, err := c.kubeclientset.CoreV1().Secrets(propagationNamespace).Get(context.TODO(), clusterCopy.Spec.UID, metav1.GetOptions{})
	if err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageMissingSecretFMAuth)
		state = federationv1alpha1.StatusReconciliation
		message = messageMissingSecretFMAuth
	}
	// Manipulate the secret to be compared with the one in the remote cluster
	authSecret.SetName("federation")
	authSecret.SetNamespace("edgenet")
	authSecret.Data["serviceaccount"] = []byte(fmt.Sprintf("system:serviceaccount:%s:%s", propagationNamespace, clusterCopy.Spec.UID))
	authSecret.Data["namespace"] = []byte(propagationNamespace)
	// Get the address of the federation manager
	var address string
	nodeRaw, _ := c.kubeclientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/control-plane"})
	for _, node := range nodeRaw.Items {
		if internal, external := multiprovider.GetNodeIPAddresses(node.DeepCopy()); external == "" && internal == "" {
			continue
		} else if external != "" {
			address = external + ":8443"
		} else {
			address = internal + ":8443"
		}
		break
	}
	authSecret.Data["server"] = []byte(address)
	authSecret.Data["remote-cluster-uid"] = []byte(fedmanagerUID)
	authSecret.Data["assigned-cluster-namespace"] = []byte(clusterCopy.GetNamespace())
	authSecret.Data["assigned-cluster-name"] = []byte(clusterCopy.GetName())
	// Prepare the config to access the remote cluster
	config, ok := c.prepareConfig(clusterCopy)
	if !ok {
		return
	}
	remotekubeclientset, err := bootstrap.CreateKubeClientset(config)
	if err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageRemoteClientFailed)
		state = federationv1alpha1.StatusReconciliation
		message = messageRemoteClientFailed
	}
	// Retrieve that secret from the remote cluster
	remoteSecretFMAuth, err := remotekubeclientset.CoreV1().Secrets(authSecret.GetNamespace()).Get(context.TODO(), authSecret.GetName(), metav1.GetOptions{})
	if err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageMissingSecretAtRemote)
		state = federationv1alpha1.StatusReconciliation
		message = messageMissingSecretAtRemote
	}
	// Compare the two secrets
	if bytes.Compare(authSecret.Data["namespace"], remoteSecretFMAuth.Data["namespace"]) != 0 || bytes.Compare(authSecret.Data["server"], remoteSecretFMAuth.Data["server"]) != 0 ||
		bytes.Compare(authSecret.Data["token"], remoteSecretFMAuth.Data["token"]) != 0 || bytes.Compare(authSecret.Data["remote-cluster-uid"], remoteSecretFMAuth.Data["remote-cluster-uid"]) != 0 ||
		bytes.Compare(authSecret.Data["assigned-cluster-namespace"], remoteSecretFMAuth.Data["assigned-cluster-namespace"]) != 0 ||
		bytes.Compare(authSecret.Data["assigned-cluster-name"], remoteSecretFMAuth.Data["assigned-cluster-name"]) != 0 {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageWrongSecretAtRemote)
		state = federationv1alpha1.StatusReconciliation
		message = messageWrongSecretAtRemote
	}

	// Check if the manager cache at the remote cluster exists and holds the correct information
	managerCache, _ := c.edgenetclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), fedmanagerUID, metav1.GetOptions{})
	if clusterCopy.Spec.Role == federationv1alpha1.FederationManagerRole {
		// Update the manager cache of the federation manager to include the newly added cluster as a child
		ok := false
		for _, child := range managerCache.Spec.Hierarchy.Children {
			if child.Name == clusterCopy.Spec.UID && child.Enabled == clusterCopy.Spec.Enabled {
				ok = true
			}
		}
		if !ok {
			c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageManagerCacheUpdateFailed)
			state = federationv1alpha1.StatusReconciliation
			message = messageManagerCacheUpdateFailed
		}
		remoteedgeclientset, err := bootstrap.CreateEdgeNetClientset(config)
		if err != nil {
			c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageRemoteClientFailed)
			state = federationv1alpha1.StatusReconciliation
			message = messageRemoteClientFailed
		}
		if remoteManagerCache, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), clusterCopy.Spec.UID, metav1.GetOptions{}); err == nil {
			if remoteManagerCache.Spec.Hierarchy.Parent.Name != fedmanagerUID || remoteManagerCache.Spec.Hierarchy.Parent.Enabled != managerCache.Spec.Enabled || remoteManagerCache.Spec.Hierarchy.Level != managerCache.Spec.Hierarchy.Level+1 {
				c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageWrongManagerCacheAtRemote)
				state = federationv1alpha1.StatusReconciliation
				message = messageWrongManagerCacheAtRemote
			}
		} else {
			c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageRemoteManagerCacheCreationFailed)
			state = federationv1alpha1.StatusReconciliation
			message = messageRemoteManagerCacheCreationFailed
		}
	} else {
		if clusterCopy.Spec.Role == federationv1alpha1.WorkloadRole {
			if managerCache.Spec.Clusters != nil {
				if clusterCache, ok := managerCache.Spec.Clusters[clusterCopy.Spec.UID]; !ok || !reflect.DeepEqual(clusterCache.AllocatableResources, clusterCopy.Status.AllocatableResources) || !reflect.DeepEqual(clusterCache.Characteristics, clusterCopy.GetLabels()) ||
					clusterCache.RelativeResourceAvailability != clusterCopy.Status.RelativeResourceAvailability {
					c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageManagerCacheUpdateFailed)
					state = federationv1alpha1.StatusReconciliation
					message = messageManagerCacheUpdateFailed
				}
			} else {
				c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageManagerCacheUpdateFailed)
				state = federationv1alpha1.StatusReconciliation
				message = messageManagerCacheUpdateFailed
			}
		}
	}
	// If the cluster status is not ready, update it
	if state != federationv1alpha1.StatusReady {
		c.updateStatus(context.TODO(), clusterCopy, state, message)
	} else {
		if clusterCopy.Status.Failed != 0 {
			clusterCopy.Status.Failed = 0
			c.updateStatus(context.TODO(), clusterCopy, state, message)
		}
	}
}

func (c *Controller) prepareConfig(clusterCopy *federationv1alpha1.Cluster) (*rest.Config, bool) {
	// Below is to prepare the config to create the remote clientsets. It gets the secret created in the cluster's namespace.
	// This secret includes necessary information for the federation manager to access the remote cluster.
	// The federation framework uses service accounts to allow clusters to access one another.
	remoteAuthSecret, err := c.kubeclientset.CoreV1().Secrets(clusterCopy.GetNamespace()).Get(context.TODO(), clusterCopy.Spec.SecretName, metav1.GetOptions{})
	if err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageMissingSecretRemoteAuth)
		c.updateStatus(context.TODO(), clusterCopy, federationv1alpha1.StatusFailed, messageMissingSecretRemoteAuth)
		return nil, false
	}
	remoteServer := clusterCopy.Spec.Server               // API server address
	remoteToken := string(remoteAuthSecret.Data["token"]) // A service account token to be consumed
	remoteCA := remoteAuthSecret.Data["ca.crt"]           // CA certificate
	config := bootstrap.PrepareRestConfig(remoteServer, remoteToken, remoteCA)
	return config, true
}

// updateStatus calls the API to update the cluster status.
func (c *Controller) updateStatus(ctx context.Context, clusterCopy *federationv1alpha1.Cluster, state, message string) {
	clusterCopy.Status.State = state
	clusterCopy.Status.Message = message
	if clusterCopy.Status.State == federationv1alpha1.StatusFailed {
		clusterCopy.Status.Failed++
		now := metav1.Now()
		clusterCopy.Status.UpdateTimestamp = &now
	}
	if _, err := c.edgenetclientset.FederationV1alpha1().Clusters(clusterCopy.GetNamespace()).UpdateStatus(ctx, clusterCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
