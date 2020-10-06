package team

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type TestGroup struct {
	authorityObj  apps_v1alpha.Authority
	teamObj       apps_v1alpha.Team
	userObj       apps_v1alpha.User
	client        kubernetes.Interface
	edgenetClient versioned.Interface
	handler       Handler
}

func TestMain(m *testing.M) {
	flag.String("dir", "../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *TestGroup) Init() {
	teamObj := apps_v1alpha.Team{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Team",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "team",
			UID:       "UID",
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
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+33NUMBER",
				Username:  "johndoe",
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
			Name:       "johnsmith",
			Namespace:  "authority-edgenet",
			UID:        "TestUID",
			Generation: 1,
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			State: success,
			Type:  "user",
			AUP:   true,
		},
	}
	g.authorityObj = authorityObj
	g.teamObj = teamObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// authorityHandler := authority.Handler{}
	// authorityHandler.Init(g.client, g.edgenetClient)
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", g.authorityObj.GetName())}}
	namespaceLabels := map[string]string{"owner": "authority", "owner-name": g.authorityObj.GetName(), "authority-name": g.authorityObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	// Create a user as admin on authority
	user := apps_v1alpha.User{}
	user.SetName(strings.ToLower(g.authorityObj.Spec.Contact.Username))
	user.Spec.Email = g.authorityObj.Spec.Contact.Email
	user.Spec.FirstName = g.authorityObj.Spec.Contact.FirstName
	user.Spec.LastName = g.authorityObj.Spec.Contact.LastName
	user.Spec.Active = true
	user.Status.AUP = true
	user.Status.Type = "admin"
	g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), user.DeepCopy(), metav1.CreateOptions{})
	// authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

// TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetClient)
	util.Equals(t, g.client, g.handler.clientset)
	util.Equals(t, g.edgenetClient, g.handler.edgenetClientset)
	util.Equals(t, "team-quota", g.handler.resourceQuota.Name)
	util.NotEquals(t, nil, g.handler.resourceQuota.Spec.Hard)
	util.Equals(t, int64(0), g.handler.resourceQuota.Spec.Hard.Pods().Value())
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Create Team
	g.edgenetClient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.teamObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.teamObj.DeepCopy())
	childNamespaceStr := fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName())
	t.Run("namespace", func(t *testing.T) {
		_, err := g.handler.clientset.CoreV1().Namespaces().Get(context.TODO(), childNamespaceStr, metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("role bindings", func(t *testing.T) {
		_, err := g.client.RbacV1().RoleBindings(childNamespaceStr).Get(context.TODO(), fmt.Sprintf("authority-%s-%s-team-%s", g.authorityObj.GetName(), g.authorityObj.Spec.Contact.Username, "admin"), metav1.GetOptions{})
		// Verifying server created rolebinding for admin user in team's child namespace
		util.OK(t, err)
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	// Create Team to update later
	team, _ := g.edgenetClient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.teamObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreated func to create a team
	g.handler.ObjectCreated(g.teamObj.DeepCopy())
	childNamespaceStr := fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName())

	// Add new users to team
	t.Run("add user", func(t *testing.T) {
		team.Spec.Users = []apps_v1alpha.TeamUsers{
			{
				Authority: g.authorityObj.GetName(),
				Username:  g.userObj.GetName(),
			},
		}
		_, err := g.client.RbacV1().RoleBindings(childNamespaceStr).Get(context.TODO(), fmt.Sprintf("%s-%s-team-%s", g.userObj.GetNamespace(), g.userObj.GetName(), "user"), metav1.GetOptions{})
		// Verifying the user is not involved in the beginning
		util.Equals(t, true, errors.IsNotFound(err))
		// Building field parameter
		var field fields
		field.users.status = true
		field.users.added = fmt.Sprintf("`[{\"Authority\": \"%s\", \"Username\": \"%s\" }]`", g.authorityObj.GetName(), g.userObj.GetName())
		// Requesting server to update internal representation of team
		_, err = g.edgenetClient.AppsV1alpha().Teams(team.GetNamespace()).Update(context.TODO(), team.DeepCopy(), metav1.UpdateOptions{})
		util.OK(t, err)

		// Invoking ObjectUpdated to send emails to users removed or added to team
		g.handler.ObjectUpdated(team.DeepCopy(), field)
		// Check user rolebinding in team child namespace
		_, err = g.client.RbacV1().RoleBindings(childNamespaceStr).Get(context.TODO(), fmt.Sprintf("%s-%s-team-%s", g.userObj.GetNamespace(), g.userObj.GetName(), "user"), metav1.GetOptions{})
		// Verifying server created rolebinding for new user in team's child namespace
		util.OK(t, err)
	})
}

func TestGetOwnerReferences(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Creating User before updating requesting server to update internal representation of team
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	g.teamObj.Spec.Users = []apps_v1alpha.TeamUsers{
		{
			Authority: g.authorityObj.GetName(),
			Username:  g.userObj.GetName(),
		},
	}
	g.handler.ObjectCreated(g.teamObj.DeepCopy())

	teamChildNamespaceStr := fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName())
	teamChildNamespace, _ := g.client.CoreV1().Namespaces().Get(context.TODO(), teamChildNamespaceStr, metav1.GetOptions{})
	ownerReferences := g.handler.getOwnerReferences(g.teamObj.DeepCopy(), teamChildNamespace)
	util.NotEquals(t, nil, ownerReferences)
	util.Equals(t, 2, len(ownerReferences))
	util.Equals(t, teamChildNamespaceStr, ownerReferences[0].Name)
	util.Equals(t, g.userObj.GetName(), ownerReferences[1].Name)
}

func TestDelete(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	teamChildNamespaceStr := fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName())
	g.teamObj.Spec.Users = []apps_v1alpha.TeamUsers{
		{
			Authority: g.authorityObj.GetName(),
			Username:  g.userObj.GetName(),
		},
	}
	// Creating team with one user
	g.edgenetClient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.teamObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.teamObj.DeepCopy())
	_, err := g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName()), metav1.GetOptions{})
	util.OK(t, err)

	g.edgenetClient.AppsV1alpha().Teams(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(context.TODO(), g.teamObj.Name, metav1.DeleteOptions{})
	var field fields
	field.users.status, field.enabled, field.users.deleted = true, true, fmt.Sprintf("`[{\"Authority\": \"%s\", \"Username\": \"%s\" }]`", g.authorityObj.GetName(), g.userObj.GetName())
	field.object = objectData{
		name:           g.teamObj.GetName(),
		ownerNamespace: g.teamObj.GetNamespace(),
		childNamespace: teamChildNamespaceStr,
	}
	g.handler.ObjectDeleted(g.teamObj.DeepCopy(), field)

	_, err = g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-team-%s", g.teamObj.GetNamespace(), g.teamObj.GetName()), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
}
