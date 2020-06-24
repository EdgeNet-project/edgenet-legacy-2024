package user

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"edgenet/pkg/controller/v1alpha/authority"

	"github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

//The main structure of test group
type UserTestGroup struct {
	authorityObj        apps_v1alpha.Authority
	authorityRequestObj apps_v1alpha.AuthorityRequest
	userObj             apps_v1alpha.User
	client              kubernetes.Interface
	edgenetclient       versioned.Interface
	handler             Handler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

//Init syncs the test group
func (g *UserTestGroup) Init() {

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
		},
		Status: apps_v1alpha.AuthorityStatus{
			Enabled: false,
		},
	}
	authorityRequestObj := apps_v1alpha.AuthorityRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AuthorityRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: apps_v1alpha.AuthorityRequestSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: apps_v1alpha.Address{
				City:    "Paris",
				Country: "France",
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
		},
		Status: apps_v1alpha.AuthorityRequestStatus{
			State: success,
		},
	}

	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "unittesting",
			Namespace:  "authority-edgenet",
			UID:        "TestUID",
			Generation: 1,
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "EdgeNet",
			LastName:  "EdgeNet",
			Roles:     []string{"Admin"},
			Email:     "unittest@edge-net.org",
		},
		Status: apps_v1alpha.UserStatus{
			State:  success,
			Active: true,
		},
	}
	g.authorityObj = authorityObj
	g.authorityRequestObj = authorityRequestObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
}

//TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	//Sync the test group
	g := UserTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)

	if g.handler.clientset != g.client {
		t.Error("Kubernetes clientset sync problem")
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error("EdgeNet clientset sync problem")
	}
}

func TestUserCreate(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	//invoke authority ObjectCreated to create namespace
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())

	t.Run("creation of user authority", func(t *testing.T) {
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
		if user == nil {
			t.Error("User generation failed when an authority created")
		}
	})
	t.Run("creation of user", func(t *testing.T) {
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		status, err := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).List(metav1.ListOptions{})
		t.Logf("\nstatusSS= %v\n", status)
		t.Logf("\nerrRR= %v\n", err)
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.GetName(), metav1.GetOptions{})
		if user == nil {
			t.Error("User creation failed")
		}
	})
	t.Run("check dublicate object", func(t *testing.T) {
		// Change the user object name to make comparison with the user-created above
		g.userObj.Name = "different"
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.GetName(), metav1.GetOptions{})
		if user != nil {
			t.Error("Duplicate value cannot be detected")
		}
	})
}

func TestUserUpdate(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())

	g.userObj.Name = "different"
	g.userObj.Spec.Email = "check"
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())

	g.userObj.Spec.Email = "unittest@edge-net.org"
	var field fields
	field.active = false
	field.aup = false
	field.roles = false
	field.email = true
	g.handler.ObjectUpdated(g.userObj.DeepCopy(), field)
	user, err2 := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.GetName(), metav1.GetOptions{})
	if user.Spec.Email == "unittest@edge-net.org" {
		t.Error("Duplicate value cannot be detected\n")
	}
	t.Logf("ERR= %v\n", err2)
	status, err := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).List(metav1.ListOptions{})
	t.Logf("status= %v\n", status)
	t.Logf("err= %v\n", err)
}

func TestDuplicateValue(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	t.Run("user : same email address", func(t *testing.T) {
		//create a user for comparison
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
		exists, _ := g.handler.checkDuplicateObject(g.userObj.DeepCopy(), fmt.Sprintf("authority-%s", g.authorityObj.GetName()))
		if exists != true {
			t.Error("User having same email address cannot be detected")
		}
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(g.userObj.GetName(), &metav1.DeleteOptions{})

	})
}

func TestGenerateRandomString(t *testing.T) {
	for i := 1; i < 5; i++ {
		origin := generateRandomString(16)
		time.Sleep(1 * time.Second)
		test := generateRandomString(16)
		if origin == test {
			t.Error("User GenerateRadnomString failed")
		}
	}
}
