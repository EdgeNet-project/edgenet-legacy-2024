package tenant

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type TestGroup struct {
	tenantObj        apps_v1alpha.Tenant
	tenantRequestObj apps_v1alpha.TenantRequest
	userObj             apps_v1alpha.User
	userRegistrationObj registrationv1alpha.UserRequest
	client              kubernetes.Interface
	edgenetClient       versioned.Interface
	handler             Handler
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
	tenantObj := apps_v1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: apps_v1alpha.TenantSpec{
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
	tenantRequestObj := apps_v1alpha.TenantRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "tenantRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet-request",
		},
		Spec: apps_v1alpha.TenantRequestSpec{
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
				Email:     "tom.public@edge-net.org",
				FirstName: "Tom",
				LastName:  "Public",
				Phone:     "+33NUMBER",
				Username:  "tompublic",
			},
		},
		Status: apps_v1alpha.TenantRequestStatus{
			State: success,
		},
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "joepublic",
			Namespace:  "tenant-edgenet",
			UID:        "TestUID",
			Generation: 1,
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "Joe",
			LastName:  "Public",
			Email:     "joe.public@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			Type: "user",
		},
	}
	URRObj := registrationv1alpha.UserRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRegistrationRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "johnsmith",
			Namespace: "tenant-edgenet",
		},
		Spec: registrationv1alpha.UserRequestSpec{
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
		},
	}
	g.tenantObj = tenantObj
	g.tenantRequestObj = tenantRequestObj
	g.userObj = userObj
	g.userRegistrationObj = URRObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
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
	util.Equals(t, "tenant-quota", g.handler.resourceQuota.Name)
	util.NotEquals(t, nil, g.handler.resourceQuota.Spec.Hard)
	util.Equals(t, int64(0), g.handler.resourceQuota.Spec.Hard.Pods().Value())
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.tenantObj.DeepCopy())

	t.Run("user creation", func(t *testing.T) {
		_, err := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("tenant-%s", g.tenantObj.GetName())).Get(context.TODO(), g.tenantObj.Spec.Contact.Username, metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("total resource quota", func(t *testing.T) {
		_, err := g.handler.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.tenantObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("cluster role", func(t *testing.T) {
		_, err := g.handler.clientset.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("tenant-%s", g.tenantObj.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
}

func TestCollision(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)

	ar1 := g.tenantRequestObj
	ar1.Spec.Contact.Email = g.tenantObj.Spec.Contact.Email
	ar2 := g.tenantRequestObj
	ar2.SetName(g.tenantObj.GetName())
	ar3 := g.tenantRequestObj

	user1 := g.userObj
	user1.SetNamespace("different")
	user1.Spec.Email = g.tenantObj.Spec.Contact.Email
	user2 := g.userObj
	user2.SetNamespace("different")

	cases := map[string]struct {
		request  interface{}
		kind     string
		expected bool
	}{
		"ar/email":   {ar1.DeepCopy(), "TenantRequest", true},
		"ar/name":    {ar2.DeepCopy(), "TenantRequest", true},
		"ar/none":    {ar3.DeepCopy(), "TenantRequest", false},
		"user/email": {user1.DeepCopy(), "User", true},
		"user/none":  {user2.DeepCopy(), "User", false},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			if tc.kind == "TenantRequest" {
				_, err := g.edgenetClient.CoreV1alpha().TenantRequests().Create(context.TODO(), tc.request.(*apps_v1alpha.TenantRequest), metav1.CreateOptions{})
				util.OK(t, err)
				defer g.edgenetClient.CoreV1alpha().TenantRequests().Delete(context.TODO(), tc.request.(*apps_v1alpha.TenantRequest).GetName(), metav1.DeleteOptions{})
				g.handler.checkDuplicateObject(g.tenantObj.DeepCopy())
				_, err = g.edgenetClient.CoreV1alpha().TenantRequests().Get(context.TODO(), tc.request.(*apps_v1alpha.TenantRequest).GetName(), metav1.GetOptions{})
				util.Equals(t, tc.expected, errors.IsNotFound(err))
			} else if tc.kind == "User" {
				_, err := g.edgenetClient.AppsV1alpha().Users(tc.request.(*apps_v1alpha.User).GetNamespace()).Create(context.TODO(), tc.request.(*apps_v1alpha.User).DeepCopy(), metav1.CreateOptions{})
				util.OK(t, err)
				defer g.edgenetClient.AppsV1alpha().Users(tc.request.(*apps_v1alpha.User).GetNamespace()).Delete(context.TODO(), tc.request.(*apps_v1alpha.User).GetName(), metav1.DeleteOptions{})
				exists, message := g.handler.checkDuplicateObject(g.tenantObj.DeepCopy())
				log.Println(message)
				util.Equals(t, tc.expected, exists)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Create an tenant to update later
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreated func to create a user
	g.handler.ObjectCreated(g.tenantObj.DeepCopy())
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	userAdmin, _ := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("tenant-%s", g.tenantObj.GetName())).Get(context.TODO(), g.tenantObj.Spec.Contact.Username, metav1.GetOptions{})
	util.Equals(t, true, userAdmin.Spec.Active)
	user, _ := g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Get(context.TODO(), g.userObj.GetName(), metav1.GetOptions{})
	util.Equals(t, true, user.Spec.Active)
	g.tenantObj.Spec.Enabled = false
	g.handler.ObjectUpdated(g.tenantObj.DeepCopy())
	userAdmin, _ = g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("tenant-%s", g.tenantObj.GetName())).Get(context.TODO(), g.tenantObj.Spec.Contact.Username, metav1.GetOptions{})
	util.Equals(t, false, userAdmin.Spec.Active)
	user, _ = g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Get(context.TODO(), g.userObj.GetName(), metav1.GetOptions{})
	util.Equals(t, false, user.Spec.Active)
}

func TestTenantPreparation(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	g.tenantObj.Spec.Enabled = true
	var tenantCopy *apps_v1alpha.Tenant
	// Test repeated demands
	for i := 1; i < 3; i++ {
		t.Run(fmt.Sprintf("preation no %d", i), func(t *testing.T) {
			if i == 1 {
				tenantCopy = g.handler.tenantPreparation(g.tenantObj.DeepCopy())
			} else {
				tenantCopy = g.handler.tenantPreparation(tenantCopy)
			}
			util.Equals(t, g.tenantObj.Spec, tenantCopy.Spec)
			util.Equals(t, established, tenantCopy.Status.State)
			util.Equals(t, true, tenantCopy.Spec.Enabled)
		})
	}
}
