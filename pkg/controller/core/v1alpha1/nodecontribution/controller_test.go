package nodecontribution

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

// The main structure of test group
type TestGroup struct {
	tenantResourceQuotaObj corev1alpha.TenantResourceQuota
	claimObj               corev1alpha.ResourceTuning
	nodeObj                corev1.Node
}

var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()

func TestMain(m *testing.M) {
	klog.SetOutput(ioutil.Discard)
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)

	stopCh := signals.SetupSignalHandler()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

	controller := NewController(kubeclientset,
		edgenetclientset,
		kubeInformerFactory.Core().V1().Nodes(),
		edgenetInformerFactory.Core().V1alpha1().NodeContributions(), "", "")

	kubeInformerFactory.Start(stopCh)
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

func (g *TestGroup) Init() {
	tenantResourceQuotaObj := corev1alpha.TenantResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "tenantResourceQuota",
			APIVersion: "apps.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
			UID:  "trq",
		},
	}
	claimObj := corev1alpha.ResourceTuning{
		ResourceList: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("12000m"),
			corev1.ResourceMemory: resource.MustParse("12Gi"),
		},
	}
	nodeObj := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fr-idf-0000.edge-net.io",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps.edgenet.io/v1alpha1",
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
	g.tenantResourceQuotaObj = tenantResourceQuotaObj
	g.claimObj = claimObj
	g.nodeObj = nodeObj
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()

	randomString := util.GenerateRandomString(6)
	// Create a resource request
	tenantResourceQuotaObj := g.tenantResourceQuotaObj
	tenantResourceQuotaObj.SetName(randomString)
	tenantResourceQuotaObj.SetUID(types.UID(randomString))
	tenantResourceQuotaObj.Spec.Claim = make(map[string]corev1alpha.ResourceTuning)
	tenantResourceQuotaObj.Spec.Claim["initial"] = g.claimObj
	edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuotaObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(250 * time.Millisecond)
	tenantResourceQuota, err := edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)

	expectedMemoryRes := g.claimObj.ResourceList["memory"]
	expectedMemory := expectedMemoryRes.Value()
	expectedMemoryRew := expectedMemory + int64(float64(g.nodeObj.Status.Capacity.Memory().Value())*1.3)
	expectedCPURes := g.claimObj.ResourceList["cpu"]
	expectedCPU := expectedCPURes.Value()
	expectedCPURew := expectedCPU + int64(float64(g.nodeObj.Status.Capacity.Cpu().Value())*1.5)

	node := g.nodeObj
	node.OwnerReferences[0].Name = randomString
	nodeCopy, _ := kubeclientset.CoreV1().Nodes().Create(context.TODO(), node.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(250 * time.Millisecond)
	tenantResourceQuota, err = edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	reward := false
	if _, claimExists := tenantResourceQuota.Spec.Claim[nodeCopy.GetName()]; claimExists {
		reward = true
	}
	util.Equals(t, true, reward)
	cpuQuota, memoryQuota := getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemoryRew, memoryQuota)
	util.Equals(t, expectedCPURew, cpuQuota)

	nodeCopy.Status.Conditions[0].Status = "False"
	kubeclientset.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(250 * time.Millisecond)
	tenantResourceQuota, err = edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemory, memoryQuota)
	util.Equals(t, expectedCPU, cpuQuota)

	nodeCopy.Status.Conditions[0].Status = "True"
	kubeclientset.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(250 * time.Millisecond)
	tenantResourceQuota, err = edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemoryRew, memoryQuota)
	util.Equals(t, expectedCPURew, cpuQuota)

	nodeCopy.Status.Conditions[0].Status = "Unknown"
	kubeclientset.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(250 * time.Millisecond)
	tenantResourceQuota, err = edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemory, memoryQuota)
	util.Equals(t, expectedCPU, cpuQuota)

	kubeclientset.CoreV1().Nodes().Delete(context.TODO(), nodeCopy.GetName(), metav1.DeleteOptions{})
	time.Sleep(250 * time.Millisecond)
	tenantResourceQuota, err = edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemory, memoryQuota)
	util.Equals(t, expectedCPU, cpuQuota)
}

func getQuotas(claimRaw map[string]corev1alpha.ResourceTuning) (int64, int64) {
	var cpuQuota int64
	var memoryQuota int64
	for _, claimRow := range claimRaw {
		CPUResource := claimRow.ResourceList["cpu"]
		cpuQuota += CPUResource.Value()
		memoryResource := claimRow.ResourceList["memory"]
		memoryQuota += memoryResource.Value()
	}
	return cpuQuota, memoryQuota
}
