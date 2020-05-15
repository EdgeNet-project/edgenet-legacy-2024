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
	"math/rand"
	"reflect"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/authorization"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/mailer"

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
	log.Info("URRHandler.Init")
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
	if URROwnerAuthority.Status.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if URRCopy.Status.Expires == nil {
			// Run timeout goroutine
			go t.runApprovalTimeout(URRCopy)
			defer t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).UpdateStatus(URRCopy)
			URRCopy.Status.Approved = false
			// Set the approval timeout which is 72 hours
			URRCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(72 * time.Hour),
			}
			URRCopy = t.setEmailVerification(URRCopy, URROwnerNamespace.Labels["authority-name"])
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
	statusChange := false
	URROwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(URRCopy.GetNamespace(), metav1.GetOptions{})
	URROwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(URROwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	if URROwnerAuthority.Status.Enabled {
		// Check again if the email address is already taken
		exists, message := t.checkDuplicateObject(URRCopy, URROwnerNamespace.Labels["authority-name"])
		if !exists {
			// Check whether the request for user registration approved
			if URRCopy.Status.Approved {
				// Create a user on authority
				user := apps_v1alpha.User{}
				user.SetName(URRCopy.GetName())
				user.Spec.Bio = URRCopy.Spec.Bio
				user.Spec.Email = URRCopy.Spec.Email
				user.Spec.FirstName = URRCopy.Spec.FirstName
				user.Spec.LastName = URRCopy.Spec.LastName
				user.Spec.Roles = URRCopy.Spec.Roles
				user.Spec.URL = URRCopy.Spec.URL
				_, err := t.edgenetClientset.AppsV1alpha().Users(URRCopy.GetNamespace()).Create(user.DeepCopy())
				if err == nil {
					t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).Delete(URRCopy.GetName(), &metav1.DeleteOptions{})
				} else {
					t.sendEmail(URRCopy, URROwnerNamespace.Labels["authority-name"], "", "user-creation-failure")
					statusChange = true
					URRCopy.Status.State = failure
					URRCopy.Status.Message = []string{"User creation failed", err.Error()}
				}
			} else if !URRCopy.Status.Approved && URRCopy.Status.State == failure {
				URRCopy = t.setEmailVerification(URRCopy, URROwnerNamespace.Labels["authority-name"])
				statusChange = true
			}
		} else if exists && !reflect.DeepEqual(URRCopy.Status.Message, message) {
			URRCopy.Status.State = failure
			URRCopy.Status.Message = message
			statusChange = true
		}
		if statusChange {
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

// setEmailVerification to provide one-time code for verification
func (t *Handler) setEmailVerification(URRCopy *apps_v1alpha.UserRegistrationRequest, authorityName string) *apps_v1alpha.UserRegistrationRequest {
	// The section below is a part of the method which provides email verification
	// Email verification code is a security point for email verification. The user
	// registration object creates an email verification object with a name which is
	// this email verification code. Only who knows the authority and the email verification
	// code can manipulate that object by using a public token.
	URROwnerReferences := t.setOwnerReferences(URRCopy)
	emailVerificationCode := "bs" + generateRandomString(16)
	emailVerification := apps_v1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: URROwnerReferences}}
	emailVerification.SetName(emailVerificationCode)
	emailVerification.Spec.Kind = "User"
	emailVerification.Spec.Identifier = URRCopy.GetName()
	_, err := t.edgenetClientset.AppsV1alpha().EmailVerifications(URRCopy.GetNamespace()).Create(emailVerification.DeepCopy())
	if err == nil {
		t.sendEmail(URRCopy, authorityName, emailVerificationCode, "user-email-verification")
		// Update the status as successful
		URRCopy.Status.State = success
		URRCopy.Status.Message = []string{"Everything is OK, verification email sent"}
	} else {
		t.sendEmail(URRCopy, authorityName, emailVerificationCode, "user-email-verification-malfunction")
		URRCopy.Status.State = issue
		URRCopy.Status.Message = []string{"Couldn't send verification email"}
	}
	return URRCopy
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(URRCopy *apps_v1alpha.UserRegistrationRequest, authorityName, emailVerificationCode, subject string) {
	// Set the HTML template variables
	var contentData interface{}
	var collective = mailer.CommonContentData{}
	collective.CommonData.Authority = authorityName
	collective.CommonData.Username = URRCopy.GetName()
	collective.CommonData.Name = fmt.Sprintf("%s %s", URRCopy.Spec.FirstName, URRCopy.Spec.LastName)
	collective.CommonData.Email = []string{URRCopy.Spec.Email}
	if emailVerificationCode != "" {
		verifyContent := mailer.VerifyContentData{}
		verifyContent.Code = emailVerificationCode
		verifyContent.CommonData = collective.CommonData
		contentData = verifyContent
	} else {
		contentData = collective
	}
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
				if status {
					if URREvent.Type == "DELETED" {
						terminated <- true
						continue
					}

					if updatedURR.Status.Approved == true {
						registrationApproved <- true
						break
					} else if updatedURR.Status.Expires != nil {
						timeout = time.After(time.Until(updatedURR.Status.Expires.Time))
						// Check whether expiration date updated
						if URRCopy.Status.Expires != nil {
							if URRCopy.Status.Expires.Time != updatedURR.Status.Expires.Time {
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
	userRaw, _ := t.edgenetClientset.AppsV1alpha().Users(URRCopy.GetNamespace()).List(
		metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", URRCopy.GetName())})
	if len(userRaw.Items) == 0 {
		// To check email address
		userRaw, _ = t.edgenetClientset.AppsV1alpha().Users("").List(metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			if userRow.Spec.Email == URRCopy.Spec.Email {
				exists = true
				message = append(message, fmt.Sprintf("Email address, %s, already exists for another user account", URRCopy.Spec.Email))
				break
			}
		}
		if !exists {
			// To check email address
			URRRaw, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests("").List(metav1.ListOptions{})
			for _, URRRow := range URRRaw.Items {
				if URRRow.Spec.Email == URRCopy.Spec.Email && URRRow.GetUID() != URRCopy.GetUID() {
					exists = true
					message = append(message, fmt.Sprintf("Email address, %s, already exists for another user registration request", URRCopy.Spec.Email))
				}
			}
			if !exists {
				// To check email address given at authorityRequest
				authorityRequestRaw, _ := t.edgenetClientset.AppsV1alpha().AuthorityRequests().List(metav1.ListOptions{})
				for _, authorityRequestRow := range authorityRequestRaw.Items {
					if authorityRequestRow.Spec.Contact.Email == URRCopy.Spec.Email {
						exists = true
						message = append(message, fmt.Sprintf("Email address, %s, already exists for another authority request", URRCopy.Spec.Email))
					}
				}
			}
		}
		if exists && !reflect.DeepEqual(URRCopy.Status.Message, message) {
			t.sendEmail(URRCopy, authorityName, "", "user-validation-failure-email")
		}
	} else {
		exists = true
		message = append(message, fmt.Sprintf("Username, %s, already exists for another user account", URRCopy.GetName()))
		if exists && !reflect.DeepEqual(URRCopy.Status.Message, message) {
			t.sendEmail(URRCopy, authorityName, "", "user-validation-failure-name")
		}
	}
	return exists, message
}

// setOwnerReferences put the userregistrationrequest as owner
func (t *Handler) setOwnerReferences(URRCopy *apps_v1alpha.UserRegistrationRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(URRCopy, apps_v1alpha.SchemeGroupVersion.WithKind("UserRegistrationRequest"))
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
