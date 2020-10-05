package emailverification

import (
	"context"
	"flag"
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
	authorityObj  apps_v1alpha.Authority
	ARObj         apps_v1alpha.AuthorityRequest
	URRObj        apps_v1alpha.UserRegistrationRequest
	userObj       apps_v1alpha.User
	EVObj         apps_v1alpha.EmailVerification
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
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "johndoe",
			Namespace:  "authority-edgenet",
			UID:        "TestUID",
			Generation: 1,
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			Type: "admin",
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
	EVObj := apps_v1alpha.EmailVerification{
		TypeMeta: metav1.TypeMeta{
			Kind:       "emailVerification",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emailverificationcode",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.EmailVerificationSpec{
			Kind:       "Email",
			Identifier: "johndoe",
			Verified:   false,
		},
	}
	g.authorityObj = authorityObj
	g.ARObj = authorityRequestObj
	g.userObj = userObj
	g.URRObj = URRObj
	g.EVObj = EVObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// authorityHandler := authority.Handler{}
	// authorityHandler.Init(g.client, g.edgenetClient)
	// Create Authority
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.EVObj.GetNamespace()}}
	namespaceLabels := map[string]string{"owner": "authority", "owner-name": g.authorityObj.GetName(), "authority-name": g.authorityObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	// Create a user as admin on authority
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
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
	reference := g.EVObj
	code := "bs" + util.GenerateRandomString(16)
	reference.SetName(code)
	g.edgenetClient.AppsV1alpha().EmailVerifications(reference.GetNamespace()).Create(context.TODO(), reference.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(reference.DeepCopy())
	t.Run("set expiry date", func(t *testing.T) {
		// Handler will update expiration time
		EVCopy, _ := g.edgenetClient.AppsV1alpha().EmailVerifications(reference.GetNamespace()).Get(context.TODO(), reference.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(24 * time.Hour),
		}
		util.Equals(t, expected.Day(), EVCopy.Status.Expires.Day())
		util.Equals(t, expected.Month(), EVCopy.Status.Expires.Month())
		util.Equals(t, expected.Year(), EVCopy.Status.Expires.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		EVCopy, _ := g.edgenetClient.AppsV1alpha().EmailVerifications(reference.GetNamespace()).Get(context.TODO(), reference.GetName(), metav1.GetOptions{})
		go g.handler.runVerificationTimeout(EVCopy)
		EVCopy.Status.Expires = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		_, err := g.edgenetClient.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Update(context.TODO(), EVCopy.DeepCopy(), metav1.UpdateOptions{})
		util.OK(t, err)
		time.Sleep(100 * time.Millisecond)
		_, err = g.edgenetClient.AppsV1alpha().EmailVerifications(EVCopy.GetNamespace()).Get(context.TODO(), EVCopy.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
	t.Run("recreate a verified object", func(t *testing.T) {
		recreate := g.EVObj
		code := "bs" + util.GenerateRandomString(16)
		recreate.SetName(code)
		recreate.Spec.Verified = true
		g.edgenetClient.AppsV1alpha().EmailVerifications(recreate.GetNamespace()).Create(context.TODO(), recreate.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(recreate.DeepCopy())
		// Handler will delete EV if it is verified
		_, err := g.edgenetClient.AppsV1alpha().EmailVerifications(recreate.GetNamespace()).Get(context.TODO(), recreate.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	t.Run("verify", func(t *testing.T) {
		cases := map[string]struct {
			kind       string
			identifier string
			expected   bool
		}{
			"email":     {"Email", "johndoe", true},
			"user":      {"User", "johnsmith", true},
			"authority": {"Authority", "edgenet-request", true},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				verify := g.EVObj
				code := "bs" + util.GenerateRandomString(16)
				verify.SetName(code)
				verify.Spec.Kind = tc.kind
				verify.Spec.Identifier = tc.identifier
				if tc.kind == "Authority" {
					g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), g.ARObj.DeepCopy(), metav1.CreateOptions{})
					defer g.edgenetClient.AppsV1alpha().AuthorityRequests().Delete(context.TODO(), g.ARObj.GetName(), metav1.DeleteOptions{})
				} else if tc.kind == "User" {
					g.edgenetClient.AppsV1alpha().UserRegistrationRequests(g.URRObj.GetNamespace()).Create(context.TODO(), g.URRObj.DeepCopy(), metav1.CreateOptions{})
					defer g.edgenetClient.AppsV1alpha().UserRegistrationRequests(g.URRObj.GetNamespace()).Delete(context.TODO(), g.URRObj.GetName(), metav1.DeleteOptions{})
				}
				g.edgenetClient.AppsV1alpha().EmailVerifications(verify.GetNamespace()).Create(context.TODO(), verify.DeepCopy(), metav1.CreateOptions{})
				g.handler.ObjectCreated(verify.DeepCopy())
				EVObj, err := g.edgenetClient.AppsV1alpha().EmailVerifications(verify.GetNamespace()).Get(context.TODO(), verify.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				EVObj.Spec.Verified = true
				var field fields
				g.handler.ObjectUpdated(EVObj, field)
				// Handler will delete EV if it is verified
				_, err = g.edgenetClient.AppsV1alpha().EmailVerifications(EVObj.GetNamespace()).Get(context.TODO(), EVObj.GetName(), metav1.GetOptions{})
				util.Equals(t, true, errors.IsNotFound(err))
			})
		}
	})

	t.Run("dubious", func(t *testing.T) {
		cases := map[string]struct {
			kind     string
			cheat    []string
			expected bool
		}{
			"email/identifier":           {"Email", []string{"Identifier", "joepublic"}, true},
			"user/identifier":            {"User", []string{"Identifier", "tompublic"}, true},
			"authority/identifier":       {"Authority", []string{"Identifier", "dubious"}, true},
			"email/kind":                 {"Email", []string{"Kind", "User"}, true},
			"user/kind":                  {"User", []string{"Kind", "Authority"}, true},
			"authority/kind":             {"Authority", []string{"Kind", "Email"}, true},
			"trustworthy/authority/kind": {"Authority", []string{"Kind", "Authority"}, false},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				dub := g.EVObj
				code := "bs" + util.GenerateRandomString(16)
				dub.SetName(code)
				dub.Spec.Kind = tc.kind
				if tc.kind == "Authority" {
					g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), g.ARObj.DeepCopy(), metav1.CreateOptions{})
					defer g.edgenetClient.AppsV1alpha().AuthorityRequests().Delete(context.TODO(), g.ARObj.GetName(), metav1.DeleteOptions{})
				} else if tc.kind == "User" {
					g.edgenetClient.AppsV1alpha().UserRegistrationRequests(g.URRObj.GetNamespace()).Create(context.TODO(), g.URRObj.DeepCopy(), metav1.CreateOptions{})
					defer g.edgenetClient.AppsV1alpha().UserRegistrationRequests(g.URRObj.GetNamespace()).Delete(context.TODO(), g.URRObj.GetName(), metav1.DeleteOptions{})
				}
				g.edgenetClient.AppsV1alpha().EmailVerifications(dub.GetNamespace()).Create(context.TODO(), dub.DeepCopy(), metav1.CreateOptions{})
				g.handler.ObjectCreated(g.EVObj.DeepCopy())
				EVObj, err := g.edgenetClient.AppsV1alpha().EmailVerifications(dub.GetNamespace()).Get(context.TODO(), dub.GetName(), metav1.GetOptions{})
				util.OK(t, err)

				var field fields
				if tc.cheat[0] == "Identifier" {
					EVObj.Spec.Identifier = tc.cheat[1]
					field.identifier = true
				} else if tc.cheat[0] == "Kind" && tc.cheat[1] != EVObj.Spec.Kind {
					EVObj.Spec.Kind = tc.cheat[1]
					field.kind = true
				}
				g.handler.ObjectUpdated(EVObj, field)
				// Handler deletes EV as it is no longer valid
				_, err = g.edgenetClient.AppsV1alpha().EmailVerifications(EVObj.GetNamespace()).Get(context.TODO(), EVObj.GetName(), metav1.GetOptions{})
				util.Equals(t, tc.expected, errors.IsNotFound(err))
			})
		}
	})
}

func TestCreateEmailVerification(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	cases := map[string]struct {
		input    interface{}
		expected bool
	}{
		"authority request":         {g.ARObj.DeepCopy(), true},
		"user registration request": {g.URRObj.DeepCopy(), true},
		"user":                      {g.userObj.DeepCopy(), true},
		"user no pointer":           {g.userObj, false},
		"user wrong obj":            {g.authorityObj, false},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			status := g.handler.Create(tc.input, []metav1.OwnerReference{})
			util.Equals(t, tc.expected, status)
		})
	}
}
