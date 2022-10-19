package slice

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
	edgenetInformerFactory.Core().V1alpha().SliceClaims(),
	edgenetInformerFactory.Core().V1alpha().Slices())

type TestGroup struct {
	sliceObj      corev1alpha.Slice
	sliceClaimObj corev1alpha.SliceClaim
}

func (g *TestGroup) Init() {
	sliceObj := corev1alpha.Slice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Slice",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenet",
			UID:       "edgenet",
			Namespace: "default",
		},
		Spec: corev1alpha.SliceSpec{
			SliceClassName: "Slice",
			ClaimRef: &corev1.ObjectReference{
				Kind:            "Sclice",
				Namespace:       "default",
				Name:            "slice-controller-test",
				UID:             "slice-controller-test",
				APIVersion:      "apps.edgenet.io/v1alpha",
				ResourceVersion: "",
				FieldPath:       "",
			},
			NodeSelector: corev1alpha.NodeSelector{
				Selector: corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "edgenet",
									Operator: corev1.NodeSelectorOpExists,
									Values:   []string{},
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
						corev1.ResourcePods:             resource.MustParse("100"),
					},
				},
			},
		},
	}
	sliceclaimObj := corev1alpha.SliceClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SliceClaim",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenet",
			UID:       "edgenet",
			Namespace: "default",
		},
		Spec: corev1alpha.SliceClaimSpec{
			SliceClassName: "Slice",
			SliceName:      "lice-controller-test",
			NodeSelector: corev1alpha.NodeSelector{
				Selector: corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "edgenet",
									Operator: corev1.NodeSelectorOpExists,
									Values:   []string{},
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
	g.sliceObj = sliceObj
	g.sliceClaimObj = sliceclaimObj
}

func getTestResource() (*corev1alpha.Slice, *corev1alpha.SliceClaim) {
	g := TestGroup{}
	g.Init()
	sliceTest := g.sliceObj.DeepCopy()
	sliceClaimTest := g.sliceClaimObj.DeepCopy()
	// Create a test object
	edgenetclientset.CoreV1alpha().Slices().Create(context.TODO(), sliceTest, metav1.CreateOptions{})
	edgenetclientset.CoreV1alpha().SliceClaims("default").Create(context.TODO(), sliceClaimTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(250 * time.Millisecond)
	// Get the object and check the status
	sliceTestObj, _ := edgenetclientset.CoreV1alpha().Slices().Get(context.TODO(), sliceTest.GetName(), metav1.GetOptions{})
	sliceClaimTestObj, _ := edgenetclientset.CoreV1alpha().SliceClaims("default").Get(context.TODO(), sliceClaimTest.GetName(), metav1.GetOptions{})
	return sliceTestObj, sliceClaimTestObj
}

func TestEnqueueSlice(t *testing.T) {
	sliceTestObj1, _ := getTestResource()
	sliceTestObj2, _ := getTestResource()
	c.enqueueSlice(sliceTestObj1)
	util.Equals(t, 1, c.workqueue.Len())
	c.enqueueSlice(sliceTestObj2)
	util.Equals(t, 2, c.workqueue.Len())
}

func TestEnqueueSliceAfter(t *testing.T) {
	sliceTestObj, _ := getTestResource()
	c.enqueueSliceAfter(sliceTestObj, 10*time.Millisecond)
	util.Equals(t, 1, c.workqueue.Len())
	time.Sleep(250 * time.Millisecond)
	util.Equals(t, 0, c.workqueue.Len())
}
