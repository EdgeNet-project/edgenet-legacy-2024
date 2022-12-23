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
	"os"
	"strings"
	"time"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/federation/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/federation/v1alpha1"
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
	successSynced = "Synced"

	messageResourceSynced          = "Cluster synced successfully"
	messageCredsPrepared           = "Credentials for federation manager access prepared successfully"
	messageSubnamespaceCreated     = "Subnamespace for object propagation created successfully"
	messageSubnamespaceFailed      = "Subnamespace for object propagation creation failed"
	messageBindingFailed           = "Role binding failed"
	messageMissingSecretAtRemote   = "Secret storing federation managers's token is missing in the remote cluster"
	messageWrongSecretAtRemote     = "Secret storing federation manager's token is wrong in the remote cluster"
	messageMissingSecretFMAuth     = "Secret storing federation manager's token is missing in the federation manager"
	messageMissingSecretRemoteAuth = "Secret storing remote cluster's token is missing in the federation manager"
	messageServiceAccountFailed    = "Service account creation failed"
	messageAuthSecretFailed        = "Secret storing federation manager's token cannot be created"
	messageRemoteClientFailed      = "Clientset for remote cluster cannot be created"
	messageReady                   = "Inter-cluster communication is established"
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
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
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
	multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
	permitted, _, namespaceLabels := multitenancyManager.EligibilityCheck(clusterCopy.GetNamespace())
	if permitted {
		propagationNamespace := fmt.Sprintf(federationv1alpha1.FederationManagerNamespace, namespaceLabels["edge-net.io/cluster-uid"])
		switch clusterCopy.Status.State {
		case federationv1alpha1.StatusReady:
			// Add manager cache to reconcile
			c.reconcile(clusterCopy, propagationNamespace, namespaceLabels["edge-net.io/cluster-uid"])
		case federationv1alpha1.StatusCredsPrepared:
			// Create the remote clientset
			remotekubeclientset, err := c.createRemoteKubeClientset(clusterCopy)
			if err != nil {
				c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusFailed, messageRemoteClientFailed)
				clusterCopy.Status.State = federationv1alpha1.StatusFailed
				clusterCopy.Status.Message = messageRemoteClientFailed
				c.updateStatus(context.TODO(), clusterCopy)
				return
			}
			remoteSecret, err := c.deployTokenToRemoteCluster(clusterCopy, propagationNamespace, namespaceLabels["edge-net.io/cluster-uid"])
			if err != nil {
				return
			}
			remotekubeclientset.CoreV1().Secrets(remoteSecret.GetNamespace()).Create(context.TODO(), remoteSecret, metav1.CreateOptions{})

			if clusterCopy.Spec.Role == "Federation" {
				managerCache, _ := c.edgenetclientset.FederationV1alpha1().ManagerCaches().Get(context.TODO(), namespaceLabels["edge-net.io/cluster-uid"], metav1.GetOptions{})
				managerCache.Spec.Hierarchy.Children = append(managerCache.Spec.Hierarchy.Children, clusterCopy.Spec.UID)
				c.edgenetclientset.FederationV1alpha1().ManagerCaches().Update(context.TODO(), managerCache, metav1.UpdateOptions{})

				remoteManagerCache := new(federationv1alpha1.ManagerCache)
				remoteManagerCache.SetName(clusterCopy.Spec.UID)
				remoteManagerCache.Spec.Hierarchy.Parent = namespaceLabels["edge-net.io/cluster-uid"]
				remoteManagerCache.Spec.Hierarchy.Level = managerCache.Spec.Hierarchy.Level + 1
				remoteedgeclientset, _ := c.createRemoteEdgeNetClientset(clusterCopy)
				remoteedgeclientset.FederationV1alpha1().ManagerCaches().Create(context.TODO(), remoteManagerCache, metav1.CreateOptions{})
			}

			c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusReady, messageReady)
			clusterCopy.Status.State = federationv1alpha1.StatusReady
			clusterCopy.Status.Message = messageReady
			c.updateStatus(context.TODO(), clusterCopy)
		default:
			/*if clusterCopy.Spec.Role == "Federation" {
				fedSubnamespace := new(corev1alpha1.SubNamespace)
				fedSubnamespace.SetName(clusterCopy.GetName())
				fedSubnamespace.SetNamespace(clusterCopy.GetNamespace())
				fedSubnamespace.Spec.Workspace = new(corev1alpha1.Workspace)
				fedSubnamespace.Spec.Workspace.Scope = "federated"
				if _, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(clusterCopy.GetNamespace()).Create(context.TODO(), fedSubnamespace, metav1.CreateOptions{}); err != nil {
					c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageSubnamespaceFailed)
					clusterCopy.Status.State = federationv1alpha1.StatusFailed
					clusterCopy.Status.Message = messageSubnamespaceFailed
					c.updateStatus(context.TODO(), clusterCopy)
				}
				c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusSubnamespaceCreated, messageSubnamespaceCreated)
			}*/

			if err := c.createTokenForRemoteCluster(clusterCopy, propagationNamespace); err != nil {
				return
			}
			c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusCredsPrepared, messageCredsPrepared)
			clusterCopy.Status.State = federationv1alpha1.StatusCredsPrepared
			clusterCopy.Status.Message = messageCredsPrepared
			c.updateStatus(context.TODO(), clusterCopy)
		}
	} else {
		c.edgenetclientset.FederationV1alpha1().Clusters(clusterCopy.GetNamespace()).Delete(context.TODO(), clusterCopy.GetName(), metav1.DeleteOptions{})
	}
}

func (c *Controller) reconcile(clusterCopy *federationv1alpha1.Cluster, propagationNamespace, fedmanagerUID string) {
	clusterSubnamespace, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(clusterCopy.GetNamespace()).Get(context.TODO(), clusterCopy.GetName(), metav1.GetOptions{})
	if err != nil || (err == nil && clusterSubnamespace.Status.Child == nil) || (err == nil && clusterSubnamespace.Status.State == corev1alpha1.StatusFailed) {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageSubnamespaceFailed)
		clusterCopy.Status.State = federationv1alpha1.StatusReconciliation
		clusterCopy.Status.Message = messageSubnamespaceFailed
	}
	if _, err := c.kubeclientset.CoreV1().ServiceAccounts(propagationNamespace).Get(context.TODO(), clusterCopy.Spec.UID, metav1.GetOptions{}); err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusReconciliation, messageServiceAccountFailed)
		clusterCopy.Status.State = federationv1alpha1.StatusReconciliation
		clusterCopy.Status.Message = messageServiceAccountFailed
	}
	authSecret, err := c.kubeclientset.CoreV1().Secrets(propagationNamespace).Get(context.TODO(), clusterCopy.Spec.UID, metav1.GetOptions{})
	if err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusFailed, messageMissingSecretFMAuth)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageMissingSecretFMAuth
	}
	authSecret.SetName("federation")
	authSecret.SetNamespace("edgenet")
	authSecret.Data["serviceaccount"] = []byte(fmt.Sprintf("system:serviceaccount:%s:%s", propagationNamespace, clusterCopy.Spec.UID))
	authSecret.Data["namespace"] = []byte(propagationNamespace)
	var authentication string
	if authentication = strings.TrimSpace(os.Getenv("AUTHENTICATION_STRATEGY")); authentication != "kubeconfig" {
		authentication = "serviceaccount"
	}
	config, err := bootstrap.GetRestConfig(authentication)
	if err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusFailed, messageMissingSecretFMAuth)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageMissingSecretFMAuth
	}
	authSecret.Data["server"] = []byte(config.Host)
	authSecret.Data["cluster-uid"] = []byte(fedmanagerUID)
	remotekubeclientset, err := c.createRemoteKubeClientset(clusterCopy)
	remoteSecretFMAuth, err := remotekubeclientset.CoreV1().Secrets(authSecret.GetNamespace()).Get(context.TODO(), authSecret.GetName(), metav1.GetOptions{})
	if err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusFailed, messageMissingSecretAtRemote)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageMissingSecretAtRemote
	}
	if bytes.Compare(authSecret.Data["username"], remoteSecretFMAuth.Data["username"]) != 0 || bytes.Compare(authSecret.Data["namespace"], remoteSecretFMAuth.Data["namespace"]) != 0 ||
		bytes.Compare(authSecret.Data["server"], remoteSecretFMAuth.Data["server"]) != 0 || bytes.Compare(authSecret.Data["token"], remoteSecretFMAuth.Data["token"]) != 0 || bytes.Compare(authSecret.Data["cluster-uid"], remoteSecretFMAuth.Data["cluster-uid"]) != 0 {
		c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusFailed, messageWrongSecretAtRemote)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageWrongSecretAtRemote
	}
	if clusterCopy.Status.State != federationv1alpha1.StatusReady {
		c.updateStatus(context.TODO(), clusterCopy)
	}
}

func (c *Controller) getPropagationNamespace(clusterCopy *federationv1alpha1.Cluster) *string {
	clusterSubnamespace, err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(clusterCopy.GetNamespace()).Get(context.TODO(), clusterCopy.GetName(), metav1.GetOptions{})
	if err != nil || (err == nil && clusterSubnamespace.Status.Child == nil) {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageSubnamespaceFailed)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageSubnamespaceFailed
		c.updateStatus(context.TODO(), clusterCopy)
		return nil
	}
	return clusterSubnamespace.Status.Child
}

// createTokenForRemoteCluster creates a service account, a secret, and required permissions for the remote cluster to access the federation manager
func (c *Controller) createTokenForRemoteCluster(clusterCopy *federationv1alpha1.Cluster, propagationNamespace string) error {
	serviceAccount := new(corev1.ServiceAccount)
	serviceAccount.SetName(clusterCopy.Spec.UID)
	serviceAccount.SetNamespace(propagationNamespace)
	if _, err := c.kubeclientset.CoreV1().ServiceAccounts(propagationNamespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{}); err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageServiceAccountFailed)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageServiceAccountFailed
		c.updateStatus(context.TODO(), clusterCopy)
		return err
	}
	authSecret := new(corev1.Secret)
	authSecret.Name = clusterCopy.Spec.UID
	authSecret.Namespace = propagationNamespace
	authSecret.Annotations = map[string]string{"kubernetes.io/service-account.name": serviceAccount.GetName()}
	if _, err := c.kubeclientset.CoreV1().Secrets(propagationNamespace).Create(context.TODO(), authSecret, metav1.CreateOptions{}); err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageAuthSecretFailed)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageAuthSecretFailed
		c.updateStatus(context.TODO(), clusterCopy)
		return err
	}

	roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: federationv1alpha1.RemoteClusterRole}
	rbSubjects := []rbacv1.Subject{{Kind: "ServiceAccount", Name: serviceAccount.GetName(), Namespace: serviceAccount.GetNamespace()}}
	roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", federationv1alpha1.RemoteClusterRole, clusterCopy.Spec.UID), Namespace: serviceAccount.GetName()},
		Subjects: rbSubjects, RoleRef: roleRef}
	roleBindLabels := map[string]string{"edge-net.io/generated": "true"}
	roleBind.SetLabels(roleBindLabels)
	if _, err := c.kubeclientset.RbacV1().RoleBindings(propagationNamespace).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(propagationNamespace).Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{}); err == nil {
				roleBindingCopy := roleBinding.DeepCopy()
				roleBindingCopy.RoleRef = roleBind.RoleRef
				roleBindingCopy.Subjects = roleBind.Subjects
				roleBindingCopy.SetLabels(roleBind.GetLabels())
				if _, err := c.kubeclientset.RbacV1().RoleBindings(propagationNamespace).Update(context.TODO(), roleBindingCopy, metav1.UpdateOptions{}); err == nil {
					return nil
				}
			}
		}
		c.recorder.Event(clusterCopy, corev1.EventTypeWarning, federationv1alpha1.StatusFailed, messageBindingFailed)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageBindingFailed
		c.updateStatus(context.TODO(), clusterCopy)
		return err
	}
	return nil
}

func (c *Controller) deployTokenToRemoteCluster(clusterCopy *federationv1alpha1.Cluster, propagationNamespace, fedmanagerUID string) (*corev1.Secret, error) {
	authSecret, err := c.kubeclientset.CoreV1().Secrets(propagationNamespace).Get(context.TODO(), clusterCopy.Spec.UID, metav1.GetOptions{})
	if err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusFailed, messageMissingSecretFMAuth)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageMissingSecretFMAuth
		c.updateStatus(context.TODO(), clusterCopy)
		return nil, err
	}
	remoteSecret := new(corev1.Secret)
	remoteSecret.SetName("federation")
	remoteSecret.SetNamespace("edgenet")
	remoteSecret.Data["token"] = authSecret.Data["token"]
	remoteSecret.Data["serviceaccount"] = []byte(fmt.Sprintf("system:serviceaccount:%s:%s", propagationNamespace, clusterCopy.Spec.UID))
	remoteSecret.Data["namespace"] = []byte(propagationNamespace)
	var authentication string
	if authentication = strings.TrimSpace(os.Getenv("AUTHENTICATION_STRATEGY")); authentication != "kubeconfig" {
		authentication = "serviceaccount"
	}
	config, err := bootstrap.GetRestConfig(authentication)
	if err != nil {
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		c.updateStatus(context.TODO(), clusterCopy)
		return nil, err
	}
	remoteSecret.Data["server"] = []byte(config.Host)
	remoteSecret.Data["cluster-uid"] = []byte(fedmanagerUID)
	return remoteSecret, nil
}

func (c *Controller) createRemoteKubeClientset(clusterCopy *federationv1alpha1.Cluster) (*kubernetes.Clientset, error) {
	remoteAuthSecret, err := c.getSecretForRemoteClusterAuth(clusterCopy)
	if err != nil {
		return nil, err
	}
	// TODO: Check if the secret is valid
	remoteUsername := string(remoteAuthSecret.Data["username"])
	remoteToken := string(remoteAuthSecret.Data["token"])

	remoteConfig := new(rest.Config)
	remoteConfig.Host = clusterCopy.Spec.Server
	remoteConfig.Username = remoteUsername
	remoteConfig.BearerToken = remoteToken
	// Create the clientset
	remotekubeclientset, err := kubernetes.NewForConfig(remoteConfig)
	if err != nil {
		klog.Infoln(err)
	}
	return remotekubeclientset, nil
}

func (c *Controller) createRemoteEdgeNetClientset(clusterCopy *federationv1alpha1.Cluster) (*clientset.Clientset, error) {
	remoteAuthSecret, err := c.getSecretForRemoteClusterAuth(clusterCopy)
	if err != nil {
		return nil, err
	}
	// TODO: Check if the secret is valid
	remoteUsername := string(remoteAuthSecret.Data["username"])
	remoteToken := string(remoteAuthSecret.Data["token"])

	remoteConfig := new(rest.Config)
	remoteConfig.Host = clusterCopy.Spec.Server
	remoteConfig.Username = remoteUsername
	remoteConfig.BearerToken = remoteToken
	// Create the clientset
	remoteedgeclientset, err := clientset.NewForConfig(remoteConfig)
	if err != nil {
		klog.Infoln(err)
	}
	return remoteedgeclientset, nil
}

func (c *Controller) getSecretForRemoteClusterAuth(clusterCopy *federationv1alpha1.Cluster) (*corev1.Secret, error) {
	remoteAuthSecret, err := c.kubeclientset.CoreV1().Secrets(clusterCopy.GetNamespace()).Get(context.TODO(), clusterCopy.Spec.SecretName, metav1.GetOptions{})
	if err != nil {
		c.recorder.Event(clusterCopy, corev1.EventTypeNormal, federationv1alpha1.StatusFailed, messageMissingSecretRemoteAuth)
		clusterCopy.Status.State = federationv1alpha1.StatusFailed
		clusterCopy.Status.Message = messageMissingSecretRemoteAuth
		c.updateStatus(context.TODO(), clusterCopy)
		return nil, err
	}
	return remoteAuthSecret, nil
}

// updateStatus calls the API to update the cluster status.
func (c *Controller) updateStatus(ctx context.Context, clusterCopy *federationv1alpha1.Cluster) {
	if clusterCopy.Status.State == federationv1alpha1.StatusFailed {
		clusterCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.FederationV1alpha1().Clusters(clusterCopy.GetNamespace()).UpdateStatus(ctx, clusterCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
