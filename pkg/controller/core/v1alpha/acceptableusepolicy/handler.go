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
	"strconv"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface)
	ObjectCreatedOrUpdated(obj interface{})
	ObjectDeleted(obj interface{})
	RunExpiryController()
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

// ObjectCreatedOrUpdated is called when an object is created
func (t *Handler) ObjectCreatedOrUpdated(obj interface{}) {
	log.Info("AUPHandler.ObjectCreatedOrUpdated")
	// Make a copy of the acceptable use policy object to make changes on it
	acceptableUsePolicy := obj.(*corev1alpha.AcceptableUsePolicy).DeepCopy()
	aupLabels := acceptableUsePolicy.GetLabels()
	tenantName := aupLabels["edge-net.io/tenant"]
	tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
	// Check if the tenant is active
	if tenant.Spec.Enabled && acceptableUsePolicy.Spec.Accepted {
		if acceptableUsePolicy.Status.Expiry == nil {
			// Set a 6-month timeout cycle
			acceptableUsePolicy.Status.Expiry = &metav1.Time{
				Time: time.Now().Add(4382 * time.Hour),
			}

			acceptableUsePolicy.Status.State = success
			acceptableUsePolicy.Status.Message = []string{statusDict["aup-ok"]}
			if _, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), acceptableUsePolicy, metav1.UpdateOptions{}); err != nil {
				// TODO: Provide more detailed information about the issue
				log.Println(statusDict["aup-set-fail"])
			}
			// Get the user who owns this acceptable use policy object
			for _, user := range tenant.Spec.User {
				if user.Username == aupLabels["edge-net.io/user"] {
					contentData := mailer.CommonContentData{}
					contentData.CommonData.Tenant = tenantName
					contentData.CommonData.Username = acceptableUsePolicy.GetName()
					contentData.CommonData.Name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
					contentData.CommonData.Email = []string{user.Email}
					mailer.Send("acceptable-use-policy-accepted", contentData)
				}
			}
		}
	} else if tenant.Spec.Enabled && !acceptableUsePolicy.Spec.Accepted {
		if acceptableUsePolicy.Status.Expiry == nil {
			acceptableUsePolicy.Status.Expiry = nil
			acceptableUsePolicy.Status.State = success
			acceptableUsePolicy.Status.Message = []string{statusDict["aup-ok"]}
		} else if acceptableUsePolicy.Status.Expiry != nil && acceptableUsePolicy.Status.Expiry.Time.Sub(time.Now()) <= 0 {
			acceptableUsePolicy.Status.State = failure
			acceptableUsePolicy.Status.Message = []string{statusDict["aup-expired"]}
		}
		t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), acceptableUsePolicy, metav1.UpdateOptions{})
	} else {
		acceptableUsePolicy.Status.State = failure
		acceptableUsePolicy.Status.Message = []string{statusDict["tenant-disabled"]}
		t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().UpdateStatus(context.TODO(), acceptableUsePolicy, metav1.UpdateOptions{})
	}

	tenantLabels := tenant.GetLabels()
	if tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted/%s", acceptableUsePolicy.GetName())] != strconv.FormatBool(acceptableUsePolicy.Spec.Accepted) {
		tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted/%s", acceptableUsePolicy.GetName())] = strconv.FormatBool(acceptableUsePolicy.Spec.Accepted)
		if _, err := t.edgenetClientset.CoreV1alpha().Tenants().Update(context.TODO(), tenant, metav1.UpdateOptions{}); err != nil {
			// TODO: Define the error precisely
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("AUPHandler.ObjectDeleted")
	// TODO: Update the tenant labels accordingly
}

// runApprovalTimeout puts a procedure in place to remove requests by approval or timeout
/*func (t *Handler) runApprovalTimeout(acceptableUsePolicy *corev1alpha.AcceptableUsePolicy) {
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if acceptableUsePolicy.Status.Expiry != nil {
		timeout = time.After(time.Until(acceptableUsePolicy.Status.Expiry.Time))
	}
	closeChannels := func() {
		close(terminated)
	}

	// Watch the events of acceptable use policy object
	watchAcceptableUsePolicy, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", acceptableUsePolicy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for acceptableUsePolicyEvent := range watchAcceptableUsePolicy.ResultChan() {
				// Get updated acceptable use policy object
				updatedAcceptableUsePolicy, status := acceptableUsePolicyEvent.Object.(*corev1alpha.AcceptableUsePolicy)
				if acceptableUsePolicy.GetUID() == updatedAcceptableUsePolicy.GetUID() {
					if status {
						if acceptableUsePolicyEvent.Type == "DELETED" || !updatedAcceptableUsePolicy.Spec.Accepted {
							terminated <- true
							continue
						}
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching acceptableusepolicy resources,
		// there is a timeout at 72 hours
		timeout = time.After(72 * time.Hour)
	}

	// Wait on multiple channel operations
	select {
	case <-timeout:
		watchAcceptableUsePolicy.Stop()

		aupLabels := acceptableUsePolicy.GetLabels()
		tenantName := aupLabels["edge-net.io/tenant"]
		tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})

		for _, user := range tenant.Spec.User {
			if user.Username == aupLabels["edge-net.io/user"] {
				contentData := mailer.CommonContentData{}
				contentData.CommonData.Tenant = tenantName
				contentData.CommonData.Username = acceptableUsePolicy.GetName()
				contentData.CommonData.Name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
				contentData.CommonData.Email = []string{user.Email}
				mailer.Send("acceptable-use-policy-expired", contentData)
			}
		}

		acceptableUsePolicy.Spec.Accepted = false
		t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy, metav1.UpdateOptions{})
		closeChannels()
	case <-terminated:
		watchAcceptableUsePolicy.Stop()
		closeChannels()
	}
}*/

// RunExpiryController puts a procedure in place to turn accepted policies into not accepted
func (t *Handler) RunExpiryController() {
	var closestExpiry time.Time
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchAcceptableUsePolicy, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchAcceptableUsePolicy watch.Interface, newExpiry *chan time.Time) {
			// Watch the events of acceptable use policy object
			// Get events from watch interface
			for acceptableUsePolicyEvent := range watchAcceptableUsePolicy.ResultChan() {
				// Get updated acceptable use policy object
				updatedAcceptableUsePolicy, status := acceptableUsePolicyEvent.Object.(*corev1alpha.AcceptableUsePolicy)
				if status {
					if updatedAcceptableUsePolicy.Status.Expiry != nil {
						log.Println(updatedAcceptableUsePolicy.Status.Expiry)
						*newExpiry <- updatedAcceptableUsePolicy.Status.Expiry.Time
					}
				}
			}
		}
		go watchEvents(watchAcceptableUsePolicy, &newExpiry)
	} else {
		go t.RunExpiryController()
		terminated <- true
	}

infiniteLoop:
	for {
		// Wait on multiple channel operations
		select {
		case timeout := <-newExpiry:
			if closestExpiry.Sub(timeout) > 0 {
				closestExpiry = timeout
				log.Printf("ExpiryController: Closest expiry date is %v", closestExpiry)
			}
		case <-time.After(time.Until(closestExpiry)):
			acceptableUsePolicyRaw, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TO-DO: Provide more information on error
				log.Println(err)
			}
			for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
				if acceptableUsePolicyRow.Status.Expiry != nil && acceptableUsePolicyRow.Status.Expiry.Time.Sub(time.Now()) <= 0 {
					acceptableUsePolicy := acceptableUsePolicyRow.DeepCopy()
					aupLabels := acceptableUsePolicy.GetLabels()
					tenantName := aupLabels["edge-net.io/tenant"]
					tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
					for _, user := range tenant.Spec.User {
						if user.Username == aupLabels["edge-net.io/user"] {
							contentData := mailer.CommonContentData{}
							contentData.CommonData.Tenant = tenantName
							contentData.CommonData.Username = acceptableUsePolicy.GetName()
							contentData.CommonData.Name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
							contentData.CommonData.Email = []string{user.Email}
							mailer.Send("acceptable-use-policy-expired", contentData)
						}
					}

					acceptableUsePolicy.Spec.Accepted = false
					go t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy, metav1.UpdateOptions{})
				}
			}
		case <-terminated:
			watchAcceptableUsePolicy.Stop()
			break infiniteLoop
		}
	}
}
