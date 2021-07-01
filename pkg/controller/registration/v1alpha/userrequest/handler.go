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

package userrequest

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	tenantv1alpha "github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenant"
	"github.com/EdgeNet-project/edgenet/pkg/controller/registration/v1alpha/emailverification"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	"github.com/EdgeNet-project/edgenet/pkg/permission"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
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
	log.Info("UserRequestHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
	permission.Clientset = t.clientset
}

// ObjectCreatedOrUpdated is called when an object is created
func (t *Handler) ObjectCreatedOrUpdated(obj interface{}) {
	log.Info("UserRequestHandler.ObjectCreated")
	// Make a copy of the user registration request object to make changes on it
	userRequest := obj.(*registrationv1alpha.UserRequest).DeepCopy()
	if userRequest.Status.State != approved {
		defer func() {
			if !reflect.DeepEqual(obj.(*registrationv1alpha.UserRequest).Status, userRequest.Status) {
				if _, err := t.edgenetClientset.RegistrationV1alpha().UserRequests().UpdateStatus(context.TODO(), userRequest, metav1.UpdateOptions{}); err != nil {
					// TO-DO: Provide more information on error
					log.Println(err)
				}
			}
		}()

		tenant, _ := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), strings.ToLower(userRequest.Spec.Tenant), metav1.GetOptions{})
		// Check if the tenant is active
		if tenant.Spec.Enabled {
			if userRequest.Spec.Approved {
				userRequest.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})
				tenantHandler := tenantv1alpha.Handler{}
				tenantHandler.Init(t.clientset, t.edgenetClientset)
				tenantHandler.ConfigurePermissions(tenant, userRequest, tenantv1alpha.SetAsOwnerReference(tenant))

				if aupFailure, _ := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["aup-rolebinding-failure"], userRequest.Spec.Email)); !aupFailure {
					if certFailure, _ := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["cert-failure"], userRequest.Spec.Email)); !certFailure {
						if kubeconfigFailure, _ := util.Contains(tenant.Status.Message, fmt.Sprintf(statusDict["kubeconfig-failure"], userRequest.Spec.Email)); !kubeconfigFailure {
							userRequest.Status.State = approved
							userRequest.Status.Message = []string{statusDict["user-approved"]}
							return
						}
					}
				}
				t.sendEmail(userRequest, tenant.GetName(), "user-creation-failure")
				userRequest.Status.State = failure
				userRequest.Status.Message = []string{statusDict["user-failed"]}
			} else {
				if userRequest.Status.Expiry == nil {
					// Set the approval timeout which is 72 hours
					userRequest.Status.Expiry = &metav1.Time{
						Time: time.Now().Add(72 * time.Hour),
					}
				}
				exists, _ := util.Contains(userRequest.Status.Message, statusDict["email-ok"])
				if !exists {
					emailVerificationHandler := emailverification.Handler{}
					emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
					created := emailVerificationHandler.Create(userRequest, SetAsOwnerReference(userRequest))
					if created {
						// Update the status as successful
						userRequest.Status.State = success
						userRequest.Status.Message = []string{statusDict["email-ok"]}
					} else {
						userRequest.Status.State = issue
						userRequest.Status.Message = []string{statusDict["email-fail"]}
					}
				}
				labels := userRequest.GetLabels()
				ownerReferences := SetAsOwnerReference(userRequest)
				if err := permission.CreateObjectSpecificClusterRole(tenant.GetName(), "registration.edgenet.io", "userrequests", fmt.Sprintf("%s-%s", userRequest.GetName(), labels["edge-net.io/user-template-hash"]), "owner", []string{"get", "update", "patch"}, ownerReferences); err != nil && !errors.IsAlreadyExists(err) {
					log.Infof("Couldn't create user request cluster role %s, %s: %s", tenant.GetName(), userRequest.GetName(), err)
					// TODO: Provide err information at the status
				}

				if acceptableUsePolicyRaw, err := t.edgenetClientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s,edge-net.io/identity=true", tenant.GetName())}); err == nil {
					for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
						aupLabels := acceptableUsePolicyRow.GetLabels()
						if aupLabels != nil && aupLabels["edge-net.io/username"] != "" && aupLabels["edge-net.io/role"] != "" {
							if aupLabels["edge-net.io/role"] == "Owner" || aupLabels["edge-net.io/role"] == "Admin" {
								clusterRoleName := fmt.Sprintf("edgenet:%s:userrequests:%s-%s", tenant.GetName(), userRequest.GetName(), "owner")
								roleBindLabels := map[string]string{"edge-net.io/tenant": tenant.GetName(), "edge-net.io/username": aupLabels["edge-net.io/username"], "edge-net.io/user-template-hash": aupLabels["edge-net.io/user-template-hash"]}
								if err := permission.CreateObjectSpecificClusterRoleBinding(tenant.GetName(), clusterRoleName, fmt.Sprintf("%s-%s", aupLabels["edge-net.io/username"], aupLabels["edge-net.io/user-template-hash"]), acceptableUsePolicyRow.Spec.Email, roleBindLabels, ownerReferences); err != nil {
									// TODO: Define the error precisely
									userRequest.Status.State = failure
									userRequest.Status.Message = []string{statusDict["role-failed"]}
								}
							}
						}
					}
				}
			}
		} else {
			t.edgenetClientset.RegistrationV1alpha().UserRequests().Delete(context.TODO(), userRequest.GetName(), metav1.DeleteOptions{})
		}

	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("UserRequestHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(userRequest *registrationv1alpha.UserRequest, tenantName, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Tenant = tenantName
	contentData.CommonData.Username = userRequest.GetName()
	contentData.CommonData.Name = fmt.Sprintf("%s %s", userRequest.Spec.FirstName, userRequest.Spec.LastName)
	contentData.CommonData.Email = []string{userRequest.Spec.Email}
	mailer.Send(subject, contentData)
}

// RunExpiryController puts a procedure in place to turn accepted policies into not accepted
func (t *Handler) RunExpiryController() {
	var closestExpiry time.Time
	terminated := make(chan bool)
	newExpiry := make(chan time.Time)
	defer close(terminated)
	defer close(newExpiry)

	watchUserRequest, err := t.edgenetClientset.RegistrationV1alpha().UserRequests().Watch(context.TODO(), metav1.ListOptions{})
	if err == nil {
		watchEvents := func(watchUserRequest watch.Interface, newExpiry *chan time.Time) {
			// Watch the events of user request object
			// Get events from watch interface
			for userRequestEvent := range watchUserRequest.ResultChan() {
				// Get updated user request object
				updatedUserRequest, status := userRequestEvent.Object.(*registrationv1alpha.UserRequest)
				if status {
					if updatedUserRequest.Status.Expiry != nil {
						*newExpiry <- updatedUserRequest.Status.Expiry.Time
					}
				}
			}
		}
		go watchEvents(watchUserRequest, &newExpiry)
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
			userRequestRaw, err := t.edgenetClientset.RegistrationV1alpha().UserRequests().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// TO-DO: Provide more information on error
				log.Println(err)
			}
			for _, userRequestRow := range userRequestRaw.Items {
				if userRequestRow.Status.Expiry != nil && userRequestRow.Status.Expiry.Time.Sub(time.Now()) <= 0 {
					t.edgenetClientset.RegistrationV1alpha().UserRequests().Delete(context.TODO(), userRequestRow.GetName(), metav1.DeleteOptions{})
				} else if userRequestRow.Status.Expiry != nil && userRequestRow.Status.Expiry.Time.Sub(time.Now()) > 0 {
					if closestExpiry.Sub(time.Now()) <= 0 || closestExpiry.Sub(userRequestRow.Status.Expiry.Time) > 0 {
						closestExpiry = userRequestRow.Status.Expiry.Time
						log.Printf("ExpiryController: Closest expiry date is %v after the expiration of a user request", closestExpiry)
					}
				}
			}

			if closestExpiry.Sub(time.Now()) <= 0 {
				closestExpiry = time.Now().AddDate(1, 0, 0)
				log.Printf("ExpiryController: Closest expiry date is %v after the expiration of a user request", closestExpiry)
			}
		case <-terminated:
			watchUserRequest.Stop()
			break infiniteLoop
		}
	}
}

// SetAsOwnerReference put the userrequest as owner
func SetAsOwnerReference(userRequest *registrationv1alpha.UserRequest) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newNamespaceRef := *metav1.NewControllerRef(userRequest, registrationv1alpha.SchemeGroupVersion.WithKind("UserRequest"))
	takeControl := false
	newNamespaceRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newNamespaceRef)
	return ownerReferences
}
