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

package login

import (
	"encoding/base64"
	"fmt"
	"regexp"
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
	ObjectUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        *kubernetes.Clientset
	edgenetClientset *versioned.Clientset
}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("LoginHandler.Init")
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
	log.Info("LoginHandler.ObjectCreated")
	// Create a copy of the login object to make changes on it
	loginCopy := obj.(*apps_v1alpha.Login).DeepCopy()
	// Find the authority from the namespace in which the object is
	loginOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(loginCopy.GetNamespace(), metav1.GetOptions{})
	loginOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(loginOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	// Check if the authority is active
	if loginOwnerAuthority.Status.Enabled {
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		credentialsMatch, user := t.validateCredentials(loginCopy)
		if credentialsMatch {
			// Run timeout goroutine
			go t.runLoginTimeout(loginCopy)
			if loginCopy.Status.Expires == nil {
				defer t.edgenetClientset.AppsV1alpha().Logins(loginCopy.GetNamespace()).UpdateStatus(loginCopy)
				if loginCopy.Status.Renew {
					loginCopy.Status.Renew = false
				}
				// Set the email verification timeout which is 1 hour
				loginCopy.Status.Expires = &metav1.Time{
					Time: time.Now().Add(1 * time.Hour),
				}
			} else if loginCopy.Status.Expires != nil && loginCopy.Status.Expires.Time.Sub(time.Now()) < 0 {
				// Run the procedures to securely logout if timeout exists and expired
				t.secureLogout(loginCopy)
				// Terminate the function
				return
			}
			// Create a service account dedicated to web login of the user
			// That brings security measures onto the email provider level by providing web token via email
			_, err := registration.CreateServiceAccount(user, "webauth")
			if err != nil {
				log.Println(err.Error())
			}
			// This function collects the bearer token from the service account created to be sent by email
			sendTokenByEmail := func() {
			checkTokenTimer:
				for {
					select {
					// Check every 30 seconds whether the secret related to the service account has been generated
					case <-time.Tick(30 * time.Second):
						webServiceAccount, _ := t.clientset.CoreV1().ServiceAccounts(user.GetNamespace()).Get(fmt.Sprintf("%s-webauth", user.GetName()), metav1.GetOptions{})
						if len(webServiceAccount.Secrets) > 0 {
							// Create kubeconfig file according to the web service account
							registration.CreateConfig(webServiceAccount)
							// Get the secret name from the service account
							tokenSecret := webServiceAccount.Secrets[0].Name
							// Double-check on the web token
							for _, accountSecret := range webServiceAccount.Secrets {
								match, _ := regexp.MatchString(fmt.Sprintf("%s-webauth-token-([a-z0-9]+)", user.GetName()), accountSecret.Name)
								if match {
									tokenSecret = accountSecret.Name
								}
							}
							secret, _ := t.clientset.CoreV1().Secrets(webServiceAccount.GetNamespace()).Get(tokenSecret, metav1.GetOptions{})
							// Set the HTML template variables including the web token
							contentData := mailer.LoginContentData{}
							contentData.CommonData.Authority = loginOwnerNamespace.Labels["authority-name"]
							contentData.CommonData.Username = user.GetName()
							contentData.CommonData.Name = fmt.Sprintf("%s %s", user.Spec.FirstName, user.Spec.LastName)
							contentData.CommonData.Email = []string{user.Spec.Email}
							contentData.Token = string(secret.Data["token"])
							mailer.Send("login", contentData)

							break checkTokenTimer
						}
					case <-time.After(15 * time.Minute):
						t.secureLogout(loginCopy)
						break checkTokenTimer
					}
				}
			}
			go sendTokenByEmail()
			// Update user status because the user has successfully signed in
			user.Status.WebAuth = true
			go t.edgenetClientset.AppsV1alpha().Users(user.GetNamespace()).UpdateStatus(user)
			// Copy the core user role bindings to grant the same access rights to the web token
			slicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(user.GetNamespace()).List(metav1.ListOptions{})
			teamsRaw, _ := t.edgenetClientset.AppsV1alpha().Teams(user.GetNamespace()).List(metav1.ListOptions{})
			t.updateRoleBindings(user, slicesRaw, teamsRaw)
		} else {
			t.secureLogout(loginCopy)
		}
	} else {
		credentialsMatch, _ := t.validateCredentials(loginCopy)
		if !credentialsMatch {
			t.secureLogout(loginCopy)
		}
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("LoginHandler.ObjectUpdated")
	// Create a copy of the login object to make changes on it
	loginCopy := obj.(*apps_v1alpha.Login).DeepCopy()
	loginOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(loginCopy.GetNamespace(), metav1.GetOptions{})
	loginOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(loginOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})

	if loginOwnerAuthority.Status.Enabled {
		defer t.edgenetClientset.AppsV1alpha().Logins(loginCopy.GetNamespace()).UpdateStatus(loginCopy)
		// Extend the expiration date
		if loginCopy.Status.Renew {
			loginCopy.Status.Expires = &metav1.Time{
				Time: time.Now().Add(1 * time.Hour),
			}
		}
		loginCopy.Status.Renew = false
	} else {
		t.secureLogout(loginCopy)
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("LoginHandler.ObjectDeleted")
	// Mail notification, TBD
}

// validateCredentials checks that those are a valid username and password
func (t *Handler) validateCredentials(loginCopy *apps_v1alpha.Login) (bool, *apps_v1alpha.User) {
	user, err := t.edgenetClientset.AppsV1alpha().Users(loginCopy.GetNamespace()).Get(loginCopy.GetName(), metav1.GetOptions{})
	if err == nil {
		if user.Status.Active && user.Status.AUP {
			secret, err := t.clientset.CoreV1().Secrets(user.GetNamespace()).Get(fmt.Sprintf("%s-pass", user.GetName()), metav1.GetOptions{})
			if err == nil {
				loginPassword := base64.StdEncoding.EncodeToString([]byte(loginCopy.Spec.Password))
				userPassword := base64.StdEncoding.EncodeToString(secret.Data["password"])
				if loginPassword == userPassword {
					return true, user
				}
			}
		}
	}
	return false, nil
}

// secureLogout applies the procedure for removing all objects related to the web login
func (t *Handler) secureLogout(loginCopy *apps_v1alpha.Login) {
	// Update user status as web authentication terminated
	user, _ := t.edgenetClientset.AppsV1alpha().Users(loginCopy.GetNamespace()).Get(loginCopy.GetName(), metav1.GetOptions{})
	user.Status.WebAuth = false
	t.edgenetClientset.AppsV1alpha().Users(user.GetNamespace()).UpdateStatus(user)
	// Delete the service account assigned for web login
	t.clientset.CoreV1().ServiceAccounts(user.GetNamespace()).Delete(fmt.Sprintf("%s-webauth", user.GetName()), &metav1.DeleteOptions{})
	// Delete the login object to allow new login attempts
	t.edgenetClientset.AppsV1alpha().Logins(loginCopy.GetNamespace()).Delete(loginCopy.GetName(), &metav1.DeleteOptions{})
}

// updateRoleBindings links the web service account up with the rolebindings of user
func (t *Handler) updateRoleBindings(userCopy *apps_v1alpha.User, slicesRaw *apps_v1alpha.SliceList, teamsRaw *apps_v1alpha.TeamList) {
	// This part puts the web service account into the rolebindings one by one
	updateLoop := func(roleBindings *rbacv1.RoleBindingList) {
		for _, roleBindingRow := range roleBindings.Items {
			for _, roleBindingSubject := range roleBindingRow.Subjects {
				if roleBindingSubject.Kind == "ServiceAccount" && roleBindingSubject.Name == userCopy.GetName() && roleBindingSubject.Namespace == userCopy.GetNamespace() {
					roleBindingCopy := roleBindingRow.DeepCopy()
					rbSubject := rbacv1.Subject{Kind: "ServiceAccount", Name: fmt.Sprintf("%s-webauth", userCopy.GetName()), Namespace: userCopy.GetNamespace()}
					roleBindingCopy.Subjects = append(roleBindingCopy.Subjects, rbSubject)
					t.clientset.RbacV1().RoleBindings(roleBindingRow.GetNamespace()).Update(roleBindingCopy)
					break
				}
			}
		}
	}
	// List the rolebindings in the authority namespace
	roleBindings, _ := t.clientset.RbacV1().RoleBindings(userCopy.GetNamespace()).List(metav1.ListOptions{})
	updateLoop(roleBindings)
	// List the rolebindings in the slice namespaces which directly created by slices in the authority namespace
	for _, sliceRow := range slicesRaw.Items {
		roleBindings, _ := t.clientset.RbacV1().RoleBindings(fmt.Sprintf("%s-slice-%s", userCopy.GetNamespace(), sliceRow.GetName())).List(metav1.ListOptions{})
		updateLoop(roleBindings)
	}
	for _, teamRow := range teamsRaw.Items {
		// List the rolebindings in the team namespace
		roleBindings, _ := t.clientset.RbacV1().RoleBindings(teamRow.GetNamespace()).List(metav1.ListOptions{})
		updateLoop(roleBindings)
		// List the rolebindings in the slice namespaces which created by slices in the team namespace
		teamSlicesRaw, _ := t.edgenetClientset.AppsV1alpha().Slices(fmt.Sprintf("%s-team-%s", userCopy.GetNamespace(), teamRow.GetName())).List(metav1.ListOptions{})
		for _, teamSliceRow := range teamSlicesRaw.Items {
			roleBindings, _ := t.clientset.RbacV1().RoleBindings(fmt.Sprintf("%s-team-%s-slice-%s", userCopy.GetNamespace(), teamRow.GetName(), teamSliceRow.GetName())).List(metav1.ListOptions{})
			updateLoop(roleBindings)
		}
	}
}

// runLoginTimeout puts a procedure in place to remove requests by approval or timeout
func (t *Handler) runLoginTimeout(loginCopy *apps_v1alpha.Login) {
	timeoutRenewed := make(chan bool, 1)
	terminated := make(chan bool, 1)
	var timeout <-chan time.Time
	if loginCopy.Status.Expires != nil {
		timeout = time.After(time.Until(loginCopy.Status.Expires.Time))
	}
	closeChannels := func() {
		close(timeoutRenewed)
		close(terminated)
	}

	// Watch the events of login object
	watchLogin, err := t.edgenetClientset.AppsV1alpha().Logins(loginCopy.GetNamespace()).Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", loginCopy.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for LoginEvent := range watchLogin.ResultChan() {
				// Get updated login object
				updatedLogin, status := LoginEvent.Object.(*apps_v1alpha.Login)
				if status {
					if LoginEvent.Type == "DELETED" {
						terminated <- true
						continue
					}

					if updatedLogin.Status.Expires != nil {
						// Check whether expiration date updated
						if loginCopy.Status.Expires != nil && timeout != nil {
							if loginCopy.Status.Expires.Time == updatedLogin.Status.Expires.Time {
								loginCopy = updatedLogin
								continue
							}
						}

						if updatedLogin.Status.Expires.Time.Sub(time.Now()) >= 0 {
							timeout = time.After(time.Until(updatedLogin.Status.Expires.Time))
							timeoutRenewed <- true
						}
					}
					loginCopy = updatedLogin
				}
			}
		}()
	} else {
		// In case of any malfunction of watching login resources,
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
			watchLogin.Stop()
			user, _ := t.edgenetClientset.AppsV1alpha().Users(loginCopy.GetNamespace()).Get(loginCopy.GetName(), metav1.GetOptions{})
			user.Status.WebAuth = false
			t.edgenetClientset.AppsV1alpha().Users(user.GetNamespace()).UpdateStatus(user)

			t.clientset.CoreV1().ServiceAccounts(user.GetNamespace()).Delete(fmt.Sprintf("%s-webauth", user.GetName()), &metav1.DeleteOptions{})
			t.edgenetClientset.AppsV1alpha().Logins(loginCopy.GetNamespace()).Delete(loginCopy.GetName(), &metav1.DeleteOptions{})
			closeChannels()
			break timeoutLoop
		case <-terminated:
			watchLogin.Stop()
			user, _ := t.edgenetClientset.AppsV1alpha().Users(loginCopy.GetNamespace()).Get(loginCopy.GetName(), metav1.GetOptions{})
			user.Status.WebAuth = false
			t.edgenetClientset.AppsV1alpha().Users(user.GetNamespace()).UpdateStatus(user)

			t.clientset.CoreV1().ServiceAccounts(user.GetNamespace()).Delete(fmt.Sprintf("%s-webauth", user.GetName()), &metav1.DeleteOptions{})
			t.edgenetClientset.AppsV1alpha().Logins(loginCopy.GetNamespace()).Delete(loginCopy.GetName(), &metav1.DeleteOptions{})
			closeChannels()
			break timeoutLoop
		}
	}
}
