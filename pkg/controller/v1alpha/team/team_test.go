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
	"edgenet/pkg/controller/v1alpha/authority"

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
	authorityObj        apps_v1alpha.Authority
	teamObj             apps_v1alpha.Team
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
	g.teamObj = teamObj
	g.authorityRequestObj = authorityRequestObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	//invoke ObjectCreated to create namespace
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
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
	// Create Team
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.teamObj.DeepCopy())
	// Creation of Team
	t.Run("creation of Team", func(t *testing.T) {
		team, err := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.teamObj.GetName(), metav1.GetOptions{})
		if team == nil {
			t.Error("Failed to create new Team when an authority created")
		}
		if err != nil {
			t.Errorf("Failed to create new team, %v", err)
		}
		g.handler.ObjectCreated(g.teamObj.DeepCopy())
	})
	t.Run("check duplicate object", func(t *testing.T) {
		// Create two teams with the same name
		team, _ := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.teamObj.DeepCopy())
		if team != nil {
			t.Error("Duplicate value cannot be detected")
		}
	})
}

func TestTeamUpdate(t *testing.T) {
	g := TeamTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Create Team to update later
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.teamObj.DeepCopy())
	// Invoke ObjectCreated func to create a team
	g.handler.ObjectCreated(g.teamObj.DeepCopy())
	// Update of team status
	t.Run("Update existing team", func(t *testing.T) {
		// Building field parameter
		g.teamObj.Status.Enabled = true
		var field fields
		field.enabled = true
		// Requesting server to Update internal representation of team
		g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(g.teamObj.DeepCopy())
		// Invoking ObjectUpdated to send emails to users added or removed from team
		g.handler.ObjectUpdated(g.teamObj.DeepCopy(), field)
		// Verifying Team status is enabled in server's representation of team
		team, _ := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.teamObj.GetName(), metav1.GetOptions{})
		if !team.Status.Enabled {
			t.Error("Failed to update status of team")
		}
	})
	// Add new users to team
	t.Run("Add new users to team", func(t *testing.T) {
		g.teamObj.Spec.Users = []apps_v1alpha.TeamUsers{
			apps_v1alpha.TeamUsers{
				Authority: g.authorityObj.GetName(),
				Username:  "user1",
			},
		}
		// Building field parameter
		var field fields
		field.users.status, field.enabled = true, true
		field.users.added = `[{"Authority": "edgenet", "Username": "user1" }]`
		g.userObj.Status.Active, g.userObj.Status.AUP = true, true
		// Creating User before updating requesting server to update internal representation of team
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
		// Requesting server to update internal representation of team
		g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(g.teamObj.DeepCopy())
		// Invoking ObjectUpdated to send emails to users removed or added to team
		g.handler.ObjectUpdated(g.teamObj.DeepCopy(), field)
		// Verifying server's representation of team contains users
		team, _ := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.teamObj.GetName(), metav1.GetOptions{})
		if len(team.Spec.Users) != 1 {
			t.Error("Failed to add user to team")
		}
	})
}

func TestTeamUserOwnerReferences(t *testing.T) {
	g := TeamTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	g.teamObj.Spec.Users = []apps_v1alpha.TeamUsers{
		apps_v1alpha.TeamUsers{
			Authority: g.authorityObj.GetName(),
			Username:  "user1",
		},
	}
	g.userObj.Status.Active, g.userObj.Status.AUP = true, true
	// Creating User before updating requesting server to update internal representation of team
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
	// Creating team with one user
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.teamObj.DeepCopy())
	// Sanity check team created
	team, _ := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.teamObj.GetName(), metav1.GetOptions{})
	if team == nil {
		t.Error("Failed to create new team")
	}
	// Setting owner references
	t.Run("Set Owner references", func(t *testing.T) {
		g.handler.setOwnerReferences(team)
	})
}

func TestTeamDelete(t *testing.T) {
	g := TeamTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	g.teamObj.Spec.Users = []apps_v1alpha.TeamUsers{
		apps_v1alpha.TeamUsers{
			Authority: g.authorityObj.GetName(),
			Username:  "user1",
		},
	}
	// Creating team with one user
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.teamObj.DeepCopy())
	// Sanity check for team creation
	team, _ := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.teamObj.GetName(), metav1.GetOptions{})
	if team == nil {
		t.Error("Failed to create new team")
	}
	// Deleting team
	t.Run("Delete team", func(t *testing.T) {
		// Requesting server to delete internal representation of team
		g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(g.teamObj.Name, &metav1.DeleteOptions{})
		// Building field parameter
		var field fields
		field.users.status, field.enabled, field.users.deleted = true, true, `[{"Authority": "edgenet", "Username": "user1" }]`
		// Invoking ObjectDeleted to send emails to users removed from deleted team
		g.handler.ObjectDeleted(g.teamObj.DeepCopy(), field)
		// Verifying server no longer has internal representation of team
		team, _ := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.teamObj.GetName(), metav1.GetOptions{})
		if team != nil {
			if len(team.Spec.Users) != 0 {
				t.Error("Failed to delete users in team")
			}
			t.Error("Failed to delete new test team")
		}
	})
}
