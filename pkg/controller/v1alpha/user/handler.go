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
	"strings"
	"time"

	apps_v1alpha "headnode/pkg/apis/apps/v1alpha"
	"headnode/pkg/authorization"
	"headnode/pkg/client/clientset/versioned"
	"headnode/pkg/mailer"
	"headnode/pkg/registration"

	log "github.com/Sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
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
	log.Info("UserHandler.Init")
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
	log.Info("UserHandler.ObjectCreated")
	// Create a copy of the user object to make changes on it
	userCopy := obj.(*apps_v1alpha.User).DeepCopy()
	// Check if the email address is already taken
	emailExists := t.checkDuplicateObject(userCopy)
	if emailExists {
		// If it is already taken, remove the user registration request object
		t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Delete(userCopy.GetName(), &metav1.DeleteOptions{})
		return
	}
	// Find the authority from the namespace in which the object is
	userOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(userCopy.GetNamespace(), metav1.GetOptions{})
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
				Verbs: []string{"get"}}, {APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"users"}, ResourceNames: []string{userCopy.GetName()}, Verbs: []string{"get"}},
				{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"logins"}, ResourceNames: []string{userCopy.GetName()}, Verbs: []string{"*"}}}
			userRole := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("user-%s", userCopy.GetName()), OwnerReferences: userOwnerReferences},
				Rules: policyRule}
			_, err := t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Create(userRole)
			if err != nil {
				log.Infof("Couldn't create user-%s role: %s", userCopy.GetName(), err)
			}
			// Create a dedicated role to allow the user access to accept/reject AUP, even if the AUP is rejected
			policyRule = []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"acceptableusepolicies", "acceptableusepolicies/status"}, ResourceNames: []string{userCopy.GetName()},
				Verbs: []string{"get", "update", "patch"}}}
			userRole = &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("user-aup-%s", userCopy.GetName()), OwnerReferences: userOwnerReferences},
				Rules: policyRule}
			_, err = t.clientset.RbacV1().Roles(userCopy.GetNamespace()).Create(userRole)
			if err != nil {
				log.Infof("Couldn't create user-aup-%s role: %s", userCopy.GetName(), err)
			}

			// Check if the password has been replaced by a secret already
			_, err = t.clientset.CoreV1().Secrets(userCopy.GetNamespace()).Get(userCopy.Spec.Password, metav1.GetOptions{})
			if err != nil {
				// Create a user-specific secret to keep the password safe
				passwordSecret := registration.CreateSecretByPassword(userCopy)
				// Update the password field as the secret's name for later use
				userCopy.Spec.Password = passwordSecret
				userCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Update(userCopy)
				if err == nil {
					// To manipulate the object later
					userCopy = userCopyUpdated
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
			}
			// This function collects the bearer token from the created service account to form kubeconfig file and send it by email
			makeConfigAvailable := func() {
				for range time.Tick(30 * time.Second) {
					serviceAccount, _ := t.clientset.CoreV1().ServiceAccounts(userCopy.GetNamespace()).Get(userCopy.GetName(), metav1.GetOptions{})
					if len(serviceAccount.Secrets) > 0 {
						registration.CreateConfig(serviceAccount)
						break
					}
				}

			checkTokenTimer:
				for {
					select {
					// Check every 30 seconds whether the secret related to the service account has been generated
					case <-time.Tick(30 * time.Second):
						serviceAccount, _ := t.clientset.CoreV1().ServiceAccounts(userCopy.GetNamespace()).Get(userCopy.GetName(), metav1.GetOptions{})
						if len(serviceAccount.Secrets) > 0 {
							// Create kubeconfig file according to the web service account
							registration.CreateConfig(serviceAccount)
							// Set the HTML template variables
							contentData := mailer.CommonContentData{}
							contentData.CommonData.Authority = userOwnerNamespace.Labels["authority-name"]
							contentData.CommonData.Username = userCopy.GetName()
							contentData.CommonData.Name = fmt.Sprintf("%s %s", userCopy.Spec.FirstName, userCopy.Spec.LastName)
							contentData.CommonData.Email = []string{userCopy.Spec.Email}
							mailer.Send("user-registration-successful", contentData)
							break checkTokenTimer
						}
					case <-time.After(15 * time.Minute):
						// Mail notification, TBD
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
		if userCopy.Status.Active && userCopy.Status.AUP {
			// To update the secret of password
			if fieldUpdated.password {
				_, err := t.clientset.CoreV1().Secrets(userCopy.GetNamespace()).Get(userCopy.Spec.Password, metav1.GetOptions{})
				if err != nil {
					t.clientset.CoreV1().Secrets(userCopy.GetNamespace()).Delete(fmt.Sprintf("%s-pass", userCopy.GetName()), &metav1.DeleteOptions{})
					passwordSecret := registration.CreateSecretByPassword(userCopy)
					userCopy.Spec.Password = passwordSecret
					userCopyUpdated, err := t.edgenetClientset.AppsV1alpha().Users(userCopy.GetNamespace()).Update(userCopy)
					if err == nil {
						userCopy = userCopyUpdated
					}
				}
			}
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
				if roleBindingSubject.Kind == "ServiceAccount" && (roleBindingSubject.Name == userCopy.GetName() || roleBindingSubject.Name == fmt.Sprintf("%s-webauth", userCopy.GetName())) &&
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
		if userCopy.Status.WebAuth {
			rbSubjectWebAuth := rbacv1.Subject{Kind: "ServiceAccount", Name: fmt.Sprintf("%s-webauth", userCopy.GetName()), Namespace: userCopy.GetNamespace()}
			rbSubjects = append(rbSubjects, rbSubjectWebAuth)
		}
		roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: userCopy.GetNamespace(), Name: fmt.Sprintf("%s-%s", userCopy.GetNamespace(), roleName)},
			Subjects: rbSubjects, RoleRef: roleRef}
		// When a user is deleted, the owner references feature allows the related role binding to be automatically removed
		userOwnerReferences := t.setOwnerReferences(userCopy)
		roleBind.ObjectMeta.OwnerReferences = userOwnerReferences
		t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).Create(roleBind)
	}
}

// checkDuplicateObject checks whether a user exists with the same username or email address
func (t *Handler) checkDuplicateObject(userCopy *apps_v1alpha.User) bool {
	exist := false
	// To check email address
	userRaw, _ := t.edgenetClientset.AppsV1alpha().Users("").List(metav1.ListOptions{})
	for _, userRow := range userRaw.Items {
		if userRow.Spec.Email == userCopy.Spec.Email && userRow.GetUID() != userCopy.GetUID() {
			exist = true
			break
		}
	}
	if !exist {
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
	}
	// Mail notification, TBD
	return exist
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
