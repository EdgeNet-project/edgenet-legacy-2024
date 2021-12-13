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
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

// The main structure of test group
type TestGroup struct {
	tenant            corev1alpha.Tenant
	tenantRequest     registrationv1alpha.TenantRequest
	userRequest       registrationv1alpha.UserRequest
	emailVerification registrationv1alpha.EmailVerification
}

var controller *Controller
var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()

func TestMain(m *testing.M) {
	klog.SetOutput(ioutil.Discard)
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)

	flag.String("dir", "../../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	go func() {
		edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

		newController := NewController(kubeclientset,
			edgenetclientset,
			edgenetInformerFactory.Registration().V1alpha().EmailVerifications())

		edgenetInformerFactory.Start(stopCh)
		controller = newController
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	os.Exit(m.Run())
	<-stopCh
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

	// Create Tenant
	edgenetclientset.CoreV1alpha().Tenants().Create(context.TODO(), g.tenant.DeepCopy(), metav1.CreateOptions{})
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()

	// Create a emailVerification object
	edgenetclientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), g.emailVerification.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	emailVerification, _ := edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), g.emailVerification.GetName(), metav1.GetOptions{})
	util.NotEquals(t, nil, emailVerification.Status.Expiry)
	// Update an emailVerification
	g.emailVerification.Spec.Verified = true
	edgenetclientset.RegistrationV1alpha().EmailVerifications().Update(context.TODO(), g.emailVerification.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	_, err := edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), g.emailVerification.GetName(), metav1.GetOptions{})
	util.Equals(t, false, errors.IsNotFound(err))
	// TODO: Check the status of the relevant object
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()

	reference := g.emailVerification
	code := "bs" + util.GenerateRandomString(16)
	reference.SetName(code)
	edgenetclientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), reference.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	t.Run("set expiry date", func(t *testing.T) {
		// Handler will update expiration time
		emailVerificationCopy, _ := edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), reference.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(24 * time.Hour),
		}
		util.Equals(t, expected.Day(), emailVerificationCopy.Status.Expiry.Day())
		util.Equals(t, expected.Month(), emailVerificationCopy.Status.Expiry.Month())
		util.Equals(t, expected.Year(), emailVerificationCopy.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		emailVerificationCopy, _ := edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), reference.GetName(), metav1.GetOptions{})
		emailVerificationCopy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		_, err := edgenetclientset.RegistrationV1alpha().EmailVerifications().Update(context.TODO(), emailVerificationCopy.DeepCopy(), metav1.UpdateOptions{})
		util.OK(t, err)
		time.Sleep(100 * time.Millisecond)
		_, err = edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), emailVerificationCopy.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
	t.Run("recreate a verified object", func(t *testing.T) {
		recreate := g.emailVerification
		code := "bs" + util.GenerateRandomString(16)
		recreate.SetName(code)
		recreate.Spec.Verified = true
		edgenetclientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), recreate.DeepCopy(), metav1.CreateOptions{})
		time.Sleep(time.Millisecond * 500)
		_, err := edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), recreate.GetName(), metav1.GetOptions{})
		util.Equals(t, false, errors.IsNotFound(err))
		// TODO: Check the status of the relevant object
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
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
					edgenetclientset.RegistrationV1alpha().TenantRequests().Create(context.TODO(), g.tenantRequest.DeepCopy(), metav1.CreateOptions{})
					defer edgenetclientset.RegistrationV1alpha().TenantRequests().Delete(context.TODO(), g.tenantRequest.GetName(), metav1.DeleteOptions{})
				} else if tc.kind == "User" {
					edgenetclientset.RegistrationV1alpha().UserRequests().Create(context.TODO(), g.userRequest.DeepCopy(), metav1.CreateOptions{})
					defer edgenetclientset.RegistrationV1alpha().UserRequests().Delete(context.TODO(), g.userRequest.GetName(), metav1.DeleteOptions{})
				}
				edgenetclientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), verify.DeepCopy(), metav1.CreateOptions{})
				time.Sleep(time.Millisecond * 500)
				emailVerification, err := edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), verify.GetName(), metav1.GetOptions{})
				util.OK(t, err)
				emailVerification.Spec.Verified = true
				time.Sleep(time.Millisecond * 500)
				_, err = edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), emailVerification.GetName(), metav1.GetOptions{})
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
					edgenetclientset.RegistrationV1alpha().TenantRequests().Create(context.TODO(), g.tenantRequest.DeepCopy(), metav1.CreateOptions{})
					defer edgenetclientset.RegistrationV1alpha().TenantRequests().Delete(context.TODO(), g.tenantRequest.GetName(), metav1.DeleteOptions{})
				} else if tc.kind == "User" {
					edgenetclientset.RegistrationV1alpha().UserRequests().Create(context.TODO(), g.userRequest.DeepCopy(), metav1.CreateOptions{})
					defer edgenetclientset.RegistrationV1alpha().UserRequests().Delete(context.TODO(), g.userRequest.GetName(), metav1.DeleteOptions{})
				}
				edgenetclientset.RegistrationV1alpha().EmailVerifications().Create(context.TODO(), dub.DeepCopy(), metav1.CreateOptions{})
				g.handler.ObjectCreatedOrUpdated(g.emailVerification.DeepCopy())
				emailVerification, err := edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), dub.GetName(), metav1.GetOptions{})
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
				_, err = edgenetclientset.RegistrationV1alpha().EmailVerifications().Get(context.TODO(), emailVerification.GetName(), metav1.GetOptions{})
				util.Equals(t, tc.expected, errors.IsNotFound(err))
			})
		}
	})*/
}
