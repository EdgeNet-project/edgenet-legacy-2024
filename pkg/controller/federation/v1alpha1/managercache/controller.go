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
	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/federation/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/federation/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

const controllerAgentName = "managercache-controller"

// Definitions of the state of the managercache resource
const (
	backoffLimit = 3

	successSynced = "Synced"

	messageResourceSynced     = "ManagerCache synced successfully"
	messageReady              = "ManagerCache is ready"
	messagePending            = "ManagerCache is pending a cluster"
	messageParentUpdated      = "ManagerCache updated at the parent federation manager"
	messageParentNotUpdated   = "ManagerCache cannot be updated at the parent federation manager"
	messageChildUpdated       = "ManagerCache updated at the child federation manager"
	messageChildNotUpdated    = "ManagerCache cannot be updated at the child federation manager"
	messageKubeSystemNotFound = "The kube-system namespace not found"
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
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events(metav1.NamespaceAll)})
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
	kubesystemNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		c.recorder.Event(managercacheCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageKubeSystemNotFound)
		c.updateStatus(context.TODO(), managercacheCopy, federationv1alpha1.StatusFailed, messageKubeSystemNotFound)
		return
	}
	if managercacheCopy.GetName() == string(kubesystemNamespace.GetUID()) {
		// Crashloop backoff limit to avoid endless loop
		if managercacheCopy.Status.UpdateTimestamp != nil && managercacheCopy.Status.UpdateTimestamp.Add(1*time.Hour).After(time.Now()) {
			if exceedsBackoffLimit := managercacheCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
				return
			}
		}
		switch managercacheCopy.Status.State {
		case federationv1alpha1.StatusReady:
			// Reconcile
			if isReconciled := c.reconcileWithChildren(); !isReconciled {
				c.recorder.Event(managercacheCopy, corev1.EventTypeNormal, federationv1alpha1.StatusReconciliation, messageChildNotUpdated)
				c.updateStatus(context.TODO(), managercacheCopy, federationv1alpha1.StatusReconciliation, messageChildNotUpdated)
				return
			}
			if managercacheCopy.Status.Failed != 0 {
				managercacheCopy.Status.Failed = 0
				c.updateStatus(context.TODO(), managercacheCopy, managercacheCopy.Status.State, managercacheCopy.Status.Message)
			}
		default:
			// A federation manager does not control any workload cluster, it falls into the pending state.
			// As manager caches are used to make scheduling decisions, we simply ignore the ones who do not hold a cluster to run workloads.
			if len(managercacheCopy.Spec.Clusters) == 0 && len(managercacheCopy.Spec.Hierarchy.Children) == 0 {
				c.recorder.Event(managercacheCopy, corev1.EventTypeNormal, federationv1alpha1.StatusPending, messagePending)
				c.updateStatus(context.TODO(), managercacheCopy, federationv1alpha1.StatusPending, messagePending)
				return
			}
			// This controller is responsible for spreading the cache to the parent and children FMs of the federation manager on which it runs.
			// For this purpose, we create a manager cache to be created at the remote clusters.

			// The code below creates/updates the caches at children FMs as well as at the current FM
			if err := c.applyCache(managercacheCopy, string(kubesystemNamespace.GetUID())); err != nil {
				klog.Infoln(err)
				c.recorder.Event(managercacheCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageChildNotUpdated)
				c.updateStatus(context.TODO(), managercacheCopy, federationv1alpha1.StatusFailed, messageChildNotUpdated)
				return
			}
			// Update status to ready
			c.recorder.Event(managercacheCopy, corev1.EventTypeNormal, federationv1alpha1.StatusReady, messageReady)
			c.updateStatus(context.TODO(), managercacheCopy, federationv1alpha1.StatusReady, messageReady)
		}
	}
}

func (c *Controller) reconcileWithParent(managerCache *federationv1alpha1.ManagerCache) bool {
	parentFedManagerSecret, err := c.kubeclientset.CoreV1().Secrets("edgenet").Get(context.TODO(), "federation", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return true
		}
		return false
	}
	if managerCache.Spec.Hierarchy.Parent.Name != string(parentFedManagerSecret.Data["remote-cluster-uid"]) {
		return false
	}
	config := bootstrap.PrepareRestConfig(string(parentFedManagerSecret.Data["server"]), string(parentFedManagerSecret.Data["token"]), parentFedManagerSecret.Data["ca.crt"])
	remoteedgeclientset, err := bootstrap.CreateEdgeNetClientset(config)
	if err != nil {
		return false
	}
	if remoteManagerCacheCopy, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), managerCache.GetName(), metav1.GetOptions{}); err == nil {
		if !reflect.DeepEqual(managerCache.Spec, remoteManagerCacheCopy.Spec) {
			return false
		}
	} else {
		return false
	}
	return true
}

func (c *Controller) reconcileWithChildren() bool {
	managercacheMap := make(map[string]federationv1alpha1.ManagerCache)
	managercacheRaw, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, managercacheRow := range managercacheRaw.Items {
			managercacheMap[managercacheRow.GetName()] = managercacheRow
		}
	} else {
		return false
	}
	clusterRaw, err := c.edgenetclientset.FederationV1alpha1().Clusters(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false
	}
	for _, clusterRow := range clusterRaw.Items {
		if clusterRow.Spec.Role == federationv1alpha1.FederationManagerRole {
			childFedManagerSecret, err := c.kubeclientset.CoreV1().Secrets(clusterRow.GetNamespace()).Get(context.TODO(), clusterRow.Spec.SecretName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			config := bootstrap.PrepareRestConfig(clusterRow.Spec.Server, string(childFedManagerSecret.Data["token"]), childFedManagerSecret.Data["ca.crt"])
			remoteedgeclientset, err := bootstrap.CreateEdgeNetClientset(config)
			if err != nil {
				return false
			}

			if remotemanagercacheRaw, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().List(context.TODO(), metav1.ListOptions{}); err == nil {
				for _, remotemanagercacheRow := range remotemanagercacheRaw.Items {
					if _, ok := managercacheMap[remotemanagercacheRow.GetName()]; ok {
						if !reflect.DeepEqual(managercacheMap[remotemanagercacheRow.GetName()].Spec, remotemanagercacheRow.Spec) || managercacheMap[remotemanagercacheRow.GetName()].Status.State != remotemanagercacheRow.Status.State {
							return false
						}
					} else {
						return false
					}
				}
			} else {
				return false
			}
		}
	}
	return true
}

func (c *Controller) applyCache(managercacheCopy *federationv1alpha1.ManagerCache, clusterUID string) error {
	managercacheRaw, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Infoln(err)
		return err
	}
	localCacheMap := c.constructCacheMap(managercacheRaw)
	// We list children federation manager clusters and create/update the cache
	clusterRaw, err := c.edgenetclientset.FederationV1alpha1().Clusters(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Infoln(err)
		return err
	}
	// We list caches in children to create/update local caches based on the source of truth caches
	localCacheMap, err = c.syncCaches(clusterRaw, localCacheMap, clusterUID, true)
	if err != nil {
		if err != nil {
			klog.Infoln(err)
			return err
		}
	}
	// Below we create/update fedeation manager specific namespaces in the local federation manager cluster
	if err := c.prepareManagerSpecificNamespaces(localCacheMap, clusterUID); err != nil {
		klog.Infoln(err)
		return err
	}
	// We list children federation manager clusters where we create/update remote caches based on the local, source of truth caches
	_, err = c.syncCaches(clusterRaw, localCacheMap, clusterUID, false)
	if err != nil {
		if err != nil {
			klog.Infoln(err)
			return err
		}
	}
	return nil
}

/*
func (c *Controller) updateChildManagerCache(managercacheCopy *federationv1alpha1.ManagerCache, server string, token, certificateAuthorityData []byte) error {
	config := bootstrap.PrepareRestConfig(server, string(token), certificateAuthorityData)
	remoteedgeclientset, err := bootstrap.CreateEdgeNetClientset(config)
	if err != nil {
		klog.Infoln(err)
		return err
	}
	// As the status is not ready in this switch statement, we assume the parent federation manager does not have this cache.
	// If it exists, we then compare the current cache's spec with the remote cache's one. Then the remote cache gets updated if these are not the same.
	remoteManagerCache := new(federationv1alpha1.ManagerCache)
	remoteManagerCache.SetName(managercacheCopy.GetName())
	remoteManagerCache.Spec = managercacheCopy.Spec
	if _, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Create(context.TODO(), remoteManagerCache, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if remoteManagerCacheCopy, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), remoteManagerCache.GetName(), metav1.GetOptions{}); err == nil {
				if !reflect.DeepEqual(remoteManagerCache.Spec, remoteManagerCacheCopy.Spec) {
					remoteManagerCacheCopy.Spec = remoteManagerCache.Spec
					if _, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Update(context.TODO(), remoteManagerCacheCopy, metav1.UpdateOptions{}); err != nil {
						klog.Infoln(err)
						return err
					}
				}
				klog.Infoln("managercacheCopy.Status: ", managercacheCopy.Status)
				klog.Infoln("remoteManagerCacheCopy.Status: ", remoteManagerCacheCopy.Status)
				if !reflect.DeepEqual(managercacheCopy.Status, remoteManagerCacheCopy.Status) || managercacheCopy.Status.State == federationv1alpha1.StatusReconciliation {
					if remoteManagerCacheCopy, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), remoteManagerCache.GetName(), metav1.GetOptions{}); err == nil {
						remoteManagerCacheCopy.Status = managercacheCopy.Status
						if managercacheCopy.Status.State == federationv1alpha1.StatusReconciliation {
							remoteManagerCacheCopy.Status.State = federationv1alpha1.StatusReady
							remoteManagerCacheCopy.Status.Failed = managercacheCopy.Status.Failed
							remoteManagerCacheCopy.Status.Message = messageReady
							remoteManagerCacheCopy.Status.UpdateTimestamp = managercacheCopy.Status.UpdateTimestamp
						}
						if _, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().UpdateStatus(context.TODO(), remoteManagerCacheCopy, metav1.UpdateOptions{}); err != nil {
							klog.Infoln(err)
							return err
						}
					}
				}
			}
		} else {
			klog.Infoln(err)
			return err
		}
	} else {
		if remoteManagerCacheCopy, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), remoteManagerCache.GetName(), metav1.GetOptions{}); err == nil {
			if reflect.DeepEqual(managercacheCopy.Status, remoteManagerCacheCopy.Status) || managercacheCopy.Status.State == federationv1alpha1.StatusReconciliation {
				return nil
			}
			remoteManagerCacheCopy.Status = managercacheCopy.Status
			if managercacheCopy.Status.State == federationv1alpha1.StatusReconciliation {
				remoteManagerCacheCopy.Status.State = federationv1alpha1.StatusReady
				remoteManagerCacheCopy.Status.Failed = managercacheCopy.Status.Failed
				remoteManagerCacheCopy.Status.Message = messageReady
				remoteManagerCacheCopy.Status.UpdateTimestamp = managercacheCopy.Status.UpdateTimestamp
			}
			if _, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().UpdateStatus(context.TODO(), remoteManagerCacheCopy, metav1.UpdateOptions{}); err == nil {
				return nil
			} else {
				klog.Infoln(err)
			}
		}
	}
	return nil
}
*/

func (c *Controller) constructCacheMap(managercacheRaw *federationv1alpha1.ManagerCacheList) map[string]federationv1alpha1.ManagerCache {
	managercacheMap := make(map[string]federationv1alpha1.ManagerCache)
	for _, managercacheRow := range managercacheRaw.Items {
		managercacheMap[managercacheRow.GetName()] = managercacheRow
	}
	return managercacheMap
}

func (c *Controller) prepareManagerSpecificNamespaces(cacheMap map[string]federationv1alpha1.ManagerCache, clusterUID string) error {
	for key := range cacheMap {
		propagationNamespace := fmt.Sprintf(federationv1alpha1.FederationManagerNamespace, key)
		remoteNamespace := new(corev1.Namespace)
		remoteNamespace.SetName(propagationNamespace)
		if _, err := c.kubeclientset.CoreV1().Namespaces().Create(context.TODO(), remoteNamespace, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			klog.Infoln(err)
			return err
		}
		fedmanagerNamespace := fmt.Sprintf(federationv1alpha1.FederationManagerNamespace, clusterUID)

		// This part binds a ClusterRole to the service account to grant the predefined permissions to the serviceaccount
		roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: federationv1alpha1.RemoteClusterRole}
		rbSubjects := []rbacv1.Subject{{Kind: "ServiceAccount", Name: key, Namespace: fedmanagerNamespace}}
		roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", federationv1alpha1.RemoteClusterRole, key), Namespace: propagationNamespace},
			Subjects: rbSubjects, RoleRef: roleRef}
		roleBindLabels := map[string]string{"edge-net.io/generated": "true"}
		roleBind.SetLabels(roleBindLabels)
		if _, err := c.kubeclientset.RbacV1().RoleBindings(propagationNamespace).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil {
			if !errors.IsAlreadyExists(err) {
				klog.Infoln(err)
				return err
			}
			if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(propagationNamespace).Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{}); err == nil {
				roleBinding.RoleRef = roleBind.RoleRef
				roleBinding.Subjects = roleBind.Subjects
				roleBinding.SetLabels(roleBind.GetLabels())
				if _, err := c.kubeclientset.RbacV1().RoleBindings(propagationNamespace).Update(context.TODO(), roleBinding, metav1.UpdateOptions{}); err != nil {
					klog.Infoln(err)
					return err
				}
			} else {
				klog.Infoln(err)
				return err
			}
		}
	}
	return nil
}

func (c *Controller) syncCaches(clusterRaw *federationv1alpha1.ClusterList, localCacheMap map[string]federationv1alpha1.ManagerCache, clusterUID string, local bool) (map[string]federationv1alpha1.ManagerCache, error) {
	for _, clusterRow := range clusterRaw.Items {
		if clusterRow.Spec.Role == federationv1alpha1.FederationManagerRole || clusterRow.Spec.Role == federationv1alpha1.PeerRole {
			fedManagerSecret, err := c.kubeclientset.CoreV1().Secrets(clusterRow.GetNamespace()).Get(context.TODO(), clusterRow.Spec.SecretName, metav1.GetOptions{})
			if err != nil {
				klog.Infoln(err)
				return nil, err
			}
			config := bootstrap.PrepareRestConfig(clusterRow.Spec.Server, string(fedManagerSecret.Data["token"]), fedManagerSecret.Data["ca.crt"])
			remoteedgeclientset, err := bootstrap.CreateEdgeNetClientset(config)
			if err != nil {
				klog.Infoln(err)
				return nil, err
			}
			remotemanagercacheRaw, err := remoteedgeclientset.FederationV1alpha1().ManagerCaches().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				klog.Infoln(err)
				return nil, err
			}
			if local {
				localCacheMap, err = c.syncLocalCaches(remotemanagercacheRaw, localCacheMap, clusterUID)
				if err != nil {
					klog.Infoln(err)
					return nil, err
				}
			} else {
				remoteCacheMap := c.constructCacheMap(remotemanagercacheRaw)
				if err := syncRemoteCaches(remoteedgeclientset, localCacheMap, remoteCacheMap, clusterUID); err != nil {
					klog.Infoln(err)
					return nil, err
				}
			}
		}
	}
	return localCacheMap, nil
}

func (c *Controller) syncLocalCaches(remotecacheRaw *federationv1alpha1.ManagerCacheList, cacheMap map[string]federationv1alpha1.ManagerCache, clusterUID string) (map[string]federationv1alpha1.ManagerCache, error) {
	var err error
	for _, remotecacheRow := range remotecacheRaw.Items {
		if remotecacheRow.GetName() == clusterUID {
			continue
		}
		if localCache, ok := cacheMap[remotecacheRow.GetName()]; ok {
			if remotecacheRow.Spec.LatestUpdateTimestamp.After(localCache.Spec.LatestUpdateTimestamp.Time) {
				localCache, err = updateCache(c.edgenetclientset, remotecacheRow, localCache)
				if err != nil {
					klog.Infoln(err)
					return nil, err
				}
				cacheMap[remotecacheRow.GetName()] = localCache
			}
		} else {
			newManagerCache := new(federationv1alpha1.ManagerCache)
			newManagerCache.SetName(remotecacheRow.GetName())
			newManagerCache.Spec = remotecacheRow.Spec
			newCache, err := createCache(c.edgenetclientset, newManagerCache, remotecacheRow.Status)
			if err != nil {
				klog.Infoln(err)
				return nil, err
			}
			cacheMap[remotecacheRow.GetName()] = *newCache
		}
	}
	return cacheMap, nil
}

// updateStatus calls the API to update the managercache status.
func (c *Controller) updateStatus(ctx context.Context, managercacheCopy *federationv1alpha1.ManagerCache, state, message string) {
	managercacheCopy.Status.State = state
	managercacheCopy.Status.Message = message
	if managercacheCopy.Status.State == federationv1alpha1.StatusFailed {
		managercacheCopy.Status.Failed++
		now := metav1.Now()
		managercacheCopy.Status.UpdateTimestamp = &now
	}
	if _, err := c.edgenetclientset.FederationV1alpha1().ManagerCaches().UpdateStatus(ctx, managercacheCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}

func syncRemoteCaches(client clientset.Interface, localCacheMap, remoteCacheMap map[string]federationv1alpha1.ManagerCache, clusterUID string) error {
	for localCacheName, localCache := range localCacheMap {
		if remoteCache, ok := remoteCacheMap[localCacheName]; ok {
			if remoteCache.Spec.LatestUpdateTimestamp.Before(localCache.Spec.LatestUpdateTimestamp) {
				if localCacheName == clusterUID && localCache.Status.State == federationv1alpha1.StatusReconciliation {
					localCache.Status.State = federationv1alpha1.StatusReady
					localCache.Status.Message = messageReady
				}
				updateCache(client, localCache, remoteCache)
			}
		} else {
			newManagerCache := new(federationv1alpha1.ManagerCache)
			newManagerCache.SetName(localCacheName)
			newManagerCache.Spec = localCache.Spec
			newManagerStatus := localCache.Status
			if localCacheName == clusterUID && localCache.Status.State == federationv1alpha1.StatusReconciliation {
				newManagerStatus.State = federationv1alpha1.StatusReady
				newManagerStatus.Message = messageReady
			}
			_, err := createCache(client, newManagerCache, newManagerStatus)
			if err != nil {
				klog.Infoln(err)
				return err
			}
		}

		/*klog.Infoln("clusterUID: ", clusterUID)
		klog.Infoln("managercacheCopy.GetName(): ", managercacheCopy.GetName())
		if managercacheCopy.GetName() == clusterUID {
			klog.Infoln("UIDs are equal")
			if err := c.updateChildManagerCache(managercacheCopy, clusterRow.Spec.Server, childFedManagerSecret.Data["token"], childFedManagerSecret.Data["ca.crt"]); err != nil {
				klog.Infoln(err)
				return err
			}
		}*/
	}
	return nil
}

func createCache(client clientset.Interface, managercache *federationv1alpha1.ManagerCache, status federationv1alpha1.ManagerCacheStatus) (*federationv1alpha1.ManagerCache, error) {
	if _, err := client.FederationV1alpha1().ManagerCaches().Create(context.TODO(), managercache, metav1.CreateOptions{}); err != nil {
		klog.Infoln(err)
		return nil, err
	}
	managerCacheCopy, err := client.FederationV1alpha1().ManagerCaches().Get(context.TODO(), managercache.GetName(), metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return nil, err
	}
	managerCacheCopy.Status = status
	if _, err := client.FederationV1alpha1().ManagerCaches().UpdateStatus(context.TODO(), managerCacheCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
		return nil, err
	}
	return managerCacheCopy, nil
}

func updateCache(client clientset.Interface, sotManagercache, currentManagercache federationv1alpha1.ManagerCache) (federationv1alpha1.ManagerCache, error) {
	// `sot` means source of truth
	if !reflect.DeepEqual(currentManagercache.Spec, sotManagercache.Spec) {
		currentManagercache.Spec = sotManagercache.Spec
		if _, err := client.FederationV1alpha1().ManagerCaches().Update(context.TODO(), &currentManagercache, metav1.UpdateOptions{}); err != nil {
			klog.Infoln(err)
			return currentManagercache, err
		}
	}
	if !reflect.DeepEqual(currentManagercache.Status, sotManagercache.Status) {
		if currentManagerCacheCopy, err := client.FederationV1alpha1().ManagerCaches().Get(context.TODO(), currentManagercache.GetName(), metav1.GetOptions{}); err == nil {
			currentManagerCacheCopy.Status = sotManagercache.Status
			if _, err := client.FederationV1alpha1().ManagerCaches().UpdateStatus(context.TODO(), currentManagerCacheCopy, metav1.UpdateOptions{}); err != nil {
				klog.Infoln(err)
				return currentManagercache, err
			}
			currentManagercache.Status = sotManagercache.Status
		}
	}
	return currentManagercache, nil
}
