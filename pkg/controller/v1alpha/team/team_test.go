package team

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/v1alpha/authority"
	"github.com/EdgeNet-project/edgenet/pkg/controller/v1alpha/user"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Dictionary for status messages
var errorDict = map[string]string{
	"k8-sync":               "Kubernetes clientset sync problem",
	"edgnet-sync":           "EdgeNet clientset sync problem",
	"quota-name":            "Wrong resource quota name",
	"quota-spec":            "Resource quota spec issue",
	"quota-pod":             "Resource quota allows pod deployment",
	"team-child-nmspce":     "Failed to create team child namespace",
	"team-status":           "Failed to update status of team",
	"team-user-rolebinding": "Failed to create Rolebinding for user in team child namespace",
	"team-get-owner-ref":    "Failed to get owner references",
	"team-set-owner-ref":    "Failed to set team namespace owner references",
	"team-fail":             "Failed to create new team",
	"team-users-del":        "Failed to delete users in team",
	"team-del-child-nmspce": "Failed to delete Team child namespace",
	"add-func":              "Add func of event handler doesn't work properly",
	"upd-func":              "Update func of event handler doesn't work properly",
	"del-func":              "Delete func of event handler doesn't work properly",
}

// Constant variables for events
const success = "Successful"

// The main structure of test group
type TeamTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	teamObj       apps_v1alpha.Team
	userObj       apps_v1alpha.User
	client        kubernetes.Interface
	edgenetclient versioned.Interface
	handler       Handler
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
			Enabled:     true,
		},
		Status: apps_v1alpha.TeamStatus{
			State: success,
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
			Type:  "Admin",
			State: success,
		},
	}
	g.authorityObj = authorityObj
	g.teamObj = teamObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreated to create namespace
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
		t.Error(errorDict["k8-sync"])
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error(errorDict["edgenet-sync"])
	}
	if g.handler.resourceQuota.Name != "team-quota" {
		t.Error(errorDict["quota-name"])
	}
	if g.handler.resourceQuota.Spec.Hard == nil {
		t.Error(errorDict["quota-spec"])
	} else {
		if g.handler.resourceQuota.Spec.Hard.Pods().Value() != 0 {
			t.Error(errorDict["quota-pod"])
		}
	}
}

func TestTeamCreate(t *testing.T) {
	g := TeamTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Create Team
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.teamObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.teamObj.DeepCopy())
	// Creation of Team
	t.Run("creation of Team", func(t *testing.T) {
		teamChildNamespace, _ := g.handler.clientset.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName()), metav1.GetOptions{})
		if teamChildNamespace == nil {
			t.Error(errorDict["team-child-nmspce"])
		}
	})
}

func TestTeamUpdate(t *testing.T) {
	g := TeamTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	userHandler := user.Handler{}
	userHandler.Init(g.client, g.edgenetclient)
	// Create Team to update later
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.teamObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreated func to create a team
	g.handler.ObjectCreated(g.teamObj.DeepCopy())
	// Update of team status
	t.Run("Update existing team", func(t *testing.T) {
		// Building field parameter
		g.teamObj.Spec.Enabled = false
		var field fields
		field.enabled = true
		// Requesting server to Update internal representation of team
		g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(context.TODO(), g.teamObj.DeepCopy(), metav1.UpdateOptions{})
		// Invoking ObjectUpdated to send emails to users added or removed from team
		g.handler.ObjectUpdated(g.teamObj.DeepCopy(), field)
		// Verifying Team status is enabled in server's representation of team
		team, _ := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.teamObj.GetName(), metav1.GetOptions{})
		if team.Spec.Enabled {
			t.Error("Failed to update status of team")
		}
		// Re-enable team for futher tests
		g.teamObj.Spec.Enabled = true
		// Requesting server to Update internal representation of team
		g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(context.TODO(), g.teamObj.DeepCopy(), metav1.UpdateOptions{})
		// Invoking ObjectUpdated to send emails to users added or removed from team
		g.handler.ObjectUpdated(g.teamObj.DeepCopy(), field)
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
		g.userObj.Spec.Active, g.userObj.Status.AUP = true, true
		// Creating User before updating requesting server to update internal representation of team
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		userHandler.ObjectCreated(g.userObj.DeepCopy())
		// Requesting server to update internal representation of team
		g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(context.TODO(), g.teamObj.DeepCopy(), metav1.UpdateOptions{})
		// Invoking ObjectUpdated to send emails to users removed or added to team
		g.handler.ObjectUpdated(g.teamObj.DeepCopy(), field)
		// Check user rolebinding in team child namespace
		user, _ := g.handler.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), "user1", metav1.GetOptions{})
		roleBindings, _ := g.client.RbacV1().RoleBindings(fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName())).Get(context.TODO(), fmt.Sprintf("%s-%s-team-%s", user.GetNamespace(), user.GetName(), "user"), metav1.GetOptions{})
		// Verifying server created rolebinding for new user in team's child namespace
		if roleBindings == nil {
			t.Error(errorDict["team-user-rolebinding"])
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
	g.userObj.Spec.Active, g.userObj.Status.AUP = true, true
	// Creating User before updating requesting server to update internal representation of team
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	// Creating team with one user
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.teamObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.teamObj.DeepCopy())
	// Setting owner references
	t.Run("Set Owner references", func(t *testing.T) {
		teamChildNamespaceStr := fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName())
		teamChildNamespace, _ := g.client.CoreV1().Namespaces().Get(context.TODO(), teamChildNamespaceStr, metav1.GetOptions{})
		if g.handler.getOwnerReferences(g.teamObj.DeepCopy(), teamChildNamespace) == nil {
			t.Error(errorDict["team-get-owner-ref"])
		}
		// Verifying team owns child namespace
		if teamChildNamespace.Labels["owner"] != "team" && teamChildNamespace.Labels["owner-name"] != "edgnetteam" {
			t.Error(errorDict["team-set-owner-ref"])
		}

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
	g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.teamObj.DeepCopy(), metav1.CreateOptions{})
	// Sanity check for team creation
	team, _ := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.teamObj.GetName(), metav1.GetOptions{})
	if team == nil {
		t.Error(errorDict["team-fail"])
	}
	// Deleting team
	t.Run("Delete team", func(t *testing.T) {
		// Requesting server to delete internal representation of team
		g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(context.TODO(), g.teamObj.Name, metav1.DeleteOptions{})
		// Building field parameter
		var field fields
		field.users.status, field.enabled, field.users.deleted = true, true, `[{"Authority": "edgenet", "Username": "user1" }]`
		// Invoking ObjectDeleted to send emails to users removed from deleted team
		g.handler.ObjectDeleted(g.teamObj.DeepCopy(), field)
		// Verifying server no longer has internal representation of team
		team, _ := g.edgenetclient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.teamObj.GetName(), metav1.GetOptions{})
		if team != nil {
			if len(team.Spec.Users) != 0 {
				t.Error(errorDict["team-users-del"])
			}
			t.Error("Failed to delete new test team")
		}
		teamChildNamespace, _ := g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName()), metav1.GetOptions{})
		if teamChildNamespace != nil {
			t.Error(errorDict["team-del-child-nmspce"])
		}
	})
}
