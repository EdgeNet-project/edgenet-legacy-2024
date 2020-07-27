package authorityrequest

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"edgenet/pkg/controller/v1alpha/authority"
	"edgenet/pkg/controller/v1alpha/user"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type ARTestGroup struct {
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

// Init syncs the test group
func (g *ARTestGroup) Init() {
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
	authorityRequestObj := apps_v1alpha.AuthorityRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "authorityRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "authorityRequestName",
		},
		Spec: apps_v1alpha.AuthorityRequestSpec{
			FullName:  "John Doe",
			ShortName: "John",
			URL:       "",
			Address: apps_v1alpha.Address{
				Street:  "",
				ZIP:     "",
				City:    "",
				Region:  "",
				Country: "",
			},
			Contact: apps_v1alpha.Contact{
				Username:  "JohnDoe",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "123456789",
				Email:     "JohnDoe@edge-net.org",
			},
			Approved: false,
		},
		Status: apps_v1alpha.AuthorityRequestStatus{
			EmailVerified: false,
			Expires:       nil,
			State:         "",
			Message:       nil,
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
			Type:  "Admin",
			State: success,
		},
	}
	g.authorityObj = authorityObj
	g.authorityRequestObj = authorityRequestObj
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
	g := ARTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error(errorDict["k8-sync"])
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error(errorDict["edgnet-sync"])
	}
}

func TestARCreate(t *testing.T) {
	g := ARTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creation of Authority request
	t.Run("creation of Authority request", func(t *testing.T) {
		g.edgenetclient.AppsV1alpha().AuthorityRequests().Create(g.authorityRequestObj.DeepCopy())
		g.handler.ObjectCreated(g.authorityRequestObj.DeepCopy())
		authorityRequest, _ := g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		if authorityRequest.Status.Expires == nil {
			t.Errorf(errorDict["auth-timeout"])
		}
	})
	t.Run("checking duplicate Authority names", func(t *testing.T) {
		// Set Authority Request object name equal to existing Authority object name
		g.authorityRequestObj.SetName("edgenet")
		// Create Authority Request
		g.edgenetclient.AppsV1alpha().AuthorityRequests().Create(g.authorityRequestObj.DeepCopy())
		g.handler.ObjectCreated(g.authorityRequestObj.DeepCopy())
		AR, _ := g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		if AR.Status.Message == nil {
			t.Error(errorDict["auth-coll"])
		}
	})

}

func TestARUpdate(t *testing.T) {
	g := ARTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	t.Run("checking duplicate emails and Authority names", func(t *testing.T) {
		// Create Authority request
		g.edgenetclient.AppsV1alpha().AuthorityRequests().Create(g.authorityRequestObj.DeepCopy())
		g.handler.ObjectCreated(g.authorityRequestObj.DeepCopy())
		// Create user
		userHandler := user.Handler{}
		userHandler.Init(g.client, g.edgenetclient)
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
		userHandler.ObjectCreated(g.userObj.DeepCopy())
		// Set contact email in authority request obj equal to user email to cause collision
		g.authorityRequestObj.Spec.Contact.Email = "userName@edge-net.org"
		g.edgenetclient.AppsV1alpha().AuthorityRequests().Update(g.authorityRequestObj.DeepCopy())
		g.handler.ObjectUpdated(g.authorityRequestObj.DeepCopy())
		AR, _ := g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		if AR.Status.Message == nil && strings.Contains(AR.Status.Message[0], "Email address") {
			t.Error(errorDict["email-coll"])
		}
	})
	t.Run("Testing Authority Request transition to Authority", func(t *testing.T) {
		// Updating authority registration status to approved
		g.authorityRequestObj.Spec.Contact.Email = "JohnDoe@edge-net.org"
		g.authorityRequestObj.Spec.Approved = true
		// Updating the authority registration object
		g.edgenetclient.AppsV1alpha().AuthorityRequests().Update(g.authorityRequestObj.DeepCopy())
		// Requesting server to update internal representation of authority registration object and transition it to authority
		g.handler.ObjectUpdated(g.authorityRequestObj.DeepCopy())
		// Checking if authority with same name as authority registration object exists
		g.edgenetclient.AppsV1alpha().AuthorityRequests().List(metav1.ListOptions{})
		authority, _ := g.edgenetclient.AppsV1alpha().Authorities().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		if authority == nil {
			t.Error(errorDict["auth-approv"])
		}
	})
}
