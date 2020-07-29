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
	"fmt"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/bootstrap"
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
	log.Info("AUPHandler.Init")
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
	log.Info("AUPHandler.ObjectCreated")
	// Create a copy of the acceptable use policy object to make changes on it
	AUPCopy := obj.(*apps_v1alpha.AcceptableUsePolicy).DeepCopy()
	// Find the authority from the namespace in which the object is
	AUPOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(AUPCopy.GetNamespace(), metav1.GetOptions{})
	AUPOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(AUPOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	// Check if the authority is active
	if AUPOwnerAuthority.Spec.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if AUPCopy.Spec.Accepted && AUPCopy.Status.Expires == nil {
			// Run timeout goroutine
			go t.runApprovalTimeout(AUPCopy)
			if AUPCopy.Spec.Renew {
				AUPCopy.Spec.Renew = false
				AUPUpdated, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Update(AUPCopy)
				if err == nil {
					AUPCopy = AUPUpdated
				}
			}
			// Set a timeout cycle which makes the acceptable use policy expires every 6 months
			AUPCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(4382 * time.Hour),
			}
			AUPCopy.Status.State = success
			AUPCopy.Status.Message = []string{statusDict["aup-ok"]}
			AUPUpdated, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(AUPCopy)
			if err == nil {
				AUPCopy = AUPUpdated
			} else {
				AUPCopy.Status.State = failure
				AUPCopy.Status.Message = []string{statusDict["aup-ok"], statusDict["aup-set-fail"]}
				t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(AUPCopy)
			}
		} else if AUPCopy.Spec.Accepted && AUPCopy.Status.Expires != nil {
			// Check if the 6 months cycle expired
			if AUPCopy.Status.Expires.Time.Sub(time.Now()) >= 0 {
				go t.runApprovalTimeout(AUPCopy)
				if AUPCopy.Spec.Renew {
					AUPCopy.Spec.Renew = false
					t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Update(AUPCopy)
				}
			} else {
				AUPCopy.Spec.Accepted = false
				AUPCopy.Spec.Renew = false
				AUPUpdated, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Update(AUPCopy)
				if err == nil {
					AUPCopy = AUPUpdated
				}
				AUPCopy.Status.State = failure
				AUPCopy.Status.Message = []string{statusDict["aup-expired"]}
				t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(AUPCopy)
			}
		}
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("AUPHandler.ObjectUpdated")
	// Create a copy of the acceptable use policy object to make changes on it
	AUPCopy := obj.(*apps_v1alpha.AcceptableUsePolicy).DeepCopy()
	AUPOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(AUPCopy.GetNamespace(), metav1.GetOptions{})
	AUPOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(AUPOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	fieldUpdated := updated.(fields)

	if AUPOwnerAuthority.Spec.Enabled {
		defer func() {
			AUPUpdated, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(AUPCopy)
			if err == nil {
				AUPCopy = AUPUpdated
			}
			if AUPCopy.Spec.Renew {
				AUPCopy.Spec.Renew = false
				t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Update(AUPCopy)
			}
		}()
		// To manipulate user object according to the changes of acceptable use policy
		if fieldUpdated.accepted {
			// Get the user who owns this acceptable use policy object
			AUPUser, _ := t.edgenetClientset.AppsV1alpha().Users(AUPCopy.GetNamespace()).Get(AUPCopy.GetName(), metav1.GetOptions{})
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
			go t.edgenetClientset.AppsV1alpha().Users(AUPUser.GetNamespace()).UpdateStatus(AUPUser)
		} else if AUPCopy.Spec.Accepted && AUPCopy.Spec.Renew {
			AUPCopy.Status.State = success
			AUPCopy.Status.Message = []string{statusDict["aup-agreed"]}
			AUPCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(4382 * time.Hour),
			}
		}
	} else {
		AUPCopy.Spec.Accepted = false
		AUPCopy.Spec.Renew = false
		AUPUpdated, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Update(AUPCopy)
		if err == nil {
			AUPCopy = AUPUpdated
		}
		AUPCopy.Status.State = failure
		AUPCopy.Status.Message = []string{statusDict["authority-disabled"]}
		t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).UpdateStatus(AUPCopy)
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
	var reminder <-chan time.Time
	if AUPCopy.Status.Expires != nil {
		timeout = time.After(time.Until(AUPCopy.Status.Expires.Time))
		reminder = time.After(time.Until(AUPCopy.Status.Expires.Time.Add(time.Hour * -168)))
	}
	closeChannels := func() {
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of acceptable use policy object
	watchAUP, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", AUPCopy.GetName())})
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
								reminder = time.After(time.Until(updatedAUP.Status.Expires.Time.Add(time.Hour * -168)))
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
		case <-reminder:
			AUPOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(AUPCopy.GetNamespace(), metav1.GetOptions{})
			AUPUser, _ := t.edgenetClientset.AppsV1alpha().Users(AUPCopy.GetNamespace()).Get(AUPCopy.GetName(), metav1.GetOptions{})
			contentData := mailer.CommonContentData{}
			contentData.CommonData.Authority = AUPOwnerNamespace.Labels["authority-name"]
			contentData.CommonData.Username = AUPCopy.GetName()
			contentData.CommonData.Name = fmt.Sprintf("%s %s", AUPUser.Spec.FirstName, AUPUser.Spec.LastName)
			contentData.CommonData.Email = []string{AUPUser.Spec.Email}
			mailer.Send("acceptable-use-policy-renewal", contentData)
			break timeoutOptions
		case <-timeout:
			watchAUP.Stop()
			AUPOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(AUPCopy.GetNamespace(), metav1.GetOptions{})
			AUPUser, _ := t.edgenetClientset.AppsV1alpha().Users(AUPCopy.GetNamespace()).Get(AUPCopy.GetName(), metav1.GetOptions{})
			contentData := mailer.CommonContentData{}
			contentData.CommonData.Authority = AUPOwnerNamespace.Labels["authority-name"]
			contentData.CommonData.Username = AUPCopy.GetName()
			contentData.CommonData.Name = fmt.Sprintf("%s %s", AUPUser.Spec.FirstName, AUPUser.Spec.LastName)
			contentData.CommonData.Email = []string{AUPUser.Spec.Email}
			mailer.Send("acceptable-use-policy-expired", contentData)
			AUPCopy.Spec.Accepted = false
			t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(AUPCopy.GetNamespace()).Update(AUPCopy)
			closeChannels()
			break timeoutLoop
		case <-terminated:
			watchAUP.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}
