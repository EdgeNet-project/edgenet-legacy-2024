package rolerequest

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	multitenancy "github.com/EdgeNet-project/edgenet/pkg/multitenancy"
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
	tenantObj      corev1alpha1.Tenant
	roleRequestObj registrationv1alpha1.RoleRequest
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
		edgenetInformerFactory.Registration().V1alpha1().RoleRequests())

	edgenetInformerFactory.Start(stopCh)

	go func() {
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	multitenancyManager := multitenancy.NewManager(kubeclientset, edgenetclientset)
	multitenancyManager.CreateClusterRoles()
	kubeSystemNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), kubeSystemNamespace, metav1.CreateOptions{})

	time.Sleep(500 * time.Millisecond)

	os.Exit(m.Run())
	<-stopCh
}

// Init syncs the test group
func (g *TestGroup) Init() {
	tenantObj := corev1alpha1.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "apps.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: corev1alpha1.TenantSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: corev1alpha1.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha1.Contact{
				Email:     "joe.public@edge-net.org",
				FirstName: "Joe",
				LastName:  "Public",
				Phone:     "+33NUMBER",
			},
			Enabled: true,
		},
	}
	roleRequestObj := registrationv1alpha1.RoleRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleRequest",
			APIVersion: "apps.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "johnsmith",
			Namespace: "edgenet",
		},
		Spec: registrationv1alpha1.RoleRequestSpec{
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
			RoleRef: registrationv1alpha1.RoleRefSpec{
				Kind: "ClusterRole",
				Name: corev1alpha1.TenantOwnerClusterRoleName,
			},
		},
	}
	g.tenantObj = tenantObj
	g.roleRequestObj = roleRequestObj

	edgenetclientset.CoreV1alpha1().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	tenantCoreNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.tenantObj.GetName()}}
	namespaceLabels := map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": g.tenantObj.GetName()}
	tenantCoreNamespace.SetLabels(namespaceLabels)
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), tenantCoreNamespace, metav1.CreateOptions{})
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	roleRequestTest := g.roleRequestObj.DeepCopy()
	roleRequestTest.SetName("role-request-controller-test")

	// Create a role request object
	edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestTest.GetNamespace()).Create(context.TODO(), roleRequestTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	roleRequest, err := edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), roleRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), roleRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), roleRequest.Status.Expiry.Year())

	util.Equals(t, registrationv1alpha1.StatusPending, roleRequest.Status.State)
	util.Equals(t, messagePending, roleRequest.Status.Message)

	roleRequest.Spec.Approved = true
	edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestTest.GetNamespace()).Update(context.TODO(), roleRequest, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	roleRequest, err = edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})

	util.OK(t, err)
	util.Equals(t, registrationv1alpha1.StatusBound, roleRequest.Status.State)
	util.Equals(t, messageRoleBound, roleRequest.Status.Message)
}

func TestTimeout(t *testing.T) {
	g := TestGroup{}
	g.Init()
	roleRequestTest := g.roleRequestObj.DeepCopy()
	roleRequestTest.SetName("role-request-timeout-test")
	edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestTest.GetNamespace()).Create(context.TODO(), roleRequestTest, metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)

	t.Run("set expiry date", func(t *testing.T) {
		roleRequest, _ := edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		util.Equals(t, expected.Day(), roleRequest.Status.Expiry.Day())
		util.Equals(t, expected.Month(), roleRequest.Status.Expiry.Month())
		util.Equals(t, expected.Year(), roleRequest.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		roleRequest, _ := edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})
		roleRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestTest.GetNamespace()).UpdateStatus(context.TODO(), roleRequest, metav1.UpdateOptions{})
		time.Sleep(100 * time.Millisecond)
		_, err := edgenetclientset.RegistrationV1alpha1().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequest.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
}
