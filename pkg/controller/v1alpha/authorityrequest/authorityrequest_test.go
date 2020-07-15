package authorityrequest

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"edgenet/pkg/controller/v1alpha/authority"
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
type ARTestGroup struct {
	authorityObj        apps_v1alpha.Authority
	authorityRequestObj apps_v1alpha.AuthorityRequest
	teamObj             apps_v1alpha.Team
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
	teamObj := apps_v1alpha.Team{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Team",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenetteam",
			UID:       "edgenetteamUID",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.TeamSpec{
			Users:       []apps_v1alpha.TeamUsers{},
			Description: "This is a test description",
		},
		Status: apps_v1alpha.TeamStatus{
			Enabled: false,
		},
	}
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
	AuthorityRequestObj := apps_v1alpha.AuthorityRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "authorityRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "authority request name",
			// Namespace: "authority-edgenet",
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
		},
		Status: apps_v1alpha.AuthorityRequestStatus{
			EmailVerify: true,
			Approved:    true,
			Expires:     nil,
			State:       "",
			Message:     nil,
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
			Roles:     []string{"Admin"},
			Email:     "userName@edge-net.org",
		},
		Status: apps_v1alpha.UserStatus{
			State:  success,
			Active: true,
		},
	}
	g.authorityObj = authorityObj
	g.authorityRequestObj = AuthorityRequestObj
	g.teamObj = teamObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	//invoke ObjectCreated to create namespace
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
	g.authorityObj.SetName("edgenet2")
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := ARTestGroup{}
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

func TestARCreate(t *testing.T) {
	g := ARTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creation of Authority request

	// Creation of Authority request
	t.Run("creation of Authority request", func(t *testing.T) {
		g.edgenetclient.AppsV1alpha().AuthorityRequests().Create(g.authorityRequestObj.DeepCopy())
		g.handler.ObjectCreated(g.authorityRequestObj.DeepCopy())
		authorityRequest, _ := g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		t.Log(authorityRequest.Status.Expires)
		if authorityRequest.Status.Expires == nil {
			t.Errorf("Failed to update approval timeout of Authority Request")
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
			t.Error("Failed to detect Authority name collision")
		}
		// t.Log(g.edgenetclient.AppsV1alpha().AuthorityRequests().List(metav1.ListOptions{}))
		// g.authorityRequestObj.SetName("authorityrequest1")
		// // Create authority request
		// g.edgenetclient.AppsV1alpha().AuthorityRequests().Create(g.authorityRequestObj.DeepCopy())
		// g.handler.ObjectCreated(g.authorityRequestObj.DeepCopy())
		// t.Logf("metadata.name==%s", g.authorityRequestObj.GetName())
		// authorityRaw, _ := g.edgenetclient.AppsV1alpha().Authorities().List(
		// 	metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", g.authorityRequestObj.GetName())})
		// t.Log(authorityRaw)
		// t.Log(g.edgenetclient.AppsV1alpha().Authorities().List(metav1.ListOptions{}))
		// t.Log(g.edgenetclient.AppsV1alpha().AuthorityRequests().List(metav1.ListOptions{}))
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

		authorityRaw, _ := g.edgenetclient.AppsV1alpha().Authorities().List(
			metav1.ListOptions{FieldSelector: "metadata.name==jibberish"})
		// should be empty
		t.Log(authorityRaw)

		// // Create user
		// userHandler := user.Handler{}
		// userHandler.Init(g.client, g.edgenetclient)
		// g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
		// userHandler.ObjectCreated(g.userObj.DeepCopy())
		// // Update User in authority request to cause collision
		// g.authorityRequestObj.Spec.Contact.Email = "userName@edge-net.org"
		// AR, err := g.edgenetclient.AppsV1alpha().AuthorityRequests().Update(g.authorityRequestObj.DeepCopy())
		// t.Log(AR, err)
		// g.handler.ObjectUpdated(g.authorityRequestObj.DeepCopy())
		// AR, err = g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		// t.Log(AR, err)
		// if AR.Status.Message == nil {
		// 	t.Error("Failed to detect Authority name collision")
		// }

		// g.authorityRequestObj.SetName("AuthorityRequest")

		// t.Log(AR.Status.Message)
		// // Checking duplicate emails and authority names

		// // AR, err := g.edgenetclient.AppsV1alpha().AuthorityRequests().Update(g.authorityRequestObj.DeepCopy())
		// // t.Log(AR, err)

		// // authorityRequest, _ := g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		// // if authorityRequest.Status.Expires == nil {
		// // 	t.Errorf("Failed to update approval timeout of authority request")
		// // }
	})
}
