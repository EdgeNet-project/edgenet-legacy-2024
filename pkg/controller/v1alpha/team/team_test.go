package team

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"

	"github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Constant variables for events
const failure = "Failure"
const success = "Successful"
const established = "Established"

// The main structure of test group
type TeamTestGroup struct {
	authorityObj        apps_v1alpha.Team
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
func (g *TeamTestGroup) Init() {
	teamObj := apps_v1alpha.Team{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Authority",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenet",
			Namespace: "TeamObj",
		},
		Spec: apps_v1alpha.TeamSpec{
			Users: []apps_v1alpha.TeamUsers{
				apps_v1alpha.TeamUsers{
					Authority: "edgenet",
					Username:  "username",
				},
			},
			Description: "This is a test description",
		},
		Status: apps_v1alpha.TeamStatus{
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
			Name: "unittesting",
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
	g.authorityObj = teamObj
	g.authorityRequestObj = authorityRequestObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
}

// TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TeamTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error("Kubernetes clientset sync problem")
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error("EdgeNet clientset sync problem")
	}
	if g.handler.resourceQuota.Name != "team-quota" {
		t.Error("Wrong resource quota name")
	}
	if g.handler.resourceQuota.Spec.Hard == nil {
		t.Error("Resource quota spec issue")
	} else {
		if g.handler.resourceQuota.Spec.Hard.Pods().Value() != 0 {
			t.Error("Resource quota allows pod deployment")
		}
	}
}

func TestTeamCreate(t *testing.T) {
	g := TeamTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	t.Run("creation of user-total resource quota-cluster role", func(t *testing.T) {
		g.handler.ObjectCreated(g.authorityObj.DeepCopy())
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Users[0].Username, metav1.GetOptions{})
		if user == nil {
			t.Error("User generation failed when an authority created")
		}

		TRQ, _ := g.handler.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Get(g.authorityObj.GetName(), metav1.GetOptions{})
		if TRQ == nil {
			t.Error("Total resource quota cannot be created")
		}

		clusterRole, _ := g.handler.clientset.RbacV1().ClusterRoles().Get(fmt.Sprintf("authority-%s", g.authorityObj.GetName()), metav1.GetOptions{})
		if clusterRole == nil {
			t.Error("Cluster role cannot be created")
		}
	})

	t.Run("check dublicate object", func(t *testing.T) {
		// Change the authority object name to make comparison with the user-created above
		g.authorityObj.Name = "different"
		g.handler.ObjectCreated(g.authorityObj.DeepCopy())
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Users[0].Username, metav1.GetOptions{})
		if user != nil {
			t.Error("Duplicate value cannot be detected")
		}
	})
}

func TestTeamUpdate(t *testing.T) {
	g := TeamTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Create an authority to update later
	g.edgenetclient.AppsV1alpha().Teams("TeamObj").Create(g.authorityObj.DeepCopy())
	var field fields
	field.enabled = false
	field.users.status = false
	field.users.deleted = ""
	field.users.added = ""
	field.object.name = "TestName"
	field.object.ownerNamespace = ""
	field.object.childNamespace = ""
	// Invoke ObjectCreated func to create a user
	g.handler.ObjectCreated(g.authorityObj.DeepCopy())
	// Create another user
	g.userObj.Spec.Email = "check"
	g.edgenetclient.AppsV1alpha().Users("default").Create(g.userObj.DeepCopy())
	// Use the same email address with the user created above
	g.authorityObj.Status.Enabled = true
	g.handler.ObjectUpdated(g.authorityObj.DeepCopy(), field)
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Users[0].Username, metav1.GetOptions{})
	if user.Spec.Email == "check" {
		t.Error("Duplicate value cannot be detected")
	}
	if user.Status.Active {
		t.Error("User cannot be deactivated")
	}
}
