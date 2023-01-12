/*
Copyright 2021 Contributors to the EdgeNet project.

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

package tenant

import (
	"context"
	"fmt"
	"time"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/multitenancy"

	antreav1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	antrea "antrea.io/antrea/pkg/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "tenant-controller"

// Definitions of the state of the tenant resource
const (
	backoffLimit = 3

	successSynced        = "Synced"
	failureCreation      = "Not Created"
	failureBinding       = "Binding Failed"
	failureNetworkPolicy = "Not Applied"
	failureDeletion      = "Not Removed"

	messageResourceSynced                   = "Tenant synced successfully"
	messageEstablished                      = "Tenant established successfully"
	messageCreated                          = "Core namespace created successfully"
	messageCreationFailed                   = "Core namespace creation failed"
	messageBindingFailed                    = "Role binding failed"
	messageNetworkPolicyFailed              = "Applying network policy failed"
	messageSliceClaimDeletionFailed         = "Slice claim clean up failed"
	messageSubNamespaceDeletionFailed       = "Subsidiary namespace clean up failed"
	messageClusterRoleDeletionFailed        = "Cluster role clean up failed"
	messageClusterRoleBindingDeletionFailed = "Cluster role binding clean up failed"
	messageRoleBindingDeletionFailed        = "Role binding clean up failed"
	messageRoleBindingCreationFailed        = "Role binding creation for tenant failed"
	messageReconciliation                   = "Reconciliation in progress"
)

// Controller is the controller implementation for Tenant resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface
	antreaclientset  antrea.Interface

	tenantsLister listers.TenantLister
	tenantsSynced cache.InformerSynced

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
	antreaclientset antrea.Interface,
	tenantInformer informers.TenantInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.Infoln("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:    kubeclientset,
		edgenetclientset: edgenetclientset,
		antreaclientset:  antreaclientset,
		tenantsLister:    tenantInformer.Lister(),
		tenantsSynced:    tenantInformer.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Tenants"),
		recorder:         recorder,
	}

	klog.Infoln("Setting up event handlers")
	tenantInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueTenant,
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueueTenant(newObj)
		},
	})

	return controller
}

// Run will set up the event handlers for the types of tenant, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting Tenant controller")

	klog.Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.tenantsSynced); !ok {
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
// converge the two. It then updates the Status block of the Tenant
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	tenant, err := c.tenantsLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("tenant '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processTenant(tenant.DeepCopy())

	c.recorder.Event(tenant, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueTenant takes a Tenant resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Tenant.
func (c *Controller) enqueueTenant(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) processTenant(tenantCopy *corev1alpha1.Tenant) {
	systemNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return
	}
	if exceedsBackoffLimit := tenantCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
		c.cleanup(tenantCopy, string(systemNamespace.GetUID()))
		return
	}

	if tenantCopy.Spec.Enabled {
		// When a tenant is deleted, the owner references feature drives the namespace to be automatically removed
		ownerReferences := []metav1.OwnerReference{tenantCopy.MakeOwnerReference()}
		switch tenantCopy.Status.State {
		case corev1alpha1.StatusEstablished:
			c.reconcile(tenantCopy)
		case corev1alpha1.StatusCoreNamespaceCreated:
			// Apply network policies
			if err := c.applyNetworkPolicy(tenantCopy.GetName(), string(tenantCopy.GetUID()), string(systemNamespace.GetUID()), tenantCopy.Spec.ClusterNetworkPolicy, ownerReferences); err != nil {
				c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureNetworkPolicy, messageNetworkPolicyFailed)
				tenantCopy.Status.State = corev1alpha1.StatusFailed
				tenantCopy.Status.Message = messageNetworkPolicyFailed
				c.updateStatus(context.TODO(), tenantCopy)
				return
			}
			// Deliver required permissions to the tenant owner
			if err := c.configureOwnerPermissions(tenantCopy); err != nil {
				return
			}
			c.recorder.Event(tenantCopy, corev1.EventTypeNormal, corev1alpha1.StatusEstablished, messageEstablished)
			tenantCopy.Status.State = corev1alpha1.StatusEstablished
			tenantCopy.Status.Message = messageEstablished
			c.updateStatus(context.TODO(), tenantCopy)
		default:
			// Create the core namespace
			if err = c.makeCoreNamespace(tenantCopy, ownerReferences, string(systemNamespace.GetUID())); err != nil {
				c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureCreation, messageCreationFailed)
				tenantCopy.Status.State = corev1alpha1.StatusFailed
				tenantCopy.Status.Message = messageCreationFailed
				c.updateStatus(context.TODO(), tenantCopy)
				return
			}
			// Create the cluster role and role binding for the tenant resource
			multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
			if err := multitenancyManager.GrantObjectOwnership("core.edgenet.io", "tenants", tenantCopy.GetName(), tenantCopy.Spec.Contact.Email, ownerReferences); err != nil {
				c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureCreation, messageRoleBindingCreationFailed)
				tenantCopy.Status.State = corev1alpha1.StatusFailed
				tenantCopy.Status.Message = messageRoleBindingCreationFailed
				c.updateStatus(context.TODO(), tenantCopy)
				return
			}
			c.recorder.Event(tenantCopy, corev1.EventTypeNormal, corev1alpha1.StatusCoreNamespaceCreated, messageCreated)
			tenantCopy.Status.State = corev1alpha1.StatusCoreNamespaceCreated
			tenantCopy.Status.Message = messageCreated
			c.updateStatus(context.TODO(), tenantCopy)
		}
	} else {
		c.cleanup(tenantCopy, string(systemNamespace.GetUID()))
	}
}

func (c *Controller) reconcile(tenantCopy *corev1alpha1.Tenant) {
	// Reconcile with the owner permissions in the core namespace
	if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(tenantCopy.GetName()).Get(context.TODO(), corev1alpha1.TenantOwnerClusterRoleName, metav1.GetOptions{}); err != nil {
		tenantCopy.Status.State = corev1alpha1.StatusCoreNamespaceCreated
		tenantCopy.Status.Message = messageCreated
	} else {
		if roleBinding.RoleRef.Kind == "ClusterRole" && roleBinding.RoleRef.Name == corev1alpha1.TenantOwnerClusterRoleName {
			isConsiled := false
			for _, subject := range roleBinding.Subjects {
				if subject.Kind == "User" && subject.Name == tenantCopy.Spec.Contact.Email {
					isConsiled = true
				}
			}
			if !isConsiled {
				tenantCopy.Status.State = corev1alpha1.StatusCoreNamespaceCreated
				tenantCopy.Status.Message = messageCreated
			}
		}
	}
	// Reconcile with the network policies
	if _, err := c.kubeclientset.NetworkingV1().NetworkPolicies(tenantCopy.GetName()).Get(context.TODO(), "baseline", metav1.GetOptions{}); err != nil {
		tenantCopy.Status.State = corev1alpha1.StatusCoreNamespaceCreated
		tenantCopy.Status.Message = messageCreated
	}
	if _, err := c.antreaclientset.CrdV1alpha1().ClusterNetworkPolicies().Get(context.TODO(), tenantCopy.GetName(), metav1.GetOptions{}); (err != nil && tenantCopy.Spec.ClusterNetworkPolicy) || (err == nil && !tenantCopy.Spec.ClusterNetworkPolicy) {
		tenantCopy.Status.State = corev1alpha1.StatusCoreNamespaceCreated
		tenantCopy.Status.Message = messageCreated
	}
	// Reconcile with the core namespace and the associated permissions of the tenant resource
	if _, err := c.kubeclientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("edgenet:tenants:%s-owner", tenantCopy.GetName()), metav1.GetOptions{}); err != nil {
		tenantCopy.Status.State = corev1alpha1.StatusReconciliation
		tenantCopy.Status.Message = messageReconciliation
	}
	if _, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), tenantCopy.GetName(), metav1.GetOptions{}); err != nil {
		tenantCopy.Status.State = corev1alpha1.StatusReconciliation
		tenantCopy.Status.Message = messageReconciliation
	}

	if tenantCopy.Status.State != corev1alpha1.StatusEstablished {
		c.updateStatus(context.TODO(), tenantCopy)
	}
}

func (c *Controller) makeCoreNamespace(tenantCopy *corev1alpha1.Tenant, ownerReferences []metav1.OwnerReference, clusterUID string) error {
	// Core namespace has the same name as the tenant
	coreNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantCopy.GetName(), OwnerReferences: ownerReferences}}
	// Namespace labels indicate this namespace created by a tenant, not by a team or slice
	labels := map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenantCopy.GetName(),
		"edge-net.io/tenant-uid": string(tenantCopy.GetUID()), "edge-net.io/cluster-uid": clusterUID}
	coreNamespace.SetLabels(labels)
	annotations := map[string]string{"scheduler.alpha.kubernetes.io/node-selector": "edge-net.io/access=public,edge-net.io/slice=none"}
	if nodeSelector, elementExists := tenantCopy.GetAnnotations()["scheduler.alpha.kubernetes.io/node-selector"]; elementExists {
		annotations["scheduler.alpha.kubernetes.io/node-selector"] = nodeSelector
	}
	coreNamespace.SetAnnotations(annotations)
	if _, err := c.kubeclientset.CoreV1().Namespaces().Create(context.TODO(), coreNamespace, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if namespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), coreNamespace.GetName(), metav1.GetOptions{}); err == nil {
				namespace.SetLabels(labels)
				namespace.SetAnnotations(annotations)
				namespace.SetOwnerReferences(ownerReferences)
				if _, err := c.kubeclientset.CoreV1().Namespaces().Update(context.TODO(), namespace, metav1.UpdateOptions{}); err == nil {
					return nil
				}
			}
		}
		return err
	}
	return nil
}

func (c *Controller) configureOwnerPermissions(tenantCopy *corev1alpha1.Tenant) error {
	roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: corev1alpha1.TenantOwnerClusterRoleName}
	rbSubjects := []rbacv1.Subject{{Kind: "User", Name: tenantCopy.Spec.Contact.Email, APIGroup: "rbac.authorization.k8s.io"}}
	roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: corev1alpha1.TenantOwnerClusterRoleName, Namespace: tenantCopy.GetName()},
		Subjects: rbSubjects, RoleRef: roleRef}
	roleBindLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/notification": "true"}
	roleBind.SetLabels(roleBindLabels)
	if _, err := c.kubeclientset.RbacV1().RoleBindings(tenantCopy.GetName()).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(tenantCopy.GetName()).Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{}); err == nil {
				roleBindingCopy := roleBinding.DeepCopy()
				roleBindingCopy.RoleRef = roleBind.RoleRef
				roleBindingCopy.Subjects = roleBind.Subjects
				roleBindingCopy.SetLabels(roleBind.GetLabels())
				if _, err := c.kubeclientset.RbacV1().RoleBindings(tenantCopy.GetName()).Update(context.TODO(), roleBindingCopy, metav1.UpdateOptions{}); err == nil {
					return nil
				}
			}
		}
		c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
		tenantCopy.Status.State = corev1alpha1.StatusFailed
		tenantCopy.Status.Message = messageBindingFailed
		c.updateStatus(context.TODO(), tenantCopy)
		return err
	}
	return nil
}

func (c *Controller) applyNetworkPolicy(tenant, tenantUID, clusterUID string, clusterNetworkPolicyEnabled bool, ownerReferences []metav1.OwnerReference) error {
	// TODO: Apply a network policy to the core namespace according to spec
	// Restricted only allows intra-tenant communication
	// Baseline allows intra-tenant communication plus ingress from external traffic
	// Privileged allows all kind of traffics
	var err error
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"edge-net.io/subtenant":   "false",
			"edge-net.io/tenant":      tenant,
			"edge-net.io/tenant-uid":  tenantUID,
			"edge-net.io/cluster-uid": clusterUID,
		},
	}
	port := intstr.IntOrString{IntVal: 1}
	endPort := int32(32768)
	networkPolicy := new(networkingv1.NetworkPolicy)
	networkPolicy.SetName("baseline")
	networkPolicy.SetNamespace(tenant)
	networkPolicy.Spec.PolicyTypes = []networkingv1.PolicyType{"Ingress"}
	networkPolicy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
		{
			From: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &labelSelector,
				},
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR:   "0.0.0.0/0",
						Except: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Port:    &port,
					EndPort: &endPort,
				},
			},
		},
	}
	if _, err = c.kubeclientset.NetworkingV1().NetworkPolicies(tenant).Create(context.TODO(), networkPolicy, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	if clusterNetworkPolicyEnabled {
		drop := antreav1alpha1.RuleActionDrop
		allow := antreav1alpha1.RuleActionAllow
		clusterNetworkPolicy := new(antreav1alpha1.ClusterNetworkPolicy)
		clusterNetworkPolicy.SetName(tenant)
		clusterNetworkPolicy.SetOwnerReferences(ownerReferences)
		clusterNetworkPolicy.Spec.Tier = "tenant"
		clusterNetworkPolicy.Spec.Priority = 5
		clusterNetworkPolicy.Spec.Ingress = []antreav1alpha1.Rule{
			{
				Action: &allow,
				From: []antreav1alpha1.NetworkPolicyPeer{
					{
						NamespaceSelector: &labelSelector,
					},
				},
				Ports: []antreav1alpha1.NetworkPolicyPort{
					{
						Port:    &port,
						EndPort: &endPort,
					},
				},
			},
			{
				Action: &drop,
				From: []antreav1alpha1.NetworkPolicyPeer{
					{
						IPBlock: &antreav1alpha1.IPBlock{
							CIDR: "10.0.0.0/8",
						},
					},
					{
						IPBlock: &antreav1alpha1.IPBlock{
							CIDR: "172.16.0.0/12",
						},
					},
					{
						IPBlock: &antreav1alpha1.IPBlock{
							CIDR: "192.168.0.0/16",
						},
					},
				},
				Ports: []antreav1alpha1.NetworkPolicyPort{
					{
						Port:    &port,
						EndPort: &endPort,
					},
				},
			},
			{
				Action: &allow,
				From: []antreav1alpha1.NetworkPolicyPeer{
					{
						IPBlock: &antreav1alpha1.IPBlock{
							CIDR: "0.0.0.0/0",
						},
					},
				},
				Ports: []antreav1alpha1.NetworkPolicyPort{
					{
						Port:    &port,
						EndPort: &endPort,
					},
				},
			},
		}
		clusterNetworkPolicy.Spec.AppliedTo = []antreav1alpha1.NetworkPolicyPeer{
			{
				NamespaceSelector: &labelSelector,
			},
		}
		if _, err = c.antreaclientset.CrdV1alpha1().ClusterNetworkPolicies().Create(context.TODO(), clusterNetworkPolicy, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	} else {
		c.antreaclientset.CrdV1alpha1().ClusterNetworkPolicies().Delete(context.TODO(), tenant, metav1.DeleteOptions{})
	}
	return nil
}

func (c *Controller) cleanup(tenantCopy *corev1alpha1.Tenant, clusterUID string) {
	// Delete all roles, role bindings, slices and subsidiary namespaces
	if err := c.kubeclientset.RbacV1().ClusterRoles().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s,edge-net.io/tenant-uid=%s,edge-net.io/cluster-uid=%s", tenantCopy.GetName(), string(tenantCopy.GetUID()), clusterUID)}); err != nil {
		c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureDeletion, messageClusterRoleDeletionFailed)
	}
	if err := c.kubeclientset.RbacV1().ClusterRoleBindings().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s,edge-net.io/tenant-uid=%s,edge-net.io/cluster-uid=%s", tenantCopy.GetName(), string(tenantCopy.GetUID()), clusterUID)}); err != nil {
		c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureDeletion, messageClusterRoleBindingDeletionFailed)
	}
	if err := c.kubeclientset.RbacV1().RoleBindings(tenantCopy.GetName()).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
		c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureDeletion, messageRoleBindingDeletionFailed)
	}
	if err := c.edgenetclientset.CoreV1alpha1().SliceClaims(tenantCopy.GetName()).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
		c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureDeletion, messageSliceClaimDeletionFailed)
	}
	if err := c.edgenetclientset.CoreV1alpha1().SubNamespaces(tenantCopy.GetName()).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
		c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureDeletion, messageSubNamespaceDeletionFailed)
	}
}

// updateStatus calls the API to update the tenant status.
func (c *Controller) updateStatus(ctx context.Context, tenantCopy *corev1alpha1.Tenant) {
	if tenantCopy.Status.State == corev1alpha1.StatusFailed {
		tenantCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.CoreV1alpha1().Tenants().UpdateStatus(ctx, tenantCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
