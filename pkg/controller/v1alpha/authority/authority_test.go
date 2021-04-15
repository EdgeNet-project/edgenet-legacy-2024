package authority

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
	authorityObj        apps_v1alpha.Authority
	authorityRequestObj apps_v1alpha.AuthorityRequest
	userObj             apps_v1alpha.User
	userRegistrationObj apps_v1alpha.UserRegistrationRequest
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
	authorityRequestObj := apps_v1alpha.AuthorityRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "authorityRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet-request",
		},
		Spec: apps_v1alpha.AuthorityRequestSpec{
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
			Name:       "joepublic",
			Namespace:  "authority-edgenet",
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
	URRObj := apps_v1alpha.UserRegistrationRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRegistrationRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "johnsmith",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserRegistrationRequestSpec{
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
		},
	}
	g.authorityObj = authorityObj
	g.authorityRequestObj = authorityRequestObj
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
	util.Equals(t, "authority-quota", g.handler.resourceQuota.Name)
	util.NotEquals(t, nil, g.handler.resourceQuota.Spec.Hard)
	util.Equals(t, int64(0), g.handler.resourceQuota.Spec.Hard.Pods().Value())
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.authorityObj.DeepCopy())

	t.Run("user creation", func(t *testing.T) {
		_, err := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("total resource quota", func(t *testing.T) {
		_, err := g.handler.edgenetClientset.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.authorityObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("cluster role", func(t *testing.T) {
		_, err := g.handler.clientset.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("authority-%s", g.authorityObj.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
}

func TestCollision(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)

	ar1 := g.authorityRequestObj
	ar1.Spec.Contact.Email = g.authorityObj.Spec.Contact.Email
	ar2 := g.authorityRequestObj
	ar2.SetName(g.authorityObj.GetName())
	ar3 := g.authorityRequestObj

	user1 := g.userObj
	user1.SetNamespace("different")
	user1.Spec.Email = g.authorityObj.Spec.Contact.Email
	user2 := g.userObj
	user2.SetNamespace("different")

	cases := map[string]struct {
		request  interface{}
		kind     string
		expected bool
	}{
		"ar/email":   {ar1.DeepCopy(), "AuthorityRequest", true},
		"ar/name":    {ar2.DeepCopy(), "AuthorityRequest", true},
		"ar/none":    {ar3.DeepCopy(), "AuthorityRequest", false},
		"user/email": {user1.DeepCopy(), "User", true},
		"user/none":  {user2.DeepCopy(), "User", false},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			if tc.kind == "AuthorityRequest" {
				_, err := g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), tc.request.(*apps_v1alpha.AuthorityRequest), metav1.CreateOptions{})
				util.OK(t, err)
				defer g.edgenetClient.AppsV1alpha().AuthorityRequests().Delete(context.TODO(), tc.request.(*apps_v1alpha.AuthorityRequest).GetName(), metav1.DeleteOptions{})
				g.handler.checkDuplicateObject(g.authorityObj.DeepCopy())
				_, err = g.edgenetClient.AppsV1alpha().AuthorityRequests().Get(context.TODO(), tc.request.(*apps_v1alpha.AuthorityRequest).GetName(), metav1.GetOptions{})
				util.Equals(t, tc.expected, errors.IsNotFound(err))
			} else if tc.kind == "User" {
				_, err := g.edgenetClient.AppsV1alpha().Users(tc.request.(*apps_v1alpha.User).GetNamespace()).Create(context.TODO(), tc.request.(*apps_v1alpha.User).DeepCopy(), metav1.CreateOptions{})
				util.OK(t, err)
				defer g.edgenetClient.AppsV1alpha().Users(tc.request.(*apps_v1alpha.User).GetNamespace()).Delete(context.TODO(), tc.request.(*apps_v1alpha.User).GetName(), metav1.DeleteOptions{})
				exists, message := g.handler.checkDuplicateObject(g.authorityObj.DeepCopy())
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
	// Create an authority to update later
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreated func to create a user
	g.handler.ObjectCreated(g.authorityObj.DeepCopy())
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	userAdmin, _ := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	util.Equals(t, true, userAdmin.Spec.Active)
	user, _ := g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Get(context.TODO(), g.userObj.GetName(), metav1.GetOptions{})
	util.Equals(t, true, user.Spec.Active)
	g.authorityObj.Spec.Enabled = false
	g.handler.ObjectUpdated(g.authorityObj.DeepCopy())
	userAdmin, _ = g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	util.Equals(t, false, userAdmin.Spec.Active)
	user, _ = g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Get(context.TODO(), g.userObj.GetName(), metav1.GetOptions{})
	util.Equals(t, false, user.Spec.Active)
}

func TestAuthorityPreparation(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
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
			util.Equals(t, g.authorityObj.Spec, authorityCopy.Spec)
			util.Equals(t, established, authorityCopy.Status.State)
			util.Equals(t, true, authorityCopy.Spec.Enabled)
		})
	}
}
