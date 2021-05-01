package authorityrequest

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
	authorityObj        corev1alpha.Authority
	authorityRequestObj corev1alpha.AuthorityRequest
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
	authorityObj := corev1alpha.Authority{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Authority",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: corev1alpha.AuthoritySpec{
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
	authorityRequestObj := corev1alpha.AuthorityRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "authorityRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet-request",
		},
		Spec: corev1alpha.AuthorityRequestSpec{
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
	URRObj := registrationv1alpha.UserRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRegistrationRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "johnsmith",
			Namespace: "authority-edgenet",
		},
		Spec: registrationv1alpha.UserRequestSpec{
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
	// authorityHandler := authority.Handler{}
	// authorityHandler.Init(g.client, g.edgenetClient)
	// Create Authority
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.userObj.GetNamespace()}}
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
	// Creation of Authority request
	g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), g.authorityRequestObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(g.authorityRequestObj.DeepCopy())
	t.Run("set expiry date", func(t *testing.T) {
		authorityRequest, _ := g.edgenetClient.AppsV1alpha().AuthorityRequests().Get(context.TODO(), g.authorityRequestObj.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		util.Equals(t, expected.Day(), authorityRequest.Status.Expiry.Day())
		util.Equals(t, expected.Month(), authorityRequest.Status.Expiry.Month())
		util.Equals(t, expected.Year(), authorityRequest.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		authorityRequest, _ := g.edgenetClient.AppsV1alpha().AuthorityRequests().Get(context.TODO(), g.authorityRequestObj.GetName(), metav1.GetOptions{})
		go g.handler.runApprovalTimeout(authorityRequest)
		authorityRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		_, err := g.edgenetClient.AppsV1alpha().AuthorityRequests().Update(context.TODO(), authorityRequest, metav1.UpdateOptions{})
		util.OK(t, err)
		time.Sleep(100 * time.Millisecond)
		_, err = g.edgenetClient.AppsV1alpha().AuthorityRequests().Get(context.TODO(), authorityRequest.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
	t.Run("collision", func(t *testing.T) {
		ar1 := g.authorityRequestObj
		ar1.SetName(g.authorityObj.GetName())
		ar2 := g.authorityRequestObj
		ar2.SetName("different")
		ar2.SetUID("UIDar2")
		ar3 := g.authorityRequestObj
		ar3.SetName("different")
		ar3.Spec.Contact.Email = g.userObj.Spec.Email
		ar4 := g.authorityRequestObj
		ar4.SetName("different")
		ar4.Spec.Contact.Email = g.userRegistrationObj.Spec.Email
		// Create a user, an authority request, and user registration request for comparison
		_, err := g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().UserRegistrationRequests(g.userRegistrationObj.GetNamespace()).Create(context.TODO(), g.userRegistrationObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), g.authorityRequestObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)

		cases := map[string]struct {
			request  corev1alpha.AuthorityRequest
			expected string
		}{
			"name/authority":                {ar1, fmt.Sprintf(statusDict["authority-taken"], ar1.GetName())},
			"email/authorityrequest":        {ar2, fmt.Sprintf(statusDict["email-used-auth"], ar2.Spec.Contact.Email)},
			"email/user":                    {ar3, fmt.Sprintf(statusDict["email-exist"], ar3.Spec.Contact.Email)},
			"email/userregistrationrequest": {ar4, fmt.Sprintf(statusDict["email-used-reg"], ar4.Spec.Contact.Email)},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				_, err := g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), tc.request.DeepCopy(), metav1.CreateOptions{})
				util.OK(t, err)
				g.handler.ObjectCreated(tc.request.DeepCopy())
				AR, err := g.edgenetClient.AppsV1alpha().AuthorityRequests().Get(context.TODO(), tc.request.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				util.Equals(t, tc.expected, AR.Status.Message[0])
				g.edgenetClient.AppsV1alpha().AuthorityRequests().Delete(context.TODO(), tc.request.GetName(), metav1.DeleteOptions{})
			})
		}
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	t.Run("collision", func(t *testing.T) {
		ar := g.authorityRequestObj
		arComparison := ar
		arComparison.SetUID("UID")
		arComparison.SetName("different")
		arComparison.Spec.Contact.Email = "duplicate@edge-net.org"
		ar1 := ar
		ar1.Spec.Contact.Email = arComparison.Spec.Contact.Email
		ar2 := ar
		ar2.Spec.Contact.Email = g.userObj.Spec.Email
		ar3 := ar
		ar3.Spec.Contact.Email = g.userRegistrationObj.Spec.Email
		ar4 := ar
		ar4.Spec.Contact.Email = "different@edge-net.org"

		// Create a user, an authority request, and user registration request for comparison
		_, err := g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().UserRegistrationRequests(g.userRegistrationObj.GetNamespace()).Create(context.TODO(), g.userRegistrationObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), arComparison.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), ar.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)

		var status = corev1alpha.AuthorityRequestStatus{}
		cases := map[string]struct {
			request  corev1alpha.AuthorityRequest
			expected []string
		}{
			"email/authorityrequest/duplicate":        {ar1, []string{failure}},
			"email/user/duplicate":                    {ar2, []string{failure}},
			"email/userregistrationrequest/duplicate": {ar3, []string{failure}},
			"email/unique":                            {ar4, []string{success, issue, ""}},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				tc.request.Status = status
				_, err := g.edgenetClient.AppsV1alpha().AuthorityRequests().Update(context.TODO(), tc.request.DeepCopy(), metav1.UpdateOptions{})
				util.OK(t, err)
				g.handler.ObjectUpdated(tc.request.DeepCopy())
				AR, err := g.edgenetClient.AppsV1alpha().AuthorityRequests().Get(context.TODO(), tc.request.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				util.EqualsMultipleExp(t, tc.expected, AR.Status.State)
				status = AR.Status
			})
		}
	})

	t.Run("approval", func(t *testing.T) {
		// Updating authority request status to approved
		g.authorityRequestObj.Spec.Approved = true
		// Requesting server to update internal representation of authority request object and transition it to authority
		g.handler.ObjectUpdated(g.authorityRequestObj.DeepCopy())
		// Checking if handler created authority from request
		_, err := g.edgenetClient.AppsV1alpha().Authorities().Get(context.TODO(), g.authorityRequestObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
	})
}
