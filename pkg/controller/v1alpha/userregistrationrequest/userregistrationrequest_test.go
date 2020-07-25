package userregistrationrequest

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"edgenet/pkg/controller/v1alpha/authority"
	"edgenet/pkg/controller/v1alpha/user"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type URRTestGroup struct {
	authorityObj        apps_v1alpha.Authority
	userObj             apps_v1alpha.User
	userRegistrationObj apps_v1alpha.UserRegistrationRequest
	client              kubernetes.Interface
	edgenetclient       versioned.Interface
	handler             Handler
}

var ErrorDict = map[string]string{
	"k8-sync":        "Kubernetes clientset sync problem",
	"edgnet-sync":    "EdgeNet clientset sync problem",
	"URR-timeout":    "Failed to update approval timeout of user Request",
	"usr-email-coll": "Failed to detect user email address collision",
	"usr-approv":     "Failed to create user from user Request after approval",
	"add-func":       "Add func of event handler authority request doesn't work properly",
	"usr-URR":        "Failed to create user from user Request after approval",
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *URRTestGroup) Init() {
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
			Enabled: true,
		},
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user1",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "user",
			LastName:  "NAME",
			Email:     "userName@edge-net.org",
			Active:    false,
		},
		Status: apps_v1alpha.UserStatus{
			State: success,
			Type:  "Admin",
		},
	}
	URRObj := apps_v1alpha.UserRegistrationRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRegistrationRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "userRegistrationRequestName",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserRegistrationRequestSpec{
			FirstName: "user",
			LastName:  "NAME",
			Email:     "URR@edge-net.org",
		},
		Status: apps_v1alpha.UserRegistrationRequestStatus{
			EmailVerified: true,
			Expires:       nil,
			Message:       nil,
		},
	}
	g.authorityObj = authorityObj
	g.userRegistrationObj = URRObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	//invoke ObjectCreated to create namespace
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := URRTestGroup{}
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

func TestURRCreate(t *testing.T) {
	g := URRTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creation of Authority reques
	t.Run("creation of Authority request", func(t *testing.T) {
		g.edgenetclient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userRegistrationObj.DeepCopy())
		g.handler.ObjectCreated(g.userRegistrationObj.DeepCopy())
		userRequest, _ := g.edgenetclient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userRegistrationObj.GetName(), metav1.GetOptions{})
		t.Log(userRequest.Status.Expires)
		if userRequest.Status.Expires == nil {
			t.Errorf("Failed to update approval timeout of user Request")
		}
	})
}

func TestURRUpdate(t *testing.T) {
	g := URRTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	t.Run("checking duplicate user names and emails", func(t *testing.T) {
		// Create user registration request
		g.edgenetclient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userRegistrationObj.DeepCopy())
		g.handler.ObjectCreated(g.userRegistrationObj.DeepCopy())
		// Create a user
		userHandler := user.Handler{}
		userHandler.Init(g.client, g.edgenetclient)
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
		userHandler.ObjectCreated(g.userObj.DeepCopy())
		// Set user Request object email equal to existing user email
		g.userRegistrationObj.Spec.Email = "userName@edge-net.org"
		// Update server's representation of user registration request
		g.edgenetclient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(g.userRegistrationObj.DeepCopy())
		g.handler.ObjectUpdated(g.userRegistrationObj.DeepCopy())
		URR, _ := g.edgenetclient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userRegistrationObj.GetName(), metav1.GetOptions{})
		if URR.Status.Message == nil {
			t.Error(ErrorDict["usr-email-coll"])
		}
	})

	t.Run("Testing Authority Request transition to Authority", func(t *testing.T) {
		// Updating user registration status to approved
		g.userRegistrationObj.Spec.Approved = true
		g.userRegistrationObj.Spec.Email = "URR@edge-net.org"
		// Updating the user registration object
		g.edgenetclient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(g.userRegistrationObj.DeepCopy())
		// Requesting server to update internal representation of user registration object and transition it to user
		g.handler.ObjectUpdated(g.userRegistrationObj.DeepCopy())
		// Checking if user with same name as user registration object exists
		User, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userRegistrationObj.GetName(), metav1.GetOptions{})
		if User == nil {
			t.Error(ErrorDict["usr-approv"])
		}
	})
}
