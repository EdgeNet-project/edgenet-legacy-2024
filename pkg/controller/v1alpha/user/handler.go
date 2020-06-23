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
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/mailer"
	"edgenet/pkg/registration"

	log "github.com/Sirupsen/logrus"
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
	fmt.Printf("Check usercopy %v\n", userCopy)

	userOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(userCopy.GetNamespace(), metav1.GetOptions{})
	// Check if the email address is already taken
	fmt.Printf("Check userOwnerNamespace=\n\n %v\n\n", userOwnerNamespace)
	emailExists, message := t.checkDuplicateObject(userCopy, userOwnerNamespace.Labels["authority-name"])

	if emailExists {
		userCopy.Status.State = failure
		userCopy.Status.Message = []string{message}
		userCopy.Status.Active = false
		t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
		return
	}
	userOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(userOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	// Check if the authority is active
	if userOwnerAuthority.Status.Enabled == true && userCopy.GetGeneration() == 1 {

		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		_, err := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(userCopy.GetNamespace()).Get(userCopy.GetName(), metav1.GetOptions{})
		if err != nil {
			// Automatically creates an acceptable use policy object belonging to the user in the authority namespace
			// When a user is deleted, the owner references feature allows the related AUP to be automatically removed
			userOwnerReferences := t.setOwnerReferences(userCopy)
			userAUP := &apps_v1alpha.AcceptableUsePolicy{TypeMeta: metav1.TypeMeta{Kind: "AcceptableUsePolicy", APIVersion: "apps.edgenet.io/v1alpha"},
				ObjectMeta: metav1.ObjectMeta{Name: userCopy.GetName(), OwnerReferences: userOwnerReferences}, Spec: apps_v1alpha.AcceptableUsePolicySpec{Accepted: false}}
			t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(userCopy.GetNamespace()).Create(userAUP)
			// Create user-specific roles regarding the resources of authority, users, and acceptableusepolicies
			policyRule := []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"authorities"}, ResourceNames: []string{userOwnerNamespace.Labels["authority-name"]},
				Verbs: []string{"get"}}, {APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"users"}, ResourceNames: []string{userCopy.GetName()}, Verbs: []string{"get"}}}
			userRole := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("user-%s", userCopy.GetName()), OwnerReferences: userOwnerReferences},
				Rules: policyRule}
			_, err := t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Create(userRole)
			if err != nil {
				log.Infof("Couldn't create user-%s role: %s", userCopy.GetName(), err)
				log.Infoln(errors.IsAlreadyExists(err))
				if errors.IsAlreadyExists(err) {
					currentUserRole, err := t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Get(userRole.GetName(), metav1.GetOptions{})
					if err == nil {
						currentUserRole.Rules = policyRule
						_, err = t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Update(currentUserRole)
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
			_, err = t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Create(userRole)
			if err != nil {
				log.Infof("Couldn't create user-aup-%s role: %s", userCopy.GetName(), err)
				log.Infoln(errors.IsAlreadyExists(err))
				if errors.IsAlreadyExists(err) {
					currentUserRole, err := t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Get(userRole.GetName(), metav1.GetOptions{})
					if err == nil {
						currentUserRole.Rules = policyRule
						_, err = t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Update(currentUserRole)
						if err == nil {
							log.Infof("User-aup-%s role updated", userCopy.GetName())
						}
					}
				}
			}
			// Activate user
			defer t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
			userCopy.Status.Active = true
			// Create the main service account for permanent use
			// In next versions, there will be a method to renew the token of this service account for security
			_, err = registration.CreateServiceAccount(userCopy, "main")
			if err != nil {
				log.Println(err.Error())
				userCopy.Status.State = failure
				userCopy.Status.Message = []string{fmt.Sprintf("Service account creation failed for user %s", userCopy.GetName())}
				t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
				t.sendEmail(userCopy, userOwnerNamespace.Labels["authority-name"], "", "user-serviceaccount-failure")
				return
			}
			// This function collects the bearer token from the created service account to form kubeconfig file and send it by email
			makeConfigAvailable := func() {
			checkTokenTimer:
				for {
					select {
					// Check every 30 seconds whether the secret related to the service account has been generated
					case <-time.Tick(30 * time.Second):
						serviceAccount, _ := t.clientset.CoreV1().ServiceAccounts(userCopy.GetNamespace()).Get(userCopy.GetName(), metav1.GetOptions{})
						if len(serviceAccount.Secrets) > 0 {
							// Create kubeconfig file according to the web service account
							registration.CreateConfig(serviceAccount)
							t.sendEmail(userCopy, userOwnerNamespace.Labels["authority-name"], "", "user-registration-successful")
							break checkTokenTimer
						}
					case <-time.After(15 * time.Minute):
						userCopy.Status.State = failure
						userCopy.Status.Message = []string{fmt.Sprintf("Kubeconfig file generation failed for user %s", userCopy.GetName())}
						t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
						t.sendEmail(userCopy, userOwnerNamespace.Labels["authority-name"], "", "user-kubeconfig-failure")
						break checkTokenTimer
					}
				}
			}
			go makeConfigAvailable()
		}
	} else if userOwnerAuthority.Status.Enabled == false && userCopy.Status.Active == true {
		defer t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
		userCopy.Status.Active = false
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj, updated interface{}) {
	log.Info("UserHandler.ObjectUpdated")
	// Create a copy of the user object to make changes on it
	userCopy := obj.(*apps_v1alpha.User).DeepCopy()
	userOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(userCopy.GetNamespace(), metav1.GetOptions{})
	// Check if the email address is already taken
	emailExists, message := t.checkDuplicateObject(userCopy, userOwnerNamespace.Labels["authority-name"])
	if emailExists {
		userCopy.Status.State = failure
		userCopy.Status.Message = []string{message}
		userCopy.Status.Active = false
		t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
		return
	}
	userOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(userOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	fieldUpdated := updated.(fields)
	// Security check to prevent any kind of manipulation on the AUP
	if fieldUpdated.aup {
		userAUP, _ := t.edgenetClientset.AppsV1alpha().AcceptableUsePolicies(userCopy.GetNamespace()).Get(userCopy.GetName(), metav1.GetOptions{})
		if userAUP.Spec.Accepted != userCopy.Status.AUP {
			userCopy.Status.AUP = userAUP.Spec.Accepted
			userCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
			if err == nil {
				userCopy = userCopyUpdated
			}
		}
	}
	if userOwnerAuthority.Status.Enabled {
		if fieldUpdated.email {
			userCopy.Status.Active = false
			userCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
			if err == nil {
				userCopy = userCopyUpdated
			} else {
				log.Infof("Couldn't deactivate user %s in %s: %s", userCopy.GetName(), userCopy.GetNamespace(), err)
				t.sendEmail(userCopy, userOwnerNamespace.Labels["authority-name"], "", "user-deactivation-failure")
			}
			t.setEmailVerification(userCopy, userOwnerNamespace.Labels["authority-name"])
		}

		if userCopy.Status.Active && userCopy.Status.AUP {
			// To manipulate role bindings according to the changes
			if fieldUpdated.active || fieldUpdated.aup || fieldUpdated.roles {
				slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(userCopy.GetNamespace()).List(metav1.ListOptions{})
				teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(userCopy.GetNamespace()).List(metav1.ListOptions{})
				if fieldUpdated.roles {
					t.deleteRoleBindings(userCopy, slicesRaw, teamsRaw)
				}
				t.createRoleBindings(userCopy, slicesRaw, teamsRaw, userOwnerAuthority.GetName())
				if fieldUpdated.active {
					t.createAUPRoleBinding(userCopy)
				}
			}
		} else if !userCopy.Status.Active || !userCopy.Status.AUP {
			// To manipulate role bindings according to the changes
			if (userCopy.Status.Active == false && fieldUpdated.active) || (userCopy.Status.AUP == false && fieldUpdated.aup) {
				slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(userCopy.GetNamespace()).List(metav1.ListOptions{})
				teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(userCopy.GetNamespace()).List(metav1.ListOptions{})
				t.deleteRoleBindings(userCopy, slicesRaw, teamsRaw)
			}
			// To create AUP role binding for the user
			if userCopy.Status.Active && fieldUpdated.active {
				t.createAUPRoleBinding(userCopy)
			}
		}
	} else if userOwnerAuthority.Status.Enabled == false && userCopy.Status.Active == true {
		defer t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).UpdateStatus(userCopy)
		userCopy.Status.Active = false
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("UserHandler.ObjectDeleted")
	// Mail notification, TBD
}

// setEmailVerification to provide one-time code for verification
func (t *Handler) setEmailVerification(userCopy *apps_v1alpha.User, authorityName string) {
	// The section below is a part of the method which provides email verification
	// Email verification code is a security point for email verification. The user
	// object creates an email verification object with a name which is
	// this email verification code. Only who knows the authority and the email verification
	// code can manipulate that object by using a public token.
	userOwnerReferences := t.setOwnerReferences(userCopy)
	emailVerificationCode := "bs" + generateRandomString(16)
	emailVerification := apps_v1alpha.EmailVerification{ObjectMeta: metav1.ObjectMeta{OwnerReferences: userOwnerReferences}}
	emailVerification.SetName(emailVerificationCode)
	emailVerification.Spec.Kind = "Email"
	emailVerification.Spec.Identifier = userCopy.GetName()
	_, err := t.edgenetClientset.AppsV1alpha().EmailVerifications(userCopy.GetNamespace()).Create(emailVerification.DeepCopy())
	if err == nil {
		t.sendEmail(userCopy, authorityName, emailVerificationCode, "user-email-verification-update")
	} else {
		t.sendEmail(userCopy, authorityName, "", "user-email-verification-update-malfunction")
	}
}

// createRoleBindings creates user role bindings according to the roles
func (t *Handler) createRoleBindings(userCopy *apps_v1alpha.User, slicesRaw *apps_v1alpha.SliceList, teamsRaw *apps_v1alpha.TeamList, ownerAuthority string) {
	// Create role bindings independent of user roles
	registration.CreateSpecificRoleBindings(userCopy)
	// This part creates the rolebindings one by one in different namespaces
	createLoop := func(slicesRaw *apps_v1alpha.SliceList, namespacePrefix string) {
		for _, sliceRow := range slicesRaw.Items {
			for _, sliceUser := range sliceRow.Spec.Users {
				// If the user participates in the slice or it is an Authority-admin or a Manager of the owner authority
				if (sliceUser.Authority == ownerAuthority && sliceUser.Username == userCopy.GetName()) ||
					(userCopy.GetNamespace() == sliceRow.GetNamespace() && (containsRole(userCopy.Spec.Roles, "admin") || containsRole(userCopy.Spec.Roles, "manager"))) {
					registration.CreateRoleBindingsByRoles(userCopy, fmt.Sprintf("%s-slice-%s", namespacePrefix, sliceRow.GetName()), "Slice")
				}
			}
		}
	}
	// Create the rolebindings in the authority namespace
	registration.CreateRoleBindingsByRoles(userCopy, userCopy.GetNamespace(), "Authority")
	createLoop(slicesRaw, userCopy.GetNamespace())
	// List the teams in the authority namespace
	for _, teamRow := range teamsRaw.Items {
		for _, teamUser := range teamRow.Spec.Users {
			// If the user participates in the team or it is an Authority-admin or a Manager of the owner authority
			if (teamUser.Authority == ownerAuthority && teamUser.Username == userCopy.GetName()) ||
				(userCopy.GetNamespace() == teamRow.GetNamespace() && (containsRole(userCopy.Spec.Roles, "admin") || containsRole(userCopy.Spec.Roles, "manager"))) {
				registration.CreateRoleBindingsByRoles(userCopy, fmt.Sprintf("%s-team-%s", userCopy.GetNamespace(), teamRow.GetName()), "Team")
			}
		}
		// List the slices in the team namespace
		teamSlicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("%s-team-%s", userCopy.GetNamespace(), teamRow.GetName())).List(metav1.ListOptions{})
		createLoop(teamSlicesRaw, fmt.Sprintf("%s-team-%s", userCopy.GetNamespace(), teamRow.GetName()))
	}
}

// deleteRoleBindings removes user role bindings in the namespaces related
func (t *Handler) deleteRoleBindings(userCopy *apps_v1alpha.User, slicesRaw *apps_v1alpha.SliceList, teamsRaw *apps_v1alpha.TeamList) {
	// To delete the cluster role binding which allows user to get the authority object
	t.clientset.RbacV1().ClusterRoleBindings().Delete(fmt.Sprintf("%s-%s-for-authority", userCopy.GetNamespace(), userCopy.GetName()), &metav1.DeleteOptions{})
	// This part deletes the rolebindings one by one
	deletionLoop := func(roleBindings *rbacv1.RoleBindingList) {
		for _, roleBindingRow := range roleBindings.Items {
			for _, roleBindingSubject := range roleBindingRow.Subjects {
				if roleBindingSubject.Kind == "ServiceAccount" && (roleBindingSubject.Name == userCopy.GetName()) &&
					roleBindingSubject.Namespace == userCopy.GetNamespace() {
					t.clientset.RbacV1().RoleBindings(roleBindingRow.GetNamespace()).Delete(roleBindingRow.GetName(), &metav1.DeleteOptions{})
					break
				}
			}
		}
	}
	// Unless the user gets deactivated it has access to edit the AUP
	roleBindingListOptions := metav1.ListOptions{}
	if userCopy.Status.Active {
		roleBindingListOptions = metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name!=%s-user-aup-%s", userCopy.GetNamespace(), userCopy.GetName())}
	}
	// List the rolebindings in the authority namespace
	roleBindings, _ := t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).List(roleBindingListOptions)
	deletionLoop(roleBindings)
	// List the rolebindings in the slice namespaces which directly created by slices in the authority namespace
	for _, sliceRow := range slicesRaw.Items {
		roleBindings, _ := t.clientset.RbacV1().RoleBindings(fmt.Sprintf("%s-slice-%s", userCopy.GetNamespace(), sliceRow.GetName())).List(metav1.ListOptions{})
		deletionLoop(roleBindings)
	}
	for _, teamRow := range teamsRaw.Items {
		// List the rolebindings in the team namespace
		roleBindings, _ := t.clientset.RbacV1().RoleBindings(teamRow.GetNamespace()).List(metav1.ListOptions{})
		deletionLoop(roleBindings)
		// List the rolebindings in the slice namespaces which created by slices in the team namespace
		teamSlicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("%s-team-%s", userCopy.GetNamespace(), teamRow.GetName())).List(metav1.ListOptions{})
		for _, teamSliceRow := range teamSlicesRaw.Items {
			roleBindings, _ := t.clientset.RbacV1().RoleBindings(fmt.Sprintf("%s-team-%s-slice-%s", userCopy.GetNamespace(), teamRow.GetName(), teamSliceRow.GetName())).List(metav1.ListOptions{})
			deletionLoop(roleBindings)
		}
	}
}

// createAUPRoleBinding links the AUP up with the user
func (t *Handler) createAUPRoleBinding(userCopy *apps_v1alpha.User) {
	_, err := t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Get(fmt.Sprintf("%s-%s", userCopy.GetNamespace(),
		fmt.Sprintf("user-aup-%s", userCopy.GetName())), metav1.GetOptions{})
	if err != nil {
		// roleName to get user-specific AUP role which allows user to only get the AUP object related to itself
		roleName := fmt.Sprintf("user-aup-%s", userCopy.GetName())
		roleRef := rbacv1.RoleRef{Kind: "Role", Name: roleName}
		rbSubjects := []rbacv1.Subject{{Kind: "ServiceAccount", Name: userCopy.GetName(), Namespace: userCopy.GetNamespace()}}
		roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: userCopy.GetNamespace(), Name: fmt.Sprintf("%s-%s", userCopy.GetNamespace(), roleName)},
			Subjects: rbSubjects, RoleRef: roleRef}
		// When a user is deleted, the owner references feature allows the related role binding to be automatically removed
		userOwnerReferences := t.setOwnerReferences(userCopy)
		roleBind.ObjectMeta.OwnerReferences = userOwnerReferences
		_, err = t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Create(roleBind)
		if err != nil {
			log.Infof("Couldn't create user-aup-%s role: %s", userCopy.GetName(), err)
		}
	}
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(userCopy *apps_v1alpha.User, authorityName, emailVerificationCode, subject string) {
	// Set the HTML template variables
	var contentData interface{}
	var collective = mailer.CommonContentData{}
	collective.CommonData.Authority = authorityName
	collective.CommonData.Username = userCopy.GetName()
	collective.CommonData.Name = fmt.Sprintf("%s %s", userCopy.Spec.FirstName, userCopy.Spec.LastName)
	collective.CommonData.Email = []string{userCopy.Spec.Email}
	if subject == "user-email-verification-update" {
		verifyContent := mailer.VerifyContentData{}
		verifyContent.Code = emailVerificationCode
		verifyContent.CommonData = collective.CommonData
		contentData = verifyContent
	} else {
		contentData = collective
	}
	mailer.Send(subject, contentData)
}

// checkDuplicateObject checks whether a user exists with the same username or email address
func (t *Handler) checkDuplicateObject(userCopy *apps_v1alpha.User, authorityName string) (bool, string) {
	exists := false
	var message string
	// To check email address
	userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(metav1.ListOptions{})
	for _, userRow := range userRaw.Items {
		if userRow.Spec.Email == userCopy.Spec.Email && userRow.GetUID() != userCopy.GetUID() {
			exists = true
			message = fmt.Sprintf("Email address, %s, already exists for another user account", userCopy.Spec.Email)
			break
		}
	}
	if !exists {
		// Delete the user registration requests which have duplicate values, if any
		URRRaw, _ := t.edgenetClientset.AppsV1alpha().UserRegistrationRequests("").List(metav1.ListOptions{})
		for _, URRRow := range URRRaw.Items {
			if URRRow.Spec.Email == userCopy.Spec.Email {
				t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRRow.GetNamespace()).Delete(URRRow.GetName(), &metav1.DeleteOptions{})
			}
		}
		// Delete the user registration requests which have duplicate values in the same namespace, if any
		URRRaw, _ = t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(userCopy.GetNamespace()).List(metav1.ListOptions{})
		for _, URRRow := range URRRaw.Items {
			if URRRow.GetName() == userCopy.GetName() || URRRow.Spec.Email == userCopy.Spec.Email {
				t.edgenetClientset.AppsV1alpha().UserRegistrationRequests(URRRow.GetNamespace()).Delete(URRRow.GetName(), &metav1.DeleteOptions{})
			}
		}
	} else if exists && !reflect.DeepEqual(userCopy.Status.Message, message) {
		t.sendEmail(userCopy, authorityName, "", "user-validation-failure")
	}
	return exists, message
}

// setOwnerReferences puts the user as owner
func (t *Handler) setOwnerReferences(userCopy *apps_v1alpha.User) []metav1.OwnerReference {
	ownerReferences := []metav1.OwnerReference{}
	newUserRef := *metav1.NewControllerRef(userCopy, apps_v1alpha.SchemeGroupVersion.WithKind("User"))
	takeControl := false
	newUserRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newUserRef)
	return ownerReferences
}

// To check whether user is holder of a role
func containsRole(roles []string, value string) bool {
	for _, ele := range roles {
		if strings.ToLower(value) == strings.ToLower(ele) {
			return true
		}
	}
	return false
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
