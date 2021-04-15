package userregistrationrequest

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

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
				Email:     "joe.public@edge-net.org",
				FirstName: "Joe",
				LastName:  "Public",
				Phone:     "+33NUMBER",
				Username:  "joepublic",
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
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "johndoe",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@edge-net.org",
			Active:    false,
		},
		Status: apps_v1alpha.UserStatus{
			State: success,
			Type:  "admin",
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
	g.userRegistrationObj = URRObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// authorityHandler := authority.Handler{}
	// authorityHandler.Init(g.client, g.edgenetClient)
	// Create Authority
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", g.authorityObj.GetName())}}
	namespaceLabels := map[string]string{"owner": "authority", "owner-name": g.authorityObj.GetName(), "authority-name": g.authorityObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	// Invoke ObjectCreated to create namespace
	// authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetClient)
	util.Equals(t, g.client, g.handler.clientset)
	util.Equals(t, g.edgenetClient, g.handler.edgenetClientset)
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	t.Run("set expiry date", func(t *testing.T) {
		g.edgenetClient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userRegistrationObj.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.userRegistrationObj.DeepCopy())
		URRCopy, _ := g.edgenetClient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userRegistrationObj.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		util.Equals(t, expected.Day(), URRCopy.Status.Expires.Day())
		util.Equals(t, expected.Month(), URRCopy.Status.Expires.Month())
		util.Equals(t, expected.Year(), URRCopy.Status.Expires.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		URRCopy, _ := g.edgenetClient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userRegistrationObj.GetName(), metav1.GetOptions{})
		go g.handler.runApprovalTimeout(URRCopy)
		URRCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		g.edgenetClient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(context.TODO(), URRCopy, metav1.UpdateOptions{})
		time.Sleep(100 * time.Millisecond)
		_, err := g.edgenetClient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userRegistrationObj.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
	t.Run("collision", func(t *testing.T) {
		urr1 := g.userRegistrationObj
		urr1.SetName(g.userObj.GetName())
		urr2 := g.userRegistrationObj
		urr2.Spec.Email = g.userObj.Spec.Email
		urr3 := g.userRegistrationObj
		urr3.Spec.Email = g.authorityRequestObj.Spec.Contact.Email
		urr4 := g.userRegistrationObj
		urr4Comparison := urr4
		urr4Comparison.SetName("duplicate")
		urr4Comparison.SetUID("UID")

		// Create a user, an authority request, and user registration request for comparison
		_, err := g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), g.authorityRequestObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().UserRegistrationRequests(urr4Comparison.GetNamespace()).Create(context.TODO(), urr4Comparison.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)

		cases := map[string]struct {
			request  apps_v1alpha.UserRegistrationRequest
			expected string
		}{
			"username/user":                 {urr1, fmt.Sprintf(statusDict["username-exist"], urr1.GetName())},
			"email/user":                    {urr2, fmt.Sprintf(statusDict["email-existuser"], urr2.Spec.Email)},
			"email/authorityrequest":        {urr3, fmt.Sprintf(statusDict["email-existauth"], urr3.Spec.Email)},
			"email/userregistrationrequest": {urr4, fmt.Sprintf(statusDict["email-existregist"], urr4.Spec.Email)},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				_, err := g.edgenetClient.AppsV1alpha().UserRegistrationRequests(tc.request.GetNamespace()).Create(context.TODO(), tc.request.DeepCopy(), metav1.CreateOptions{})
				util.OK(t, err)
				g.handler.ObjectCreated(tc.request.DeepCopy())
				URR, err := g.edgenetClient.AppsV1alpha().UserRegistrationRequests(tc.request.GetNamespace()).Get(context.TODO(), tc.request.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				util.Equals(t, tc.expected, URR.Status.Message[0])
				g.edgenetClient.AppsV1alpha().UserRegistrationRequests(tc.request.GetNamespace()).Delete(context.TODO(), tc.request.GetName(), metav1.DeleteOptions{})
			})
		}
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	t.Run("collision", func(t *testing.T) {
		urr := g.userRegistrationObj
		urrComparison := urr
		urrComparison.SetUID("UID")
		urrComparison.SetName("different")
		urrComparison.Spec.Email = "duplicate@edge-net.org"
		urrUpdate1 := urr
		urrUpdate1.Spec.Email = urrComparison.Spec.Email
		urrUpdate2 := urr
		urrUpdate2.Spec.Email = g.userObj.Spec.Email
		urrUpdate3 := urr
		urrUpdate3.Spec.Email = g.authorityRequestObj.Spec.Contact.Email
		urrUpdate4 := urr
		urrUpdate4.Spec.Email = "different@edge-net.org"

		// Create a user, an authority request, and user registration request for comparison
		_, err := g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), g.authorityRequestObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().UserRegistrationRequests(urrComparison.GetNamespace()).Create(context.TODO(), urrComparison.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().UserRegistrationRequests(urr.GetNamespace()).Create(context.TODO(), urr.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)

		var status = apps_v1alpha.UserRegistrationRequestStatus{}
		cases := map[string]struct {
			request  apps_v1alpha.UserRegistrationRequest
			expected []string
		}{
			"email/userregistrationrequest/duplicate": {urrUpdate1, []string{failure}},
			"email/user/duplicate":                    {urrUpdate2, []string{failure}},
			"email/authorityrequest/duplicate":        {urrUpdate3, []string{failure}},
			"email/unique":                            {urrUpdate4, []string{success, issue, ""}},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				tc.request.Status = status
				_, err := g.edgenetClient.AppsV1alpha().UserRegistrationRequests(tc.request.GetNamespace()).Update(context.TODO(), tc.request.DeepCopy(), metav1.UpdateOptions{})
				util.OK(t, err)
				g.handler.ObjectUpdated(tc.request.DeepCopy())
				URR, err := g.edgenetClient.AppsV1alpha().UserRegistrationRequests(tc.request.GetNamespace()).Get(context.TODO(), tc.request.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				util.EqualsMultipleExp(t, tc.expected, URR.Status.State)
				status = URR.Status
			})
		}
	})

	t.Run("approval", func(t *testing.T) {
		// Updating user registration status to approved
		g.userRegistrationObj.Spec.Approved = true
		// Requesting server to update internal representation of user registration object and transition it to user
		g.handler.ObjectUpdated(g.userRegistrationObj.DeepCopy())
		// Checking if handler created user from user registration
		_, err := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userRegistrationObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
	})
}
