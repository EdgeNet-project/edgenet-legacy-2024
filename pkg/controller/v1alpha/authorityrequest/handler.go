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
	"math/rand"
	"time"

	apps_v1alpha "headnode/pkg/apis/apps/v1alpha"
	"headnode/pkg/authorization"
	"headnode/pkg/client/clientset/versioned"
	"headnode/pkg/mailer"

	log "github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init() error
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        *kubernetes.Clientset
	edgenetClientset *versioned.Clientset
}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("authorityRequestHandler.Init")
	var err error
	t.clientset, err = authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	t.edgenetClientset, err = authorization.CreateEdgeNetClientSet()
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
	// Check if the email address of user is already taken
	exist := t.checkDuplicateObject(authorityRequestCopy)
	if exist {
		// If it is already taken, remove the authority request object
		t.edgenetClientset.AppsV1alpha().AuthorityRequests().Delete(authorityRequestCopy.GetName(), &metav1.DeleteOptions{})
		return
	}
	defer t.edgenetClientset.AppsV1alpha().AuthorityRequests().UpdateStatus(authorityRequestCopy)
	authorityRequestCopy.Status.Approved = false
	// If the service restarts, it creates all objects again
	// Because of that, this section covers a variety of possibilities
	if authorityRequestCopy.Status.Expires == nil {
		// Run timeout goroutine
		go t.runApprovalTimeout(authorityRequestCopy)
		// Set the approval timeout which is 72 hours
		authorityRequestCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		// The section below is a part of the method which provides email verification
		// Email verification code is a security point for email verification. The user
		// registration object creates an email verification object with a name which is
		// this email verification code. Only who knows the authority and the email verification
		// code can manipulate that object by using a public token.
		authorityRequestOwnerReferences := t.setOwnerReferences(authorityRequestCopy)
		emailVerificationCode := "bs" + generateRandomString(16)
		emailVerification := apps_v1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: authorityRequestOwnerReferences}}
		emailVerification.SetName(emailVerificationCode)
		emailVerification.Spec.Kind = "Authority"
		emailVerification.Spec.Identifier = authorityRequestCopy.GetName()
		_, err := t.edgenetClientset.AppsV1alpha().EmailVerifications("registration").Create(emailVerification.DeepCopy())
		if err == nil {
			// Set the HTML template variables
			contentData := mailer.VerifyContentData{}
			contentData.CommonData.Authority = authorityRequestCopy.GetName()
			contentData.CommonData.Username = authorityRequestCopy.Spec.Contact.Username
			contentData.CommonData.Name = fmt.Sprintf("%s %s", authorityRequestCopy.Spec.Contact.FirstName, authorityRequestCopy.Spec.Contact.LastName)
			contentData.CommonData.Email = []string{authorityRequestCopy.Spec.Contact.Email}
			contentData.Code = emailVerificationCode
			mailer.Send("authority-email-verification", contentData)
		}
	} else {
		go t.runApprovalTimeout(authorityRequestCopy)
	}
	// Send en email to inform admins of cluster, TBD
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("authorityRequestHandler.ObjectUpdated")
	// Create a copy of the authority request object to make changes on it
	authorityRequestCopy := obj.(*apps_v1alpha.AuthorityRequest).DeepCopy()
	// Check whether the request for authority creation approved
	if authorityRequestCopy.Status.Approved {
		// Check if the email address of user is already taken
		exist := t.checkDuplicateObject(authorityRequestCopy)
		log.Println(exist)
		if !exist {
			// Create a authority on the cluster
			authority := apps_v1alpha.Authority{}
			authority.SetName(authorityRequestCopy.GetName())
			authority.Spec.Address = authorityRequestCopy.Spec.Address
			authority.Spec.Contact = authorityRequestCopy.Spec.Contact
			authority.Spec.FullName = authorityRequestCopy.Spec.FullName
			authority.Spec.ShortName = authorityRequestCopy.Spec.ShortName
			authority.Spec.URL = authorityRequestCopy.Spec.URL
			t.edgenetClientset.AppsV1alpha().Authorities().Create(authority.DeepCopy())
		}
		t.edgenetClientset.AppsV1alpha().AuthorityRequests().Delete(authorityRequestCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("authorityRequestHandler.ObjectDeleted")
	// Mail notification, TBD
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
				updatedauthorityRequest, status := authorityRequestEvent.Object.(*apps_v1alpha.AuthorityRequest)
				if status {
					if authorityRequestEvent.Type == "DELETED" {
						terminated <- true
						continue
					}

					if updatedauthorityRequest.Status.Approved == true {
						registrationApproved <- true
						break
					} else if updatedauthorityRequest.Status.Expires != nil {
						timeout = time.After(time.Until(updatedauthorityRequest.Status.Expires.Time))
						// Check whether expiration date updated
						if authorityRequestCopy.Status.Expires != nil {
							if authorityRequestCopy.Status.Expires.Time != updatedauthorityRequest.Status.Expires.Time {
								timeoutRenewed <- true
							}
						} else {
							timeoutRenewed <- true
						}
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

// checkDuplicateObject checks whether a user exists with the same email address
func (t *Handler) checkDuplicateObject(authorityRequestCopy *apps_v1alpha.AuthorityRequest) bool {
	exist := false
	// To check username on the users resource
	authorityRaw, _ := t.edgenetClientset.AppsV1alpha().Authorities().List(
		metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", authorityRequestCopy.GetName())})
	if len(authorityRaw.Items) == 0 {
		// To check email address
		userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			if userRow.Spec.Email == authorityRequestCopy.Spec.Contact.Email {
				exist = true
				break
			}
		}

		if !exist {
			// To check email address
			URRRaw, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests("").List(metav1.ListOptions{})
			for _, URRRow := range URRRaw.Items {
				if URRRow.Spec.Email == authorityRequestCopy.Spec.Contact.Email {
					exist = true
				}
			}
			if !exist {
				// To check username and email address given at authorityRequest
				authorityRequestRaw, _ := t.edgenetClientset.AppsV1alpha().AuthorityRequests().List(metav1.ListOptions{})
				for _, authorityRequestRow := range authorityRequestRaw.Items {
					if authorityRequestRow.Spec.Contact.Email == authorityRequestCopy.Spec.Contact.Email && authorityRequestRow.GetUID() != authorityRequestCopy.GetUID() {
						exist = true
					}
				}
			}
		}
	} else {
		exist = true
	}
	// Mail notification, TBD
	return exist
}

// setOwnerReferences put the authorityrequest as owner
func (t *Handler) setOwnerReferences(authorityRequestCopy *apps_v1alpha.AuthorityRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(authorityRequestCopy, apps_v1alpha.SchemeGroupVersion.WithKind("AuthorityRequest"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}

// generateRandomString to have a unique string
func generateRandomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
