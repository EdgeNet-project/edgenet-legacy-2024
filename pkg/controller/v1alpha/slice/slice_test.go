package slice

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

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
	"k8-sync":                "Kubernetes clientset sync problem",
	"edgnet-sync":            "EdgeNet clientset sync problem",
	"quota-name":             "Wrong resource quota name",
	"quota-spec":             "Resource quota spec issue",
	"quota-pod":              "Resource quota allows pod deployment",
	"slice-child-nmspce":     "Failed to create slice child namespace",
	"slice-quota":            "Failed to create slice resource quota",
	"slice-prof":             "Failed to update profile of slice",
	"slice-exp":              "Failed to update expiration time of slice",
	"slice-user-rolebinding": "Failed to create Rolebinding for user in slice child namespace",
	"add-func":               "Add func of event handler doesn't work properly",
	"upd-func":               "Update func of event handler doesn't work properly",
	"del-func":               "Delete func of event handler doesn't work properly",
}

// Constant variables for events
const success = "Successful"

// The main structure of test group
type SliceTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	userObj       apps_v1alpha.User
	sliceObj      apps_v1alpha.Slice
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
func (g *SliceTestGroup) Init() {
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
			Type:  "admin",
			State: success,
		},
	}
	sliceObj := apps_v1alpha.Slice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Slice",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "Slice1",
			Namespace:       "authority-edgenet",
			OwnerReferences: []metav1.OwnerReference{},
		},
		Spec: apps_v1alpha.SliceSpec{
			Profile:     "Low",
			Users:       []apps_v1alpha.SliceUsers{},
			Description: "This is a test description",
			Renew:       true,
		},
		Status: apps_v1alpha.SliceStatus{
			Expires: nil,
		},
	}
	g.authorityObj = authorityObj
	g.userObj = userObj
	g.sliceObj = sliceObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreated to create namespace
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := SliceTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error(errorDict["k8-sync"])
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error(errorDict["edgenet-sync"])
	}
	if g.handler.lowResourceQuota.Name != "slice-low-quota" || g.handler.medResourceQuota.Name != "slice-medium-quota" || g.handler.highResourceQuota.Name != "slice-high-quota" {
		t.Error(errorDict["quota-name"])
	}
	if g.handler.lowResourceQuota.Spec.Hard == nil || g.handler.medResourceQuota.Spec.Hard == nil || g.handler.highResourceQuota.Spec.Hard == nil {
		t.Error(errorDict["quota-spec"])
	} else {
		if g.handler.highResourceQuota.Spec.Hard.Pods().Value() != 0 {
			t.Error(errorDict["quota-pod"])
		}
	}
}

func TestSliceCreate(t *testing.T) {
	g := SliceTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Create Slice
	g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.sliceObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.sliceObj.DeepCopy())
	// Creation of slice
	t.Run("creation of slice", func(t *testing.T) {
		sliceChildNamespace, _ := g.handler.clientset.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName()), metav1.GetOptions{})
		if sliceChildNamespace == nil {
			t.Error(errorDict["slice-child-nmspce"])
		}
		resourceQuota, _ := g.client.CoreV1().ResourceQuotas(sliceChildNamespace.GetName()).List(context.TODO(), metav1.ListOptions{})
		if resourceQuota == nil {
			t.Error(errorDict["slice-quota"])
		}
	})
}

func TestSliceUpdate(t *testing.T) {
	g := SliceTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	userHandler := user.Handler{}
	userHandler.Init(g.client, g.edgenetclient)
	// Create slice to update later
	g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.sliceObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreated func to create a slice
	g.handler.ObjectCreated(g.sliceObj.DeepCopy())
	t.Run("Update existing slice profile ", func(t *testing.T) {
		// Building field parameter
		g.sliceObj.Spec.Profile = "Medium"
		var field fields
		field.profile.status, field.profile.old = true, "Low"
		// Requesting server to Update internal representation of slice
		g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(context.TODO(), g.sliceObj.DeepCopy(), metav1.UpdateOptions{})
		// Invoking ObjectUpdated to update slice resource quota
		g.handler.ObjectUpdated(g.sliceObj.DeepCopy(), field)
		// Verifying slice expiration time is updated in server's representation of slice
		slice, _ := g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.sliceObj.GetName(), metav1.GetOptions{})
		if slice.Spec.Profile != "Medium" {
			t.Error(errorDict["slice-prof"])
		}
		if slice.Status.Expires == nil {
			t.Error(errorDict["slice-exp"])
		}
		testTime := &metav1.Time{
			Time: time.Now().Add(672 * time.Hour),
		}
		yy1, mm1, dd1 := slice.Status.Expires.Date()
		yy2, mm2, dd2 := testTime.Date()
		if yy1 != yy2 && mm1 != mm2 && dd1 != dd2 {
			t.Error(errorDict["slice-exp"])
		}
	})
	t.Run("Add users to slice ", func(t *testing.T) {
		g.sliceObj.Spec.Users = []apps_v1alpha.SliceUsers{
			apps_v1alpha.SliceUsers{
				Authority: g.authorityObj.GetName(),
				Username:  "user1",
			},
		}
		// Building field parameter
		var field fields
		field.users.status = true
		field.users.added = `[{"Authority": "edgenet", "Username": "user1" }]`
		g.userObj.Spec.Active, g.userObj.Status.AUP = true, true
		// Creating User before updating requesting server to update internal representation of slice
		g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		userHandler.ObjectCreated(g.userObj.DeepCopy())
		// Requesting server to update internal representation of slice
		g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(context.TODO(), g.sliceObj.DeepCopy(), metav1.UpdateOptions{})
		// Invoking ObjectUpdated to send emails to users removed or added to slice
		g.handler.ObjectUpdated(g.sliceObj.DeepCopy(), field)
		// Check user rolebinding in slice child namespace
		user, _ := g.handler.edgenetClientset.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), "user1", metav1.GetOptions{})
		roleBindings, _ := g.client.RbacV1().RoleBindings(fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName())).Get(context.TODO(), fmt.Sprintf("%s-%s-slice-%s", user.GetNamespace(), user.GetName(), "user"), metav1.GetOptions{})
		// Verifying server created rolebinding for new user in slice's child namespace
		if roleBindings == nil {
			t.Error(errorDict["slice-user-rolebinding"])
		}
	})
}
