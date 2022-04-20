package rolerequest

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
	tenantObj      corev1alpha.Tenant
	roleRequestObj registrationv1alpha.RoleRequest
}

var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()
var edgenetInformerFactory = informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

var c = NewController(kubeclientset,
	edgenetclientset,
	edgenetInformerFactory.Registration().V1alpha().RoleRequests())

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
		edgenetInformerFactory.Registration().V1alpha().RoleRequests())

	edgenetInformerFactory.Start(stopCh)

	go func() {
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	access.Clientset = kubeclientset
	access.CreateClusterRoles()
	kubeSystemNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), kubeSystemNamespace, metav1.CreateOptions{})

	time.Sleep(500 * time.Millisecond)

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
				Email:     "joe.public@edge-net.org",
				FirstName: "Joe",
				LastName:  "Public",
				Phone:     "+33NUMBER",
			},
			Enabled: true,
		},
	}
	roleRequestObj := registrationv1alpha.RoleRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "johnsmith",
			Namespace: "edgenet",
		},
		Spec: registrationv1alpha.RoleRequestSpec{
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
			RoleRef: registrationv1alpha.RoleRefSpec{
				Kind: "ClusterRole",
				Name: "edgenet:tenant-admin",
			},
		},
	}
	g.tenantObj = tenantObj
	g.roleRequestObj = roleRequestObj

	edgenetclientset.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
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
	edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Create(context.TODO(), roleRequestTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	roleRequest, err := edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), roleRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), roleRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), roleRequest.Status.Expiry.Year())

	util.Equals(t, pending, roleRequest.Status.State)
	util.Equals(t, messageRoleNotApproved, roleRequest.Status.Message)

	roleRequest.Spec.Approved = true
	edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Update(context.TODO(), roleRequest, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	roleRequest, err = edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})

	util.OK(t, err)
	util.Equals(t, approved, roleRequest.Status.State)
	util.Equals(t, messageRoleApproved, roleRequest.Status.Message)
}

func TestTimeout(t *testing.T) {
	g := TestGroup{}
	g.Init()
	roleRequestTest := g.roleRequestObj.DeepCopy()
	roleRequestTest.SetName("role-request-timeout-test")
	edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Create(context.TODO(), roleRequestTest, metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)

	t.Run("set expiry date", func(t *testing.T) {
		roleRequest, _ := edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})
		expected := metav1.Time{
			Time: time.Now().Add(72 * time.Hour),
		}
		util.Equals(t, expected.Day(), roleRequest.Status.Expiry.Day())
		util.Equals(t, expected.Month(), roleRequest.Status.Expiry.Month())
		util.Equals(t, expected.Year(), roleRequest.Status.Expiry.Year())
	})
	t.Run("timeout", func(t *testing.T) {
		roleRequest, _ := edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})
		roleRequest.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).UpdateStatus(context.TODO(), roleRequest, metav1.UpdateOptions{})
		time.Sleep(100 * time.Millisecond)
		_, err := edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequest.GetName(), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
}

//TODO: test failed
func TestCheckForRequestedRole(t *testing.T) {
	roleRequest := getTestResource("role-request-controller-test")
	ret := c.checkForRequestedRole(roleRequest)
	util.Equals(t, ret, true)
	roleRequest.Spec.RoleRef.Kind = "Role"
	ret = c.checkForRequestedRole(roleRequest)
	util.Equals(t, ret, true)
	roleRequest.Spec.RoleRef.Kind = "Illegal"
	ret = c.checkForRequestedRole(roleRequest)
	util.Equals(t, ret, false)
}

// TODO: more testcases, e.g. for variation situation of roleRequestCopy.Spec.Approved=true
func TestProcessRoleRequest(t *testing.T) {
	roleRequest := getTestResource("role-request-controller-test")
	roleRequest.Status.Expiry = nil
	c.processRoleRequest(roleRequest)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), roleRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), roleRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), roleRequest.Status.Expiry.Year())

	roleRequest.Status.Expiry = &metav1.Time{
		Time: time.Now(),
	}
	c.processRoleRequest(roleRequest)
	_, err := c.edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequest.GetNamespace()).Get(context.TODO(), roleRequest.GetName(), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))

	roleRequest = getTestResource("role-request-controller-test")
	roleRequest.Spec.Approved = false
	c.processRoleRequest(roleRequest)
	util.Equals(t, pending, roleRequest.Status.State)
	util.Equals(t, messageRoleNotApproved, roleRequest.Status.Message)

	roleRequest.Spec.Approved = true
	c.processRoleRequest(roleRequest)
	util.Equals(t, approved, roleRequest.Status.State)
	util.Equals(t, messageRoleApproved, roleRequest.Status.Message)
}

func TestEnqueueRoleRequestAfter(t *testing.T) {
	roleRequest := getTestResource("role-request-controller-test")
	c.enqueueRoleRequestAfter(roleRequest, 10*time.Millisecond)
	util.Equals(t, 1, c.workqueue.Len())
	time.Sleep(250 * time.Millisecond)
	util.Equals(t, 0, c.workqueue.Len())
}

func TestEnqueueRoleRequest(t *testing.T) {
	roleRequest_1 := getTestResource("role-request-controller-test-1")
	roleRequest_2 := getTestResource("role-request-controller-test-2")

	c.enqueueRoleRequest(roleRequest_1)
	util.Equals(t, 1, c.workqueue.Len())
	c.enqueueRoleRequest(roleRequest_2)
	util.Equals(t, 2, c.workqueue.Len())
}

//TODO: test failed
func TestSyncHandler(t *testing.T) {
	key := "default/role-request-controller-test"
	roleRequest := getTestResource("role-request-controller-test")
	roleRequest.Status.State = pending
	err := c.syncHandler(key)
	util.OK(t, err)
	util.Equals(t, approved, roleRequest.Status.State)
}

// TODO: More test cases
func TestProcessNextWorkItem(t *testing.T) {
	roleRequest := getTestResource("role-request-controller-test")
	c.enqueueRoleRequest(roleRequest)
	util.Equals(t, 1, c.workqueue.Len())
	c.processNextWorkItem()
	util.Equals(t, 0, c.workqueue.Len())
}

func getTestResource(name string) *registrationv1alpha.RoleRequest {
	g := TestGroup{}
	g.Init()
	roleRequestTest := g.roleRequestObj.DeepCopy()
	roleRequestTest.SetName(name)

	// Create a role request object
	edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Create(context.TODO(), roleRequestTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	roleRequest, _ := edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})
	return roleRequest
}
