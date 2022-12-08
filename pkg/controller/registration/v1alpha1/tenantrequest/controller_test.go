package tenantrequest

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

type TestGroup struct {
	tenantRequestObj registrationv1alpha1.TenantRequest
}

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

	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

	controller := NewController(kubeclientset,
		edgenetclientset,
		edgenetInformerFactory.Registration().V1alpha1().TenantRequests())

	edgenetInformerFactory.Start(stopCh)

	go func() {
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	kubeSystemNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), kubeSystemNamespace, metav1.CreateOptions{})

	time.Sleep(500 * time.Millisecond)

	os.Exit(m.Run())
	<-stopCh
}

// Init syncs the test group
func (g *TestGroup) Init() {
	tenantRequestObj := registrationv1alpha1.TenantRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "tenantRequest",
			APIVersion: "apps.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet-request",
			UID:  "requestUID",
		},
		Spec: registrationv1alpha1.TenantRequestSpec{
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
			},
			ResourceAllocation: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("12000m"),
				corev1.ResourceMemory: resource.MustParse("12Gi"),
			},
		},
	}
	g.tenantRequestObj = tenantRequestObj
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	tenantRequestTest := g.tenantRequestObj.DeepCopy()
	tenantRequestTest.SetName("tenant-request-controller-test")
	tenantRequestTest.SetUID("tenant-request-controller-test")

	// Create a tenant request
	edgenetclientset.RegistrationV1alpha1().TenantRequests().Create(context.TODO(), tenantRequestTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(250 * time.Millisecond)
	// Get the object and check the status
	tenantRequest, err := edgenetclientset.RegistrationV1alpha1().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), tenantRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), tenantRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), tenantRequest.Status.Expiry.Year())

	util.Equals(t, registrationv1alpha1.StatusPending, tenantRequest.Status.State)
	util.Equals(t, messagePending, tenantRequest.Status.Message)

	tenantRequest.Spec.Approved = true
	edgenetclientset.RegistrationV1alpha1().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantRequest.GetName()}}, metav1.CreateOptions{})
	time.Sleep(250 * time.Millisecond)
	tenantRequest, err = edgenetclientset.RegistrationV1alpha1().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, registrationv1alpha1.StatusCreated, tenantRequest.Status.State)
	util.Equals(t, messageCreated, tenantRequest.Status.Message)
}

func TestTimeout(t *testing.T) {
	g := TestGroup{}
	g.Init()
	tenantRequestTest := g.tenantRequestObj.DeepCopy()
	tenantRequestTest.SetName("tenant-request-timeout-test")
	tenantRequestTest.SetUID("tenant-request-timeout-test")

	edgenetclientset.RegistrationV1alpha1().TenantRequests().Create(context.TODO(), tenantRequestTest, metav1.CreateOptions{})
	time.Sleep(250 * time.Millisecond)

	t.Run("set expiry date", func(t *testing.T) {
		tenantRequest, _ := edgenetclientset.RegistrationV1alpha1().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		util.Equals(t, expected.Day(), tenantRequest.Status.Expiry.Day())
		util.Equals(t, expected.Month(), tenantRequest.Status.Expiry.Month())
		util.Equals(t, expected.Year(), tenantRequest.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		tenantRequest, _ := edgenetclientset.RegistrationV1alpha1().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
		tenantRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		_, err := edgenetclientset.RegistrationV1alpha1().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
		util.OK(t, err)
		time.Sleep(250 * time.Millisecond)
		_, err = edgenetclientset.RegistrationV1alpha1().TenantRequests().Get(context.TODO(), tenantRequest.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	tenantRequestTest := g.tenantRequestObj.DeepCopy()
	tenantRequestTest.SetName("tenant-request-approval-test")
	tenantRequestTest.SetUID("tenant-request-approval-test")

	edgenetclientset.RegistrationV1alpha1().TenantRequests().Create(context.TODO(), tenantRequestTest, metav1.CreateOptions{})
	time.Sleep(250 * time.Millisecond)

	t.Run("approval", func(t *testing.T) {
		tenantRequest, err := edgenetclientset.RegistrationV1alpha1().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		tenantRequest.Spec.Approved = true
		edgenetclientset.RegistrationV1alpha1().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
		// Requesting server to update internal representation of tenant request object and transition it to tenant
		time.Sleep(250 * time.Millisecond)
		// Checking if handler created tenant from request
		_, err = edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), tenantRequest.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		kubeclientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantRequestTest.GetName()}}, metav1.CreateOptions{})
		time.Sleep(250 * time.Millisecond)
		t.Run("tenant resource quota", func(t *testing.T) {
			tenantResourceQuota, err := edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, tenantRequestTest.Spec.ResourceAllocation, tenantResourceQuota.Spec.Claim["initial"].ResourceList)
		})
	})
}
