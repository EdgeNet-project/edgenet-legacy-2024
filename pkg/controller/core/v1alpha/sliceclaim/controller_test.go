package sliceclaim

import (
	"context"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"

	"github.com/EdgeNet-project/edgenet/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()
var edgenetInformerFactory = informers.NewSharedInformerFactory(edgenetclientset, 0)

var c = NewController(
	kubeclientset,
	edgenetclientset,
	edgenetInformerFactory.Core().V1alpha().SubNamespaces(),
	edgenetInformerFactory.Core().V1alpha().SliceClaims(),
	"Dynamic")

type TestGroup struct {
	sliceClaimObj    corev1alpha.SliceClaim
	subNamespaceObj  corev1alpha.SubNamespace
	resourceQuotaObj corev1.ResourceQuota
	tenantObj        corev1alpha.Tenant
}

func (g *TestGroup) Init() {
	sliceclaimObj := corev1alpha.SliceClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SliceClaim",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sliceclaim-controller-test",
			UID:       "sliceclaim-controller-test",
			Namespace: "edgenet-sub",
		},
		Spec: corev1alpha.SliceClaimSpec{
			SliceClassName: "Slice",
			SliceName:      "slice-controller-test",
			NodeSelector: corev1alpha.NodeSelector{
				Selector: corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "edgenet",
									Operator: corev1.NodeSelectorOpLt,
									Values:   []string{"1"},
								},
							},
							MatchFields: []corev1.NodeSelectorRequirement{},
						},
					},
				},
				Count: 2,
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory:           resource.MustParse("4Gi"),
						corev1.ResourceCPU:              resource.MustParse("2"),
						corev1.ResourceEphemeralStorage: resource.MustParse("51493088"),
						corev1.ResourcePods:             resource.MustParse("100"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory:           resource.MustParse("2Gi"),
						corev1.ResourceCPU:              resource.MustParse("1"),
						corev1.ResourceEphemeralStorage: resource.MustParse("25746544"),
						corev1.ResourcePods:             resource.MustParse("50"),
					},
				},
			},
			SliceExpiry: &metav1.Time{
				Time: time.Now().Add(72 * time.Hour),
			},
		},
	}
	subNamespaceObj := corev1alpha.SubNamespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SubNamespace",
			APIVersion: "core.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenet-sub",
			Namespace: "edgenet",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps.edgenet.io/v1alpha",
					Kind:       "SliceClaim",
					Name:       "sliceclaim-controller-test",
					UID:        "sliceclaim-controller-test",
				},
			},
		},
		Spec: corev1alpha.SubNamespaceSpec{
			Workspace: &corev1alpha.Workspace{
				ResourceAllocation: map[corev1.ResourceName]resource.Quantity{
					"cpu":    resource.MustParse("6000m"),
					"memory": resource.MustParse("6Gi"),
				},
				Inheritance: map[string]bool{
					"rbac":           true,
					"networkpolicy":  true,
					"limitrange":     false,
					"configmap":      false,
					"secret":         false,
					"serviceaccount": false,
				},
				Scope: "local",
			},
		},
	}
	resourceQuotaObj := corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: "core-quota",
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: map[corev1.ResourceName]resource.Quantity{
				"cpu":              resource.MustParse("8000m"),
				"memory":           resource.MustParse("8192Mi"),
				"requests.storage": resource.MustParse("8Gi"),
			},
		},
	}
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "core.edgenet.io/v1alpha",
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
			Enabled: true,
		},
	}
	g.sliceClaimObj = sliceclaimObj
	g.subNamespaceObj = subNamespaceObj
	g.resourceQuotaObj = resourceQuotaObj
	g.tenantObj = tenantObj
}

func getTestResource() (*corev1alpha.SliceClaim, *corev1alpha.SubNamespace, *corev1.ResourceQuota) {
	g := TestGroup{}
	g.Init()
	sliceClaimTest := g.sliceClaimObj.DeepCopy()
	subNamespaceTest := g.subNamespaceObj.DeepCopy()
	resourceQuotaTest := g.resourceQuotaObj.DeepCopy()
	// Create a test object
	edgenetclientset.CoreV1alpha().SliceClaims("edgenet-sub").Create(context.TODO(), sliceClaimTest, metav1.CreateOptions{})
	edgenetclientset.CoreV1alpha().SubNamespaces("edgenet-sub").Create(context.TODO(), subNamespaceTest, metav1.CreateOptions{})

	tenantCoreNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.tenantObj.GetName()}}
	namespaceLabels := map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": g.tenantObj.GetName()}
	tenantCoreNamespace.SetLabels(namespaceLabels)
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), tenantCoreNamespace, metav1.CreateOptions{})
	kubeclientset.CoreV1().ResourceQuotas(tenantCoreNamespace.GetName()).Create(context.TODO(), resourceQuotaTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(250 * time.Millisecond)
	// Get the object and check the status
	sliceClaimTestObj, _ := edgenetclientset.CoreV1alpha().SliceClaims("edgenet-sub").Get(context.TODO(), sliceClaimTest.GetName(), metav1.GetOptions{})
	subNamespaceTestObj, _ := edgenetclientset.CoreV1alpha().SubNamespaces("edgenet-sub").Get(context.TODO(), subNamespaceTest.GetName(), metav1.GetOptions{})
	resourceQuotaTestObj, _ := kubeclientset.CoreV1().ResourceQuotas(tenantCoreNamespace.GetName()).Get(context.TODO(), resourceQuotaTest.GetName(), metav1.GetOptions{})
	return sliceClaimTestObj, subNamespaceTestObj, resourceQuotaTestObj
}

func TestEnqueueSliceClaim(t *testing.T) {
	sliceClaimTestObj, _, _ := getTestResource()
	util.Equals(t, 0, c.workqueue.Len())
	c.enqueueSliceClaim(sliceClaimTestObj)
	util.Equals(t, 1, c.workqueue.Len())
}
func TestProcessNextWorkItem(t *testing.T) {
	sliceClaimTestObj, _, _ := getTestResource()
	c.enqueueSliceClaim(sliceClaimTestObj)
	c.processNextWorkItem()
	util.Equals(t, 0, c.workqueue.Len())
}

//TODO: Status.State/Message checking for equals
func TestSyncHandler(t *testing.T) {
	key := "edgenet-sub/sliceclaim-controller-test"
	sliceClaimTestObj, _, _ := getTestResource()
	sliceClaimTestObj.Status.State = pending
	sliceClaimTestObj.Status.Message = dynamic
	c.enqueueSliceClaim(sliceClaimTestObj)
	err := c.syncHandler(key)
	util.OK(t, err)
	util.Equals(t, pending, sliceClaimTestObj.Status.State)
	util.Equals(t, dynamic, sliceClaimTestObj.Status.Message)
}

//TODO: test failed
func TestHandleSubNamespace(t *testing.T) {
	_, subNamespaceTestObj, _ := getTestResource()
	util.Equals(t, 0, c.workqueue.Len())
	c.handleSubNamespace(subNamespaceTestObj)
	util.Equals(t, 1, c.workqueue.Len())
}

//TODO: test failure
func TestProcessSliceClaim(t *testing.T) {
	sliceClaimTestObj, _, _ := getTestResource()
	sliceClaimTestObj.Status.State = pending
	sliceClaimTestObj.Status.Message = dynamic
	c.enqueueSliceClaim(sliceClaimTestObj)
	c.processSliceClaim(sliceClaimTestObj)
	util.Equals(t, pending, sliceClaimTestObj.Status.State)
	util.Equals(t, dynamic, sliceClaimTestObj.Status.Message)
}

func TestClaimSlice(t *testing.T) {
	updated := make(chan bool, 1)
	sliceClaimTestObj, _, _ := getTestResource()
	done := c.claimSlice(sliceClaimTestObj, updated)
	util.Equals(t, true, done)
}

func TestSetAsOwnerReference(t *testing.T) {
	sliceClaimTestObj, _, _ := getTestResource()
	done := c.setAsOwnerReference(sliceClaimTestObj)
	util.Equals(t, false, done)
	// sliceClaimName := sliceClaimTestObj.GetName()
	// subNamespaceObj.Spec.Workspace.SliceClaim = &sliceClaimName	//TODO: panic: runtime error: invalid memory address or nil pointer dereference
	// done = c.setAsOwnerReference(sliceClaimTestObj)
	// util.Equals(t, true, done)
}

func TestTuneParentResourceQuota(t *testing.T) {
	sliceClaimTestObj, _, resourceQuotaTestObj := getTestResource()
	done := c.tuneParentResourceQuota(sliceClaimTestObj, resourceQuotaTestObj)
	util.Equals(t, true, done)
	sliceClaimTestObj.Spec.NodeSelector.Count = 3
	done = c.tuneParentResourceQuota(sliceClaimTestObj, resourceQuotaTestObj)
	util.Equals(t, false, done)
}
