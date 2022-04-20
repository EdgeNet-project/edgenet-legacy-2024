package nodecontribution

import (
	"io/ioutil"
	"os"

	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"

	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"

	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeinformers "k8s.io/client-go/informers"

	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"

	testclient "k8s.io/client-go/kubernetes/fake"
)

var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()
var kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeclientset, 0)

var edgenetInformerFactory = informers.NewSharedInformerFactory(edgenetclientset, 0)
var nodeInformer coreinformers.NodeInformer = kubeInformerFactory.Core().V1().Nodes()
var nodecontributionInformer = edgenetInformerFactory.Core().V1alpha().NodeContributions()

var c = NewController(
	kubeclientset,
	edgenetclientset,
	nodeInformer,
	nodecontributionInformer)

type testGroup struct {
	tenantObj           corev1alpha.Tenant
	nodeContributionObj corev1alpha.NodeContribution
	nodeObj             corev1.Node
}

func (g *testGroup) Init() {
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
	nodeContributionObj := corev1alpha.NodeContribution{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NodeContribution",
			APIVersion: "core.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenet",
			UID:       "edgenet",
			Namespace: "default",
			// OwnerReferences: []metav1.OwnerReference{
			// 	{
			// 		APIVersion: "core.edgenet.io/v1alpha",
			// 		Kind:       "NodeContribution",
			// 		Name:       "edgenet",
			// 		UID:        "edgenet",
			// 	},
			// },
		},
		Spec: corev1alpha.NodeContributionSpec{
			Tenant:  &tenantObj.Name,
			Host:    "fr-idf-0000.edge-net.io",
			Port:    22,
			User:    "edgenetAdmin",
			Enabled: true,
		},
	}
	nodeObj := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fr-idf-0000.edge-net.io",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps.edgenet.io/v1alpha",
					Kind:       "NodeContribution",
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
	g.nodeContributionObj = nodeContributionObj
	g.nodeObj = nodeObj
}
func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestSetAsOwnerReference(t *testing.T) {
	_, nodeContribution, _ := getTestResource()
	ownerReference := SetAsOwnerReference(nodeContribution)
	flag := false
	for _, ref := range ownerReference {
		if ref.Name == nodeContribution.GetName() && ref.UID == nodeContribution.GetUID() {
			flag = true
		}
	}
	util.Equals(t, true, flag)
}

// TODO: runing failed
/*
panic: ssh: no key found
exit status 2
*/
func TestEnqueueNodeContribution(t *testing.T) {
	_, nodeContribution_1, _ := getTestResource()
	_, nodeContribution_2, _ := getTestResource()
	c.enqueueNodeContribution(nodeContribution_1)
	util.Equals(t, 1, c.workqueue.Len())
	c.enqueueNodeContribution(nodeContribution_2)
	util.Equals(t, 2, c.workqueue.Len())
}

//TODO: to have more testing cases, like more nodes for same nodeContribution, variation of nodeContribution status
// TODO: running failed
/*
panic: ssh: no key found
*/
func TestHandleObject(t *testing.T) {
	_, _, node_1 := getTestResource()
	c.handleObject(node_1)
	util.Equals(t, 1, c.workqueue.Len())
}

func getTestResource() (*corev1alpha.Tenant, *corev1alpha.NodeContribution, *corev1.Node) {
	g := testGroup{}
	g.Init()
	// Create a tenant
	tenantCopy := g.tenantObj.DeepCopy()
	edgenetclientset.CoreV1alpha().Tenants().Create(context.TODO(), tenantCopy, metav1.CreateOptions{})
	// Wait for the status update of the created object
	time.Sleep(250 * time.Millisecond)
	// Create a nodeContribution
	nodeContributionCopy := g.nodeContributionObj.DeepCopy()
	edgenetclientset.CoreV1alpha().NodeContributions().Create(context.TODO(), nodeContributionCopy, metav1.CreateOptions{})
	time.Sleep(250 * time.Millisecond)
	// Create a node
	node := g.nodeObj
	nodeCopy, _ := kubeclientset.CoreV1().Nodes().Create(context.TODO(), node.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(250 * time.Millisecond)
	return tenantCopy, nodeContributionCopy, nodeCopy
}
