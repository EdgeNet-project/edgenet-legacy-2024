package tenant

import (
	"context"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var edgenetInformerFactory = informers.NewSharedInformerFactory(edgenetclientset, 0)
var c = NewController(kubeclientset,
	edgenetclientset,
	antreaclientset,
	edgenetInformerFactory.Core().V1alpha().Tenants())

type UnitTestGroup struct {
	tenantObj corev1alpha.Tenant
	nodeObj   corev1.Node
}

func (g *UnitTestGroup) Init() {
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenet",
			UID:       "edgenet",
			Namespace: "default",
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
			Enabled: true,
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
	g.tenantObj = tenantObj
	g.nodeObj = nodeObj
}

func getTestResource() (*corev1alpha.Tenant, *corev1.Node) {
	g := UnitTestGroup{}
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

//TODO: run failed
// === RUN   TestApplyNetworkPolicy
// --- FAIL: TestApplyNetworkPolicy (0.25s)
// panic: runtime error: invalid memory address or nil pointer dereference [recovered]
//         panic: runtime error: invalid memory address or nil pointer dereference
// [signal 0xc0000005 code=0x0 addr=0x20 pc=0x1247402]

// goroutine 21 [running]:
// testing.tRunner.func1.2({0x13db2a0, 0x2359420})
//         C:/Program Files/Go/src/testing/testing.go:1389 +0x24e
// testing.tRunner.func1()
//         C:/Program Files/Go/src/testing/testing.go:1392 +0x39f
// panic({0x13db2a0, 0x2359420})
//         C:/Program Files/Go/src/runtime/panic.go:838 +0x207
// github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenant.TestApplyNetworkPolicy(0x0?)
//         C:/Users/tfa/Documents/GitHub/edgenet/pkg/controller/core/v1alpha/tenant/unit_test.go:122 +0xc2
// testing.tRunner(0xc000061520, 0x16a5210)
//         C:/Program Files/Go/src/testing/testing.go:1439 +0x102
// created by testing.(*T).Run
//         C:/Program Files/Go/src/testing/testing.go:1486 +0x35f
// exit status 2
func TestApplyNetworkPolicy(t *testing.T) {
	tenantControllerTest, node := getTestResource()
	tenant, _ := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantControllerTest.GetName(), metav1.GetOptions{})
	tenantUID := string(tenantControllerTest.ObjectMeta.UID)
	clusterNetworkPolicyEnabled := true
	ownerReferences := node.OwnerReferences
	clusterUID := string(node.GetUID())
	// case 1: should success
	// TODO: run failed, need bugging
	err := c.applyNetworkPolicy(tenant.GetName(), tenantUID, clusterUID, clusterNetworkPolicyEnabled, ownerReferences)
	util.OK(t, err)
	// case 2: should fail

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
