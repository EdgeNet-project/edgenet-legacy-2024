/*
Copyright 2020 Sorbonne Université

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

package tenantrequest

import (
	"context"
	"fmt"
	"reflect"
	"time"

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenant"
	"github.com/EdgeNet-project/edgenet/pkg/controller/registration/v1alpha/emailverification"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface)
	ObjectCreatedOrUpdated(obj interface{})
	ObjectDeleted(obj interface{})
	RunExpiryController()
}

// Handler implementation
type Handler struct {
	clientset        kubernetes.Interface
	edgenetClientset versioned.Interface
}

// Init handles any handler initialization
func (t *Handler) Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) {
	log.Info("TenantRequestHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreatedOrUpdated is called when an object is created
func (t *Handler) ObjectCreatedOrUpdated(obj interface{}) {
	log.Info("TenantRequestHandler.ObjectCreatedOrUpdated")
	// Make a copy of the tenant request object to make changes on it
	tenantRequest := obj.(*registrationv1alpha.TenantRequest).DeepCopy()
	if tenantRequest.Status.State != approved {
		defer func() {
			if !reflect.DeepEqual(obj.(*registrationv1alpha.TenantRequest).Status, tenantRequest.Status) {
				if _, err := t.edgenetClientset.RegistrationV1alpha().TenantRequests().UpdateStatus(context.TODO(), tenantRequest, metav1.UpdateOptions{}); err != nil {
					// TO-DO: Provide more information on error
					log.Println(err)
				}
			}
		}()
		if tenantRequest.Spec.Approved {
			tenantHandler := tenant.Handler{}
			tenantHandler.Init(t.clientset, t.edgenetClientset)
			created := tenantHandler.Create(tenantRequest)
			if created {
				tenantRequest.Status.State = approved
				tenantRequest.Status.Message = []string{statusDict["tenant-approved"]}
			} else {
				t.sendEmail("tenant-creation-failure", tenantRequest)
				tenantRequest.Status.State = failure
				tenantRequest.Status.Message = []string{statusDict["tenant-failed"]}
			}
		} else {
			if tenantRequest.Status.Expiry == nil {
				// Set the approval timeout which is 72 hours
				tenantRequest.Status.Expiry = &metav1.Time{
					Time: time.Now().Add(72 * time.Hour),
				}
			}
			isCreated := false
			labels := tenantRequest.GetLabels()
			if labels != nil && labels[fmt.Sprintf("edge-net.io/emailverification-%s", tenantRequest.Spec.Contact.Username)] != "" {
				if _, err := t.edgenetClientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), labels[fmt.Sprintf("edge-net.io/emailverification-%s", tenantRequest.Spec.Contact.Username)], metav1.GetOptions{}); err == nil {
					isCreated = true
				}
			}
			if !isCreated {
				emailVerificationHandler := emailverification.Handler{}
				emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
				code, created := emailVerificationHandler.Create(tenantRequest, SetAsOwnerReference(tenantRequest))
				if created {
					if labels == nil {
						labels = map[string]string{fmt.Sprintf("edge-net.io/emailverification-%s", tenantRequest.Spec.Contact.Username): code}
					} else if labels[fmt.Sprintf("edge-net.io/emailverification-%s", tenantRequest.Spec.Contact.Username)] == "" {
						labels[fmt.Sprintf("edge-net.io/emailverification-%s", tenantRequest.Spec.Contact.Username)] = code
					}
					tenantRequest.SetLabels(labels)
					tenantRequestUpdated, err := t.edgenetClientset.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
					if err == nil {
						tenantRequest = tenantRequestUpdated
					}
					// Update the status as successful
					tenantRequest.Status.State = success
					tenantRequest.Status.Message = []string{statusDict["email-ok"]}
				} else {
					// TO-DO: Define error message more precisely
					tenantRequest.Status.State = issue
					tenantRequest.Status.Message = []string{statusDict["email-fail"]}
				}
			} else if isCreated && tenantRequest.Status.State == failure {
				// Update the status as successful
				tenantRequest.Status.State = success
				tenantRequest.Status.Message = []string{statusDict["email-ok"]}
			}
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("TenantRequestHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(subject string, tenantRequest *registrationv1alpha.TenantRequest) {
	// Set the HTML template variables
	var contentData = mailer.CommonContentData{}
	contentData.CommonData.Tenant = tenantRequest.GetName()
	contentData.CommonData.Username = tenantRequest.Spec.Contact.Username
	contentData.CommonData.Name = fmt.Sprintf("%s %s", tenantRequest.Spec.Contact.FirstName, tenantRequest.Spec.Contact.LastName)
	contentData.CommonData.Email = []string{tenantRequest.Spec.Contact.Email}
	mailer.Send(subject, contentData)
}

// RunExpiryController puts a procedure in place to turn accepted policies into not accepted
func (t *Handler) RunExpiryController() {
	var closestExpiry time.Time
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchTenantRequest, err := t.edgenetClientset.RegistrationV1alpha().TenantRequests().Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchTenantRequest watch.Interface, newExpiry *chan time.Time) {
			// Watch the events of tenant request object
			// Get events from watch interface
			for tenantRequestEvent := range watchTenantRequest.ResultChan() {
				// Get updated tenant request object
				updatedTenantRequest, status := tenantRequestEvent.Object.(*registrationv1alpha.TenantRequest)
				if status {
					if updatedTenantRequest.Status.Expiry != nil {
						*newExpiry <- updatedTenantRequest.Status.Expiry.Time
					}
				}
			}
		}
		go watchEvents(watchTenantRequest, &newExpiry)
	} else {
		go t.RunExpiryController()
		terminated <- true
	}

infiniteLoop:
	for {
		// Wait on multiple channel operations
		select {
		case timeout := <-newExpiry:
			if closestExpiry.Sub(timeout) > 0 {
				closestExpiry = timeout
				log.Printf("ExpiryController: Closest expiry date is %v", closestExpiry)
			}
		case <-time.After(time.Until(closestExpiry)):
			tenantRequestRaw, err := t.edgenetClientset.RegistrationV1alpha().TenantRequests().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TO-DO: Provide more information on error
				log.Println(err)
			}
			for _, tenantRequestRow := range tenantRequestRaw.Items {
				if tenantRequestRow.Status.Expiry != nil && tenantRequestRow.Status.Expiry.Time.Sub(time.Now()) <= 0 {
					t.edgenetClientset.RegistrationV1alpha().TenantRequests().Delete(context.TODO(), tenantRequestRow.GetName(), metav1.DeleteOptions{})
				}
			}
		case <-terminated:
			watchTenantRequest.Stop()
			break infiniteLoop
		}
	}
}

// SetAsOwnerReference put the tenantrequest as owner
func SetAsOwnerReference(tenantRequest *registrationv1alpha.TenantRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(tenantRequest, registrationv1alpha.SchemeGroupVersion.WithKind("TenantRequest"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
