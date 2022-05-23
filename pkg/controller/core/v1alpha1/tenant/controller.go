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
	"reflect"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha1"

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
	successSynced                           = "Synced"
	messageResourceSynced                   = "Tenant synced successfully"
	successEstablished                      = "Established"
	messageEstablished                      = "Tenant established successfully"
	failureCreation                         = "Not Created"
	messageCreationFailed                   = "Core namespace creation failed"
	failureBinding                          = "Binding Failed"
	messageBindingFailed                    = "Role binding failed"
	failureNetworkPolicy                    = "Not Applied"
	messageNetworkPolicyFailed              = "Applying network policy failed"
	failureSubNamespaceDeletion             = "Not Removed"
	messageSubNamespaceDeletionFailed       = "Subsidiary namespace clean up failed"
	failureClusterRoleDeletion              = "Not Removed"
	messageClusterRoleDeletionFailed        = "Cluster role clean up failed"
	failureClusterRoleBindingDeletion       = "Not Removed"
	messageClusterRoleBindingDeletionFailed = "Cluster role binding clean up failed"
	failureRoleBindingDeletion              = "Not Removed"
	messageRoleBindingDeletionFailed        = "Role binding clean up failed"
	failureRoleBindingCreation              = "Not Created"
	messageRoleBindingCreationFailed        = "Role binding creation for tenant failed"
	failure                                 = "Failure"
	pending                                 = "Pending"
	established                             = "Established"
)

// The main structure of controller
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

	access.Clientset = kubeclientset
	access.EdgenetClientset = edgenetclientset

	access.CreateClusterRoles()

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

	c.ProcessTenant(tenant.DeepCopy())

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

func (c *Controller) ProcessTenant(tenantCopy *corev1alpha1.Tenant) {
	oldStatus := tenantCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, tenantCopy.Status) {
			if _, err := c.edgenetclientset.CoreV1alpha1().Tenants().UpdateStatus(context.TODO(), tenantCopy, metav1.UpdateOptions{}); err != nil {
				klog.Infoln(err)
			}
		}
	}
	defer statusUpdate()

	systemNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		klog.Infoln(err)
		return
	}

	if tenantCopy.Spec.Enabled {
		// When a tenant is deleted, the owner references feature drives the namespace to be automatically removed
		ownerReferences := []metav1.OwnerReference{tenantCopy.MakeOwnerReference()}
		// Create the cluster roles
		tenantOwnerClusterRole, err := access.CreateObjectSpecificClusterRole(tenantCopy.GetName(), "core.edgenet.io", "tenants", tenantCopy.GetName(), "owner", []string{"get", "update", "patch"}, ownerReferences)
		if err != nil && !errors.IsAlreadyExists(err) {
			klog.Infof("Couldn't create owner cluster role %s: %s", tenantCopy.GetName(), err)
			// TODO: Provide err information at the EVENTS
		}
		err = c.createCoreNamespace(tenantCopy, ownerReferences, string(systemNamespace.GetUID()))
		if err == nil || errors.IsAlreadyExists(err) {
			// Apply network policies
			err = c.applyNetworkPolicy(tenantCopy.GetName(), string(tenantCopy.GetUID()), string(systemNamespace.GetUID()), tenantCopy.Spec.ClusterNetworkPolicy, ownerReferences)
			if err != nil && !errors.IsAlreadyExists(err) {
				c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureNetworkPolicy, messageNetworkPolicyFailed)
			}

			// Cluster role binding
			if err := access.CreateObjectSpecificClusterRoleBinding(tenantOwnerClusterRole, tenantCopy.Spec.Contact.Email, map[string]string{"edge-net.io/generated": "true"}, []metav1.OwnerReference{}); err != nil {
				c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureRoleBindingCreation, messageRoleBindingCreationFailed)
			}
			// Role binding
			clusterRoleName := "edgenet:tenant-owner"
			roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: clusterRoleName}
			rbSubjects := []rbacv1.Subject{{Kind: "User", Name: tenantCopy.Spec.Contact.Email, APIGroup: "rbac.authorization.k8s.io"}}
			roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName, Namespace: tenantCopy.GetName()},
				Subjects: rbSubjects, RoleRef: roleRef}
			roleBindLabels := map[string]string{"edge-net.io/generated": "true"}
			roleBind.SetLabels(roleBindLabels)
			if _, err := c.kubeclientset.RbacV1().RoleBindings(tenantCopy.GetName()).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
				c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureBinding, messageBindingFailed)
				tenantCopy.Status.State = failure
				tenantCopy.Status.Message = messageBindingFailed
				klog.Infoln(err)
			} else if errors.IsAlreadyExists(err) {
				if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(tenantCopy.GetName()).Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{}); err == nil {
					roleBindingCopy := roleBinding.DeepCopy()
					roleBindingCopy.RoleRef = roleBind.RoleRef
					roleBindingCopy.Subjects = roleBind.Subjects
					roleBindingCopy.SetLabels(roleBind.GetLabels())
					c.kubeclientset.RbacV1().RoleBindings(tenantCopy.GetName()).Update(context.TODO(), roleBindingCopy, metav1.UpdateOptions{})
				}
			} else {
				c.recorder.Event(tenantCopy, corev1.EventTypeNormal, successEstablished, messageEstablished)
				tenantCopy.Status.State = established
				tenantCopy.Status.Message = successEstablished
			}
		}
	} else {
		// Delete all subsidiary namespaces
		if namespaceRaw, err := c.kubeclientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s,edge-net.io/tenant-uid=%s,edge-net.io/cluster-uid=%s,edge-net.io/kind=sub", tenantCopy.GetName(), string(tenantCopy.GetUID()), string(systemNamespace.GetUID()))}); err == nil {
			for _, namespaceRow := range namespaceRaw.Items {
				c.kubeclientset.CoreV1().Namespaces().Delete(context.TODO(), namespaceRow.GetName(), metav1.DeleteOptions{})
			}
		} else {
			c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureSubNamespaceDeletion, messageSubNamespaceDeletionFailed)
		}
		// Delete all roles, role bindings, and subsidiary namespaces
		if err := c.kubeclientset.RbacV1().ClusterRoles().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s,edge-net.io/tenant-uid=%s,edge-net.io/cluster-uid=%s", tenantCopy.GetName(), string(tenantCopy.GetUID()), string(systemNamespace.GetUID()))}); err != nil {
			c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureClusterRoleDeletion, messageClusterRoleDeletionFailed)
		}
		if err := c.kubeclientset.RbacV1().ClusterRoleBindings().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s,edge-net.io/tenant-uid=%s,edge-net.io/cluster-uid=%s", tenantCopy.GetName(), string(tenantCopy.GetUID()), string(systemNamespace.GetUID()))}); err != nil {
			c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureClusterRoleBindingDeletion, messageClusterRoleBindingDeletionFailed)
		}
		if err := c.kubeclientset.RbacV1().RoleBindings(tenantCopy.GetName()).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureRoleBindingDeletion, messageRoleBindingDeletionFailed)
		}
	}
}

func (c *Controller) createCoreNamespace(tenantCopy *corev1alpha1.Tenant, ownerReferences []metav1.OwnerReference, clusterUID string) error {
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
	_, err := c.kubeclientset.CoreV1().Namespaces().Create(context.TODO(), coreNamespace, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		c.recorder.Event(tenantCopy, corev1.EventTypeWarning, failureCreation, messageCreationFailed)
		tenantCopy.Status.State = failure
		tenantCopy.Status.Message = messageCreationFailed
		return err
	} else if errors.IsAlreadyExists(err) {
		if namespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), coreNamespace.GetName(), metav1.GetOptions{}); err == nil {
			namespace.SetLabels(labels)
			namespace.SetAnnotations(annotations)
			c.kubeclientset.CoreV1().Namespaces().Update(context.TODO(), namespace, metav1.UpdateOptions{})
		}
		return nil
	} else {
		return nil
	}
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
	_, err = c.kubeclientset.NetworkingV1().NetworkPolicies(tenant).Create(context.TODO(), networkPolicy, metav1.CreateOptions{})
	klog.Infoln(err)
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

		_, err = c.antreaclientset.CrdV1alpha1().ClusterNetworkPolicies().Create(context.TODO(), clusterNetworkPolicy, metav1.CreateOptions{})
		klog.Infoln(err)
	} else {
		c.antreaclientset.CrdV1alpha1().ClusterNetworkPolicies().Delete(context.TODO(), tenant, metav1.DeleteOptions{})
	}
	return err
}
