package registration

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/controller/v1alpha/authority"
	"flag"
	"fmt"
	"reflect"

	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"io/ioutil"
	"log"
	"os"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

type RegistrationTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	userObj       apps_v1alpha.User
	client        kubernetes.Interface
	edgenetclient versioned.Interface
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Args = []string{"-headnode-path", "../../configs/headnode_template.yaml"}

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
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

func TestMakeUser(t *testing.T) {
	g := RegistrationTestGroup{}
	g.Init()
	// Get the user object
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	// Find the authority from the namespace in which the object is (needed for invoking MakeUser)
	userOwnerNamespace, _ := g.client.CoreV1().Namespaces().Get(user.GetNamespace(), metav1.GetOptions{})
	_, _, err := MakeUser(userOwnerNamespace.Labels["authority-name"], user.GetName(), user.Spec.Email)
	if err != nil {
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
	// Get the user object
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	// Find the authority from the namespace in which the object is (needed for invoking MakeUser)
	userOwnerNamespace, _ := g.client.CoreV1().Namespaces().Get(user.GetNamespace(), metav1.GetOptions{})
	serviceAccount, _ := CreateServiceAccount(g.userObj.DeepCopy(), "User", userOwnerNamespace.GetObjectMeta().GetOwnerReferences())
	output := CreateConfig(serviceAccount)
	if !reflect.DeepEqual(output, "Serviceaccount unittesting doesn't have a serviceaccount token\n") {
		t.Errorf("failed")
	}
}
