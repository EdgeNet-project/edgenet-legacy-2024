package authority

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"

	"github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Dictionary for status messages
var errorDict = map[string]string{
	"k8-sync":              "Kubernetes clientset sync problem",
	"edgnet-sync":          "EdgeNet clientset sync problem",
	"quota-name":           "Wrong resource quota name",
	"quota-spec":           "Resource quota spec issue",
	"quota-pod":            "Resource quota allows pod deployment",
	"user-gen":             "User generation failed when an authority created",
	"TRQ-create":           "Total resource quota cannot be created",
	"cluster-role":         "Cluster role cannot be created",
	"dupl-val":             "Duplicate value cannot be detected",
	"user-deact":           "User cannot be deactivated",
	"auth-name-coll":       "Authority creation is broken by an authority request due to the name is the same",
	"auth-req-coll":        "Authority request having same name still exists",
	"auht-email-coll":      "Authority request having an admin with same email address still exists",
	"auth-info-del":        "Authority request with different information has been deleted",
	"user-coll-failed":     "User having same email address cannot be detected",
	"user-conflict":        "User with different information has created a conflict",
	"auth-create-failed":   "Authority cannot be created properly",
	"auth-est-failed":      "Authority establishment failed",
	"auth-disbld-creation": "Authority is disabled after creation",
	"add-func":             "Add func of event handler doesn't work properly",
	"upd-func":             "Update func of event handler doesn't work properly",
	"del-func":             "Delete func of event handler doesn't work properly",
}

// The main structure of test group
type AuthorityTestGroup struct {
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
func (g *AuthorityTestGroup) Init() {
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
			Email:     "unittest@edge-net.org",
			Active:    true,
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
}

// TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := AuthorityTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error(errorDict["k8-sync"])
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error(errorDict["edgenet-sync"])
	}
	if g.handler.resourceQuota.Name != "authority-quota" {
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

func TestAuthorityCreate(t *testing.T) {
	g := AuthorityTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	t.Run("creation of user, total resource quota, cluster role", func(t *testing.T) {
		g.handler.ObjectCreated(g.authorityObj.DeepCopy())
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
		if user == nil {
			t.Error(errorDict["user-gen"])
		}

		TRQ, _ := g.handler.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Get(g.authorityObj.GetName(), metav1.GetOptions{})
		if TRQ == nil {
			t.Error(errorDict["TRQ-create"])
		}

		clusterRole, _ := g.handler.clientset.RbacV1().ClusterRoles().Get(fmt.Sprintf("authority-%s", g.authorityObj.GetName()), metav1.GetOptions{})
		if clusterRole == nil {
			t.Error(errorDict["cluster-role"])
		}
	})

	t.Run("check dublicate object", func(t *testing.T) {
		// Change the authority object name to make comparison with the user-created above
		g.authorityObj.Name = "different"
		g.handler.ObjectCreated(g.authorityObj.DeepCopy())
		user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
		if user != nil {
			t.Error(errorDict["dupl-val"])
		}
	})
}

func TestAuthorityUpdate(t *testing.T) {
	g := AuthorityTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Create an authority to update later
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	// Invoke ObjectCreated func to create a user
	g.handler.ObjectCreated(g.authorityObj.DeepCopy())
	// Create another user
	g.userObj.Spec.Email = "check"
	g.edgenetclient.AppsV1alpha().Users("default").Create(g.userObj.DeepCopy())
	// Use the same email address with the user created above
	g.authorityObj.Spec.Contact.Email = "check"
	g.authorityObj.Spec.Enabled = true
	g.handler.ObjectUpdated(g.authorityObj.DeepCopy())
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	if user.Spec.Email == "check" {
		t.Error(errorDict["dupl-val"])
	}
	if user.Spec.Active {
		t.Error(errorDict["user-deact"])
	}
}

func TestDuplicateValue(t *testing.T) {
	g := AuthorityTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	t.Run("authority request: same name", func(t *testing.T) {
		// Create an authority request for comparison
		g.edgenetclient.AppsV1alpha().AuthorityRequests().Create(g.authorityRequestObj.DeepCopy())
		exists, _ := g.handler.checkDuplicateObject(g.authorityObj.DeepCopy())
		if exists == true {
			t.Error(errorDict["auth-name-coll"])
		}
		// Check if the authority request exists
		authorityRequest, _ := g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		if authorityRequest != nil {
			t.Error(errorDict["auth-req-coll"])
			g.edgenetclient.AppsV1alpha().AuthorityRequests().Delete(g.authorityRequestObj.GetName(), &metav1.DeleteOptions{})
		}
	})
	t.Run("authority request: same email address", func(t *testing.T) {
		g.authorityRequestObj.Name = "different"
		g.edgenetclient.AppsV1alpha().AuthorityRequests().Create(g.authorityRequestObj.DeepCopy())
		exists, _ := g.handler.checkDuplicateObject(g.authorityObj.DeepCopy())
		if exists == true {
			t.Error(errorDict["auth-name-coll"])
		}
		authorityRequest, _ := g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		if authorityRequest != nil {
			t.Error(errorDict["auth-email-coll"])
			g.edgenetclient.AppsV1alpha().AuthorityRequests().Delete(g.authorityRequestObj.GetName(), &metav1.DeleteOptions{})
		}
	})
	t.Run("authority request: different", func(t *testing.T) {
		g.authorityRequestObj.Name = "different"
		g.authorityRequestObj.Spec.Contact.Email = "different"
		// Create another authority request with a different name and an email address
		g.edgenetclient.AppsV1alpha().AuthorityRequests().Create(g.authorityRequestObj.DeepCopy())
		exists, _ := g.handler.checkDuplicateObject(g.authorityObj.DeepCopy())
		if exists == true {
			t.Error(errorDict["auth-name-coll"])
		}
		authorityRequest, _ := g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
		if authorityRequest == nil {
			t.Error(errorDict["auth-info-del"])
		} else {
			g.edgenetclient.AppsV1alpha().AuthorityRequests().Delete(g.authorityRequestObj.GetName(), &metav1.DeleteOptions{})
		}
	})
	t.Run("user: same email address", func(t *testing.T) {
		// Create a user for comparison
		g.edgenetclient.AppsV1alpha().Users("default").Create(g.userObj.DeepCopy())
		exists, _ := g.handler.checkDuplicateObject(g.authorityObj.DeepCopy())
		if exists != true {
			t.Error(errorDict["user-coll-failed"])
		}
		g.edgenetclient.AppsV1alpha().Users("default").Delete(g.userObj.GetName(), &metav1.DeleteOptions{})
	})

	t.Run("user: different", func(t *testing.T) {
		// Create a user for comparison with different email address
		g.userObj.Spec.Email = "different"
		g.edgenetclient.AppsV1alpha().Users("default").Create(g.userObj.DeepCopy())
		exists, _ := g.handler.checkDuplicateObject(g.authorityObj.DeepCopy())
		if exists == true {
			t.Error(errorDict["user-conflict"])
		}
		g.edgenetclient.AppsV1alpha().Users("default").Delete(g.userObj.GetName(), &metav1.DeleteOptions{})
	})
}

func TestAuthorityPreparation(t *testing.T) {
	g := AuthorityTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	g.authorityObj.Spec.Enabled = true
	var authorityCopy *apps_v1alpha.Authority
	// Test repeated demands
	for i := 1; i < 3; i++ {
		t.Run(fmt.Sprintf("preation no %d", i), func(t *testing.T) {
			if i == 1 {
				authorityCopy = g.handler.authorityPreparation(g.authorityObj.DeepCopy())
			} else {
				authorityCopy = g.handler.authorityPreparation(authorityCopy)
			}
			if !reflect.DeepEqual(g.authorityObj.Spec, authorityCopy.Spec) {
				t.Error(errorDict["auth-create-failed"])
			}
			if authorityCopy.Status.State != established {
				t.Error(errorDict["auth-est-failed"])
			}
			if authorityCopy.Spec.Enabled != true {
				t.Error(errorDict["auth-disbld-creation"])
			}
		})
	}
}
