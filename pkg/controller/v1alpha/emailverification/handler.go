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

package emailverification

import (
	"context"
	"fmt"
	"strings"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface)
	ObjectCreated(obj interface{})
	ObjectUpdated(obj, updated interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        kubernetes.Interface
	edgenetClientset versioned.Interface
}

// Init handles any handler initialization
func (t *Handler) Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) {
	log.Info("EVHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("EVHandler.ObjectCreated")
	// Create a copy of the email verification object to make changes on it
	EVCopy := obj.(*apps_v1alpha.EmailVerification).DeepCopy()
	// Find the authority from the namespace in which the object is
	EVOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), EVCopy.GetNamespace(), metav1.GetOptions{})
	// If the object's kind is AuthorityRequest, `registration` namespace hosts the email verification object.
	// Otherwise, the object belongs to the namespace that the authority created.
	var authorityEnabled bool
	if EVOwnerNamespace.GetName() == "registration" {
		authorityEnabled = true
	} else {
		EVOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), EVOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
		authorityEnabled = EVOwnerAuthority.Spec.Enabled
	}
	// Check if the authority is active
	if authorityEnabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if EVCopy.Spec.Verified {
			t.objectConfiguration(EVCopy, EVOwnerNamespace.Labels["authority-name"])
		} else if !EVCopy.Spec.Verified && EVCopy.Status.Expires == nil {
			// Run timeout goroutine
			go t.runVerificationTimeout(EVCopy)
			defer t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).UpdateStatus(context.TODO(), EVCopy, metav1.UpdateOptions{})
			// Set the email verification timeout which is 24 hours
			EVCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(24 * time.Hour),
			}
		} else if !EVCopy.Spec.Verified && EVCopy.Status.Expires != nil {
			// Check if the email verification expired
			if EVCopy.Status.Expires.Time.Sub(time.Now()) >= 0 {
				go t.runVerificationTimeout(EVCopy)
			} else {
				t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(context.TODO(), EVCopy.GetName(), metav1.DeleteOptions{})
			}
		}
	} else {
		t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(context.TODO(), EVCopy.GetName(), metav1.DeleteOptions{})
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("EVHandler.ObjectUpdated")
	// Create a copy of the email verification object to make changes on it
	EVCopy := obj.(*apps_v1alpha.EmailVerification).DeepCopy()
	EVOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), EVCopy.GetNamespace(), metav1.GetOptions{})
	// Security check to prevent any kind of manipulation on the email verification
	fieldUpdated := updated.(fields)
	if fieldUpdated.kind || fieldUpdated.identifier {
		t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(context.TODO(), EVCopy.GetName(), metav1.DeleteOptions{})
		if strings.ToLower(EVCopy.Spec.Kind) == "authority" {
			t.sendEmail("authority-email-verification-dubious", EVCopy.Spec.Identifier, EVCopy.GetNamespace(), "", "", "", "")
		} else if strings.ToLower(EVCopy.Spec.Kind) == "user" || strings.ToLower(EVCopy.Spec.Kind) == "email" {
			t.sendEmail("user-email-verification-dubious", EVOwnerNamespace.Labels["authority-name"], EVCopy.GetNamespace(), EVCopy.Spec.Identifier, "", "", "")
		}
		return
	}
	var authorityEnabled bool
	if EVOwnerNamespace.GetName() == "registration" {
		authorityEnabled = true
	} else {
		EVOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), EVOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
		authorityEnabled = EVOwnerAuthority.Spec.Enabled
	}
	// Check whether the authority enabled
	if authorityEnabled {
		// Check whether the email verification is done
		if EVCopy.Spec.Verified {
			t.objectConfiguration(EVCopy, EVOwnerNamespace.Labels["authority-name"])
		}
	} else {
		t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(context.TODO(), EVCopy.GetName(), metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("EVHandler.ObjectDeleted")
	// Mail notification, TBD
}

// Create to provide one-time code for verification
func (t *Handler) Create(obj interface{}, ownerReferences []metav1.OwnerReference) bool {
	// The section below is a part of the method which provides email verification
	// Email verification code is a security point for email verification. The user
	// registration object creates an email verification object with a name which is
	// this email verification code. Only who knows the email verification
	// code can manipulate that object by using a public token.
	created := false
	switch obj.(type) {
	case *apps_v1alpha.AuthorityRequest:
		authorityRequestCopy := obj.(*apps_v1alpha.AuthorityRequest)
		code := "bs" + util.GenerateRandomString(16)
		emailVerification := apps_v1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Kind = "Authority"
		emailVerification.Spec.Identifier = authorityRequestCopy.GetName()
		_, err := t.edgenetClientset.AppsV1alpha().EmailVerifications("registration").Create(context.TODO(), emailVerification.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			created = true
			t.sendEmail("authority-email-verification", authorityRequestCopy.GetName(), "", authorityRequestCopy.Spec.Contact.Username,
				fmt.Sprintf("%s %s", authorityRequestCopy.Spec.Contact.FirstName, authorityRequestCopy.Spec.Contact.LastName), authorityRequestCopy.Spec.Contact.Email, code)
		} else {
			t.sendEmail("authority-email-verification-malfunction", authorityRequestCopy.GetName(), "", authorityRequestCopy.Spec.Contact.Username,
				fmt.Sprintf("%s %s", authorityRequestCopy.Spec.Contact.FirstName, authorityRequestCopy.Spec.Contact.LastName), authorityRequestCopy.Spec.Contact.Email, "")
		}
	case *apps_v1alpha.User:
		userCopy := obj.(*apps_v1alpha.User)
		userOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), userCopy.GetNamespace(), metav1.GetOptions{})
		code := "bs" + util.GenerateRandomString(16)
		emailVerification := apps_v1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Kind = "Email"
		emailVerification.Spec.Identifier = userCopy.GetName()
		_, err := t.edgenetClientset.AppsV1alpha().EmailVerifications(userCopy.GetNamespace()).Create(context.TODO(), emailVerification.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			t.sendEmail("user-email-verification-update", userOwnerNamespace.Labels["authority-name"], userCopy.GetNamespace(), userCopy.GetName(),
				fmt.Sprintf("%s %s", userCopy.Spec.FirstName, userCopy.Spec.LastName), userCopy.Spec.Email, code)
		} else {
			t.sendEmail("user-email-verification-update-malfunction", userOwnerNamespace.Labels["authority-name"], userCopy.GetNamespace(), userCopy.GetName(),
				fmt.Sprintf("%s %s", userCopy.Spec.FirstName, userCopy.Spec.LastName), userCopy.Spec.Email, "")
		}
	case *apps_v1alpha.UserRegistrationRequest:
		URRCopy := obj.(*apps_v1alpha.UserRegistrationRequest)
		URROwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), URRCopy.GetNamespace(), metav1.GetOptions{})
		code := "bs" + util.GenerateRandomString(16)
		emailVerification := apps_v1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Kind = "User"
		emailVerification.Spec.Identifier = URRCopy.GetName()
		_, err := t.edgenetClientset.AppsV1alpha().EmailVerifications(URRCopy.GetNamespace()).Create(context.TODO(), emailVerification.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			t.sendEmail("user-email-verification-update", URROwnerNamespace.Labels["authority-name"], URRCopy.GetNamespace(), URRCopy.GetName(),
				fmt.Sprintf("%s %s", URRCopy.Spec.FirstName, URRCopy.Spec.LastName), URRCopy.Spec.Email, code)
		} else {
			t.sendEmail("user-email-verification-malfunction", URROwnerNamespace.Labels["authority-name"], URRCopy.GetNamespace(), URRCopy.GetName(),
				fmt.Sprintf("%s %s", URRCopy.Spec.FirstName, URRCopy.Spec.LastName), URRCopy.Spec.Email, "")
		}
	}
	return created
}

// sendEmail to send notification to authority-admins and authorized users about email verification
func (t *Handler) sendEmail(subject, authority, namespace, username, fullname, email, code string) {
	// Set the HTML template variables
	var contentData interface{}

	collective := mailer.CommonContentData{}
	collective.CommonData.Authority = authority
	collective.CommonData.Username = username
	collective.CommonData.Name = fullname
	collective.CommonData.Email = []string{}
	if subject == "authority-email-verification" || subject == "user-email-verification-update" ||
		subject == "user-email-verification" {
		collective.CommonData.Email = []string{email}
		verifyContent := mailer.VerifyContentData{}
		verifyContent.Code = code
		verifyContent.CommonData = collective.CommonData
		contentData = verifyContent
	} else if subject == "user-email-verified-alert" {
		// Put the email addresses of the authority-admins and authorized users in the email to be sent list
		userRaw, _ := t.edgenetClientset.AppsV1alpha().Users(namespace).List(context.TODO(), metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			if strings.ToLower(userRow.Status.Type) == "admin" {
				collective.CommonData.Email = append(collective.CommonData.Email, userRow.Spec.Email)
			}
		}
		contentData = collective
	} else {
		collective.CommonData.Email = []string{email}
		contentData = collective
	}

	mailer.Send(subject, contentData)
}

// objectConfiguration to update the objects that are relevant the request and send email
func (t *Handler) objectConfiguration(EVCopy *apps_v1alpha.EmailVerification, authorityName string) {
	// Update the status of request related to email verification
	if strings.ToLower(EVCopy.Spec.Kind) == "authority" {
		SRRObj, _ := t.edgenetClientset.AppsV1alpha().AuthorityRequests().Get(context.TODO(), EVCopy.Spec.Identifier, metav1.GetOptions{})
		SRRObj.Status.EmailVerified = true
		t.edgenetClientset.AppsV1alpha().AuthorityRequests().UpdateStatus(context.TODO(), SRRObj, metav1.UpdateOptions{})
		// Send email to inform admins of the cluster
		t.sendEmail("authority-email-verified-alert", EVCopy.Spec.Identifier, EVCopy.GetNamespace(), SRRObj.Spec.Contact.Username,
			fmt.Sprintf("%s %s", SRRObj.Spec.Contact.FirstName, SRRObj.Spec.Contact.LastName), "", "")
	} else if strings.ToLower(EVCopy.Spec.Kind) == "user" {
		URRObj, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(EVCopy.GetNamespace()).Get(context.TODO(), EVCopy.Spec.Identifier, metav1.GetOptions{})
		URRObj.Status.EmailVerified = true
		t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRObj.GetNamespace()).UpdateStatus(context.TODO(), URRObj, metav1.UpdateOptions{})
		// Send email to inform authority-admins and authorized users
		t.sendEmail("user-email-verified-alert", authorityName, EVCopy.GetNamespace(), EVCopy.Spec.Identifier,
			fmt.Sprintf("%s %s", URRObj.Spec.FirstName, URRObj.Spec.LastName), "", "")
	} else if strings.ToLower(EVCopy.Spec.Kind) == "email" {
		userObj, _ := t.edgenetClientset.AppsV1alpha().Users(EVCopy.GetNamespace()).Get(context.TODO(), EVCopy.Spec.Identifier, metav1.GetOptions{})
		userObj.Spec.Active = true
		t.edgenetClientset.AppsV1alpha().Users(userObj.GetNamespace()).UpdateStatus(context.TODO(), userObj, metav1.UpdateOptions{})
		if userObj.Status.Type == "admin" {
			authorityObj, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), authorityName, metav1.GetOptions{})
			if authorityObj.Spec.Contact.Username == userObj.GetName() {
				authorityObj.Spec.Contact.Email = userObj.Spec.Email
				t.edgenetClientset.AppsV1alpha().Authorities().Update(context.TODO(), authorityObj, metav1.UpdateOptions{})
			}
		}
		// Send email to inform user
		t.sendEmail("user-email-verified-notification", authorityName, EVCopy.GetNamespace(), EVCopy.Spec.Identifier,
			fmt.Sprintf("%s %s", userObj.Spec.FirstName, userObj.Spec.LastName), userObj.Spec.Email, "")
	}
	// Delete the unique email verification object as it gets verified
	t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(context.TODO(), EVCopy.GetName(), metav1.DeleteOptions{})
}

// runVerificationTimeout puts a procedure in place to remove requests by verification or timeout
func (t *Handler) runVerificationTimeout(EVCopy *apps_v1alpha.EmailVerification) {
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if EVCopy.Status.Expires != nil {
		timeout = time.After(time.Until(EVCopy.Status.Expires.Time))
	}
	closeChannels := func() {
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of email verification object
	watchEV, err := t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", EVCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for EVEvent := range watchEV.ResultChan() {
				// Get updated email verification object
				updatedEV, status := EVEvent.Object.(*apps_v1alpha.EmailVerification)
				if EVCopy.GetUID() == updatedEV.GetUID() {
					if status {
						if EVEvent.Type == "DELETED" {
							terminated <- true
							continue
						}

						if updatedEV.Status.Expires != nil {
							// Check whether expiration date updated - TBD
							/*if EVCopy.Status.Expires != nil && timeout != nil {
								if EVCopy.Status.Expires.Time == updatedEV.Status.Expires.Time {
									EVCopy = updatedEV
									continue
								}
							}*/

							if updatedEV.Status.Expires.Time.Sub(time.Now()) >= 0 {
								timeout = time.After(time.Until(updatedEV.Status.Expires.Time))
								timeoutRenewed <- true
							} else {
								terminated <- true
							}
						}
						EVCopy = updatedEV
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching emailverification resources,
		// there is a timeout at 3 hours
		timeout = time.After(3 * time.Hour)
	}

	// Infinite loop
timeoutLoop:
	for {
		// Wait on multiple channel operations
	timeoutOptions:
		select {
		case <-timeoutRenewed:
			break timeoutOptions
		case <-timeout:
			watchEV.Stop()
			t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(context.TODO(), EVCopy.GetName(), metav1.DeleteOptions{})
			closeChannels()
			break timeoutLoop
		case <-terminated:
			watchEV.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}

// To check whether user is holder of a role
func containsRole(roles []string, value string) bool {
	for _, ele := range roles {
		if strings.ToLower(value) == strings.ToLower(ele) {
			return true
		}
	}
	return false
}
