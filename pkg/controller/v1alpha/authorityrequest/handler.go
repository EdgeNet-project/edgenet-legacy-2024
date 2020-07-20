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
	"fmt"
	"reflect"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/bootstrap"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/controller/v1alpha/authority"
	"edgenet/pkg/controller/v1alpha/emailverification"
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
	log.Info("authorityRequestHandler.Init")
	var err error
	t.clientset, err = bootstrap.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	t.edgenetClientset, err = bootstrap.CreateEdgeNetClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	return err
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("authorityRequestHandler.ObjectCreated")
	// Create a copy of the authority request object to make changes on it
	authorityRequestCopy := obj.(*apps_v1alpha.AuthorityRequest).DeepCopy()
	defer t.edgenetClientset.AppsV1alpha().AuthorityRequests().UpdateStatus(authorityRequestCopy)
	// Check if the email address of user or authority name is already taken
	exists, message := t.checkDuplicateObject(authorityRequestCopy)
	if exists {
		authorityRequestCopy.Status.State = failure
		authorityRequestCopy.Status.Message = message
		// Run timeout goroutine
		go t.runApprovalTimeout(authorityRequestCopy)
		// Set the approval timeout which is 24 hours
		authorityRequestCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(24 * time.Hour),
		}
		return
	}
	if authorityRequestCopy.Spec.Approved {
		authorityHandler := authority.Handler{}
		err := authorityHandler.Init()
		if err == nil {
			created := !authorityHandler.Create(authorityRequestCopy)
			if created {
				return
			} else {
				t.sendEmail("authority-creation-failure", authorityRequestCopy)
				authorityRequestCopy.Status.State = failure
				authorityRequestCopy.Status.Message = []string{"Authority establishment failed", err.Error()}
			}
		}
	}
	// If the service restarts, it creates all objects again
	// Because of that, this section covers a variety of possibilities
	if authorityRequestCopy.Status.Expires == nil {
		// Run timeout goroutine
		go t.runApprovalTimeout(authorityRequestCopy)
		// Set the approval timeout which is 72 hours
		authorityRequestCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		emailVerificationHandler := emailverification.Handler{}
		err := emailVerificationHandler.Init()
		if err == nil {
			created := emailVerificationHandler.Create(authorityRequestCopy, SetAsOwnerReference(authorityRequestCopy))
			if created {
				// Update the status as successful
				authorityRequestCopy.Status.State = success
				authorityRequestCopy.Status.Message = []string{"Everything is OK, verification email sent"}
			} else {
				authorityRequestCopy.Status.State = issue
				authorityRequestCopy.Status.Message = []string{"Couldn't send verification email"}
			}
		}
	} else {
		go t.runApprovalTimeout(authorityRequestCopy)
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("authorityRequestHandler.ObjectUpdated")
	// Create a copy of the authority request object to make changes on it
	authorityRequestCopy := obj.(*apps_v1alpha.AuthorityRequest).DeepCopy()
	changeStatus := false
	// Check if the email address of user or authority name is already taken
	exists, message := t.checkDuplicateObject(authorityRequestCopy)
	if !exists {
		// Check whether the request for authority creation approved
		if authorityRequestCopy.Spec.Approved {
			authorityHandler := authority.Handler{}
			err := authorityHandler.Init()
			if err == nil {
				changeStatus := authorityHandler.Create(authorityRequestCopy)
				if changeStatus {
					t.sendEmail("authority-creation-failure", authorityRequestCopy)
					authorityRequestCopy.Status.State = failure
					authorityRequestCopy.Status.Message = []string{"Authority establishment failed", err.Error()}
				}
			}
		} else if !authorityRequestCopy.Spec.Approved && authorityRequestCopy.Status.State == failure {
			emailVerificationHandler := emailverification.Handler{}
			err := emailVerificationHandler.Init()
			if err == nil {
				created := emailVerificationHandler.Create(authorityRequestCopy, SetAsOwnerReference(authorityRequestCopy))
				if created {
					// Update the status as successful
					authorityRequestCopy.Status.State = success
					authorityRequestCopy.Status.Message = []string{"Everything is OK, verification email sent"}
				} else {
					authorityRequestCopy.Status.State = issue
					authorityRequestCopy.Status.Message = []string{"Couldn't send verification email"}
				}
			}
			changeStatus = true
		}
	} else if exists && !reflect.DeepEqual(authorityRequestCopy.Status.Message, message) {
		authorityRequestCopy.Status.State = failure
		authorityRequestCopy.Status.Message = message
		changeStatus = true
	}
	if changeStatus {
		t.edgenetClientset.AppsV1alpha().AuthorityRequests().UpdateStatus(authorityRequestCopy)
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("authorityRequestHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(subject string, authorityRequestCopy *apps_v1alpha.AuthorityRequest) {
	// Set the HTML template variables
	var contentData = mailer.CommonContentData{}
	contentData.CommonData.Authority = authorityRequestCopy.GetName()
	contentData.CommonData.Username = authorityRequestCopy.Spec.Contact.Username
	contentData.CommonData.Name = fmt.Sprintf("%s %s", authorityRequestCopy.Spec.Contact.FirstName, authorityRequestCopy.Spec.Contact.LastName)
	contentData.CommonData.Email = []string{authorityRequestCopy.Spec.Contact.Email}
	mailer.Send(subject, contentData)
}

// checkDuplicateObject checks whether a user exists with the same email address
func (t *Handler) checkDuplicateObject(authorityRequestCopy *apps_v1alpha.AuthorityRequest) (bool, []string) {
	exists := false
	message := []string{}
	// To check username on the users resource
	authorityRaw, _ := t.edgenetClientset.AppsV1alpha().Authorities().List(
		metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", authorityRequestCopy.GetName())})
	if len(authorityRaw.Items) == 0 {
		// To check email address among users
		userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			if userRow.Spec.Email == authorityRequestCopy.Spec.Contact.Email {
				exists = true
				message = append(message, fmt.Sprintf("Email address, %s, already exists for another user account", authorityRequestCopy.Spec.Contact.Email))
				break
			}
		}
		// To check email address among user registration requests
		URRRaw, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests("").List(metav1.ListOptions{})
		for _, URRRow := range URRRaw.Items {
			if URRRow.Spec.Email == authorityRequestCopy.Spec.Contact.Email {
				exists = true
				message = append(message, fmt.Sprintf("Email address, %s, has already been used in a user registration request", authorityRequestCopy.Spec.Contact.Email))
				break
			}
		}
		// To check email address given at authority request
		authorityRequestRaw, _ := t.edgenetClientset.AppsV1alpha().AuthorityRequests().List(metav1.ListOptions{})
		for _, authorityRequestRow := range authorityRequestRaw.Items {
			if authorityRequestRow.Spec.Contact.Email == authorityRequestCopy.Spec.Contact.Email && authorityRequestRow.GetUID() != authorityRequestCopy.GetUID() {
				exists = true
				message = append(message, fmt.Sprintf("Email address, %s, has already been used in another authority request", authorityRequestCopy.Spec.Contact.Email))
				break
			}
		}
		if exists && !reflect.DeepEqual(authorityRequestCopy.Status.Message, message) {
			t.sendEmail("authority-validation-failure-email", authorityRequestCopy)

		}
	} else {
		exists = true
		message = append(message, fmt.Sprintf("Authority name, %s, is already taken", authorityRequestCopy.GetName()))
		if !reflect.DeepEqual(authorityRequestCopy.Status.Message, message) {
			t.sendEmail("authority-validation-failure-name", authorityRequestCopy)
		}
	}
	return exists, message
}

// runApprovalTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) runApprovalTimeout(authorityRequestCopy *apps_v1alpha.AuthorityRequest) {
	registrationApproved := make(chan bool, 1)
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if authorityRequestCopy.Status.Expires != nil {
		timeout = time.After(time.Until(authorityRequestCopy.Status.Expires.Time))
	}
	closeChannels := func() {
		close(registrationApproved)
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of authority request object
	watchauthorityRequest, err := t.edgenetClientset.AppsV1alpha().AuthorityRequests().Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", authorityRequestCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for authorityRequestEvent := range watchauthorityRequest.ResultChan() {
				// Get updated authority request object
				updatedAuthorityRequest, status := authorityRequestEvent.Object.(*apps_v1alpha.AuthorityRequest)
				if authorityRequestCopy.GetUID() == updatedAuthorityRequest.GetUID() {
					if status {
						if authorityRequestEvent.Type == "DELETED" {
							terminated <- true
							continue
						}

						if updatedAuthorityRequest.Spec.Approved == true {
							registrationApproved <- true
							break
						} else if updatedAuthorityRequest.Status.Expires != nil {
							// Check whether expiration date updated - TBD
							if updatedAuthorityRequest.Status.Expires.Time.Sub(time.Now()) >= 0 {
								timeout = time.After(time.Until(updatedAuthorityRequest.Status.Expires.Time))
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
			watchauthorityRequest.Stop()
			closeChannels()
			break timeoutLoop
		case <-timeoutRenewed:
			break timeoutOptions
		case <-timeout:
			watchauthorityRequest.Stop()
			closeChannels()
			t.edgenetClientset.AppsV1alpha().AuthorityRequests().Delete(authorityRequestCopy.GetName(), &metav1.DeleteOptions{})
			break timeoutLoop
		case <-terminated:
			watchauthorityRequest.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}

// SetOwnerReference put the authorityrequest as owner
func SetAsOwnerReference(authorityRequestCopy *apps_v1alpha.AuthorityRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(authorityRequestCopy, apps_v1alpha.SchemeGroupVersion.WithKind("AuthorityRequest"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
