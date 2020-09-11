package user

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/v1alpha/authority"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Dictionary for error messages
var errorDict = map[string]string{
	"k8-sync":                 "Kubernetes clientset sync problem",
	"edgnet-sync":             "EdgeNet clientset sync problem",
	"dupl-val":                "Duplicate value cannot be detected",
	"user-gen":                "User generation failed when an authority created",
	"user-role":               "User role cannot be created",
	"user-rolebinding":        "RoleBinding Creation failed",
	"user-rolebinding-delete": "RoleBinding deletion failed",
	"user-deact":              "User cannot be deactivated",
	"user-email":              "Updating user email failed",
	"user-active":             "User is still Active after changing its email",
	"user-create":             "User creation failed by Create function which can be used by other resources",
	"AUP-binding":             "Create AUPRolebinding failed",
	"add-func":                "Add func of event handler doesn't work properly",
	"upd-func":                "Update func of event handler doesn't work properly",
	"del-func":                "Delete func of event handler doesn't work properly",
}

//The main structure of test group
type UserTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	teamList      apps_v1alpha.TeamList
	sliceList     apps_v1alpha.SliceList
	userObj       apps_v1alpha.User
	urrObj        apps_v1alpha.UserRegistrationRequest
	client        kubernetes.Interface
	edgenetclient versioned.Interface
	handler       Handler
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
			Enabled: true,
		},
	}
	teamList := apps_v1alpha.TeamList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TeamList",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ListMeta: metav1.ListMeta{
			SelfLink:        "teamSelfLink",
			ResourceVersion: "1",
		},
		Items: []apps_v1alpha.Team{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Team",
					APIVersion: "apps.edgenet.io/v1alpha",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "edgenetTeam",
				},
				Spec: apps_v1alpha.TeamSpec{
					Users: []apps_v1alpha.TeamUsers{
						{
							Authority: "authority-edgenet",
							Username:  "unittestingTeamObj",
						},
					},
					Description: "This is a Teamtest description",
					Enabled:     true,
				},
				Status: apps_v1alpha.TeamStatus{
					State: success,
				},
			},
		},
	}
	sliceList := apps_v1alpha.SliceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SliceList",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ListMeta: metav1.ListMeta{
			SelfLink:        "sliceSelfLink",
			ResourceVersion: "1",
		},
		Items: []apps_v1alpha.Slice{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Slice",
					APIVersion: "apps.edgenet.io/v1alpha",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "edgenetSlice",
				},
				Spec: apps_v1alpha.SliceSpec{
					Users: []apps_v1alpha.SliceUsers{
						{
							Authority: "authority-edgenet",
							Username:  "unittestingSliceObj",
						},
					},
					Description: "This is a Slicetest description",
				},
			},
		},
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "unittestingObj",
			Namespace:  "authority-edgenet",
			UID:        "TestUID",
			Generation: 1,
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "EdgeNetFirstName",
			LastName:  "EdgeNetLastName",
			Email:     "userObj@email.com",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			State: success,
			Type:  "Admin",
		},
	}
	urrObj := apps_v1alpha.UserRegistrationRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRegistrationRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "urrName",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserRegistrationRequestSpec{
			Bio:       "urrBio",
			Email:     "urrEmail",
			FirstName: "URRFirstName",
			LastName:  "URRLastname",
			URL:       "",
		},
		Status: apps_v1alpha.UserRegistrationRequestStatus{
			EmailVerified: false,
		},
	}
	g.authorityObj = authorityObj
	g.teamList = teamList
	g.sliceList = sliceList
	g.userObj = userObj
	g.urrObj = urrObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	// Invoke authority ObjectCreated to create namespace
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	g.edgenetclient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

// TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := UserTestGroup{}
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

func TestUserCreate(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	t.Run("Creation of user from authority", func(t *testing.T) {
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
		if user == nil {
			t.Error(errorDict["user-gen"])
		}
	})
	t.Run("Creation of user", func(t *testing.T) {
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userObj.GetName(), metav1.GetOptions{})
		user.Spec.Active = true
		var field fields
		field.active = true
		g.handler.ObjectUpdated(user.DeepCopy(), field)
		currentUserRole, _ := g.handler.clientset.RbacV1().Roles(user.GetNamespace()).Get(context.TODO(), fmt.Sprintf("user-%s", user.GetName()), metav1.GetOptions{})
		if currentUserRole == nil {
			t.Error(errorDict["user-role"])
		}
		roleBinding, _ := g.handler.clientset.RbacV1().RoleBindings(user.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-user-aup-%s", user.GetNamespace(), user.GetName()), metav1.GetOptions{})
		if roleBinding == nil {
			t.Error(errorDict["user-rolebinding"])
		}
	})
	t.Run("Check dublicate object", func(t *testing.T) {
		// Change the user object name to make comparison with the user-created above
		g.userObj.Name = "different"
		g.userObj.UID = "differentUID"
		// Create user
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		// Invoke the ObjectCreated()
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		// Check if the user created successfully or not
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userObj.Name, metav1.GetOptions{})
		if user.Status.Message == nil {
			t.Error(errorDict["dupl-val"])
		}
	})
}

func TestUserUpdate(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	t.Run("Updating Email and Checking the Duplication", func(t *testing.T) {
		// Creating a user
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		// Changing the user email as the same as the authority default created user
		g.userObj.Spec.Email = "unittest@edge-net.org"
		var field fields
		g.handler.ObjectUpdated(g.userObj.DeepCopy(), field)
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userObj.Name, metav1.GetOptions{})
		if user.Status.Message == nil {
			t.Error(errorDict["dupl-val"])
		}
		if user.Spec.Active {
			t.Error(errorDict["user-deact"])
		}
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(context.TODO(), g.userObj.Name, metav1.DeleteOptions{})
	})
	t.Run("Updating Email", func(t *testing.T) {
		// Creating a user
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.userObj.DeepCopy())
		// Updateing the email
		g.userObj.Spec.Email = "NewUserObj@email.com"
		var field fields
		field.email = true
		g.handler.ObjectUpdated(g.userObj.DeepCopy(), field)
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userObj.Name, metav1.GetOptions{})
		if user.Spec.Email != "NewUserObj@email.com" {
			t.Error(errorDict["user-rolebinding"])
		}
		if user.Spec.Active != false {
			t.Error(errorDict["user-active"])
		}
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(context.TODO(), g.userObj.Name, metav1.DeleteOptions{})
	})
}

func TestCreate(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	result := g.handler.Create(g.urrObj.DeepCopy())
	if result {
		t.Errorf(errorDict["user-create"])
	}
}

func TestCreateRoleBindings(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creating User from userObj
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.userObj.DeepCopy())
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userObj.Name, metav1.GetOptions{})
	// Invoking createRoleBindings
	g.handler.createRoleBindings(user.DeepCopy(), g.sliceList.DeepCopy(), g.teamList.DeepCopy(), fmt.Sprintf("authority-%s", g.authorityObj.GetName()))
	// Check the creation of use role Binding
	roleBindings, _ := g.handler.clientset.RbacV1().RoleBindings(user.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-user-%s", user.GetNamespace(), user.GetName()), metav1.GetOptions{})
	if roleBindings == nil {
		t.Error(errorDict["user-rolebinding"])
	}
}

func TestDeleteRoleBindings(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creating User from userObj
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.userObj.DeepCopy())
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userObj.Name, metav1.GetOptions{})
	// Invoking createRoleBindings
	g.handler.createRoleBindings(user, g.sliceList.DeepCopy(), g.teamList.DeepCopy(), fmt.Sprintf("authority-%s", g.authorityObj.GetName()))
	// Check the creation of use role Binding
	roleBindings, _ := g.handler.clientset.RbacV1().RoleBindings(user.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-user-%s", user.GetNamespace(), user.GetName()), metav1.GetOptions{})
	if roleBindings == nil {
		t.Error(errorDict["user-rolebinding"])
	}
	g.handler.deleteRoleBindings(user, g.sliceList.DeepCopy(), g.teamList.DeepCopy())
	roleBindingsResult, _ := g.handler.clientset.RbacV1().RoleBindings(user.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-user-aup-%s", user.GetNamespace(), user.GetName()), metav1.GetOptions{})
	if roleBindingsResult != nil {
		t.Error(errorDict["user-rolebinding-delete"])
	}
}

func TestCreateAUPRoleBinding(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.userObj.DeepCopy())
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userObj.GetName(), metav1.GetOptions{})
	err := g.handler.createAUPRoleBinding(user)
	if err != nil {
		t.Error(errorDict["AUP-binding"])
	}
}
