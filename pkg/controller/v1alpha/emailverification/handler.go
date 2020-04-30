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
	"fmt"
	"strings"
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
	ObjectUpdated(obj, updated interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        *kubernetes.Clientset
	edgenetClientset *versioned.Clientset
}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("EVHandler.Init")
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
	log.Info("EVHandler.ObjectCreated")
	// Create a copy of the email verification object to make changes on it
	EVCopy := obj.(*apps_v1alpha.EmailVerification).DeepCopy()
	// Find the site from the namespace in which the object is
	EVOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(EVCopy.GetNamespace(), metav1.GetOptions{})
	// If the object's kind is SiteRegistrationRequest, `site-edgenet` namespace hosts the email verification object.
	// Otherwise, the object belongs to the namespace that the site created.
	var siteEnabled bool
	if EVOwnerNamespace.GetName() == "site-edgenet" {
		siteEnabled = true
	} else {
		EVOwnerSite, _ := t.edgenetClientset.AppsV1alpha().Sites().Get(EVOwnerNamespace.Labels["site-name"], metav1.GetOptions{})
		siteEnabled = EVOwnerSite.Status.Enabled
	}
	// Check if the site is active
	if siteEnabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if EVCopy.Spec.Verified {
			// Update the status of request related to email verification
			if strings.ToLower(EVCopy.Spec.Kind) == "site" {
				SRRObj, _ := t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Get(EVCopy.Spec.Identifier, metav1.GetOptions{})
				SRRObj.Status.EmailVerify = true
				t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().UpdateStatus(SRRObj)
				// Send email to inform admins of the cluster
				t.sendEmail("site-email-verified-alert", EVCopy.Spec.Identifier, EVOwnerNamespace.GetName(), "")
			} else if strings.ToLower(EVCopy.Spec.Kind) == "user" {
				URRObj, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(EVCopy.GetNamespace()).Get(EVCopy.Spec.Identifier, metav1.GetOptions{})
				URRObj.Status.EmailVerify = true
				t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRObj.GetNamespace()).UpdateStatus(URRObj)
				// Send email to inform PIs and managers
				t.sendEmail("user-email-verified-alert", EVOwnerNamespace.Labels["site-name"], EVOwnerNamespace.GetName(), EVCopy.Spec.Identifier)
			}
			// Delete the unique email verification object as it gets verified
			t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(EVCopy.GetName(), &metav1.DeleteOptions{})
		} else if !EVCopy.Spec.Verified && EVCopy.Status.Expires == nil {
			// Run timeout goroutine
			go t.runVerificationTimeout(EVCopy)
			defer t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).UpdateStatus(EVCopy)
			if EVCopy.Status.Renew {
				EVCopy.Status.Renew = false
			}
			// Set the email verification timeout which is 24 hours
			EVCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(24 * time.Hour),
			}
		} else if !EVCopy.Spec.Verified && EVCopy.Status.Expires != nil {
			// Check if the email verification expired
			if EVCopy.Status.Expires.Time.Sub(time.Now()) >= 0 {
				go t.runVerificationTimeout(EVCopy)
				if EVCopy.Status.Renew {
					EVCopy.Status.Renew = false
					t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).UpdateStatus(EVCopy)
				}
			} else {
				t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(EVCopy.GetName(), &metav1.DeleteOptions{})
			}
		}
	} else {
		t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(EVCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("EVHandler.ObjectUpdated")
	// Create a copy of the email verification object to make changes on it
	EVCopy := obj.(*apps_v1alpha.EmailVerification).DeepCopy()
	// Security check to prevent any kind of manipulation on the email verification
	fieldUpdated := updated.(fields)
	if fieldUpdated.kind || fieldUpdated.identifier {
		t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(EVCopy.GetName(), &metav1.DeleteOptions{})
		return
	}
	EVOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(EVCopy.GetNamespace(), metav1.GetOptions{})
	var siteEnabled bool
	if EVOwnerNamespace.GetName() == "site-edgenet" {
		siteEnabled = true
	} else {
		EVOwnerSite, _ := t.edgenetClientset.AppsV1alpha().Sites().Get(EVOwnerNamespace.Labels["site-name"], metav1.GetOptions{})
		siteEnabled = EVOwnerSite.Status.Enabled
	}
	// Check whether the site enabled
	if siteEnabled {
		// Check whether the email verification is done
		if EVCopy.Spec.Verified {
			if strings.ToLower(EVCopy.Spec.Kind) == "site" {
				SRRObj, _ := t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().Get(EVCopy.Spec.Identifier, metav1.GetOptions{})
				SRRObj.Status.EmailVerify = true
				t.edgenetClientset.AppsV1alpha().SiteRegistrationRequests().UpdateStatus(SRRObj)
				t.sendEmail("site-email-verified-alert", EVCopy.Spec.Identifier, EVOwnerNamespace.GetName(), "")
			} else if strings.ToLower(EVCopy.Spec.Kind) == "user" {
				URRObj, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(EVCopy.GetNamespace()).Get(EVCopy.Spec.Identifier, metav1.GetOptions{})
				URRObj.Status.EmailVerify = true
				t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRObj.GetNamespace()).UpdateStatus(URRObj)
				t.sendEmail("user-email-verified-alert", EVOwnerNamespace.Labels["site-name"], EVOwnerNamespace.GetName(), EVCopy.Spec.Identifier)
			}
			t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(EVCopy.GetName(), &metav1.DeleteOptions{})
		} else {
			defer t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).UpdateStatus(EVCopy)
			// Extend the expiration date
			if EVCopy.Status.Renew {
				EVCopy.Status.Expires = &metav1.Time{
					Time: time.Now().Add(24 * time.Hour),
				}
			}
			EVCopy.Status.Renew = false
		}
	} else {
		t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(EVCopy.GetName(), &metav1.DeleteOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("EVHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to PIs and managers about email verification
func (t *Handler) sendEmail(kind, site, namespace, username string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Site = site
	contentData.CommonData.Username = username
	contentData.CommonData.Email = []string{}
	if kind == "user-email-verified-alert" {
		// Put the email addresses of the site PI and managers in the email to be sent list
		userRaw, _ := t.edgenetClientset.AppsV1alpha().Users(namespace).List(metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			for _, userRole := range userRow.Spec.Roles {
				if strings.ToLower(userRole) == "pi" || strings.ToLower(userRole) == "manager" {
					contentData.CommonData.Email = append(contentData.CommonData.Email, userRow.Spec.Email)
				}
			}
		}
	}
	mailer.Send(kind, contentData)
}

// runVerificationTimeout puts a procedure in place to remove requests by verification or timeout
func (t *Handler) runVerificationTimeout(EVCopy *apps_v1alpha.EmailVerification) {
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	closeChannels := func() {
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of email verification object
	watchEV, err := t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", EVCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for EVEvent := range watchEV.ResultChan() {
				// Get updated email verification object
				updatedEV, status := EVEvent.Object.(*apps_v1alpha.EmailVerification)
				if status {
					if EVEvent.Type == "DELETED" {
						terminated <- true
						continue
					}

					if updatedEV.Status.Expires != nil {
						// Check whether expiration date updated
						if EVCopy.Status.Expires != nil && timeout != nil {
							if EVCopy.Status.Expires.Time == updatedEV.Status.Expires.Time {
								EVCopy = updatedEV
								continue
							}
						}

						if updatedEV.Status.Expires.Time.Sub(time.Now()) >= 0 {
							timeout = time.After(time.Until(updatedEV.Status.Expires.Time))
							timeoutRenewed <- true
						}
					}
					EVCopy = updatedEV
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
			t.edgenetClientset.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Delete(EVCopy.GetName(), &metav1.DeleteOptions{})
			closeChannels()
			break timeoutLoop
		case <-terminated:
			watchEV.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}
