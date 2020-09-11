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

package user

import (
	"context"
	"fmt"
	"reflect"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/v1alpha/emailverification"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	"github.com/EdgeNet-project/edgenet/pkg/permission"
	"github.com/EdgeNet-project/edgenet/pkg/registration"

	log "github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	log.Info("UserHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("UserHandler.ObjectCreated")
	// Create a copy of the user object to make changes on it
	userCopy := obj.(*apps_v1alpha.User).DeepCopy()
	// Find the authority from the namespace in which the object is
	userOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), userCopy.GetNamespace(), metav1.GetOptions{})
	// Check if the email address is already taken
	emailExists, message := t.checkDuplicateObject(userCopy, userOwnerNamespace.Labels["authority-name"])

	if emailExists {
		userCopy.Spec.Active = false
		userUpdated, err := t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Update(context.TODO(), userCopy, metav1.UpdateOptions{})
		if err == nil {
			userCopy = userUpdated
		}
		userCopy.Status.State = failure
		userCopy.Status.Message = []string{message}
		t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(context.TODO(), userCopy, metav1.UpdateOptions{})
		return
	}
	userOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), userOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})

	_, serviceAccountErr := t.clientset.CoreV1().ServiceAccounts(userCopy.GetNamespace()).Get(context.TODO(), userCopy.GetName(), metav1.GetOptions{})
	if !errors.IsNotFound(serviceAccountErr) {
		// To demission service accounts providing user authentication
		t.clientset.CoreV1().ServiceAccounts(userCopy.GetNamespace()).Delete(context.TODO(), userCopy.GetName(), metav1.DeleteOptions{})
	}
	// Check if the authority is active
	if (userOwnerAuthority.Spec.Enabled && userCopy.Spec.Active) || (userOwnerAuthority.Spec.Enabled && userCopy.Spec.Active && !errors.IsNotFound(serviceAccountErr)) {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		_, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(userCopy.GetNamespace()).Get(context.TODO(), userCopy.GetName(), metav1.GetOptions{})
		if err != nil || !errors.IsNotFound(serviceAccountErr) {
			// Automatically creates an acceptable use policy object belonging to the user in the authority namespace
			// When a user is deleted, the owner references feature allows the related AUP to be automatically removed
			userOwnerReferences := SetAsOwnerReference(userCopy)
			userAUP := &apps_v1alpha.AcceptableUsePolicy{TypeMeta: metav1.TypeMeta{Kind: "AcceptableUsePolicy", APIVersion: "apps.edgenet.io/v1alpha"},
				ObjectMeta: metav1.ObjectMeta{Name: userCopy.GetName(), OwnerReferences: userOwnerReferences}, Spec: apps_v1alpha.AcceptableUsePolicySpec{Accepted: false}}
			t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(userCopy.GetNamespace()).Create(context.TODO(), userAUP, metav1.CreateOptions{})
			// Create user-specific roles regarding the resources of authority, users, and acceptableusepolicies
			policyRule := []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"authorities"}, ResourceNames: []string{userOwnerNamespace.Labels["authority-name"]},
				Verbs: []string{"get"}}, {APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"users"}, ResourceNames: []string{userCopy.GetName()}, Verbs: []string{"get", "update", "patch"}}}
			userRole := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("user-%s", userCopy.GetName()), OwnerReferences: userOwnerReferences},
				Rules: policyRule}
			_, err := t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Create(context.TODO(), userRole, metav1.CreateOptions{})
			if err != nil {
				log.Infof("Couldn't create user-%s role: %s", userCopy.GetName(), err)
				if errors.IsAlreadyExists(err) {
					currentUserRole, err := t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Get(context.TODO(), userRole.GetName(), metav1.GetOptions{})
					if err == nil {
						currentUserRole.Rules = policyRule
						_, err = t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Update(context.TODO(), currentUserRole, metav1.UpdateOptions{})
						if err == nil {
							log.Infof("User-%s role updated", userCopy.GetName())
						}
					}
				}
			}
			// Create a dedicated role to allow the user access to accept/reject AUP, even if the AUP is rejected
			policyRule = []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"acceptableusepolicies", "acceptableusepolicies/status"}, ResourceNames: []string{userCopy.GetName()},
				Verbs: []string{"get", "update", "patch"}}}
			userRole = &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("user-aup-%s", userCopy.GetName()), OwnerReferences: userOwnerReferences},
				Rules: policyRule}
			_, err = t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Create(context.TODO(), userRole, metav1.CreateOptions{})
			if err != nil {
				log.Infof("Couldn't create user-aup-%s role: %s", userCopy.GetName(), err)
				if errors.IsAlreadyExists(err) {
					currentUserRole, err := t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Get(context.TODO(), userRole.GetName(), metav1.GetOptions{})
					if err == nil {
						currentUserRole.Rules = policyRule
						_, err = t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Update(context.TODO(), currentUserRole, metav1.UpdateOptions{})
						if err == nil {
							log.Infof("User-aup-%s role updated", userCopy.GetName())
						}
					}
				}
			}
			defer t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(context.TODO(), userCopy, metav1.UpdateOptions{})
			if userOwnerAuthority.Spec.Contact.Username == userCopy.GetName() && userOwnerAuthority.Spec.Contact.Email == userCopy.Spec.Email {
				userCopy.Status.Type = "admin"
			} else {
				userCopy.Status.Type = "user"
			}
			// Create the client certs for permanent use
			// In next versions, there will be a method to renew the certs for security
			crt, key, err := registration.MakeUser(userOwnerNamespace.Labels["authority-name"], userCopy.GetName(), userCopy.Spec.Email)
			if err != nil {
				log.Println(err.Error())
				userCopy.Status.State = failure
				userCopy.Status.Message = []string{fmt.Sprintf(statusDict["cert-fail"], userCopy.GetName())}
				t.sendEmail(userCopy, userOwnerNamespace.Labels["authority-name"], "user-cert-failure")
				return
			}
			err = registration.MakeConfig(userOwnerNamespace.Labels["authority-name"], userCopy.GetName(), userCopy.Spec.Email, crt, key)
			if err != nil {
				log.Println(err.Error())
				userCopy.Status.State = failure
				userCopy.Status.Message = []string{fmt.Sprintf(statusDict["kubeconfig-fail"], userCopy.GetName())}
				t.sendEmail(userCopy, userOwnerNamespace.Labels["authority-name"], "user-kubeconfig-failure")
			}
			userCopy.Status.State = success
			userCopy.Status.Message = []string{statusDict["cert-ok"]}
			t.sendEmail(userCopy, userOwnerNamespace.Labels["authority-name"], "user-registration-successful")

			slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(userCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
			teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(userCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
			t.createRoleBindings(userCopy, slicesRaw, teamsRaw, userOwnerAuthority.GetName())
			t.createAUPRoleBinding(userCopy)
		}
	} else if userOwnerAuthority.Spec.Enabled == false && userCopy.Spec.Active == true {
		defer t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Update(context.TODO(), userCopy, metav1.UpdateOptions{})
		userCopy.Spec.Active = false
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("UserHandler.ObjectUpdated")
	// Create a copy of the user object to make changes on it
	userCopy := obj.(*apps_v1alpha.User).DeepCopy()
	userOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), userCopy.GetNamespace(), metav1.GetOptions{})
	// Check if the email address is already taken
	emailExists, message := t.checkDuplicateObject(userCopy, userOwnerNamespace.Labels["authority-name"])
	if emailExists {
		userCopy.Spec.Active = false
		userUpdated, err := t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Update(context.TODO(), userCopy, metav1.UpdateOptions{})
		if err == nil {
			userCopy = userUpdated
		}
		userCopy.Status.State = failure
		userCopy.Status.Message = []string{message}
		t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(context.TODO(), userCopy, metav1.UpdateOptions{})
		return
	}
	userOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), userOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	fieldUpdated := updated.(fields)
	// Security check to prevent any kind of manipulation on the AUP
	if fieldUpdated.aup {
		userAUP, _ := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(userCopy.GetNamespace()).Get(context.TODO(), userCopy.GetName(), metav1.GetOptions{})
		if userAUP.Spec.Accepted != userCopy.Status.AUP {
			userCopy.Status.AUP = userAUP.Spec.Accepted
			userCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(context.TODO(), userCopy, metav1.UpdateOptions{})
			if err == nil {
				userCopy = userCopyUpdated
			}
		}
	}
	if userOwnerAuthority.Spec.Enabled {
		if fieldUpdated.email {
			userCopy.Spec.Active = false
			userCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Update(context.TODO(), userCopy, metav1.UpdateOptions{})
			if err == nil {
				userCopy = userCopyUpdated
			} else {
				log.Infof("Couldn't deactivate user %s in %s: %s", userCopy.GetName(), userCopy.GetNamespace(), err)
				t.sendEmail(userCopy, userOwnerNamespace.Labels["authority-name"], "user-deactivation-failure")
			}
			emailVerificationHandler := emailverification.Handler{}
			emailVerificationHandler.Init(t.clientset, t.edgenetClientset)
			created := emailVerificationHandler.Create(userCopy, SetAsOwnerReference(userCopy))
			if created {
				// Update the status as successful
				userCopy.Status.State = success
				userCopy.Status.Message = []string{statusDict["email-ok"]}
			} else {
				userCopy.Status.State = failure
				userCopy.Status.Message = []string{statusDict["email-fail"]}
			}

			userCopyUpdated, err = t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(context.TODO(), userCopy, metav1.UpdateOptions{})
			if err == nil {
				userCopy = userCopyUpdated
			}
		}

		if userCopy.Spec.Active && userCopy.Status.AUP {
			// To manipulate role bindings according to the changes
			if fieldUpdated.active || fieldUpdated.aup || fieldUpdated.role {
				slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(userCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
				teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(userCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
				if fieldUpdated.role {
					t.deleteRoleBindings(userCopy, slicesRaw, teamsRaw)
				}
				t.createRoleBindings(userCopy, slicesRaw, teamsRaw, userOwnerAuthority.GetName())
				if fieldUpdated.active {
					t.createAUPRoleBinding(userCopy)
				}
			}
		} else if !userCopy.Spec.Active || !userCopy.Status.AUP {
			// To manipulate role bindings according to the changes
			if (userCopy.Spec.Active == false && fieldUpdated.active) || (userCopy.Status.AUP == false && fieldUpdated.aup) {
				slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(userCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
				teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(userCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
				t.deleteRoleBindings(userCopy, slicesRaw, teamsRaw)
			}
			// To create AUP role binding for the user
			if userCopy.Spec.Active && fieldUpdated.active {
				t.createAUPRoleBinding(userCopy)
			}
		}
	} else if userOwnerAuthority.Spec.Enabled == false && userCopy.Spec.Active == true {
		defer t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Update(context.TODO(), userCopy, metav1.UpdateOptions{})
		userCopy.Spec.Active = false
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("UserHandler.ObjectDeleted")
	// Mail notification, TBD
}

// Create function is for being used by other resources to create an authority
func (t *Handler) Create(obj interface{}) bool {
	failed := true
	switch obj.(type) {
	case *apps_v1alpha.UserRegistrationRequest:
		URRCopy := obj.(*apps_v1alpha.UserRegistrationRequest)
		// Create a user on the cluster
		user := apps_v1alpha.User{}
		user.SetName(URRCopy.GetName())
		user.Spec.Bio = URRCopy.Spec.Bio
		user.Spec.Email = URRCopy.Spec.Email
		user.Spec.FirstName = URRCopy.Spec.FirstName
		user.Spec.LastName = URRCopy.Spec.LastName
		user.Spec.URL = URRCopy.Spec.URL
		user.Spec.Active = true
		_, err := t.edgenetClientset.AppsV1alpha().Users(URRCopy.GetNamespace()).Create(context.TODO(), user.DeepCopy(), metav1.CreateOptions{})
		if err == nil {
			failed = false
			t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRCopy.GetNamespace()).Delete(context.TODO(), URRCopy.GetName(), metav1.DeleteOptions{})
		}
	}

	return failed
}

// createRoleBindings creates user role bindings according to the roles
func (t *Handler) createRoleBindings(userCopy *apps_v1alpha.User, slicesRaw *apps_v1alpha.SliceList, teamsRaw *apps_v1alpha.TeamList, ownerAuthority string) {
	// Create role bindings independent of user roles
	permission.EstablishPrivateRoleBindings(userCopy)
	// This part creates the rolebindings one by one in different namespaces
	createLoop := func(slicesRaw *apps_v1alpha.SliceList, namespacePrefix string) {
		for _, sliceRow := range slicesRaw.Items {
			for _, sliceUser := range sliceRow.Spec.Users {
				// If the user participates in the slice or it is an Authority-admin of the owner authority
				if (sliceUser.Authority == ownerAuthority && sliceUser.Username == userCopy.GetName()) ||
					(userCopy.GetNamespace() == sliceRow.GetNamespace() && userCopy.Status.Type == "admin") ||
					permission.CheckAuthorization(namespacePrefix, userCopy.Spec.Email, "slices", sliceRow.GetName()) {
					permission.EstablishRoleBindings(userCopy, fmt.Sprintf("%s-slice-%s", namespacePrefix, sliceRow.GetName()), "Slice")
				}
			}
		}
	}
	// Create the rolebindings in the authority namespace
	permission.EstablishRoleBindings(userCopy, userCopy.GetNamespace(), "Authority")
	createLoop(slicesRaw, userCopy.GetNamespace())
	// List the teams in the authority namespace
	for _, teamRow := range teamsRaw.Items {
		for _, teamUser := range teamRow.Spec.Users {
			// If the user participates in the team or it is an Authority-admin of the owner authority
			if (teamUser.Authority == ownerAuthority && teamUser.Username == userCopy.GetName()) ||
				(userCopy.GetNamespace() == teamRow.GetNamespace() && userCopy.Status.Type == "admin") ||
				permission.CheckAuthorization(userCopy.GetNamespace(), userCopy.Spec.Email, "teams", teamRow.GetName()) {
				permission.EstablishRoleBindings(userCopy, fmt.Sprintf("%s-team-%s", userCopy.GetNamespace(), teamRow.GetName()), "Team")
			}
		}
		// List the slices in the team namespace
		teamSlicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("%s-team-%s", userCopy.GetNamespace(), teamRow.GetName())).List(context.TODO(), metav1.ListOptions{})
		createLoop(teamSlicesRaw, fmt.Sprintf("%s-team-%s", userCopy.GetNamespace(), teamRow.GetName()))
	}
}

// deleteRoleBindings removes user role bindings in the namespaces related
func (t *Handler) deleteRoleBindings(userCopy *apps_v1alpha.User, slicesRaw *apps_v1alpha.SliceList, teamsRaw *apps_v1alpha.TeamList) {
	// To delete the cluster role binding which allows user to get the authority object
	t.clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), fmt.Sprintf("%s-%s-for-authority", userCopy.GetNamespace(), userCopy.GetName()), metav1.DeleteOptions{})
	// This part deletes the rolebindings one by one
	deletionLoop := func(roleBindings *rbacv1.RoleBindingList) {
		for _, roleBindingRow := range roleBindings.Items {
			for _, roleBindingSubject := range roleBindingRow.Subjects {
				if roleBindingSubject.Kind == "User" && (roleBindingSubject.Name == userCopy.Spec.Email) &&
					roleBindingSubject.Namespace == userCopy.GetNamespace() {
					t.clientset.RbacV1().RoleBindings(roleBindingRow.GetNamespace()).Delete(context.TODO(), roleBindingRow.GetName(), metav1.DeleteOptions{})
					break
				}
			}
		}
	}
	// Unless the user gets deactivated it has access to edit the AUP
	roleBindingListOptions := metav1.ListOptions{}
	if userCopy.Spec.Active {
		roleBindingListOptions = metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name!=%s-user-aup-%s", userCopy.GetNamespace(), userCopy.GetName())}
	}
	// List the rolebindings in the authority namespace
	roleBindings, _ := t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).List(context.TODO(), roleBindingListOptions)
	deletionLoop(roleBindings)
	// List the rolebindings in the slice namespaces which directly created by slices in the authority namespace
	for _, sliceRow := range slicesRaw.Items {
		roleBindings, _ := t.clientset.RbacV1().RoleBindings(fmt.Sprintf("%s-slice-%s", userCopy.GetNamespace(), sliceRow.GetName())).List(context.TODO(), metav1.ListOptions{})
		deletionLoop(roleBindings)
	}
	for _, teamRow := range teamsRaw.Items {
		// List the rolebindings in the team namespace
		roleBindings, _ := t.clientset.RbacV1().RoleBindings(teamRow.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
		deletionLoop(roleBindings)
		// List the rolebindings in the slice namespaces which created by slices in the team namespace
		teamSlicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("%s-team-%s", userCopy.GetNamespace(), teamRow.GetName())).List(context.TODO(), metav1.ListOptions{})
		for _, teamSliceRow := range teamSlicesRaw.Items {
			roleBindings, _ := t.clientset.RbacV1().RoleBindings(fmt.Sprintf("%s-team-%s-slice-%s", userCopy.GetNamespace(), teamRow.GetName(), teamSliceRow.GetName())).List(context.TODO(), metav1.ListOptions{})
			deletionLoop(roleBindings)
		}
	}
}

// createAUPRoleBinding links the AUP up with the user
func (t *Handler) createAUPRoleBinding(userCopy *apps_v1alpha.User) error {
	_, err := t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-%s", userCopy.GetNamespace(),
		fmt.Sprintf("user-aup-%s", userCopy.GetName())), metav1.GetOptions{})
	if err != nil {
		// roleName to get user-specific AUP role which allows user to only get the AUP object related to itself
		roleName := fmt.Sprintf("user-aup-%s", userCopy.GetName())
		roleRef := rbacv1.RoleRef{Kind: "Role", Name: roleName}
		rbSubjects := []rbacv1.Subject{{Kind: "User", Name: userCopy.Spec.Email, APIGroup: "rbac.authorization.k8s.io"}}
		roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: userCopy.GetNamespace(), Name: fmt.Sprintf("%s-%s", userCopy.GetNamespace(), roleName)},
			Subjects: rbSubjects, RoleRef: roleRef}
		// When a user is deleted, the owner references feature allows the related role binding to be automatically removed
		userOwnerReferences := SetAsOwnerReference(userCopy)
		roleBind.ObjectMeta.OwnerReferences = userOwnerReferences
		_, err = t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Create(context.TODO(), roleBind, metav1.CreateOptions{})
		if err != nil {
			log.Infof("Couldn't create user-aup-%s role: %s", userCopy.GetName(), err)
			if errors.IsAlreadyExists(err) {
				userRoleBind, err := t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{})
				if err == nil {
					userRoleBind.Subjects = rbSubjects
					userRoleBind.RoleRef = roleRef
					_, err = t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Update(context.TODO(), userRoleBind, metav1.UpdateOptions{})
					if err == nil {
						log.Infof("Completed: user-aup-%s role updated", userCopy.GetName())
					}
				}
			}
		}
	}
	return err
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(userCopy *apps_v1alpha.User, authorityName, subject string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Authority = authorityName
	contentData.CommonData.Username = userCopy.GetName()
	contentData.CommonData.Name = fmt.Sprintf("%s %s", userCopy.Spec.FirstName, userCopy.Spec.LastName)
	contentData.CommonData.Email = []string{userCopy.Spec.Email}
	mailer.Send(subject, contentData)
}

// checkDuplicateObject checks whether a user exists with the same username or email address
func (t *Handler) checkDuplicateObject(userCopy *apps_v1alpha.User, authorityName string) (bool, string) {
	exists := false
	var message string
	// To check email address
	userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(context.TODO(), metav1.ListOptions{})
	for _, userRow := range userRaw.Items {
		if userRow.Spec.Email == userCopy.Spec.Email && userRow.GetUID() != userCopy.GetUID() {
			exists = true
			message = fmt.Sprintf("Email address, %s, already exists for another user account", userCopy.Spec.Email)
			break
		}
	}
	if !exists {
		// Delete the user registration requests which have duplicate values, if any
		URRRaw, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests("").List(context.TODO(), metav1.ListOptions{})
		for _, URRRow := range URRRaw.Items {
			if URRRow.Spec.Email == userCopy.Spec.Email {
				t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRRow.GetNamespace()).Delete(context.TODO(), URRRow.GetName(), metav1.DeleteOptions{})
			}
		}
		// Delete the user registration requests which have duplicate values in the same namespace, if any
		URRRaw, _ = t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(userCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
		for _, URRRow := range URRRaw.Items {
			if URRRow.GetName() == userCopy.GetName() || URRRow.Spec.Email == userCopy.Spec.Email {
				t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRRow.GetNamespace()).Delete(context.TODO(), URRRow.GetName(), metav1.DeleteOptions{})
			}
		}
	} else if exists && !reflect.DeepEqual(userCopy.Status.Message, message) {
		t.sendEmail(userCopy, authorityName, "user-validation-failure")
	}
	return exists, message
}

// SetAsOwnerReference puts the user as owner
func SetAsOwnerReference(userCopy *apps_v1alpha.User) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newUserRef := *metav1.NewControllerRef(userCopy, apps_v1alpha.SchemeGroupVersion.WithKind("User"))
	takeControl := false
	newUserRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newUserRef)
	return ownerReferences
}
