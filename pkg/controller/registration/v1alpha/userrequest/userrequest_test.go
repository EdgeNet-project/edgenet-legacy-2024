package userrequest

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
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
	tenantObj        corev1alpha.Tenant
	tenantRequestObj registrationv1alpha.TenantRequest
	userObj          corev1alpha.User
	userRequestObj   registrationv1alpha.UserRequest
	client           kubernetes.Interface
	edgenetClient    versioned.Interface
	handler          Handler
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
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: corev1alpha.TenantSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: corev1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha.User{
				Email:     "joe.public@edge-net.org",
				FirstName: "Joe",
				LastName:  "Public",
				Phone:     "+33NUMBER",
				Username:  "joepublic",
			},
			Enabled: true,
		},
	}
	tenantRequestObj := registrationv1alpha.TenantRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "tenantRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet-request",
		},
		Spec: registrationv1alpha.TenantRequestSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: corev1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha.User{
				Email:     "tom.public@edge-net.org",
				FirstName: "Tom",
				LastName:  "Public",
				Phone:     "+33NUMBER",
				Username:  "tompublic",
			},
		},
	}
	userObj := corev1alpha.User{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@edge-net.org",
	}
	userRequestObj := registrationv1alpha.UserRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "johnsmith",
		},
		Spec: registrationv1alpha.UserRequestSpec{
			Tenant:    "edgenet",
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
		},
	}
	g.tenantObj = tenantObj
	g.tenantRequestObj = tenantRequestObj
	g.userRequestObj = userRequestObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// tenantHandler := tenant.Handler{}
	// tenantHandler.Init(g.client, g.edgenetClient)
	// Create Tenant
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	/*namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: }}
	namespaceLabels := map[string]string{"owner": "tenant", "owner-name": g.tenantObj.GetName(), "tenant-name": g.tenantObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})*/
	// Invoke ObjectCreated to create namespace
	// tenantHandler.ObjectCreated(g.tenantObj.DeepCopy())
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
	go g.handler.RunExpiryController()
	t.Run("set expiry date", func(t *testing.T) {
		g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), g.userRequestObj.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreatedOrUpdated(g.userRequestObj.DeepCopy())
		userRequest, _ := g.edgenetClient.RegistrationV1alpha().UserRequests().Get(context.TODO(), g.userRequestObj.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		util.Equals(t, expected.Day(), userRequest.Status.Expiry.Day())
		util.Equals(t, expected.Month(), userRequest.Status.Expiry.Month())
		util.Equals(t, expected.Year(), userRequest.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		userRequest, _ := g.edgenetClient.RegistrationV1alpha().UserRequests().Get(context.TODO(), g.userRequestObj.GetName(), metav1.GetOptions{})
		userRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		g.edgenetClient.RegistrationV1alpha().UserRequests().Update(context.TODO(), userRequest, metav1.UpdateOptions{})
		time.Sleep(100 * time.Millisecond)
		_, err := g.edgenetClient.RegistrationV1alpha().UserRequests().Get(context.TODO(), g.userRequestObj.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
	t.Run("collision", func(t *testing.T) {
		urr1 := g.userRequestObj
		urr1.SetName(g.userObj.GetName())
		urr2 := g.userRequestObj
		urr2.Spec.Email = g.userObj.Email
		urr3 := g.userRequestObj
		urr3.Spec.Email = g.tenantRequestObj.Spec.Contact.Email
		urr4 := g.userRequestObj
		urr4Comparison := urr4
		urr4Comparison.SetName("duplicate")
		urr4Comparison.SetUID("UID")

		// Create a user, a tenant request, and user registration request for comparison
		tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), g.tenantObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		tenant.Spec.User = append(tenant.Spec.User, g.userObj)
		_, err = g.edgenetClient.CoreV1alpha().Tenants().Update(context.TODO(), tenant, metav1.UpdateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), g.tenantRequestObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), urr4Comparison.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)

		cases := map[string]struct {
			request  registrationv1alpha.UserRequest
			expected string
		}{
			"username/user":       {urr1, fmt.Sprintf(statusDict["username-exist"], urr1.GetName())},
			"email/user":          {urr2, fmt.Sprintf(statusDict["email-exist"], urr2.Spec.Email)},
			"email/tenantrequest": {urr3, fmt.Sprintf(statusDict["email-existauth"], urr3.Spec.Email)},
			"email/userrequest":   {urr4, fmt.Sprintf(statusDict["email-existregist"], urr4.Spec.Email)},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				_, err := g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), tc.request.DeepCopy(), metav1.CreateOptions{})
				util.OK(t, err)
				g.handler.ObjectCreatedOrUpdated(tc.request.DeepCopy())
				userRequest, err := g.edgenetClient.RegistrationV1alpha().UserRequests().Get(context.TODO(), tc.request.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				util.Equals(t, tc.expected, userRequest.Status.Message[0])
				g.edgenetClient.RegistrationV1alpha().UserRequests().Delete(context.TODO(), tc.request.GetName(), metav1.DeleteOptions{})
			})
		}
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	t.Run("collision", func(t *testing.T) {
		urr := g.userRequestObj
		urrComparison := urr
		urrComparison.SetUID("UID")
		urrComparison.SetName("different")
		urrComparison.Spec.Email = "duplicate@edge-net.org"
		urrUpdate1 := urr
		urrUpdate1.Spec.Email = urrComparison.Spec.Email
		urrUpdate2 := urr
		urrUpdate2.Spec.Email = g.userObj.Email
		urrUpdate3 := urr
		urrUpdate3.Spec.Email = g.tenantRequestObj.Spec.Contact.Email
		urrUpdate4 := urr
		urrUpdate4.Spec.Email = "different@edge-net.org"

		// Create a user, a tenant request, and user registration request for comparison
		tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), g.tenantObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		tenant.Spec.User = append(tenant.Spec.User, g.userObj)
		_, err = g.edgenetClient.CoreV1alpha().Tenants().Update(context.TODO(), tenant, metav1.UpdateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), g.tenantRequestObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), urrComparison.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), urr.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)

		var status = registrationv1alpha.UserRequestStatus{}
		cases := map[string]struct {
			request  registrationv1alpha.UserRequest
			expected []string
		}{
			"email/userrequest/duplicate":   {urrUpdate1, []string{failure}},
			"email/user/duplicate":          {urrUpdate2, []string{failure}},
			"email/tenantrequest/duplicate": {urrUpdate3, []string{failure}},
			"email/unique":                  {urrUpdate4, []string{success, issue, ""}},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				tc.request.Status = status
				_, err := g.edgenetClient.RegistrationV1alpha().UserRequests().Update(context.TODO(), tc.request.DeepCopy(), metav1.UpdateOptions{})
				util.OK(t, err)
				g.handler.ObjectCreatedOrUpdated(tc.request.DeepCopy())
				userRequest, err := g.edgenetClient.RegistrationV1alpha().UserRequests().Get(context.TODO(), tc.request.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				util.EqualsMultipleExp(t, tc.expected, userRequest.Status.State)
				status = userRequest.Status
			})
		}
	})

	t.Run("approval", func(t *testing.T) {
		// Updating user registration status to approved
		g.userRequestObj.Spec.Approved = true
		// Requesting server to update internal representation of user registration object and transition it to user
		g.handler.ObjectCreatedOrUpdated(g.userRequestObj.DeepCopy())
		// Checking if handler created user from user registration
		//_, err := g.edgenetClient.AppsV1alpha().Users().Get(context.TODO(), g.userRequestObj.GetName(), metav1.GetOptions{})
		//util.OK(t, err)
	})
}
