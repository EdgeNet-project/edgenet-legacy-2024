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

// TODO: This entity should be implemented by a CRD where notification medium and events can be declared.
package notifier

import (
	"context"
	"fmt"
	"net/mail"
	"reflect"
	"regexp"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/registration/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/registration/v1alpha"

	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

const controllerAgentName = "notifier-controller"

// Definitions of the state of the tenantrequest resource
const (
	failure = "Failure"
	pending = "Pending"
)

// The main structure of controller
type Controller struct {
	kubeclientset    kubernetes.Interface
	edgenetclientset clientset.Interface

	tenantrequestsLister listers.TenantRequestLister
	tenantrequestsSynced cache.InformerSynced
	rolerequestsLister   listers.RoleRequestLister
	rolerequestsSynced   cache.InformerSynced

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
	tenantrequestInformer informers.TenantRequestInformer,
	rolerequestInformer informers.RoleRequestInformer) *Controller {
	// Create event broadcaster
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	klog.V(4).Infoln("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:        kubeclientset,
		edgenetclientset:     edgenetclientset,
		tenantrequestsLister: tenantrequestInformer.Lister(),
		tenantrequestsSynced: tenantrequestInformer.Informer().HasSynced,
		rolerequestsLister:   rolerequestInformer.Lister(),
		rolerequestsSynced:   rolerequestInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Notifier"),
		recorder:             recorder,
	}

	klog.Infoln("Setting up event handlers")

	// Event handlers deal with events of resources.
	tenantrequestInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newTenantRequest := new.(*registrationv1alpha.TenantRequest)
			oldTenantRequest := old.(*registrationv1alpha.TenantRequest)
			if !reflect.DeepEqual(newTenantRequest.Status, oldTenantRequest.Status) {
				controller.enqueueNotifier(new)
			}
		},
	})
	rolerequestInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newRoleRequest := new.(*registrationv1alpha.TenantRequest)
			oldRoleRequest := old.(*registrationv1alpha.TenantRequest)
			if !reflect.DeepEqual(newRoleRequest.Status, oldRoleRequest.Status) {
				controller.enqueueNotifier(new)
			}
		},
	})

	return controller
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Infoln("Starting Notifier Controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh,
		c.tenantrequestsSynced,
		c.rolerequestsSynced); !ok {
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
		switch obj.(type) {
		case registrationv1alpha.TenantRequest:
			if err := c.syncTenantRequestHandler(key); err != nil {
				c.workqueue.AddRateLimited(key)
				return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
			}
		case registrationv1alpha.RoleRequest:
			if err := c.syncRoleRequestHandler(key); err != nil {
				c.workqueue.AddRateLimited(key)
				return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
			}
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

// syncTenantRequestHandler looks at the actual state and sends a notification if desired.
func (c *Controller) syncTenantRequestHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	tenantrequest, err := c.tenantrequestsLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("tenant request '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	klog.V(4).Infof("processNextItem: object created/updated detected: %s", key)
	c.processTenantRequest(tenantrequest)

	return nil
}

// syncRoleRequestHandler looks at the actual state and sends a notification if desired.
func (c *Controller) syncRoleRequestHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	rolerequest, err := c.rolerequestsLister.RoleRequests(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("role request '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	klog.V(4).Infof("processNextItem: object created/updated detected: %s", key)
	c.processRoleRequest(rolerequest)

	return nil
}

func (c *Controller) enqueueNotifier(obj interface{}) {
	// Put the resource object into a key
	var key string
	var err error

	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}

func (c *Controller) processTenantRequest(tenantrequest *registrationv1alpha.TenantRequest) {
	klog.V(4).Infoln("Handler.ObjectCreated")
	//nodeObj := obj.(*corev1.Node)

	systemNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		return
	}
	if tenantrequest.Status.State == failure || tenantrequest.Status.State == "" {
		return
	} else if tenantrequest.Status.State == pending {
		// The function below notifies those who have the right to approve this tenant request.
		// As tenant requests are cluster-wide resources, we check the permissions granted by Cluster Role Binding following a pattern to avoid overhead.
		// Furthermore, only those to which the system has granted permission, by attaching the "edge-net.io/generated=true" label, receive a notification email.
		emailList := []string{}
		if clusterRoleBindingRaw, err := c.kubeclientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
			r, _ := regexp.Compile("(.*)(edgenet:clusteradministration)(.*)(admin|manager|deputy)(.*)")
			for _, clusterRoleBindingRow := range clusterRoleBindingRaw.Items {
				if match := r.MatchString(clusterRoleBindingRow.GetName()); !match {
					continue
				}
				for _, subjectRow := range clusterRoleBindingRow.Subjects {
					if subjectRow.Kind == "User" {
						_, err := mail.ParseAddress(subjectRow.Name)
						if err == nil {
							subjectAccessReview := new(authorizationv1.SubjectAccessReview)
							subjectAccessReview.Spec.ResourceAttributes.Resource = "tenantrequests"
							subjectAccessReview.Spec.ResourceAttributes.Verb = "UPDATE"
							subjectAccessReview.Spec.ResourceAttributes.Name = tenantrequest.GetName()
							if subjectAccessReviewResult, err := c.kubeclientset.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), subjectAccessReview, metav1.CreateOptions{}); err == nil {
								if subjectAccessReviewResult.Status.Allowed {
									emailList = append(emailList, subjectRow.Name)
								}
							}
						}
					}
				}
			}
		}
		if len(emailList) > 0 {
			access.SendEmailForTenantRequest(tenantrequest, "tenant-request-made", "[EdgeNet Admin] A tenant request made",
				string(systemNamespace.GetUID()), emailList)
			access.SendSlackNotificationForTenantRequest(tenantrequest, "tenant-request-made", "[EdgeNet Admin] A tenant request made",
				string(systemNamespace.GetUID()))
		}
	} else {
		access.SendEmailForTenantRequest(tenantrequest, "tenant-request-approved", "[EdgeNet] Tenant request approved",
			string(systemNamespace.GetUID()), []string{tenantrequest.Spec.Contact.Email})
		access.SendSlackNotificationForTenantRequest(tenantrequest, "tenant-request-approved", "[EdgeNet] Tenant request approved",
			string(systemNamespace.GetUID()))
	}
}

func (c *Controller) processRoleRequest(rolerequest *registrationv1alpha.RoleRequest) {
	klog.V(4).Infoln("Handler.ObjectCreated")

	systemNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		return
	}
	if rolerequest.Status.State == failure || rolerequest.Status.State == "" {
		return
	} else if rolerequest.Status.State == pending {
		// The function below notifies those who have the right to approve this role request.
		// As role requests run on the layer of namespaces, we here ignore the permissions granted by Cluster Role Binding to avoid email floods.
		// Furthermore, only those to which the system has granted permission, by attaching the "edge-net.io/generated=true" label, receive a notification email.
		emailList := []string{}
		if roleBindingRaw, err := c.kubeclientset.RbacV1().RoleBindings(rolerequest.GetNamespace()).List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
			r, _ := regexp.Compile("(.*)(owner|admin|manager|deputy)(.*)")
			for _, roleBindingRow := range roleBindingRaw.Items {
				if match := r.MatchString(roleBindingRow.GetName()); !match {
					continue
				}
				for _, subjectRow := range roleBindingRow.Subjects {
					if subjectRow.Kind == "User" {
						_, err := mail.ParseAddress(subjectRow.Name)
						if err == nil {
							subjectAccessReview := new(authorizationv1.SubjectAccessReview)
							subjectAccessReview.Spec.ResourceAttributes = new(authorizationv1.ResourceAttributes)
							subjectAccessReview.Spec.ResourceAttributes.Resource = "rolerequests"
							subjectAccessReview.Spec.ResourceAttributes.Namespace = rolerequest.GetNamespace()
							subjectAccessReview.Spec.ResourceAttributes.Verb = "UPDATE"
							subjectAccessReview.Spec.ResourceAttributes.Name = rolerequest.GetName()
							if subjectAccessReviewResult, err := c.kubeclientset.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), subjectAccessReview, metav1.CreateOptions{}); err == nil {
								if subjectAccessReviewResult.Status.Allowed {
									emailList = append(emailList, subjectRow.Name)
								}
							}
						}
					}
				}
			}
		}
		if len(emailList) > 0 {
			access.SendEmailForRoleRequest(rolerequest, "role-request-made", "[EdgeNet] A role request made",
				string(systemNamespace.GetUID()), emailList)
			access.SendSlackNotificationForRoleRequest(rolerequest, "role-request-made", "[EdgeNet] A role request made",
				string(systemNamespace.GetUID()))
		}
	} else {
		access.SendEmailForRoleRequest(rolerequest, "role-request-approved", "[EdgeNet] Role request approved",
			string(systemNamespace.GetUID()), []string{rolerequest.Spec.Email})
		access.SendSlackNotificationForRoleRequest(rolerequest, "role-request-approved", "[EdgeNet] Role request approved",
			string(systemNamespace.GetUID()))
	}
}

func (c *Controller) processClusterRoleRequest(clusterRolerequest *registrationv1alpha.ClusterRoleRequest) {
	klog.V(4).Infoln("Handler.ObjectCreated")

	systemNamespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		return
	}
	if clusterRolerequest.Status.State == failure || clusterRolerequest.Status.State == "" {
		return
	} else if clusterRolerequest.Status.State == pending {
		// The function below notifies those who have the right to approve this role request.
		// As role requests run on the layer of namespaces, we here ignore the permissions granted by Cluster Role Binding to avoid email floods.
		// Furthermore, only those to which the system has granted permission, by attaching the "edge-net.io/generated=true" label, receive a notification email.
		emailList := []string{}
		if roleBindingRaw, err := c.kubeclientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{LabelSelector: "edge-net.io/generated=true"}); err == nil {
			r, _ := regexp.Compile("(.*)(owner|admin|manager|deputy)(.*)")
			for _, roleBindingRow := range roleBindingRaw.Items {
				if match := r.MatchString(roleBindingRow.GetName()); !match {
					continue
				}
				for _, subjectRow := range roleBindingRow.Subjects {
					if subjectRow.Kind == "User" {
						_, err := mail.ParseAddress(subjectRow.Name)
						if err == nil {
							subjectAccessReview := new(authorizationv1.SubjectAccessReview)
							subjectAccessReview.Spec.ResourceAttributes = new(authorizationv1.ResourceAttributes)
							subjectAccessReview.Spec.ResourceAttributes.Resource = "rolerequests"
							subjectAccessReview.Spec.ResourceAttributes.Namespace = clusterRolerequest.GetNamespace()
							subjectAccessReview.Spec.ResourceAttributes.Verb = "UPDATE"
							subjectAccessReview.Spec.ResourceAttributes.Name = clusterRolerequest.GetName()
							if subjectAccessReviewResult, err := c.kubeclientset.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), subjectAccessReview, metav1.CreateOptions{}); err == nil {
								if subjectAccessReviewResult.Status.Allowed {
									emailList = append(emailList, subjectRow.Name)
								}
							}
						}
					}
				}
			}
		}
		if len(emailList) > 0 {
			access.SendEmailForClusterRoleRequest(clusterRolerequest, "clusterrole-request-made", "[EdgeNet] A cluster role request made",
				string(systemNamespace.GetUID()), emailList)
			access.SendSlackNotificationForClusterRoleRequest(clusterRolerequest, "clusterrole-request-made", "[EdgeNet] A cluster role request made",
				string(systemNamespace.GetUID()))
		}
	} else {
		access.SendEmailForClusterRoleRequest(clusterRolerequest, "clusterrole-request-approved", "[EdgeNet] Cluster role request approved",
			string(systemNamespace.GetUID()), []string{clusterRolerequest.Spec.Email})
		access.SendSlackNotificationForClusterRoleRequest(clusterRolerequest, "clusterrole-request-approved", "[EdgeNet] Cluster role request approved",
			string(systemNamespace.GetUID()))
	}
}
