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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface)
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
	ObjectDeleted(obj interface{})
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

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("TenantRequestHandler.ObjectCreated")
	// Make a copy of the tenant request object to make changes on it
	tenantRequest := obj.(*registrationv1alpha.TenantRequest).DeepCopy()
	defer t.edgenetClientset.RegistrationV1alpha().TenantRequests().UpdateStatus(context.TODO(), tenantRequest, metav1.UpdateOptions{})
	// Check if the email address of user or tenant name is already taken
	exists, message := t.checkDuplicateObject(tenantRequest)
	if exists {
		tenantRequest.Status.State = failure
		tenantRequest.Status.Message = message
		// Run timeout goroutine
		go t.runApprovalTimeout(tenantRequest)
		// Set the approval timeout which is 72 hours
		tenantRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		return
	}
	if tenantRequest.Spec.Approved {
		tenantHandler := tenant.Handler{}
		tenantHandler.Init(t.clientset, t.edgenetClientset)
		created := !tenantHandler.Create(tenantRequest)
		if created {
			return
		}
		t.sendEmail("tenant-creation-failure", tenantRequest)
		tenantRequest.Status.State = failure
		tenantRequest.Status.Message = []string{statusDict["tenant-failed"]}
	}
	// If the service restarts, it creates all objects again
	// Because of that, this section covers a variety of possibilities
	if tenantRequest.Status.Expiry == nil {
		// Run timeout goroutine
		go t.runApprovalTimeout(tenantRequest)
		// Set the approval timeout which is 72 hours
		tenantRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		emailVerificationHandler := emailverification.Handler{}
		emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
		code, created := emailVerificationHandler.Create(tenantRequest, SetAsOwnerReference(tenantRequest))

		labels := tenantRequest.GetLabels()
		if labels == nil {
			labels = map[string]string{fmt.Sprintf("edge-net.io/emailverification/%s", tenantRequest.Spec.Contact.Username): code}
		} else {
			labels[fmt.Sprintf("edge-net.io/emailverification/%s", tenantRequest.Spec.Contact.Username)] = code
		}
		tenantRequest.SetLabels(labels)
		tenantRequestUpdated, err := t.edgenetClientset.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
		if err == nil {
			tenantRequest = tenantRequestUpdated
		}

		if created {
			// Update the status as successful
			tenantRequest.Status.State = success
			tenantRequest.Status.Message = []string{statusDict["email-ok"]}
		} else {
			// TO-DO: Define error message more precisely
			tenantRequest.Status.State = issue
			tenantRequest.Status.Message = []string{statusDict["email-fail"]}
		}
	} else {
		go t.runApprovalTimeout(tenantRequest)
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("TenantRequestHandler.ObjectUpdated")
	// Make a copy of the tenant request object to make changes on it
	tenantRequest := obj.(*registrationv1alpha.TenantRequest).DeepCopy()
	changeStatus := false
	// Check if the email address of user or tenant name is already taken
	exists, message := t.checkDuplicateObject(tenantRequest)
	if !exists {
		// Check whether the request for tenant creation approved
		if tenantRequest.Spec.Approved {
			tenantHandler := tenant.Handler{}
			tenantHandler.Init(t.clientset, t.edgenetClientset)
			changeStatus := tenantHandler.Create(tenantRequest)
			if changeStatus {
				t.sendEmail("tenant-creation-failure", tenantRequest)
				tenantRequest.Status.State = failure
				tenantRequest.Status.Message = []string{statusDict["tenant-failed"]}
			}
		} else if !tenantRequest.Spec.Approved && tenantRequest.Status.State == failure {
			emailVerificationHandler := emailverification.Handler{}
			emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
			code, created := emailVerificationHandler.Create(tenantRequest, SetAsOwnerReference(tenantRequest))

			labels := tenantRequest.GetLabels()
			if labels == nil {
				labels = map[string]string{fmt.Sprintf("edge-net.io/emailverification/%s", tenantRequest.Spec.Contact.Username): code}
			} else {
				labels[fmt.Sprintf("edge-net.io/emailverification/%s", tenantRequest.Spec.Contact.Username)] = code
			}
			tenantRequest.SetLabels(labels)
			tenantRequestUpdated, err := t.edgenetClientset.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
			if err == nil {
				tenantRequest = tenantRequestUpdated
			}

			if created {
				// Update the status as successful
				tenantRequest.Status.State = success
				tenantRequest.Status.Message = []string{statusDict["email-ok"]}
			} else {
				tenantRequest.Status.State = issue
				tenantRequest.Status.Message = []string{statusDict["email-fail"]}
			}
			changeStatus = true
		}
	} else if exists && !reflect.DeepEqual(tenantRequest.Status.Message, message) {
		tenantRequest.Status.State = failure
		tenantRequest.Status.Message = message
		changeStatus = true
	}
	if changeStatus {
		t.edgenetClientset.RegistrationV1alpha().TenantRequests().UpdateStatus(context.TODO(), tenantRequest, metav1.UpdateOptions{})
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

// checkDuplicateObject checks whether a user exists with the same email address
func (t *Handler) checkDuplicateObject(tenantRequest *registrationv1alpha.TenantRequest) (bool, []string) {
	exists := false
	message := []string{}
	// To check username on the users resource
	_, err := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantRequest.GetName(), metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		exists = true
		message = append(message, fmt.Sprintf(statusDict["tenant-taken"], tenantRequest.GetName()))
		if !reflect.DeepEqual(tenantRequest.Status.Message, message) {
			t.sendEmail("tenant-validation-failure-name", tenantRequest)
		}
	} else {
		// To check email address among users
		tenantRaw, _ := t.edgenetClientset.CoreV1alpha().Tenants().List(context.TODO(), metav1.ListOptions{})
		for _, tenantRow := range tenantRaw.Items {
			if tenantRow.Spec.Contact.Email == tenantRequest.Spec.Contact.Email {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["email-exist"], tenantRequest.Spec.Contact.Email))
				break
			} else {
				for _, userRow := range tenantRow.Spec.User {
					if userRow.Email == tenantRequest.Spec.Contact.Email {
						exists = true
						message = append(message, fmt.Sprintf(statusDict["email-exist"], tenantRequest.Spec.Contact.Email))
						break
					}
				}
			}
		}
		// To check email address among user registration requests
		userRequestRaw, _ := t.edgenetClientset.RegistrationV1alpha().UserRequests().List(context.TODO(), metav1.ListOptions{})
		for _, userRequestRow := range userRequestRaw.Items {
			if userRequestRow.Spec.Email == tenantRequest.Spec.Contact.Email {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["email-used-reg"], tenantRequest.Spec.Contact.Email))
				break
			}
		}
		// To check email address given at tenant request
		tenantRequestRaw, _ := t.edgenetClientset.RegistrationV1alpha().TenantRequests().List(context.TODO(), metav1.ListOptions{})
		for _, tenantRequestRow := range tenantRequestRaw.Items {
			if tenantRequestRow.Spec.Contact.Email == tenantRequest.Spec.Contact.Email && tenantRequestRow.GetUID() != tenantRequest.GetUID() {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["email-used-auth"], tenantRequest.Spec.Contact.Email))
				break
			}
		}
		if exists && !reflect.DeepEqual(tenantRequest.Status.Message, message) {
			t.sendEmail("tenant-validation-failure-email", tenantRequest)
		}
	}
	return exists, message
}

// runApprovalTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) runApprovalTimeout(tenantRequest *registrationv1alpha.TenantRequest) {
	registrationApproved := make(chan bool, 1)
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if tenantRequest.Status.Expiry != nil {
		timeout = time.After(time.Until(tenantRequest.Status.Expiry.Time))
	}
	closeChannels := func() {
		close(registrationApproved)
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of tenant request object
	watchTenantRequest, err := t.edgenetClientset.RegistrationV1alpha().TenantRequests().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", tenantRequest.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for tenantRequestEvent := range watchTenantRequest.ResultChan() {
				// Get updated tenant request object
				updatedTenantRequest, status := tenantRequestEvent.Object.(*registrationv1alpha.TenantRequest)
				if tenantRequest.GetUID() == updatedTenantRequest.GetUID() {
					if status {
						if tenantRequestEvent.Type == "DELETED" {
							terminated <- true
							continue
						}

						if updatedTenantRequest.Spec.Approved == true {
							registrationApproved <- true
							break
						} else if updatedTenantRequest.Status.Expiry != nil {
							// Check whether expiration date updated - TBD
							if updatedTenantRequest.Status.Expiry.Time.Sub(time.Now()) >= 0 {
								timeout = time.After(time.Until(updatedTenantRequest.Status.Expiry.Time))
								timeoutRenewed <- true
							} else {
								terminated <- true
							}
						}
						tenantRequest = updatedTenantRequest
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching tenantrequest resources,
		// there is a timeout at 72 hours
		timeout = time.After(72 * time.Hour)
	}

	// Infinite loop
timeoutLoop:
	for {
		// Wait on multiple channel operations
	timeoutOptions:
		select {
		case <-registrationApproved:
			watchTenantRequest.Stop()
			closeChannels()
			break timeoutLoop
		case <-timeoutRenewed:
			break timeoutOptions
		case <-timeout:
			watchTenantRequest.Stop()
			closeChannels()
			t.edgenetClientset.RegistrationV1alpha().TenantRequests().Delete(context.TODO(), tenantRequest.GetName(), metav1.DeleteOptions{})
			break timeoutLoop
		case <-terminated:
			watchTenantRequest.Stop()
			closeChannels()
			break timeoutLoop
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
