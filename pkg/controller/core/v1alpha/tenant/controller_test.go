package tenant

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/access"
	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	crdv1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	antreaversioned "antrea.io/antrea/pkg/client/clientset/versioned"
	antreatestclient "antrea.io/antrea/pkg/client/clientset/versioned/fake"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

type TestGroup struct {
	tenantObj corev1alpha.Tenant
	nodeObj   corev1.Node
}

var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()
var antreaclientset antreaversioned.Interface = antreatestclient.NewSimpleClientset(&crdv1alpha1.ClusterNetworkPolicy{
	ObjectMeta: metav1.ObjectMeta{Namespace: "", Name: "test", UID: "test"},
	Spec: crdv1alpha1.ClusterNetworkPolicySpec{
		Ingress: []crdv1alpha1.Rule{
			{},
		},
	},
})
var edgenetInformerFactory = informers.NewSharedInformerFactory(edgenetclientset, 0)
var c = NewController(kubeclientset,
	edgenetclientset,
	antreaclientset,
	edgenetInformerFactory.Core().V1alpha().Tenants())

func TestMain(m *testing.M) {
	klog.SetOutput(ioutil.Discard)
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)

	flag.String("dir", "../../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, 0)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, 0)

	controller := NewController(kubeclientset,
		edgenetclientset,
		antreaclientset,
		edgenetInformerFactory.Core().V1alpha().Tenants())

	kubeInformerFactory.Start(stopCh)
	edgenetInformerFactory.Start(stopCh)
	go func() {
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	access.Clientset = kubeclientset
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
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+33NUMBER",
			},
			Enabled:              true,
			ClusterNetworkPolicy: false,
		},
	}
	nodeObj := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fr-idf-0000.edge-net.io",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps.edgenet.io/v1alpha",
					Kind:       "Tenant",
					Name:       "edgenet",
					UID:        "edgenet",
				},
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("4Gi"),
				corev1.ResourceCPU:              resource.MustParse("2"),
				corev1.ResourceEphemeralStorage: resource.MustParse("51493088"),
				corev1.ResourcePods:             resource.MustParse("100"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("4Gi"),
				corev1.ResourceCPU:              resource.MustParse("2"),
				corev1.ResourceEphemeralStorage: resource.MustParse("51493088"),
				corev1.ResourcePods:             resource.MustParse("100"),
			},
			Conditions: []corev1.NodeCondition{
				{
					Type:   "Ready",
					Status: "True",
				},
			},
		},
	}
	g.nodeObj = nodeObj
	g.tenantObj = tenantObj
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()

	// Create a tenant
	tenantControllerTest := g.tenantObj.DeepCopy()
	tenantControllerTest.SetName("tenant-controller")

	edgenetclientset.CoreV1alpha().Tenants().Create(context.TODO(), tenantControllerTest, metav1.CreateOptions{})

	// Wait for the status update of the created object
	time.Sleep(250 * time.Millisecond)

	// Get the object and check the status
	tenant, err := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantControllerTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)

	tenant.Spec.Enabled = false
	edgenetclientset.CoreV1alpha().Tenants().Update(context.TODO(), tenant, metav1.UpdateOptions{})
	time.Sleep(250 * time.Millisecond)
	_, err = kubeclientset.RbacV1().Roles(tenant.GetName()).Get(context.TODO(), "edgenet:tenant-owner", metav1.GetOptions{})
	util.Equals(t, "roles.rbac.authorization.k8s.io \"edgenet:tenant-owner\" not found", err.Error())
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()

	tenant := g.tenantObj.DeepCopy()
	tenant.SetName("creation-test")

	edgenetclientset.CoreV1alpha().Tenants().Create(context.TODO(), tenant, metav1.CreateOptions{})
	time.Sleep(250 * time.Millisecond)
	t.Run("owner role configuration", func(t *testing.T) {
		tenant, err := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenant.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		t.Run("cluster role binding", func(t *testing.T) {
			_, err := kubeclientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-owner", tenant.GetName(), tenant.GetName()), metav1.GetOptions{})
			util.OK(t, err)
		})
		t.Run("role binding", func(t *testing.T) {
			_, err := kubeclientset.RbacV1().RoleBindings(tenant.GetName()).Get(context.TODO(), "edgenet:tenant-owner", metav1.GetOptions{})
			util.OK(t, err)
		})
	})
	t.Run("cluster roles", func(t *testing.T) {
		_, err := kubeclientset.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-owner", tenant.GetName(), tenant.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
}

func TestApplyNetworkPolicy(t *testing.T) {
	tenantControllerTest, node := getTestResource()
	tenant, _ := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantControllerTest.GetName(), metav1.GetOptions{})
	tenantUID := string(tenantControllerTest.ObjectMeta.UID)
	clusterNetworkPolicyEnabled := true
	ownerReferences := node.OwnerReferences
	clusterUID := string(node.GetUID())
	err := c.applyNetworkPolicy(tenant.GetName(), tenantUID, clusterUID, clusterNetworkPolicyEnabled, ownerReferences)
	util.OK(t, err)
	// TODO case 2: should fail
}

func TestCreateCoreNamespace(t *testing.T) {
	tennant, node := getTestResource()
	ownerReferences := node.OwnerReferences
	clusterUID := string(node.GetUID())
	err := c.createCoreNamespace(tennant, ownerReferences, clusterUID)
	// Case 1: should success
	util.OK(t, err)
	// Case 2: AlreadyExists, should success
	err = c.createCoreNamespace(tennant, ownerReferences, clusterUID)
	util.OK(t, err)
	// TODO: Case 3: should fail

}

func TestProcessTenant(t *testing.T) {
	tenantCopy, _ := getTestResource()
	tenantCopy.Status.State = "Failure"
	tenantCopy.Spec.Enabled = true
	c.ProcessTenant(tenantCopy)
	tenantCopy.Spec.Enabled = false
	c.ProcessTenant(tenantCopy)
	//TODO how to check if run success?
}

func getTestResource() (*corev1alpha.Tenant, *corev1.Node) {
	g := TestGroup{}
	g.Init()
	// Create a tenant
	tenantCopy := g.tenantObj.DeepCopy()
	tenantCopy.SetName("tenant-controller")
	edgenetclientset.CoreV1alpha().Tenants().Create(context.TODO(), tenantCopy, metav1.CreateOptions{})
	// Wait for the status update of the created object
	time.Sleep(250 * time.Millisecond)
	// Create a node
	node := g.nodeObj
	nodeCopy, _ := kubeclientset.CoreV1().Nodes().Create(context.TODO(), node.DeepCopy(), metav1.CreateOptions{})
	return tenantCopy, nodeCopy
}
