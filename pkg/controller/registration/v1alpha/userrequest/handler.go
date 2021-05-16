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

package userrequest

import (
	"context"
	"fmt"
	"reflect"
	"time"

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/registration/v1alpha/emailverification"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"

	log "github.com/sirupsen/logrus"
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
	log.Info("UserRequestHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("UserRequestHandler.ObjectCreated")
	// Make a copy of the user registration request object to make changes on it
	userRequest := obj.(*registrationv1alpha.UserRequest).DeepCopy()
	// Check if the email address is already taken
	exists, message := t.checkDuplicateObject(userRequest, "tenant-name")
	if exists {
		userRequest.Status.State = failure
		userRequest.Status.Message = message
		// Set the approval timeout which is 24 hours
		userRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(24 * time.Hour),
		}
		t.edgenetClientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequest, metav1.UpdateOptions{})
		// Run timeout goroutine
		go t.runApprovalTimeout(userRequest)
		return
	}
	tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), userRequest.Spec.Tenant, metav1.GetOptions{})
	// Check if the tenant is active
	if tenant.Spec.Enabled {
		if userRequest.Spec.Approved {
			/*userHandler := user.Handler{}
			userHandler.Init(t.clientset, t.edgenetClientset)
			created := !userHandler.Create(userRequest)
			if created {
				return
			}
			t.sendEmail(userRequest, userRequestOwnerNamespace.Labels["tenant-name"], "user-creation-failure")*/
			userRequest.Status.State = failure
			userRequest.Status.Message = []string{statusDict["user-failed"]}
			userRequestUpdated, err := t.edgenetClientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequest, metav1.UpdateOptions{})
			if err == nil {
				userRequest = userRequestUpdated
			}
		}
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if userRequest.Status.Expiry == nil {
			// Set the approval timeout which is 72 hours
			userRequest.Status.Expiry = &metav1.Time{
				Time: time.Now().Add(72 * time.Hour),
			}
			emailVerificationHandler := emailverification.Handler{}
			emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
			created := emailVerificationHandler.Create(userRequest, SetAsOwnerReference(userRequest))
			if created {
				// Update the status as successful
				userRequest.Status.State = success
				userRequest.Status.Message = []string{statusDict["email-ok"]}
			} else {
				userRequest.Status.State = issue
				userRequest.Status.Message = []string{statusDict["email-fail"]}
			}
			t.edgenetClientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequest, metav1.UpdateOptions{})

			// Run timeout goroutine
			go t.runApprovalTimeout(userRequest)
		}
	} else {
		t.edgenetClientset.RegistrationV1alpha().UserRequests().Delete(context.TODO(), userRequest.GetName(), metav1.DeleteOptions{})
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("UserRequestHandler.ObjectUpdated")
	// Make a copy of the user registration request object to make changes on it
	userRequest := obj.(*registrationv1alpha.UserRequest).DeepCopy()
	changeStatus := false
	tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), userRequest.Spec.Tenant, metav1.GetOptions{})
	if tenant.Spec.Enabled {
		// Check again if the email address is already taken
		exists, message := t.checkDuplicateObject(userRequest, userRequest.Spec.Tenant)
		if !exists {
			// Check whether the request for user registration approved
			if userRequest.Spec.Approved {
				/*userHandler := user.Handler{}
				userHandler.Init(t.clientset, t.edgenetClientset)
				changeStatus := userHandler.Create(userRequest)
				if changeStatus {
					t.sendEmail(userRequest, userRequestOwnerNamespace.Labels["tenant-name"], "user-creation-failure")
					userRequest.Status.State = failure
					userRequest.Status.Message = []string{statusDict["user-failed"]}
				}*/
			} else if !userRequest.Spec.Approved && userRequest.Status.State == failure {
				emailVerificationHandler := emailverification.Handler{}
				emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
				created := emailVerificationHandler.Create(userRequest, SetAsOwnerReference(userRequest))
				if created {
					// Update the status as successful
					userRequest.Status.State = success
					userRequest.Status.Message = []string{statusDict["email-ok"]}
				} else {
					userRequest.Status.State = issue
					userRequest.Status.Message = []string{statusDict["email-fail"]}
				}
				changeStatus = true
			}
		} else if exists && !reflect.DeepEqual(userRequest.Status.Message, message) {
			userRequest.Status.State = failure
			userRequest.Status.Message = message
			changeStatus = true
		}
		if changeStatus {
			t.edgenetClientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequest, metav1.UpdateOptions{})
		}
	} else {
		t.edgenetClientset.RegistrationV1alpha().UserRequests().Delete(context.TODO(), userRequest.GetName(), metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("UserRequestHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(userRequest *registrationv1alpha.UserRequest, tenantName, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Tenant = tenantName
	contentData.CommonData.Username = userRequest.GetName()
	contentData.CommonData.Name = fmt.Sprintf("%s %s", userRequest.Spec.FirstName, userRequest.Spec.LastName)
	contentData.CommonData.Email = []string{userRequest.Spec.Email}
	mailer.Send(subject, contentData)
}

// runApprovalTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) runApprovalTimeout(userRequest *registrationv1alpha.UserRequest) {
	registrationApproved := make(chan bool, 1)
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if userRequest.Status.Expiry != nil {
		timeout = time.After(time.Until(userRequest.Status.Expiry.Time))
	}
	closeChannels := func() {
		close(registrationApproved)
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of user registration request object
	watchUserRequest, err := t.edgenetClientset.RegistrationV1alpha().UserRequests().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", userRequest.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for userRequestEvent := range watchUserRequest.ResultChan() {
				// Get updated user registration request object
				updatedUserRequest, status := userRequestEvent.Object.(*registrationv1alpha.UserRequest)
				// FieldSelector doesn't work properly, and will be checked in for next releases.
				if userRequest.GetUID() == updatedUserRequest.GetUID() {
					if status {
						if userRequestEvent.Type == "DELETED" {
							terminated <- true
							continue
						}

						if updatedUserRequest.Spec.Approved == true {
							registrationApproved <- true
							break
						} else if !updatedUserRequest.Spec.Approved && updatedUserRequest.Status.Expiry != nil {
							// Check whether expiration date updated - TBD
							if updatedUserRequest.Status.Expiry.Time.Sub(time.Now()) >= 0 {
								timeout = time.After(time.Until(updatedUserRequest.Status.Expiry.Time))
								timeoutRenewed <- true
							} else {
								terminated <- true
							}
						}
						userRequest = updatedUserRequest
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching userrequest resources,
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
			watchUserRequest.Stop()
			closeChannels()
			break timeoutLoop
		case <-timeoutRenewed:
			break timeoutOptions
		case <-timeout:
			watchUserRequest.Stop()
			t.edgenetClientset.RegistrationV1alpha().UserRequests().Delete(context.TODO(), userRequest.GetName(), metav1.DeleteOptions{})
			closeChannels()
			break timeoutLoop
		case <-terminated:
			watchUserRequest.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}

// checkDuplicateObject checks whether a user exists with the same username or email address
func (t *Handler) checkDuplicateObject(userRequest *registrationv1alpha.UserRequest, tenantName string) (bool, []string) {
	exists := false
	message := []string{}

	// To check email address among users
	tenantRaw, _ := t.edgenetClientset.CoreV1alpha().Tenants().List(context.TODO(), metav1.ListOptions{})
	for _, tenantRow := range tenantRaw.Items {
		for _, userRow := range tenantRow.Spec.User {
			if tenantRow.GetName() == userRequest.Spec.Tenant && userRow.GetName() == userRequest.GetName() {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["username-exist"], userRequest.GetName()))
				if exists && !reflect.DeepEqual(userRequest.Status.Message, message) {
					t.sendEmail(userRequest, tenantName, "user-validation-failure-name")
				}
				break
			}

			if tenantRow.Spec.Contact.Email == userRequest.Spec.Email || userRow.Email == userRequest.Spec.Email {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["email-exist"], userRequest.Spec.Email))
				break
			}
		}
	}

	if !exists {
		// To check email address
		userRequestRaw, _ := t.edgenetClientset.RegistrationV1alpha().UserRequests().List(context.TODO(), metav1.ListOptions{})
		for _, userRequestRow := range userRequestRaw.Items {
			if userRequestRow.Spec.Email == userRequest.Spec.Email && userRequestRow.GetUID() != userRequest.GetUID() {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["email-existregist"], userRequest.Spec.Email))
			}
		}
		if !exists {
			// To check email address given at tenantRequest
			tenantRequestRaw, _ := t.edgenetClientset.RegistrationV1alpha().TenantRequests().List(context.TODO(), metav1.ListOptions{})
			for _, tenantRequestRow := range tenantRequestRaw.Items {
				if tenantRequestRow.Spec.Contact.Email == userRequest.Spec.Email {
					exists = true
					message = append(message, fmt.Sprintf(statusDict["email-existauth"], userRequest.Spec.Email))
				}
			}
		}
		if exists && !reflect.DeepEqual(userRequest.Status.Message, message) {
			t.sendEmail(userRequest, tenantName, "user-validation-failure-email")
		}
	}
	return exists, message
}

// SetAsOwnerReference put the userrequest as owner
func SetAsOwnerReference(userRequest *registrationv1alpha.UserRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(userRequest, registrationv1alpha.SchemeGroupVersion.WithKind("UserRequest"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
