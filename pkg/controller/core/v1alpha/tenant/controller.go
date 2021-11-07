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

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenantresourcequota"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/permission"
	"github.com/EdgeNet-project/edgenet/pkg/registration"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "tenant-controller"

// Constant variables for events
const create = "create"
const update = "update"
const delete = "delete"
const failure = "failure"
const approved = "approved"
const established = "established"

// The main structure of controller
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	tenantLister listers.TenantLister
	tenantSynced cache.InformerSynced

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

// Dictionary of status messages
var statusDict = map[string]string{
	"request-approved":                  "Tenant request has been approved",
	"tenant-established":                "Tenant successfully established",
	"namespace-failure":                 "Tenant core namespace cannot be created",
	"resource-quota-failure":            "Assigning tenant resource quota failed, user: %s",
	"aup-rolebinding-failure":           "AUP role binding creation failed, user: %s",
	"permission-rolebinding-failure":    "Permission role binding creation failed, user: %s",
	"administrator-rolebinding-failure": "Administrator role binding creation failed, user: %s",
	"user-failure":                      "User creation failed due to lack of labels, user: %s",
	"cert-failure":                      "Client cert generation failed, user: %s",
	"kubeconfig-failure":                "Kubeconfig file creation failed, user: %s",
}

func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	tenantInformer informers.TenantInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Infoln("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:    kubeclientset,
		edgenetclientset: edgenetclientset,
		tenantLister:     tenantInformer.Lister(),
		tenantSynced:     tenantInformer.Informer().HasSynced,
		workqueue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		recorder:         recorder,
	}

	klog.V(4).Infoln("Setting up event handlers")

	tenantInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueTenant,
		UpdateFunc: func(oldObj, newObj interface{}) {
			if reflect.DeepEqual(oldObj.(*corev1alpha.Tenant).Status, newObj.(*corev1alpha.Tenant).Status) {
				controller.enqueueTenant(newObj)
			}
		},
	})

	permission.Clientset = kubeclientset
	permission.EdgenetClientset = edgenetclientset

	registration.Clientset = kubeclientset
	registration.EdgenetClientset = edgenetclientset

	permission.CreateClusterRoles()

	return controller
}

// enqueueFoo takes a Tenant resource and converts it into a namespace/name
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

// Run will set up the event handlers for the types of tenant and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting Tenant controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.tenantSynced); !ok {
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

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// This function deals with the queue and sends each item in it to the specified handler to be processed.
func (c *Controller) processNextWorkItem() bool {
	// Fetch the item from workqueue
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
			utilruntime.HandleError(fmt.Errorf("expected `string` in workqueue but got %#v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.workqueue.Forget(key)
		klog.V(4).Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)

	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	tenant, err := c.tenantLister.Get(name)

	if err != nil {
		utilruntime.HandleError(fmt.Errorf("tenant '%s' in work queue no longer exists", name))

		// In this case we assume the resource is deleted
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	c.TuneTenant(tenant)
	return nil
}

func (c *Controller) TuneTenant(tenant *corev1alpha.Tenant) {
	klog.V(4).Infoln("TenantHandler.TuneTenant")

	tenantCopy := tenant.DeepCopy()

	if tenantCopy.Spec.Enabled {
		defer func() {
			if !reflect.DeepEqual(tenant.Status, tenantCopy.Status) {
				if _, err := c.edgenetclientset.CoreV1alpha().Tenants().UpdateStatus(context.TODO(), tenantCopy, metav1.UpdateOptions{}); err != nil {
					// TODO: Provide more information on error
					klog.V(4).Info(err)
				}
			}
		}()
		// When a tenant is deleted, the owner references feature allows the namespace to be automatically removed
		ownerReferences := SetAsOwnerReference(tenantCopy)
		err := c.createCoreNamespace(tenantCopy, ownerReferences)
		if err == nil || errors.IsAlreadyExists(err) {
			c.applyQuota(tenantCopy)

			// Reconfigure permissions
			if acceptableUsePolicyRaw, err := c.edgenetclientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s,edge-net.io/identity=true", tenant.GetName())}); err == nil {
				for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
					aupLabels := acceptableUsePolicyRow.GetLabels()
					if aupLabels != nil && aupLabels["edge-net.io/username"] != "" && aupLabels["edge-net.io/role"] != "" && aupLabels["edge-net.io/firstname"] != "" && aupLabels["edge-net.io/lastname"] != "" && aupLabels["edge-net.io/user-template-hash"] != "" {
						user := registrationv1alpha.UserRequest{}
						user.SetName(aupLabels["edge-net.io/username"])
						user.Spec.Tenant = tenantCopy.GetName()
						user.Spec.Email = acceptableUsePolicyRow.Spec.Email
						user.Spec.FirstName = aupLabels["edge-net.io/firstname"]
						user.Spec.LastName = aupLabels["edge-net.io/lastname"]
						user.Spec.Role = aupLabels["edge-net.io/role"]
						user.SetLabels(map[string]string{"edge-net.io/user-template-hash": aupLabels["edge-net.io/user-template-hash"]})
						permission.ConfigureTenantPermissions(tenantCopy, user.DeepCopy(), SetAsOwnerReference(tenantCopy))
					}
				}
			}

			// Create the cluster roles
			if err := permission.CreateObjectSpecificClusterRole(tenantCopy.GetName(), "core.edgenet.io", "tenants", tenantCopy.GetName(), "owner", []string{"get", "update", "patch"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
				klog.V(4).Infof("Couldn't create owner cluster role %s: %s", tenantCopy.GetName(), err)
				// TODO: Provide err information at the status
			}
			if err := permission.CreateObjectSpecificClusterRole(tenantCopy.GetName(), "core.edgenet.io", "tenants", tenantCopy.GetName(), "admin", []string{"get"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
				klog.V(4).Infof("Couldn't create admin cluster role %s: %s", tenantCopy.GetName(), err)
				// TODO: Provide err information at the status
			}
		}

		exists, _ := util.Contains(tenantCopy.Status.Message, statusDict["tenant-established"])
		if !exists && len(tenant.Status.Message) == 0 {
			tenantCopy.Status.State = established
			tenantCopy.Status.Message = []string{statusDict["tenant-established"]}
			permission.SendTenantEmail(tenantCopy, nil, "tenant-creation-successful")
		}
	} else {
		// Delete all subsidiary namespaces
		if namespaceRaw, err := c.kubeclientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s,edge-net.io/kind=sub", tenant.GetName())}); err == nil {
			for _, namespaceRow := range namespaceRaw.Items {
				c.kubeclientset.CoreV1().Namespaces().Delete(context.TODO(), namespaceRow.GetName(), metav1.DeleteOptions{})
			}
		} else {
			// TODO: Provide err information at the status
		}
		// Delete all roles, role bindings, and subsidiary namespaces
		if err := c.kubeclientset.RbacV1().ClusterRoles().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
			// TODO: Provide err information at the status

		}
		if err := c.kubeclientset.RbacV1().ClusterRoleBindings().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/tenant=%s", tenant.GetName())}); err != nil {
			// TODO: Provide err information at the status
		}
		if err := c.kubeclientset.RbacV1().RoleBindings(tenant.GetName()).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			// TODO: Provide err information at the status
		}
	}
}

func (c *Controller) createCoreNamespace(tenant *corev1alpha.Tenant, ownerReferences []metav1.OwnerReference) error {
	// Core namespace has the same name as the tenant
	tenantCoreNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenant.GetName(), OwnerReferences: ownerReferences}}
	// Namespace labels indicate this namespace created by a tenant, not by a team or slice
	namespaceLabels := map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName()}
	tenantCoreNamespace.SetLabels(namespaceLabels)
	exists, index := util.Contains(tenant.Status.Message, statusDict["namespace-failure"])
	_, err := c.kubeclientset.CoreV1().Namespaces().Create(context.TODO(), tenantCoreNamespace, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		klog.V(4).Infof("Couldn't create namespace for %s: %s", tenant.GetName(), err)
		if !exists {
			tenant.Status.State = failure
			tenant.Status.Message = append(tenant.Status.Message, statusDict["namespace-failure"])
		}
	} else if (err == nil || errors.IsAlreadyExists(err)) && exists {
		tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
	}
	return err
}

func (c *Controller) applyQuota(tenant *corev1alpha.Tenant) error {
	trqHandler := tenantresourcequota.Handler{}
	trqHandler.Init(c.kubeclientset, c.edgenetclientset)
	cpuQuota, memoryQuota := trqHandler.Create(tenant.GetName(), SetAsOwnerReference(tenant))

	resourceQuota := corev1.ResourceQuota{}
	resourceQuota.Name = "core-quota"
	resourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              resource.MustParse(cpuQuota),
			"memory":           resource.MustParse(memoryQuota),
			"requests.storage": resource.MustParse("8Gi"),
		},
	}
	exists, index := util.Contains(tenant.Status.Message, statusDict["resource-quota-failure"])
	// Create the resource quota to prevent users from using this namespace for their applications
	_, err := c.kubeclientset.CoreV1().ResourceQuotas(tenant.GetName()).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		klog.V(4).Infof("Couldn't create resource quota in %s: %s", tenant.GetName(), err)
		if !exists {
			tenant.Status.State = failure
			tenant.Status.Message = append(tenant.Status.Message, statusDict["resource-quota-failure"])
		}
	} else if (err == nil || errors.IsAlreadyExists(err)) && exists {
		tenant.Status.Message = append(tenant.Status.Message[:index], tenant.Status.Message[index+1:]...)
	}
	return err
}

// SetAsOwnerReference returns the tenant as owner
func SetAsOwnerReference(tenant *corev1alpha.Tenant) []metav1.OwnerReference {
	// The following section makes tenant become the owner
	ownerReferences := []metav1.OwnerReference{}
	newTenantRef := *metav1.NewControllerRef(tenant, corev1alpha.SchemeGroupVersion.WithKind("Tenant"))
	takeControl := false
	newTenantRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newTenantRef)
	return ownerReferences
}
