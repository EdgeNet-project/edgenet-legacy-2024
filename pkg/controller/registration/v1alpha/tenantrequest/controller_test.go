package tenantrequest

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

type TestGroup struct {
	tenantObj        corev1alpha.Tenant
	tenantRequestObj registrationv1alpha.TenantRequest
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
			edgenetInformerFactory.Registration().V1alpha().TenantRequests())

		edgenetInformerFactory.Start(stopCh)
		controller = newController
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	access.Clientset = kubeclientset
	access.CreateClusterRoles()
	kubeSystemNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), kubeSystemNamespace, metav1.CreateOptions{})

	os.Exit(m.Run())
	<-stopCh
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
	g.tenantObj = tenantObj
	g.tenantRequestObj = tenantRequestObj
	edgenetclientset.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	tenantRequestTest := g.tenantRequestObj.DeepCopy()
	tenantRequestTest.SetName("tenant-request-controller-test")

	// Create a tenant request
	edgenetclientset.RegistrationV1alpha().TenantRequests().Create(context.TODO(), tenantRequestTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	tenantRequest, err := edgenetclientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), tenantRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), tenantRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), tenantRequest.Status.Expiry.Year())

	util.Equals(t, pending, tenantRequest.Status.State)
	util.Equals(t, messageAUPNotAgreed, tenantRequest.Status.Message)

	tenantRequest.Spec.Contact.Email = "different-email@edge-net.org"
	edgenetclientset.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	tenantRequest, err = edgenetclientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, pending, tenantRequest.Status.State)
	util.Equals(t, messageAUPNotAgreed, tenantRequest.Status.Message)
	// Approve a tenant request
	if acceptableUsePolicyRaw, err := edgenetclientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
			if acceptableUsePolicyRow.Spec.Email == tenantRequest.Spec.Contact.Email {
				acceptableUsePolicyRow.Spec.Accepted = true
				edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), &acceptableUsePolicyRow, metav1.UpdateOptions{})
			}
		}
	}
	tenantRequest.Spec.Approved = true
	edgenetclientset.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	tenantRequest, err = edgenetclientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, approved, tenantRequest.Status.State)
	util.Equals(t, messageRoleApproved, tenantRequest.Status.Message)
}

func TestTimeout(t *testing.T) {
	g := TestGroup{}
	g.Init()
	tenantRequestTest := g.tenantRequestObj.DeepCopy()
	tenantRequestTest.SetName("tenant-request-timeout-test")
	edgenetclientset.RegistrationV1alpha().TenantRequests().Create(context.TODO(), tenantRequestTest, metav1.CreateOptions{})
	time.Sleep(500 * time.Millisecond)

	t.Run("set expiry date", func(t *testing.T) {
		tenantRequest, _ := edgenetclientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		util.Equals(t, expected.Day(), tenantRequest.Status.Expiry.Day())
		util.Equals(t, expected.Month(), tenantRequest.Status.Expiry.Month())
		util.Equals(t, expected.Year(), tenantRequest.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		tenantRequest, _ := edgenetclientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
		tenantRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		_, err := edgenetclientset.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
		util.OK(t, err)
		time.Sleep(100 * time.Millisecond)
		_, err = edgenetclientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequest.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	tenantRequestTest := g.tenantRequestObj.DeepCopy()
	tenantRequestTest.SetName("tenant-request-approval-test")
	edgenetclientset.RegistrationV1alpha().TenantRequests().Create(context.TODO(), tenantRequestTest, metav1.CreateOptions{})
	time.Sleep(500 * time.Millisecond)

	t.Run("approval", func(t *testing.T) {
		// Updating tenant request status to approved
		if acceptableUsePolicyRaw, err := edgenetclientset.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{}); err == nil {
			for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
				if acceptableUsePolicyRow.Spec.Email == tenantRequestTest.Spec.Contact.Email {
					acceptableUsePolicyRow.Spec.Accepted = true
					edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), &acceptableUsePolicyRow, metav1.UpdateOptions{})
				}
			}
		}
		tenantRequestTest.Spec.Approved = true
		edgenetclientset.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequestTest, metav1.UpdateOptions{})
		// Requesting server to update internal representation of tenant request object and transition it to tenant
		time.Sleep(500 * time.Millisecond)
		// Checking if handler created tenant from request
		_, err := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
		util.OK(t, err)
	})
}
