package registration

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/controller/v1alpha/authority"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"

	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"io/ioutil"
	"log"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"k8s.io/client-go/kubernetes"
)

type RegistrationTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	userObj       apps_v1alpha.User
	client        kubernetes.Interface
	edgenetclient versioned.Interface
}

func (g *RegistrationTestGroup) Init() {
	authorityObj := apps_v1alpha.Authority{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Authority",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: apps_v1alpha.AuthoritySpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: apps_v1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: apps_v1alpha.Contact{
				Email:     "unittest@edge-net.org",
				FirstName: "unit",
				LastName:  "testing",
				Phone:     "+33NUMBER",
				Username:  "unittesting",
			},
			Enabled: false,
		},
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unittesting",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "EdgeNet",
			LastName:  "EdgeNet",
			Email:     "unittest@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			Type: "Admin",
		},
	}
	g.authorityObj = authorityObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	// Invoke authority ObjectCreated to create namespace
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
	// Sync Clientset with fake client
	Clientset = g.client
}

func TestMakeConfig(t *testing.T) {
	g := RegistrationTestGroup{}
	g.Init()
	// Get the user object
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	// Get the client certificate and key
	clientcert, err := ioutil.ReadFile("../../assets/certs/unittest@edge-net.org.crt")
	if err != nil {
		log.Printf("Registration: unexpected error executing command: %v", err)
	}
	clientkey, err := ioutil.ReadFile("../../assets/certs/unittest@edge-net.org.key")
	if err != nil {
		log.Printf("Registration: unexpected error executing command: %v", err)
	}
	// call the MakeConfig function
	err = MakeConfig(g.authorityObj.GetName(), user.GetName(), user.Spec.Email, clientcert, clientkey)
	if err != nil {
		t.Errorf("MakeConfig Failed")
	}

}

func TestUserCreation(t *testing.T) {
	g := RegistrationTestGroup{}
	g.Init()
	// Get the user object
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	// Find the authority from the namespace in which the object is located (needed for invoking MakeUser)
	userOwnerNamespace, _ := g.client.CoreV1().Namespaces().Get(user.GetNamespace(), metav1.GetOptions{})
	// Mock the signer
	go func() {
		timeout := time.After(10 * time.Second)
		ticker := time.Tick(1 * time.Second)
	check:
		for {
			select {
			case <-timeout:
				break check
			case <-ticker:
				CSRObj, getErr := Clientset.CertificatesV1beta1().CertificateSigningRequests().Get(fmt.Sprintf("%s-%s", userOwnerNamespace.Labels["authority-name"], user.GetName()), metav1.GetOptions{})
				if getErr == nil {
					CSRObj.Status.Certificate = CSRObj.Spec.Request
					_, updateErr := Clientset.CertificatesV1beta1().CertificateSigningRequests().UpdateStatus(CSRObj)
					if updateErr == nil {
						break check
					}
				}
			}
		}
	}()
	_, _, err := MakeUser(userOwnerNamespace.Labels["authority-name"], user.GetName(), user.Spec.Email)
	if err != nil {
		t.Log(err)
		t.Errorf("MakeUser Failed")
	}
}

func TestCreateServiceAccount(t *testing.T) {
	g := RegistrationTestGroup{}
	g.Init()
	// Get the user object
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	// Find the authority from the namespace in which the object is (needed for invoking MakeUser)
	userOwnerNamespace, _ := g.client.CoreV1().Namespaces().Get(user.GetNamespace(), metav1.GetOptions{})
	_, err := CreateServiceAccount(g.userObj.DeepCopy(), "User", userOwnerNamespace.GetObjectMeta().GetOwnerReferences())
	if err != nil {
		t.Errorf("Create service Account Failed")
	}
}

func TestCreateConfig(t *testing.T) {
	g := RegistrationTestGroup{}
	g.Init()
	t.Run("Create config while we don't have secrets in service account", func(t *testing.T) {
		// Get the user object
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
		// Find the userOwner from the namespace in which the object is (needed for invoking MakeUser)
		userOwnerNamespace, _ := g.client.CoreV1().Namespaces().Get(user.GetNamespace(), metav1.GetOptions{})
		serviceAccount, _ := CreateServiceAccount(g.userObj.DeepCopy(), "User", userOwnerNamespace.GetObjectMeta().GetOwnerReferences())

		output := CreateConfig(serviceAccount)
		if !reflect.DeepEqual(output, "Serviceaccount unittesting doesn't have a serviceaccount token\n") {
			t.Log(output)
			t.Errorf("failed")
		}
	})
	t.Run("service account has a secrets", func(t *testing.T) {
		// Get the user object
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
		ownerReferences := []metav1.OwnerReference{
			metav1.OwnerReference{
				Kind:       "OwnerRefrence",
				APIVersion: "apps.edgenet.io/v1alpha",
				Name:       "unittest",
			},
		}
		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:            user.GetName(),
				OwnerReferences: ownerReferences,
			},
			Secrets: []corev1.ObjectReference{
				corev1.ObjectReference{
					Name: "test-token-test",
				},
			},
		}
		output := CreateConfig(serviceAccount)
		if !reflect.DeepEqual(output, "Secret unittesting not found\n") {
			t.Log(output)
			t.Errorf("failed")
		}
	})
}
