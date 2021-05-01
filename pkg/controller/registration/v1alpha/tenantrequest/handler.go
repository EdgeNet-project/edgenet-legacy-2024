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

package authorityrequest

import (
	"context"
	"fmt"
	"reflect"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/v1alpha/authority"
	"github.com/EdgeNet-project/edgenet/pkg/controller/v1alpha/emailverification"
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
	log.Info("authorityRequestHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("authorityRequestHandler.ObjectCreated")
	// Create a copy of the authority request object to make changes on it
	authorityRequestCopy := obj.(*corev1alpha.AuthorityRequest).DeepCopy()
	defer t.edgenetClientset.AppsV1alpha().AuthorityRequests().UpdateStatus(context.TODO(), authorityRequestCopy, metav1.UpdateOptions{})
	// Check if the email address of user or authority name is already taken
	exists, message := t.checkDuplicateObject(authorityRequestCopy)
	if exists {
		authorityRequestCopy.Status.State = failure
		authorityRequestCopy.Status.Message = message
		// Run timeout goroutine
		go t.runApprovalTimeout(authorityRequestCopy)
		// Set the approval timeout which is 24 hours
		authorityRequestCopy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(24 * time.Hour),
		}
		return
	}
	if authorityRequestCopy.Spec.Approved {
		authorityHandler := authority.Handler{}
		authorityHandler.Init(t.clientset, t.edgenetClientset)
		created := !authorityHandler.Create(authorityRequestCopy)
		if created {
			return
		} else {
			t.sendEmail("authority-creation-failure", authorityRequestCopy)
			authorityRequestCopy.Status.State = failure
			authorityRequestCopy.Status.Message = []string{statusDict["authority-failed"]}
		}

	}
	// If the service restarts, it creates all objects again
	// Because of that, this section covers a variety of possibilities
	if authorityRequestCopy.Status.Expiry == nil {
		// Run timeout goroutine
		go t.runApprovalTimeout(authorityRequestCopy)
		// Set the approval timeout which is 72 hours
		authorityRequestCopy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		emailVerificationHandler := emailverification.Handler{}
		emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
		created := emailVerificationHandler.Create(authorityRequestCopy, SetAsOwnerReference(authorityRequestCopy))
		if created {
			// Update the status as successful
			authorityRequestCopy.Status.State = success
			authorityRequestCopy.Status.Message = []string{statusDict["email-ok"]}
		} else {
			authorityRequestCopy.Status.State = issue
			authorityRequestCopy.Status.Message = []string{statusDict["email-fail"]}
		}

	} else {
		go t.runApprovalTimeout(authorityRequestCopy)
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("authorityRequestHandler.ObjectUpdated")
	// Create a copy of the authority request object to make changes on it
	authorityRequestCopy := obj.(*corev1alpha.AuthorityRequest).DeepCopy()
	changeStatus := false
	// Check if the email address of user or authority name is already taken
	exists, message := t.checkDuplicateObject(authorityRequestCopy)
	if !exists {
		// Check whether the request for authority creation approved
		if authorityRequestCopy.Spec.Approved {
			authorityHandler := authority.Handler{}
			authorityHandler.Init(t.clientset, t.edgenetClientset)
			changeStatus := authorityHandler.Create(authorityRequestCopy)
			if changeStatus {
				t.sendEmail("authority-creation-failure", authorityRequestCopy)
				authorityRequestCopy.Status.State = failure
				authorityRequestCopy.Status.Message = []string{statusDict["authority-failed"]}
			}
		} else if !authorityRequestCopy.Spec.Approved && authorityRequestCopy.Status.State == failure {
			emailVerificationHandler := emailverification.Handler{}
			emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
			created := emailVerificationHandler.Create(authorityRequestCopy, SetAsOwnerReference(authorityRequestCopy))
			if created {
				// Update the status as successful
				authorityRequestCopy.Status.State = success
				authorityRequestCopy.Status.Message = []string{statusDict["email-ok"]}
			} else {
				authorityRequestCopy.Status.State = issue
				authorityRequestCopy.Status.Message = []string{statusDict["email-fail"]}
			}
			changeStatus = true
		}
	} else if exists && !reflect.DeepEqual(authorityRequestCopy.Status.Message, message) {
		authorityRequestCopy.Status.State = failure
		authorityRequestCopy.Status.Message = message
		changeStatus = true
	}
	if changeStatus {
		t.edgenetClientset.AppsV1alpha().AuthorityRequests().UpdateStatus(context.TODO(), authorityRequestCopy, metav1.UpdateOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("authorityRequestHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(subject string, authorityRequestCopy *corev1alpha.AuthorityRequest) {
	// Set the HTML template variables
	var contentData = mailer.CommonContentData{}
	contentData.CommonData.Authority = authorityRequestCopy.GetName()
	contentData.CommonData.Username = authorityRequestCopy.Spec.Contact.Username
	contentData.CommonData.Name = fmt.Sprintf("%s %s", authorityRequestCopy.Spec.Contact.FirstName, authorityRequestCopy.Spec.Contact.LastName)
	contentData.CommonData.Email = []string{authorityRequestCopy.Spec.Contact.Email}
	mailer.Send(subject, contentData)
}

// checkDuplicateObject checks whether a user exists with the same email address
func (t *Handler) checkDuplicateObject(authorityRequestCopy *corev1alpha.AuthorityRequest) (bool, []string) {
	exists := false
	message := []string{}
	// To check username on the users resource
	_, err := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), authorityRequestCopy.GetName(), metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		exists = true
		message = append(message, fmt.Sprintf(statusDict["authority-taken"], authorityRequestCopy.GetName()))
		if !reflect.DeepEqual(authorityRequestCopy.Status.Message, message) {
			t.sendEmail("authority-validation-failure-name", authorityRequestCopy)
		}
	} else {
		// To check email address among users
		userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(context.TODO(), metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			if userRow.Spec.Email == authorityRequestCopy.Spec.Contact.Email {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["email-exist"], authorityRequestCopy.Spec.Contact.Email))
				break
			}
		}
		// To check email address among user registration requests
		URRRaw, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests("").List(context.TODO(), metav1.ListOptions{})
		for _, URRRow := range URRRaw.Items {
			if URRRow.Spec.Email == authorityRequestCopy.Spec.Contact.Email {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["email-used-reg"], authorityRequestCopy.Spec.Contact.Email))
				break
			}
		}
		// To check email address given at authority request
		authorityRequestRaw, _ := t.edgenetClientset.AppsV1alpha().AuthorityRequests().List(context.TODO(), metav1.ListOptions{})
		for _, authorityRequestRow := range authorityRequestRaw.Items {
			if authorityRequestRow.Spec.Contact.Email == authorityRequestCopy.Spec.Contact.Email && authorityRequestRow.GetUID() != authorityRequestCopy.GetUID() {
				exists = true
				message = append(message, fmt.Sprintf(statusDict["email-used-auth"], authorityRequestCopy.Spec.Contact.Email))
				break
			}
		}
		if exists && !reflect.DeepEqual(authorityRequestCopy.Status.Message, message) {
			t.sendEmail("authority-validation-failure-email", authorityRequestCopy)
		}
	}
	return exists, message
}

// runApprovalTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) runApprovalTimeout(authorityRequestCopy *corev1alpha.AuthorityRequest) {
	registrationApproved := make(chan bool, 1)
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if authorityRequestCopy.Status.Expiry != nil {
		timeout = time.After(time.Until(authorityRequestCopy.Status.Expiry.Time))
	}
	closeChannels := func() {
		close(registrationApproved)
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of authority request object
	watchAuthorityRequest, err := t.edgenetClientset.AppsV1alpha().AuthorityRequests().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", authorityRequestCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for authorityRequestEvent := range watchAuthorityRequest.ResultChan() {
				// Get updated authority request object
				updatedAuthorityRequest, status := authorityRequestEvent.Object.(*corev1alpha.AuthorityRequest)
				if authorityRequestCopy.GetUID() == updatedAuthorityRequest.GetUID() {
					if status {
						if authorityRequestEvent.Type == "DELETED" {
							terminated <- true
							continue
						}

						if updatedAuthorityRequest.Spec.Approved == true {
							registrationApproved <- true
							break
						} else if updatedAuthorityRequest.Status.Expiry != nil {
							// Check whether expiration date updated - TBD
							if updatedAuthorityRequest.Status.Expiry.Time.Sub(time.Now()) >= 0 {
								timeout = time.After(time.Until(updatedAuthorityRequest.Status.Expiry.Time))
								timeoutRenewed <- true
							} else {
								terminated <- true
							}
						}
						authorityRequestCopy = updatedAuthorityRequest
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching authorityrequest resources,
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
			watchAuthorityRequest.Stop()
			closeChannels()
			break timeoutLoop
		case <-timeoutRenewed:
			break timeoutOptions
		case <-timeout:
			watchAuthorityRequest.Stop()
			closeChannels()
			t.edgenetClientset.AppsV1alpha().AuthorityRequests().Delete(context.TODO(), authorityRequestCopy.GetName(), metav1.DeleteOptions{})
			break timeoutLoop
		case <-terminated:
			watchAuthorityRequest.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}

// SetAsOwnerReference put the authorityrequest as owner
func SetAsOwnerReference(authorityRequestCopy *corev1alpha.AuthorityRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(authorityRequestCopy, apps_v1alpha.SchemeGroupVersion.WithKind("AuthorityRequest"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
