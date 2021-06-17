package emailverification

import (
	"context"
	"flag"
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
	tenant            corev1alpha.Tenant
	tenantRequest     registrationv1alpha.TenantRequest
	userRequest       registrationv1alpha.UserRequest
	emailVerification registrationv1alpha.EmailVerification
	client            kubernetes.Interface
	edgenetClient     versioned.Interface
	handler           Handler
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
	tenant := corev1alpha.Tenant{
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
			Contact: corev1alpha.Contact{
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
			Contact: corev1alpha.Contact{
				Email:     "tom.public@edge-net.org",
				FirstName: "Tom",
				LastName:  "Public",
				Phone:     "+33NUMBER",
				Username:  "tompublic",
			},
		},
	}
	userRequest := registrationv1alpha.UserRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRequest",
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
	emailVerification := registrationv1alpha.EmailVerification{
		TypeMeta: metav1.TypeMeta{
			Kind:       "emailVerification",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "emailverificationcode",
		},
		Spec: registrationv1alpha.EmailVerificationSpec{
			Email:    "john.doe@edge-net.org",
			Verified: false,
		},
	}
	g.tenant = tenant
	g.tenantRequest = tenantRequestObj
	g.userRequest = userRequest
	g.emailVerification = emailVerification
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// tenantHandler := tenant.Handler{}
	// tenantHandler.Init(g.client, g.edgenetClient)
	// Create Tenant
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenant.DeepCopy(), metav1.CreateOptions{})
	//namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.}}
	//namespaceLabels := map[string]string{"owner": "tenant", "owner-name": g.tenant.GetName(), "tenant-name": g.tenant.GetName()}
	//namespace.SetLabels(namespaceLabels)
	//g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	// Create a user as admin on tenant
	// g.edgenetClient.AppsV1alpha().Users(g.user.GetNamespace()).Create(context.TODO(), g.user.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreated to create namespace
	// tenantHandler.ObjectCreated(g.tenant.DeepCopy())
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
	reference := g.emailVerification
	code := "bs" + util.GenerateRandomString(16)
	reference.SetName(code)
	g.edgenetClient.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), reference.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreatedOrUpdated(reference.DeepCopy())
	t.Run("set expiry date", func(t *testing.T) {
		// Handler will update expiration time
		emailVerificationCopy, _ := g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), reference.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(24 * time.Hour),
		}
		util.Equals(t, expected.Day(), emailVerificationCopy.Status.Expiry.Day())
		util.Equals(t, expected.Month(), emailVerificationCopy.Status.Expiry.Month())
		util.Equals(t, expected.Year(), emailVerificationCopy.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		emailVerificationCopy, _ := g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), reference.GetName(), metav1.GetOptions{})
		go g.handler.RunExpiryController()
		emailVerificationCopy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		_, err := g.edgenetClient.RegistrationV1alpha().EmailVerifications().Update(context.TODO(), emailVerificationCopy.DeepCopy(), metav1.UpdateOptions{})
		util.OK(t, err)
		time.Sleep(100 * time.Millisecond)
		_, err = g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), emailVerificationCopy.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
	t.Run("recreate a verified object", func(t *testing.T) {
		recreate := g.emailVerification
		code := "bs" + util.GenerateRandomString(16)
		recreate.SetName(code)
		recreate.Spec.Verified = true
		g.edgenetClient.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), recreate.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreatedOrUpdated(recreate.DeepCopy())
		_, err := g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), recreate.GetName(), metav1.GetOptions{})
		util.Equals(t, false, errors.IsNotFound(err))
		// TODO: Check the status of the relevant object
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
			"email":  {"Email", "johndoe", true},
			"user":   {"User", "johnsmith", true},
			"tenant": {"Tenant", "edgenet-request", true},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				verify := g.emailVerification
				code := "bs" + util.GenerateRandomString(16)
				verify.SetName(code)
				if tc.kind == "Tenant" {
					g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), g.tenantRequest.DeepCopy(), metav1.CreateOptions{})
					defer g.edgenetClient.RegistrationV1alpha().TenantRequests().Delete(context.TODO(), g.tenantRequest.GetName(), metav1.DeleteOptions{})
				} else if tc.kind == "User" {
					g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), g.userRequest.DeepCopy(), metav1.CreateOptions{})
					defer g.edgenetClient.RegistrationV1alpha().UserRequests().Delete(context.TODO(), g.userRequest.GetName(), metav1.DeleteOptions{})
				}
				g.edgenetClient.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), verify.DeepCopy(), metav1.CreateOptions{})
				g.handler.ObjectCreatedOrUpdated(verify.DeepCopy())
				emailVerification, err := g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), verify.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				emailVerification.Spec.Verified = true
				g.handler.ObjectCreatedOrUpdated(emailVerification)
				_, err = g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), emailVerification.GetName(), metav1.GetOptions{})
				util.Equals(t, false, errors.IsNotFound(err))
				// TODO: Check the status of the relevant object
			})
		}
	})

	/*t.Run("dubious", func(t *testing.T) {
		cases := map[string]struct {
			kind     string
			cheat    []string
			expected bool
		}{
			"email/identifier":        {"Email", []string{"Identifier", "joepublic"}, true},
			"user/identifier":         {"User", []string{"Identifier", "tompublic"}, true},
			"tenant/identifier":       {"Tenant", []string{"Identifier", "dubious"}, true},
			"email/kind":              {"Email", []string{"Kind", "User"}, true},
			"user/kind":               {"User", []string{"Kind", "Tenant"}, true},
			"tenant/kind":             {"Tenant", []string{"Kind", "Email"}, true},
			"trustworthy/tenant/kind": {"Tenant", []string{"Kind", "Tenant"}, false},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				dub := g.emailVerification
				code := "bs" + util.GenerateRandomString(16)
				dub.SetName(code)
				if tc.kind == "Tenant" {
					g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), g.tenantRequest.DeepCopy(), metav1.CreateOptions{})
					defer g.edgenetClient.RegistrationV1alpha().TenantRequests().Delete(context.TODO(), g.tenantRequest.GetName(), metav1.DeleteOptions{})
				} else if tc.kind == "User" {
					g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), g.userRequest.DeepCopy(), metav1.CreateOptions{})
					defer g.edgenetClient.RegistrationV1alpha().UserRequests().Delete(context.TODO(), g.userRequest.GetName(), metav1.DeleteOptions{})
				}
				g.edgenetClient.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), dub.DeepCopy(), metav1.CreateOptions{})
				g.handler.ObjectCreatedOrUpdated(g.emailVerification.DeepCopy())
				emailVerification, err := g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), dub.GetName(), metav1.GetOptions{})
				util.OK(t, err)

								if tc.cheat[0] == "Identifier" {
									emailVerification.Spec.Identifier = tc.cheat[1]
									field.identifier = true
								} else if tc.cheat[0] == "Kind" && tc.cheat[1] != emailVerification.Spec.Kind {
									emailVerification.Spec.Kind = tc.cheat[1]
									field.kind = true
								}
								g.handler.ObjectUpdated(emailVerification, field)
				// Handler deletes emailVerification as it is no longer valid
				_, err = g.edgenetClient.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), emailVerification.GetName(), metav1.GetOptions{})
				util.Equals(t, tc.expected, errors.IsNotFound(err))
			})
		}
	})*/
}

func TestCreateEmailVerification(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	cases := map[string]struct {
		input    interface{}
		expected bool
	}{
		"tenant request":            {g.tenantRequest.DeepCopy(), true},
		"user registration request": {g.userRequest.DeepCopy(), true},
		"user wrong obj":            {g.tenant, false},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			status := g.handler.Create(tc.input, []metav1.OwnerReference{})
			util.Equals(t, tc.expected, status)
		})
	}
}
