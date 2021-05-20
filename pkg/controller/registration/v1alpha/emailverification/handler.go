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

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
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
	ObjectCreatedOrUpdated(obj interface{})
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

// ObjectCreatedOrUpdated is called when an object is created
func (t *Handler) ObjectCreatedOrUpdated(obj interface{}) {
	log.Info("EVHandler.ObjectCreated")
	// Make a copy of the email verification object to make changes on it
	emailVerification := obj.(*registrationv1alpha.EmailVerification).DeepCopy()

	/*emailVerificationLabels := emailVerification.GetLabels()
	tenant := emailVerificationLabels["edge-net.io/tenant"]

	kind := emailVerificationLabels["edge-net.io/kind"]
	if kind == "tenant" {

	} else if kind == "user" {

	}*/

	if emailVerification.Spec.Verified {
		t.statusUpdate(emailVerification)
	} else {
		if emailVerification.Status.Expiry == nil {
			// Set the email verification timeout which is 24 hours
			emailVerification.Status.Expiry = &metav1.Time{
				Time: time.Now().Add(24 * time.Hour),
			}
			t.edgenetClientset.RegistrationV1alpha().EmailVerifications().UpdateStatus(context.TODO(), emailVerification, metav1.UpdateOptions{})
		}
		// Run timeout goroutine
		go t.runVerificationTimeout(emailVerification)
	}
}

// t.edgenetClientset.RegistrationV1alpha().EmailVerifications(emailVerification.GetNamespace()).Delete(context.TODO(), emailVerification.GetName(), metav1.DeleteOptions{})
// t.sendEmail("tenant-email-verification-dubious", emailVerification.Spec.Identifier, emailVerification.GetNamespace(), "", "", "", "")
// t.sendEmail("user-email-verification-dubious", EVOwnerNamespace.Labels["tenant-name"], emailVerification.GetNamespace(), emailVerification.Spec.Identifier, "", "", "")

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("EVHandler.ObjectDeleted")
	// Mail notification, TBD
}

// Create to provide one-time code for verification
func (t *Handler) Create(obj interface{}, ownerReferences []metav1.OwnerReference) (string, bool) {
	// The section below is a part of the method which provides email verification
	// Email verification code is a security point for email verification. The user
	// registration object creates an email verification object with a name which is
	// this email verification code. Only who knows the email verification
	// code can manipulate that object by using a public token.
	created := false
	code := ""
	switch obj.(type) {
	case *registrationv1alpha.TenantRequest:
		tenantRequest := obj.(*registrationv1alpha.TenantRequest)

		code = "tr-" + util.GenerateRandomString(16)
		emailVerification := registrationv1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Email = tenantRequest.Spec.Contact.Email
		// labels: tenant, user, code - attach to the email verification and tenant
		labels := map[string]string{"edge-net.io/tenant": tenantRequest.GetName(), "edge-net.io/user": tenantRequest.Spec.Contact.Username, "edge-net.io/registration": "tenant"}
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
	case corev1alpha.User:
		user := obj.(corev1alpha.User)
		code = "u-" + util.GenerateRandomString(16)
		emailVerification := registrationv1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Email = user.Email
		// labels: tenant, user, code - attach to the email verification and tenant
		labels := map[string]string{"edge-net.io/tenant": user.Tenant, "edge-net.io/user": user.GetName(), "edge-net.io/registration": "email"}
		emailVerification.SetLabels(labels)

		_, err := t.edgenetClientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), emailVerification.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			created = true
			t.sendEmail("user-email-verification-update", user.GetTenant(), "", user.GetName(),
				fmt.Sprintf("%s %s", user.FirstName, user.LastName), user.Email, code)
		} else {
			t.sendEmail("user-email-verification-update-malfunction", user.GetTenant(), "", user.GetName(),
				fmt.Sprintf("%s %s", user.FirstName, user.LastName), user.Email, "")
		}
	case *registrationv1alpha.UserRequest:
		userRequest := obj.(*registrationv1alpha.UserRequest)
		code = "ur-" + util.GenerateRandomString(16)
		emailVerification := registrationv1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: ownerReferences}}
		emailVerification.SetName(code)
		emailVerification.Spec.Email = userRequest.Spec.Email
		// labels: tenant, user, code - attach to the email verification and tenant
		labels := map[string]string{"edge-net.io/tenant": userRequest.Spec.Tenant, "edge-net.io/user": userRequest.GetName(), "edge-net.io/registration": "user"}
		emailVerification.SetLabels(labels)
		_, err := t.edgenetClientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), emailVerification.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			created = true
			t.sendEmail("user-email-verification", userRequest.Spec.Tenant, userRequest.GetNamespace(), userRequest.GetName(),
				fmt.Sprintf("%s %s", userRequest.Spec.FirstName, userRequest.Spec.LastName), userRequest.Spec.Email, code)
		} else {
			t.sendEmail("user-email-verification-malfunction", userRequest.Spec.Tenant, userRequest.GetNamespace(), userRequest.GetName(),
				fmt.Sprintf("%s %s", userRequest.Spec.FirstName, userRequest.Spec.LastName), userRequest.Spec.Email, "")
		}
	}
	return code, created
}

// sendEmail to send notification to tenant-admins and authorized users about email verification
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
		// Put the email addresses of the tenant-admins and authorized users in the email to be sent list
		/*userRaw, _ := t.edgenetClientset.RegistrationV1alpha().Users(namespace).List(context.TODO(), metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			if strings.ToLower(userRow.Status.Type) == "admin" {
				collective.CommonData.Email = append(collective.CommonData.Email, userRow.Spec.Email)
			}
		}*/
		contentData = collective
	} else {
		collective.CommonData.Email = []string{email}
		contentData = collective
	}

	mailer.Send(subject, contentData)
}

// statusUpdate to update the objects that are relevant the request and send email
func (t *Handler) statusUpdate(emailVerification *registrationv1alpha.EmailVerification) {
	labels := emailVerification.GetLabels()
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
		userRequestObj, _ := t.edgenetClientset.RegistrationV1alpha().UserRequests().Get(context.TODO(), labels["edge-net.io/user"], metav1.GetOptions{})
		userRequestObj.Status.EmailVerified = true
		t.edgenetClientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequestObj, metav1.UpdateOptions{})
		// Send email to inform tenant-admins and authorized users
		t.sendEmail("user-email-verified-alert", labels["edge-net.io/tenant"], "", labels["edge-net.io/user"],
			fmt.Sprintf("%s %s", userRequestObj.Spec.FirstName, userRequestObj.Spec.LastName), "", "")
	} else if strings.ToLower(labels["edge-net.io/registration"]) == "email" {
		acceptableUsePolicy, _ := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), labels["edge-net.io/user"], metav1.GetOptions{})
		acceptableUsePolicy.Spec.Accepted = true
		t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy, metav1.UpdateOptions{})

		// TO-DO: Get user contact information
		// Send email to inform user
		// t.sendEmail("user-email-verified-notification", labels["edge-net.io/tenant"], "", labels["edge-net.io/user"],
		// fmt.Sprintf("%s %s", userObj.Spec.FirstName, userObj.Spec.LastName), userObj.Spec.Email, "")
	}
}

// runVerificationTimeout puts a procedure in place to remove requests by verification or timeout
func (t *Handler) runVerificationTimeout(emailVerification *registrationv1alpha.EmailVerification) {
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if emailVerification.Status.Expiry != nil {
		timeout = time.After(time.Until(emailVerification.Status.Expiry.Time))
	}
	closeChannels := func() {
		close(terminated)
	}

	// Watch the events of email verification object
	watchEmailVerifiation, err := t.edgenetClientset.RegistrationV1alpha().EmailVerifications().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", emailVerification.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for emailVerificationEvent := range watchEmailVerifiation.ResultChan() {
				// Get updated email verification object
				updatedEmailVerification, status := emailVerificationEvent.Object.(*registrationv1alpha.EmailVerification)
				if emailVerification.GetUID() == updatedEmailVerification.GetUID() {
					if status {
						if emailVerificationEvent.Type == "DELETED" || updatedEmailVerification.Spec.Verified {
							terminated <- true
							continue
						}
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching emailverification resources,
		// there is a timeout at 3 hours
		timeout = time.After(3 * time.Hour)
	}

	// Wait on multiple channel operations
	select {
	case <-timeout:
		watchEmailVerifiation.Stop()
		t.edgenetClientset.RegistrationV1alpha().EmailVerifications().Delete(context.TODO(), emailVerification.GetName(), metav1.DeleteOptions{})
		closeChannels()
	case <-terminated:
		watchEmailVerifiation.Stop()
		closeChannels()
	}
}
