package tenantrequest

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
	flag.String("dir", "../../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../../configs/smtp_test.yaml", "Set SMTP path.")
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
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+33NUMBER",
				Username:  "johndoe",
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
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
		},
	}
	g.tenantObj = tenantObj
	g.tenantRequestObj = tenantRequestObj
	g.userObj = userObj
	g.userRequestObj = userRequestObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// tenantHandler := tenant.Handler{}
	// tenantHandler.Init(g.client, g.edgenetClient)
	// Create Tenant
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	/*namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.userObj.GetNamespace()}}
	namespaceLabels := map[string]string{"owner": "tenant", "owner-name": g.tenantObj.GetName(), "tenant-name": g.tenantObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})*/
	// Invoke ObjectCreatedOrUpdated to create namespace
	// tenantHandler.ObjectCreatedOrUpdated(g.tenantObj.DeepCopy())
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
	// Creation of Tenant request
	g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), g.tenantRequestObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreatedOrUpdated(g.tenantRequestObj.DeepCopy())
	t.Run("set expiry date", func(t *testing.T) {
		tenantRequest, _ := g.edgenetClient.RegistrationV1alpha().TenantRequests().Get(context.TODO(), g.tenantRequestObj.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		util.Equals(t, expected.Day(), tenantRequest.Status.Expiry.Day())
		util.Equals(t, expected.Month(), tenantRequest.Status.Expiry.Month())
		util.Equals(t, expected.Year(), tenantRequest.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		tenantRequest, _ := g.edgenetClient.RegistrationV1alpha().TenantRequests().Get(context.TODO(), g.tenantRequestObj.GetName(), metav1.GetOptions{})
		tenantRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		_, err := g.edgenetClient.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
		util.OK(t, err)
		time.Sleep(100 * time.Millisecond)
		_, err = g.edgenetClient.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequest.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
	t.Run("collision", func(t *testing.T) {
		tenantRequest1 := g.tenantRequestObj.DeepCopy()
		tenantRequest1.SetName(g.tenantObj.GetName())
		tenantRequest2 := g.tenantRequestObj.DeepCopy()
		tenantRequest2.SetName("different")
		tenantRequest2.SetUID("UIDtenantRequest2")
		tenantRequest3 := g.tenantRequestObj.DeepCopy()
		tenantRequest3.SetName("different")
		tenantRequest3.Spec.Contact.Email = g.userObj.Email
		tenantRequest4 := g.tenantRequestObj.DeepCopy()
		tenantRequest4.SetName("different")
		tenantRequest4.Spec.Contact.Email = g.userRequestObj.Spec.Email
		// Create a user, an tenant request, and user registration request for comparison
		tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), g.tenantObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		tenant.Spec.User = append(tenant.Spec.User, g.userObj)
		_, err = g.edgenetClient.CoreV1alpha().Tenants().Update(context.TODO(), tenant, metav1.UpdateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), g.userRequestObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), g.tenantRequestObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)

		cases := map[string]struct {
			request  *registrationv1alpha.TenantRequest
			expected string
		}{
			"name/tenant":                   {tenantRequest1, fmt.Sprintf(statusDict["tenant-taken"], tenantRequest1.GetName())},
			"email/tenantrequest":           {tenantRequest2, fmt.Sprintf(statusDict["email-used-auth"], tenantRequest2.Spec.Contact.Email)},
			"email/user":                    {tenantRequest3, fmt.Sprintf(statusDict["email-exist"], tenantRequest3.Spec.Contact.Email)},
			"email/userregistrationrequest": {tenantRequest4, fmt.Sprintf(statusDict["email-used-reg"], tenantRequest4.Spec.Contact.Email)},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				_, err := g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), tc.request.DeepCopy(), metav1.CreateOptions{})
				util.OK(t, err)
				g.handler.ObjectCreatedOrUpdated(tc.request.DeepCopy())
				tenantRequest, err := g.edgenetClient.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tc.request.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				util.Equals(t, tc.expected, tenantRequest.Status.Message[0])
				g.edgenetClient.RegistrationV1alpha().TenantRequests().Delete(context.TODO(), tc.request.GetName(), metav1.DeleteOptions{})
			})
		}
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	t.Run("collision", func(t *testing.T) {
		tenantRequest := g.tenantRequestObj
		tenantRequestComparison := tenantRequest.DeepCopy()
		tenantRequestComparison.SetUID("UID")
		tenantRequestComparison.SetName("different")
		tenantRequestComparison.Spec.Contact.Email = "duplicate@edge-net.org"
		tenantRequest1 := tenantRequest.DeepCopy()
		tenantRequest1.Spec.Contact.Email = tenantRequestComparison.Spec.Contact.Email
		tenantRequest2 := tenantRequest.DeepCopy()
		tenantRequest2.Spec.Contact.Email = g.userObj.Email
		tenantRequest3 := tenantRequest.DeepCopy()
		tenantRequest3.Spec.Contact.Email = g.userRequestObj.Spec.Email
		tenantRequest4 := tenantRequest.DeepCopy()
		tenantRequest4.Spec.Contact.Email = "different@edge-net.org"

		// Create a user, an tenant request, and user registration request for comparison
		_, err := g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), g.userRequestObj.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), tenantRequestComparison.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		_, err = g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), tenantRequest.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)

		var status = registrationv1alpha.TenantRequestStatus{}
		cases := map[string]struct {
			request  *registrationv1alpha.TenantRequest
			expected []string
		}{
			"email/tenantrequest/duplicate":           {tenantRequest1, []string{failure}},
			"email/user/duplicate":                    {tenantRequest2, []string{failure}},
			"email/userregistrationrequest/duplicate": {tenantRequest3, []string{failure}},
			"email/unique":                            {tenantRequest4, []string{success, issue, ""}},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				tc.request.Status = status
				_, err := g.edgenetClient.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tc.request.DeepCopy(), metav1.UpdateOptions{})
				util.OK(t, err)
				g.handler.ObjectCreatedOrUpdated(tc.request.DeepCopy())
				tenantRequest, err := g.edgenetClient.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tc.request.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				util.EqualsMultipleExp(t, tc.expected, tenantRequest.Status.State)
				status = tenantRequest.Status
			})
		}
	})

	t.Run("approval", func(t *testing.T) {
		// Updating tenant request status to approved
		g.tenantRequestObj.Spec.Approved = true
		// Requesting server to update internal representation of tenant request object and transition it to tenant
		g.handler.ObjectCreatedOrUpdated(g.tenantRequestObj.DeepCopy())
		// Checking if handler created tenant from request
		_, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), g.tenantRequestObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
	})
}
