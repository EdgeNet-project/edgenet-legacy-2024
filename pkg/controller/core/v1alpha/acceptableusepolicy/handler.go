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

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
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
	AUPCopy := obj.(*corev1alpha.AcceptableUsePolicy).DeepCopy()
	// Find the authority from the namespace in which the object is
	tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), AUPCopy.Status.Tenant, metav1.GetOptions{})
	// Check if the authority is active
	if tenant.Spec.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		if AUPCopy.Spec.Accepted && AUPCopy.Status.Expiry == nil {
			// Run timeout goroutine
			go t.runApprovalTimeout(AUPCopy)
			// Set a timeout cycle which makes the acceptable use policy expires every 6 months
			AUPCopy.Status.Expiry = &metav1.Time{
				Time: time.Now().Add(4382 * time.Hour),
			}
			AUPCopy.Status.State = success
			AUPCopy.Status.Message = []string{statusDict["aup-ok"]}
			_, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
			if err != nil {
				AUPCopy.Status.State = failure
				AUPCopy.Status.Message = []string{statusDict["aup-ok"], statusDict["aup-set-fail"]}
				t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
			}
		} else if AUPCopy.Spec.Accepted && AUPCopy.Status.Expiry != nil {
			// Check if the 6 months cycle expired
			if AUPCopy.Status.Expiry.Time.Sub(time.Now()) >= 0 {
				go t.runApprovalTimeout(AUPCopy)
			} else {
				AUPCopy.Spec.Accepted = false
				AUPUpdated, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), AUPCopy, metav1.UpdateOptions{})
				if err == nil {
					AUPCopy = AUPUpdated
				}
				AUPCopy.Status.State = failure
				AUPCopy.Status.Message = []string{statusDict["aup-expired"]}
				t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
			}
		} else if !AUPCopy.Spec.Accepted && AUPCopy.Status.Expiry == nil {
			AUPCopy.Status.State = success
			AUPCopy.Status.Message = []string{statusDict["aup-ok"]}
			t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
		}
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("AUPHandler.ObjectUpdated")
	// Create a copy of the acceptable use policy object to make changes on it
	AUPCopy := obj.(*corev1alpha.AcceptableUsePolicy).DeepCopy()
	AUPOwnerAuthority, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), AUPCopy.Labels["tenant-name"], metav1.GetOptions{})
	fieldUpdated := updated.(fields)

	if AUPOwnerAuthority.Spec.Enabled {
		// To manipulate user object according to the changes of acceptable use policy
		if fieldUpdated.accepted {
			defer func() {
				AUPUpdated, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
				if err == nil {
					AUPCopy = AUPUpdated
				}
			}()
			// Get the user who owns this acceptable use policy object
			tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), AUPCopy.Status.Tenant, metav1.GetOptions{})
			if AUPCopy.Spec.Accepted {
				go t.runApprovalTimeout(AUPCopy)
				// Set the expiration date according to the 6-month cycle
				AUPCopy.Status.Expiry = &metav1.Time{
					Time: time.Now().Add(4382 * time.Hour),
				}

				for _, user := range tenant.Spec.User {
					if user.Username == AUPCopy.GetName() {
						contentData := mailer.CommonContentData{}
						contentData.CommonData.Authority = AUPCopy.Status.Tenant
						contentData.CommonData.Username = AUPCopy.GetName()
						contentData.CommonData.Name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
						contentData.CommonData.Email = []string{user.Email}
						mailer.Send("acceptable-use-policy-accepted", contentData)
					}
				}

			}
		}
	} else {
		AUPCopy.Spec.Accepted = false
		AUPUpdated, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), AUPCopy, metav1.UpdateOptions{})
		if err == nil {
			AUPCopy = AUPUpdated
		}
		AUPCopy.Status.State = failure
		AUPCopy.Status.Message = []string{statusDict["authority-disabled"]}
		t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), AUPCopy, metav1.UpdateOptions{})
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("AUPHandler.ObjectDeleted")
	// Mail notification, TBD
}

// runApprovalTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) runApprovalTimeout(AUPCopy *corev1alpha.AcceptableUsePolicy) {
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if AUPCopy.Status.Expiry != nil {
		timeout = time.After(time.Until(AUPCopy.Status.Expiry.Time))
	}
	closeChannels := func() {
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of acceptable use policy object
	watchAUP, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", AUPCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for AUPEvent := range watchAUP.ResultChan() {
				// Get updated acceptable use policy object
				updatedAUP, status := AUPEvent.Object.(*corev1alpha.AcceptableUsePolicy)
				if AUPCopy.GetUID() == updatedAUP.GetUID() {
					if status {
						if AUPEvent.Type == "DELETED" || !updatedAUP.Spec.Accepted {
							terminated <- true
							continue
						}

						if updatedAUP.Status.Expiry != nil {
							// Check whether expiration date updated - TBD
							/*if AUPCopy.Status.Expiry != nil && timeout != nil {
								if AUPCopy.Status.Expiry.Time == updatedAUP.Status.Expiry.Time {
									AUPCopy = updatedAUP
									continue
								}
							}*/
							if updatedAUP.Status.Expiry.Time.Sub(time.Now()) >= 0 {
								timeout = time.After(time.Until(updatedAUP.Status.Expiry.Time))
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

			tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), AUPCopy.Status.Tenant, metav1.GetOptions{})

			for _, user := range tenant.Spec.User {
				if user.Username == AUPCopy.GetName() {
					contentData := mailer.CommonContentData{}
					contentData.CommonData.Authority = AUPCopy.Status.Tenant
					contentData.CommonData.Username = AUPCopy.GetName()
					contentData.CommonData.Name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
					contentData.CommonData.Email = []string{user.Email}
					mailer.Send("acceptable-use-policy-expired", contentData)
				}
			}

			AUPCopy.Spec.Accepted = false
			t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), AUPCopy, metav1.UpdateOptions{})
			closeChannels()
			break timeoutLoop
		case <-terminated:
			watchAUP.Stop()
			closeChannels()
			break timeoutLoop
		}
	}
}
