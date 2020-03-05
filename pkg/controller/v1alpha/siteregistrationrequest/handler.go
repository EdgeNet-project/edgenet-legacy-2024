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

package siteregistrationrequest

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
	log.Info("SRRHandler.Init")
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
	log.Info("SRRHandler.ObjectCreated")
	// Create a copy of the site registration request object to make changes on it
	SRRCopy := obj.(*apps_v1alpha.SiteRegistrationRequest).DeepCopy()
	// Check if the email address of user is already taken
	exist := t.checkDuplicateObject(SRRCopy)
	if exist {
		// If it is already taken, remove the site registration request object
		t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Delete(SRRCopy.GetName(), &metav1.DeleteOptions{})
		return
	}
	defer t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().UpdateStatus(SRRCopy)
	SRRCopy.Status.Approved = false
	// If the service restarts, it creates all objects again
	// Because of that, this section covers a variety of possibilities
	if SRRCopy.Status.Expires == nil {
		// Run timeout goroutine
		go t.runApprovalTimeout(SRRCopy)
		// Set the approval timeout which is 72 hours
		SRRCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		// The section below is a part of the method which provides email verification
		// Email verification code is a security point for email verification. The user
		// registration object creates an email verification object with a name which is
		// this email verification code. Only who knows the site and the email verification
		// code can manipulate that object by using a public token.
		SRROwnerReferences := t.setOwnerReferences(SRRCopy)
		emailVerificationCode := "bs" + generateRandomString(16)
		emailVerification := apps_v1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: SRROwnerReferences}}
		emailVerification.SetName(emailVerificationCode)
		emailVerification.Spec.Kind = "Site"
		emailVerification.Spec.Identifier = SRRCopy.GetName()
		_, err := t.edgenetClientset.AppsV1alpha().EmailVerifications("site-edgenet").Create(emailVerification.DeepCopy())
		if err == nil {
			// Set the HTML template variables
			contentData := mailer.VerifyContentData{}
			contentData.CommonData.Site = SRRCopy.GetName()
			contentData.CommonData.Username = SRRCopy.Spec.Contact.Username
			contentData.CommonData.Name = fmt.Sprintf("%s %s", SRRCopy.Spec.Contact.FirstName, SRRCopy.Spec.Contact.LastName)
			contentData.CommonData.Email = []string{SRRCopy.Spec.Contact.Email}
			contentData.Code = emailVerificationCode
			mailer.Send("site-email-verification", contentData)
		}
	} else {
		go t.runApprovalTimeout(SRRCopy)
	}
	// Send en email to inform admins of cluster, TBD
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("SRRHandler.ObjectUpdated")
	// Create a copy of the site registration request object to make changes on it
	SRRCopy := obj.(*apps_v1alpha.SiteRegistrationRequest).DeepCopy()
	// Check whether the request for site registration approved
	if SRRCopy.Status.Approved {
		// Check if the email address of user is already taken
		exist := t.checkDuplicateObject(SRRCopy)
		log.Println(exist)
		if !exist {
			// Create a site on the cluster
			site := apps_v1alpha.Site{}
			site.SetName(SRRCopy.GetName())
			site.Spec.Address = SRRCopy.Spec.Address
			site.Spec.Contact = SRRCopy.Spec.Contact
			site.Spec.FullName = SRRCopy.Spec.FullName
			site.Spec.ShortName = SRRCopy.Spec.ShortName
			site.Spec.URL = SRRCopy.Spec.URL
			t.edgenetClientset.AppsV1alpha().Sites().Create(site.DeepCopy())
		}
		t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Delete(SRRCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("SRRHandler.ObjectDeleted")
	// Mail notification, TBD
}

// runApprovalTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) runApprovalTimeout(SRRCopy *apps_v1alpha.SiteRegistrationRequest) {
	registrationApproved := make(chan bool, 1)
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if SRRCopy.Status.Expires != nil {
		timeout = time.After(time.Until(SRRCopy.Status.Expires.Time))
	}
	closeChannels := func() {
		close(registrationApproved)
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of site registration request object
	watchSRR, err := t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", SRRCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for SRREvent := range watchSRR.ResultChan() {
				// Get updated site registration request object
				updatedSRR, status := SRREvent.Object.(*apps_v1alpha.SiteRegistrationRequest)
				if status {
					if SRREvent.Type == "DELETED" {
						terminated <- true
						continue
					}

					if updatedSRR.Status.Approved == true {
						registrationApproved <- true
						break
					} else if updatedSRR.Status.Expires != nil {
						timeout = time.After(time.Until(updatedSRR.Status.Expires.Time))
						// Check whether expiration date updated
						if SRRCopy.Status.Expires != nil {
							if SRRCopy.Status.Expires.Time != updatedSRR.Status.Expires.Time {
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
		// In case of any malfunction of watching siteregistrationrequest resources,
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
			watchSRR.Stop()
			closeChannels()
			break timeoutLoop
		case <-timeoutRenewed:
			break timeoutOptions
		case <-timeout:
			watchSRR.Stop()
			closeChannels()
			t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Delete(SRRCopy.GetName(), &metav1.DeleteOptions{})
			break timeoutLoop
		case <-terminated:
			watchSRR.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}

// checkDuplicateObject checks whether a user exists with the same email address
func (t *Handler) checkDuplicateObject(SRRCopy *apps_v1alpha.SiteRegistrationRequest) bool {
	exist := false
	// To check username on the users resource
	siteRaw, _ := t.edgenetClientset.AppsV1alpha().Sites().List(
		metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", SRRCopy.GetName())})
	if len(siteRaw.Items) == 0 {
		// To check email address
		userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			if userRow.Spec.Email == SRRCopy.Spec.Contact.Email {
				exist = true
				break
			}
		}

		if !exist {
			// To check email address
			URRRaw, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests("").List(metav1.ListOptions{})
			for _, URRRow := range URRRaw.Items {
				if URRRow.Spec.Email == SRRCopy.Spec.Contact.Email {
					exist = true
				}
			}
			if !exist {
				// To check username and email address given at SRR
				SRRRaw, _ := t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().List(metav1.ListOptions{})
				for _, SRRRow := range SRRRaw.Items {
					if SRRRow.Spec.Contact.Email == SRRCopy.Spec.Contact.Email && SRRRow.GetUID() != SRRCopy.GetUID() {
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

// setOwnerReferences put the siteregistrationrequest as owner
func (t *Handler) setOwnerReferences(SRRCopy *apps_v1alpha.SiteRegistrationRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(SRRCopy, apps_v1alpha.SchemeGroupVersion.WithKind("SiteRegistrationRequest"))
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
