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

package acceptableusepolicy

import (
	"context"
	"fmt"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
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
	log.Info("AUPHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("AUPHandler.ObjectCreated")
	// Create a copy of the acceptable use policy object to make changes on it
	AUPCopy := obj.(*apps_v1alpha.AcceptableUsePolicy).DeepCopy()
	// Find the authority from the namespace in which the object is
	AUPOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), AUPCopy.GetNamespace(), metav1.GetOptions{})
	AUPOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), AUPOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	// Check if the authority is active
	if AUPOwnerAuthority.Spec.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if AUPCopy.Spec.Accepted && AUPCopy.Status.Expires == nil {
			// Run timeout goroutine
			go t.runApprovalTimeout(AUPCopy)
			// Set a timeout cycle which makes the acceptable use policy expires every 6 months
			AUPCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(4382 * time.Hour),
			}
			AUPCopy.Status.State = success
			AUPCopy.Status.Message = []string{statusDict["aup-ok"]}
			_, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
			if err != nil {
				AUPCopy.Status.State = failure
				AUPCopy.Status.Message = []string{statusDict["aup-ok"], statusDict["aup-set-fail"]}
				t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
			} else {
				user, _ := t.edgenetClientset.AppsV1alpha().Users(AUPCopy.GetNamespace()).Get(context.TODO(), AUPCopy.GetName(), metav1.GetOptions{})
				if !user.Status.AUP {
					user.Status.AUP = true
					t.edgenetClientset.AppsV1alpha().Users(user.GetNamespace()).Update(context.TODO(), user, metav1.UpdateOptions{})
				}
			}
		} else if AUPCopy.Spec.Accepted && AUPCopy.Status.Expires != nil {
			// Check if the 6 months cycle expired
			if AUPCopy.Status.Expires.Time.Sub(time.Now()) >= 0 {
				go t.runApprovalTimeout(AUPCopy)
				user, _ := t.edgenetClientset.AppsV1alpha().Users(AUPCopy.GetNamespace()).Get(context.TODO(), AUPCopy.GetName(), metav1.GetOptions{})
				if !user.Status.AUP {
					user.Status.AUP = true
					t.edgenetClientset.AppsV1alpha().Users(user.GetNamespace()).Update(context.TODO(), user, metav1.UpdateOptions{})
				}
			} else {
				AUPCopy.Spec.Accepted = false
				AUPUpdated, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Update(context.TODO(), AUPCopy, metav1.UpdateOptions{})
				if err == nil {
					AUPCopy = AUPUpdated
				}
				AUPCopy.Status.State = failure
				AUPCopy.Status.Message = []string{statusDict["aup-expired"]}
				t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
				user, _ := t.edgenetClientset.AppsV1alpha().Users(AUPCopy.GetNamespace()).Get(context.TODO(), AUPCopy.GetName(), metav1.GetOptions{})
				if user.Status.AUP {
					user.Status.AUP = false
					t.edgenetClientset.AppsV1alpha().Users(user.GetNamespace()).Update(context.TODO(), user, metav1.UpdateOptions{})
				}
			}
		} else if !AUPCopy.Spec.Accepted && AUPCopy.Status.Expires == nil {
			AUPCopy.Status.State = success
			AUPCopy.Status.Message = []string{statusDict["aup-ok"]}
			t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
			user, _ := t.edgenetClientset.AppsV1alpha().Users(AUPCopy.GetNamespace()).Get(context.TODO(), AUPCopy.GetName(), metav1.GetOptions{})
			if user.Status.AUP {
				user.Status.AUP = false
				t.edgenetClientset.AppsV1alpha().Users(user.GetNamespace()).Update(context.TODO(), user, metav1.UpdateOptions{})
			}
		}
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("AUPHandler.ObjectUpdated")
	// Create a copy of the acceptable use policy object to make changes on it
	AUPCopy := obj.(*apps_v1alpha.AcceptableUsePolicy).DeepCopy()
	AUPOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), AUPCopy.GetNamespace(), metav1.GetOptions{})
	AUPOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), AUPOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	fieldUpdated := updated.(fields)

	if AUPOwnerAuthority.Spec.Enabled {
		// To manipulate user object according to the changes of acceptable use policy
		if fieldUpdated.accepted {
			defer func() {
				AUPUpdated, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
				if err == nil {
					AUPCopy = AUPUpdated
				}
			}()
			// Get the user who owns this acceptable use policy object
			AUPUser, _ := t.edgenetClientset.AppsV1alpha().Users(AUPCopy.GetNamespace()).Get(context.TODO(), AUPCopy.GetName(), metav1.GetOptions{})
			if AUPCopy.Spec.Accepted {
				AUPUser.Status.AUP = true
				go t.runApprovalTimeout(AUPCopy)
				// Set the expiration date according to the 6-month cycle
				AUPCopy.Status.Expires = &metav1.Time{
					Time: time.Now().Add(4382 * time.Hour),
				}

				contentData := mailer.CommonContentData{}
				contentData.CommonData.Authority = AUPOwnerNamespace.Labels["authority-name"]
				contentData.CommonData.Username = AUPCopy.GetName()
				contentData.CommonData.Name = fmt.Sprintf("%s %s", AUPUser.Spec.FirstName, AUPUser.Spec.LastName)
				contentData.CommonData.Email = []string{AUPUser.Spec.Email}
				mailer.Send("acceptable-use-policy-accepted", contentData)
			} else {
				AUPUser.Status.AUP = false
			}
			go t.edgenetClientset.AppsV1alpha().Users(AUPUser.GetNamespace()).UpdateStatus(context.TODO(), AUPUser, metav1.UpdateOptions{})
		}
	} else {
		AUPCopy.Spec.Accepted = false
		AUPUpdated, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Update(context.TODO(), AUPCopy, metav1.UpdateOptions{})
		if err == nil {
			AUPCopy = AUPUpdated
		}
		AUPCopy.Status.State = failure
		AUPCopy.Status.Message = []string{statusDict["authority-disabled"]}
		t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("AUPHandler.ObjectDeleted")
	// Mail notification, TBD
}

// runApprovalTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) runApprovalTimeout(AUPCopy *apps_v1alpha.AcceptableUsePolicy) {
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if AUPCopy.Status.Expires != nil {
		timeout = time.After(time.Until(AUPCopy.Status.Expires.Time))
	}
	closeChannels := func() {
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of acceptable use policy object
	watchAUP, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", AUPCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for AUPEvent := range watchAUP.ResultChan() {
				// Get updated acceptable use policy object
				updatedAUP, status := AUPEvent.Object.(*apps_v1alpha.AcceptableUsePolicy)
				if AUPCopy.GetUID() == updatedAUP.GetUID() {
					if status {
						if AUPEvent.Type == "DELETED" || !updatedAUP.Spec.Accepted {
							terminated <- true
							continue
						}

						if updatedAUP.Status.Expires != nil {
							// Check whether expiration date updated - TBD
							/*if AUPCopy.Status.Expires != nil && timeout != nil {
								if AUPCopy.Status.Expires.Time == updatedAUP.Status.Expires.Time {
									AUPCopy = updatedAUP
									continue
								}
							}*/
							if updatedAUP.Status.Expires.Time.Sub(time.Now()) >= 0 {
								timeout = time.After(time.Until(updatedAUP.Status.Expires.Time))
								timeoutRenewed <- true
							} else {
								terminated <- true
							}
						}
						AUPCopy = updatedAUP
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching acceptableusepolicy resources,
		// there is a timeout at 72 hours
		timeout = time.After(72 * time.Hour)
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
			watchAUP.Stop()
			AUPOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), AUPCopy.GetNamespace(), metav1.GetOptions{})
			AUPUser, _ := t.edgenetClientset.AppsV1alpha().Users(AUPCopy.GetNamespace()).Get(context.TODO(), AUPCopy.GetName(), metav1.GetOptions{})
			contentData := mailer.CommonContentData{}
			contentData.CommonData.Authority = AUPOwnerNamespace.Labels["authority-name"]
			contentData.CommonData.Username = AUPCopy.GetName()
			contentData.CommonData.Name = fmt.Sprintf("%s %s", AUPUser.Spec.FirstName, AUPUser.Spec.LastName)
			contentData.CommonData.Email = []string{AUPUser.Spec.Email}
			mailer.Send("acceptable-use-policy-expired", contentData)
			AUPUser.Status.AUP = false
			t.edgenetClientset.AppsV1alpha().Users(AUPUser.GetNamespace()).Update(context.TODO(), AUPUser, metav1.UpdateOptions{})
			AUPCopy.Spec.Accepted = false
			t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Update(context.TODO(), AUPCopy, metav1.UpdateOptions{})
			closeChannels()
			break timeoutLoop
		case <-terminated:
			watchAUP.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}
