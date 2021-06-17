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

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	"github.com/EdgeNet-project/edgenet/pkg/permission"
	"github.com/EdgeNet-project/edgenet/pkg/util"

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
	log.Info("EVHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreatedOrUpdated is called when an object is created
func (t *Handler) ObjectCreatedOrUpdated(obj interface{}) {
	log.Info("EVHandler.ObjectCreated")
	// Make a copy of the email verification object to make changes on it
	emailVerification := obj.(*registrationv1alpha.EmailVerification).DeepCopy()
	if emailVerification.Status.State != verified {
		if emailVerification.Spec.Verified {
			emailVerification.Status.State = verified
			t.edgenetClientset.RegistrationV1alpha().EmailVerifications().UpdateStatus(context.TODO(), emailVerification, metav1.UpdateOptions{})

			t.statusUpdate(emailVerification.GetLabels())
		} else {
			if emailVerification.Status.Expiry == nil {
				// Set the email verification timeout which is 24 hours
				emailVerification.Status.Expiry = &metav1.Time{
					Time: time.Now().Add(24 * time.Hour),
				}
				t.edgenetClientset.RegistrationV1alpha().EmailVerifications().UpdateStatus(context.TODO(), emailVerification, metav1.UpdateOptions{})
			}
		}
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
	case *registrationv1alpha.TenantRequest:
		tenantRequest := obj.(*registrationv1alpha.TenantRequest)

		code := "tr-" + util.GenerateRandomString(16)
		emailVerification := registrationv1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Email = tenantRequest.Spec.Contact.Email
		// labels: tenant, user, code - attach to the email verification and tenant
		labels := map[string]string{"edge-net.io/tenant": tenantRequest.GetName(), "edge-net.io/username": tenantRequest.Spec.Contact.Username, "edge-net.io/registration": "tenant"}
		emailVerification.SetLabels(labels)

		_, err := t.edgenetClientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), emailVerification.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			created = true
			t.sendEmail("tenant-email-verification", tenantRequest.GetName(), "", tenantRequest.Spec.Contact.Username,
				fmt.Sprintf("%s %s", tenantRequest.Spec.Contact.FirstName, tenantRequest.Spec.Contact.LastName), tenantRequest.Spec.Contact.Email, code)
		} else {
			t.sendEmail("tenant-email-verification-malfunction", tenantRequest.GetName(), "", tenantRequest.Spec.Contact.Username,
				fmt.Sprintf("%s %s", tenantRequest.Spec.Contact.FirstName, tenantRequest.Spec.Contact.LastName), tenantRequest.Spec.Contact.Email, "")
		}
	case *registrationv1alpha.UserRequest:
		userRequest := obj.(*registrationv1alpha.UserRequest)
		code := "ur-" + util.GenerateRandomString(16)
		emailVerification := registrationv1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Email = userRequest.Spec.Email
		// labels: tenant, user, code - attach to the email verification and tenant
		labels := map[string]string{"edge-net.io/tenant": strings.ToLower(userRequest.Spec.Tenant), "edge-net.io/username": userRequest.GetName(), "edge-net.io/registration": "user"}
		emailVerification.SetLabels(labels)
		_, err := t.edgenetClientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), emailVerification.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			created = true
			t.sendEmail("user-email-verification", strings.ToLower(userRequest.Spec.Tenant), userRequest.GetNamespace(), userRequest.GetName(),
				fmt.Sprintf("%s %s", userRequest.Spec.FirstName, userRequest.Spec.LastName), userRequest.Spec.Email, code)
		} else {
			t.sendEmail("user-email-verification-malfunction", strings.ToLower(userRequest.Spec.Tenant), userRequest.GetNamespace(), userRequest.GetName(),
				fmt.Sprintf("%s %s", userRequest.Spec.FirstName, userRequest.Spec.LastName), userRequest.Spec.Email, "")
		}
	}
	return created
}

// sendEmail to send notification to tenant admins and authorized users about email verification
func (t *Handler) sendEmail(subject, tenant, namespace, username, fullname, email, code string) {
	// Set the HTML template variables
	var contentData interface{}

	collective := mailer.CommonContentData{}
	collective.CommonData.Tenant = tenant
	collective.CommonData.Username = username
	collective.CommonData.Name = fullname
	collective.CommonData.Email = []string{}
	if subject == "tenant-email-verification" || subject == "user-email-verification-update" ||
		subject == "user-email-verification" {
		collective.CommonData.Email = []string{email}
		verifyContent := mailer.VerifyContentData{}
		verifyContent.Code = code
		verifyContent.CommonData = collective.CommonData
		contentData = verifyContent
	} else if subject == "user-email-verified-alert" {
		// Put the email addresses of the tenant admins and authorized users in the email to be sent list
		tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenant, metav1.GetOptions{})

		if acceptableUsePolicyRaw, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s,edge-net.io/identity=true", tenant.GetName())}); err == nil {
			for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
				aupLabels := acceptableUsePolicyRow.GetLabels()
				if aupLabels != nil && aupLabels["edge-net.io/username"] != "" && aupLabels["edge-net.io/firstname"] != "" && aupLabels["edge-net.io/lastname"] != "" {
					authorized := permission.CheckAuthorization("", acceptableUsePolicyRow.Spec.Email, "UserRequest", username, "cluster")
					if authorized {
						collective.CommonData.Email = append(collective.CommonData.Email, acceptableUsePolicyRow.Spec.Email)
					}
				}
			}
		}
		contentData = collective
	} else {
		collective.CommonData.Email = []string{email}
		contentData = collective
	}

	mailer.Send(subject, contentData)
}

// statusUpdate to update the objects that are relevant the request and send email
func (t *Handler) statusUpdate(labels map[string]string) {
	// Update the status of request related to email verification
	if strings.ToLower(labels["edge-net.io/registration"]) == "tenant" {
		tenantRequest, _ := t.edgenetClientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), labels["edge-net.io/tenant"], metav1.GetOptions{})
		// TO-DO: Check dubious activity here
		// labels := tenantRequest.GetLabels()
		tenantRequest.Status.EmailVerified = true
		t.edgenetClientset.RegistrationV1alpha().TenantRequests().UpdateStatus(context.TODO(), tenantRequest, metav1.UpdateOptions{})
		// Send email to inform admins of the cluster
		t.sendEmail("tenant-email-verified-alert", labels["edge-net.io/tenant"], "", tenantRequest.Spec.Contact.Username,
			fmt.Sprintf("%s %s", tenantRequest.Spec.Contact.FirstName, tenantRequest.Spec.Contact.LastName), "", "")
	} else if strings.ToLower(labels["edge-net.io/registration"]) == "user" {
		userRequestObj, _ := t.edgenetClientset.RegistrationV1alpha().UserRequests().Get(context.TODO(), labels["edge-net.io/username"], metav1.GetOptions{})
		userRequestObj.Status.EmailVerified = true
		t.edgenetClientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequestObj, metav1.UpdateOptions{})
		// Send email to inform edgenet tenant admins and authorized users
		t.sendEmail("user-email-verified-alert", labels["edge-net.io/tenant"], "", labels["edge-net.io/username"],
			fmt.Sprintf("%s %s", userRequestObj.Spec.FirstName, userRequestObj.Spec.LastName), "", "")
	} else if strings.ToLower(labels["edge-net.io/registration"]) == "email" {
		acceptableUsePolicy, _ := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), labels["edge-net.io/username"], metav1.GetOptions{})
		acceptableUsePolicy.Spec.Accepted = true
		t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy, metav1.UpdateOptions{})

		// TO-DO: Get user contact information
		// Send email to inform user
		// t.sendEmail("user-email-verified-notification", labels["edge-net.io/tenant"], "", labels["edge-net.io/username"],
		// fmt.Sprintf("%s %s", userObj.Spec.FirstName, userObj.Spec.LastName), userObj.Spec.Email, "")
	}
}

// RunExpiryController puts a procedure in place to remove requests by verification or timeout
func (t *Handler) RunExpiryController() {
	var closestExpiry time.Time
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchEmailVerifiation, err := t.edgenetClientset.RegistrationV1alpha().EmailVerifications().Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchEmailVerifiation watch.Interface, newExpiry *chan time.Time) {
			// Watch the events of user request object
			// Get events from watch interface
			for emailVerificationEvent := range watchEmailVerifiation.ResultChan() {
				// Get updated user request object
				updatedEmailVerification, status := emailVerificationEvent.Object.(*registrationv1alpha.EmailVerification)
				if status {
					if updatedEmailVerification.Status.Expiry != nil {
						*newExpiry <- updatedEmailVerification.Status.Expiry.Time
					}
				}
			}
		}
		go watchEvents(watchEmailVerifiation, &newExpiry)
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
			emailVerificationRaw, err := t.edgenetClientset.RegistrationV1alpha().EmailVerifications().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TO-DO: Provide more information on error
				log.Println(err)
			}
			for _, emailVerificationRow := range emailVerificationRaw.Items {
				if emailVerificationRow.Status.Expiry != nil && emailVerificationRow.Status.Expiry.Time.Sub(time.Now()) <= 0 {
					t.edgenetClientset.RegistrationV1alpha().EmailVerifications().Delete(context.TODO(), emailVerificationRow.GetName(), metav1.DeleteOptions{})
				} else if emailVerificationRow.Status.Expiry != nil && emailVerificationRow.Status.Expiry.Time.Sub(time.Now()) > 0 {
					if closestExpiry.Sub(time.Now()) <= 0 || closestExpiry.Sub(emailVerificationRow.Status.Expiry.Time) > 0 {
						closestExpiry = emailVerificationRow.Status.Expiry.Time
						log.Printf("ExpiryController: Closest expiry date is %v after the expiration of a user request", closestExpiry)
					}
				}
			}

			if closestExpiry.Sub(time.Now()) <= 0 {
				closestExpiry = time.Now().AddDate(1, 0, 0)
				log.Printf("ExpiryController: Closest expiry date is %v after the expiration of a user request", closestExpiry)
			}
		case <-terminated:
			watchEmailVerifiation.Stop()
			break infiniteLoop
		}
	}
}
