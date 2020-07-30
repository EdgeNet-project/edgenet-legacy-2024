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

package userregistrationrequest

import (
	"fmt"
	"reflect"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/controller/v1alpha/emailverification"
	"edgenet/pkg/controller/v1alpha/user"
	"edgenet/pkg/mailer"

	log "github.com/Sirupsen/logrus"
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
	log.Info("URRHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("URRHandler.ObjectCreated")
	// Create a copy of the user registration request object to make changes on it
	URRCopy := obj.(*apps_v1alpha.UserRegistrationRequest).DeepCopy()
	// Find the authority from the namespace in which the object is
	URROwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(URRCopy.GetNamespace(), metav1.GetOptions{})
	// Check if the email address is already taken
	exists, message := t.checkDuplicateObject(URRCopy, URROwnerNamespace.Labels["authority-name"])
	if exists {
		URRCopy.Status.State = failure
		URRCopy.Status.Message = message
		// Run timeout goroutine
		go t.runApprovalTimeout(URRCopy)
		// Set the approval timeout which is 72 hours
		URRCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(24 * time.Hour),
		}
		t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).UpdateStatus(URRCopy)
		return
	}
	URROwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(URROwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	// Check if the authority is active
	if URROwnerAuthority.Spec.Enabled {
		if URRCopy.Spec.Approved {
			userHandler := user.Handler{}
			userHandler.Init(t.clientset, t.edgenetClientset)
			created := !userHandler.Create(URRCopy)
			if created {
				return
			} else {
				t.sendEmail(URRCopy, URROwnerNamespace.Labels["authority-name"], "user-creation-failure")
				URRCopy.Status.State = failure
				URRCopy.Status.Message = []string{statusDict["user-failed"]}
				URRCopyUpdated, err := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).UpdateStatus(URRCopy)
				if err == nil {
					URRCopy = URRCopyUpdated
				}
			}
		}
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if URRCopy.Status.Expires == nil {
			// Run timeout goroutine
			go t.runApprovalTimeout(URRCopy)
			defer t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).UpdateStatus(URRCopy)
			// Set the approval timeout which is 72 hours
			URRCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(72 * time.Hour),
			}
			emailVerificationHandler := emailverification.Handler{}
			emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
			created := emailVerificationHandler.Create(URRCopy, SetAsOwnerReference(URRCopy))
			if created {
				// Update the status as successful
				URRCopy.Status.State = success
				URRCopy.Status.Message = []string{statusDict["email-ok"]}
			} else {
				URRCopy.Status.State = issue
				URRCopy.Status.Message = []string{statusDict["email-fail"]}
			}
		} else {
			go t.runApprovalTimeout(URRCopy)
		}
	} else {
		t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).Delete(URRCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("URRHandler.ObjectUpdated")
	// Create a copy of the user registration request object to make changes on it
	URRCopy := obj.(*apps_v1alpha.UserRegistrationRequest).DeepCopy()
	changeStatus := false
	URROwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(URRCopy.GetNamespace(), metav1.GetOptions{})
	URROwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(URROwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	if URROwnerAuthority.Spec.Enabled {
		// Check again if the email address is already taken
		exists, message := t.checkDuplicateObject(URRCopy, URROwnerNamespace.Labels["authority-name"])
		if !exists {
			// Check whether the request for user registration approved
			if URRCopy.Spec.Approved {
				userHandler := user.Handler{}
				userHandler.Init(t.clientset, t.edgenetClientset)
				changeStatus := userHandler.Create(URRCopy)
				if changeStatus {
					t.sendEmail(URRCopy, URROwnerNamespace.Labels["authority-name"], "user-creation-failure")
					URRCopy.Status.State = failure
					URRCopy.Status.Message = []string{statusDict["user-failed"]}
				}
			} else if !URRCopy.Spec.Approved && URRCopy.Status.State == failure {
				emailVerificationHandler := emailverification.Handler{}
				emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
				created := emailVerificationHandler.Create(URRCopy, SetAsOwnerReference(URRCopy))
				if created {
					// Update the status as successful
					URRCopy.Status.State = success
					URRCopy.Status.Message = []string{statusDict["email-ok"]}
				} else {
					URRCopy.Status.State = issue
					URRCopy.Status.Message = []string{statusDict["email-fail"]}
				}
				changeStatus = true
			}
		} else if exists && !reflect.DeepEqual(URRCopy.Status.Message, message) {
			URRCopy.Status.State = failure
			URRCopy.Status.Message = message
			changeStatus = true
		}
		if changeStatus {
			t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).UpdateStatus(URRCopy)
		}
	} else {
		t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).Delete(URRCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("URRHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(URRCopy *apps_v1alpha.UserRegistrationRequest, authorityName, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Authority = authorityName
	contentData.CommonData.Username = URRCopy.GetName()
	contentData.CommonData.Name = fmt.Sprintf("%s %s", URRCopy.Spec.FirstName, URRCopy.Spec.LastName)
	contentData.CommonData.Email = []string{URRCopy.Spec.Email}
	mailer.Send(subject, contentData)
}

// runApprovalTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) runApprovalTimeout(URRCopy *apps_v1alpha.UserRegistrationRequest) {
	registrationApproved := make(chan bool, 1)
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if URRCopy.Status.Expires != nil {
		timeout = time.After(time.Until(URRCopy.Status.Expires.Time))
	}
	closeChannels := func() {
		close(registrationApproved)
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of user registration request object
	watchURR, err := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", URRCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for URREvent := range watchURR.ResultChan() {
				// Get updated user registration request object
				updatedURR, status := URREvent.Object.(*apps_v1alpha.UserRegistrationRequest)
				// FieldSelector doesn't work properly, and will be checked in for next releases.
				if URRCopy.GetUID() == updatedURR.GetUID() {
					if status {
						if URREvent.Type == "DELETED" {
							terminated <- true
							continue
						}

						if updatedURR.Spec.Approved == true {
							registrationApproved <- true
							break
						} else if !updatedURR.Spec.Approved && updatedURR.Status.Expires != nil {
							// Check whether expiration date updated - TBD
							if updatedURR.Status.Expires.Time.Sub(time.Now()) >= 0 {
								timeout = time.After(time.Until(updatedURR.Status.Expires.Time))
								timeoutRenewed <- true
							} else {
								terminated <- true
							}
						}
						URRCopy = updatedURR
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching userregistrationrequest resources,
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
			watchURR.Stop()
			closeChannels()
			break timeoutLoop
		case <-timeoutRenewed:
			break timeoutOptions
		case <-timeout:
			watchURR.Stop()
			t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).Delete(URRCopy.GetName(), &metav1.DeleteOptions{})
			closeChannels()
			break timeoutLoop
		case <-terminated:
			watchURR.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}

// checkDuplicateObject checks whether a user exists with the same username or email address
func (t *Handler) checkDuplicateObject(URRCopy *apps_v1alpha.UserRegistrationRequest, authorityName string) (bool, []string) {
	exists := false
	message := []string{}
	// To check username on the users resource
	userObj, _ := t.edgenetClientset.AppsV1alpha().Users(URRCopy.GetNamespace()).Get(URRCopy.GetName(), metav1.GetOptions{})
	//userRaw, _ := t.edgenetClientset.AppsV1alpha().Users(URRCopy.GetNamespace()).List(
	//	metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", URRCopy.GetName())})
	if userObj == nil {
		// To check email address
		userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			if userRow.Spec.Email == URRCopy.Spec.Email {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["email-existuser"], URRCopy.Spec.Email))
				break
			}
		}
		if !exists {
			// To check email address
			URRRaw, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests("").List(metav1.ListOptions{})
			for _, URRRow := range URRRaw.Items {
				if URRRow.Spec.Email == URRCopy.Spec.Email && URRRow.GetUID() != URRCopy.GetUID() {
					exists = true
					message = append(message, fmt.Sprintf(statusDict["email-existregist"], URRCopy.Spec.Email))
				}
			}
			if !exists {
				// To check email address given at authorityRequest
				authorityRequestRaw, _ := t.edgenetClientset.AppsV1alpha().AuthorityRequests().List(metav1.ListOptions{})
				for _, authorityRequestRow := range authorityRequestRaw.Items {
					if authorityRequestRow.Spec.Contact.Email == URRCopy.Spec.Email {
						exists = true
						message = append(message, fmt.Sprintf(statusDict["email-existauth"], URRCopy.Spec.Email))
					}
				}
			}
		}
		if exists && !reflect.DeepEqual(URRCopy.Status.Message, message) {
			t.sendEmail(URRCopy, authorityName, "user-validation-failure-email")
		}
	} else {
		exists = true
		message = append(message, fmt.Sprintf(statusDict["username-exist"], URRCopy.GetName()))
		if exists && !reflect.DeepEqual(URRCopy.Status.Message, message) {
			t.sendEmail(URRCopy, authorityName, "user-validation-failure-name")
		}
	}
	return exists, message
}

// SetAsOwnerReference put the userregistrationrequest as owner
func SetAsOwnerReference(URRCopy *apps_v1alpha.UserRegistrationRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(URRCopy, apps_v1alpha.SchemeGroupVersion.WithKind("UserRegistrationRequest"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
