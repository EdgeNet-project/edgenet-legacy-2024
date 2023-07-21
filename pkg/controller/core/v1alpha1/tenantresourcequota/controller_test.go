package tenantresourcequota

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/google/uuid"
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
	dropObj                corev1alpha.ResourceTuning
	tenantObj              corev1alpha.Tenant
	subNamespaceObj        corev1alpha.SubNamespace
	nodeObj                corev1.Node
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

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

	controller := NewController(kubeclientset,
		edgenetclientset,
		kubeInformerFactory.Core().V1().Nodes(),
		edgenetInformerFactory.Core().V1alpha1().TenantResourceQuotas())

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
	dropObj := corev1alpha.ResourceTuning{
		ResourceList: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("10000m"),
			corev1.ResourceMemory: resource.MustParse("10Gi"),
		},
	}
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "apps.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
			UID:  "edgenet",
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
	subNamespaceObj := corev1alpha.SubNamespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SubNamespace",
			APIVersion: "core.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sub",
			Namespace: "edgenet",
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
		Status: corev1alpha.SubNamespaceStatus{
			State: corev1alpha1.StatusEstablished,
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
	g.dropObj = dropObj
	g.tenantObj = tenantObj
	g.subNamespaceObj = subNamespaceObj
	g.nodeObj = nodeObj
}

// Imitate tenant creation processes
func (g *TestGroup) CreateTenant(tenantName string) {
	tenant := g.tenantObj.DeepCopy()
	tenant.SetName(tenantName)
	uid := types.UID(uuid.New().String())
	tenant.SetUID(uid)
	edgenetclientset.CoreV1alpha1().Tenants().Create(context.TODO(), tenant, metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenant.GetName()}}
	namespaceLabels := map[string]string{"edge-net.io/kind": "core", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/tenant-uid": string(uid)}
	namespace.SetLabels(namespaceLabels)
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	resourceQuota := corev1.ResourceQuota{}
	resourceQuota.Name = "core-quota"
	resourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              resource.MustParse("8000m"),
			"memory":           resource.MustParse("8192Mi"),
			"requests.storage": resource.MustParse("8Gi"),
		},
	}
	kubeclientset.CoreV1().ResourceQuotas(namespace.GetName()).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{})
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()

	randomString := util.GenerateRandomString(6)
	g.CreateTenant(randomString)
	// Create a resource request
	tenantResourceQuotaObj := g.tenantResourceQuotaObj
	tenantResourceQuotaObj.SetName(randomString)
	tenantResourceQuotaObj.SetUID(types.UID(randomString))
	tenantResourceQuotaObj.Spec.Claim = make(map[string]corev1alpha.ResourceTuning)
	tenantResourceQuotaObj.Spec.Claim["initial"] = g.claimObj
	edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuotaObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(250 * time.Millisecond)
	// Get the object and check the status
	tenantResourceQuota, err := edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, corev1alpha1.StatusApplied, tenantResourceQuota.Status.State)

	// Update the tenant resource quota
	drop := g.dropObj
	drop.Expiry = &metav1.Time{
		Time: time.Now().Add(400 * time.Millisecond),
	}
	tenantResourceQuota.Spec.Drop = make(map[string]corev1alpha.ResourceTuning)
	tenantResourceQuota.Spec.Drop["initial"] = drop
	edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(250 * time.Millisecond)
	tenantResourceQuota, err = edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, 1, len(tenantResourceQuota.Spec.Drop))
	coreResourceQuota, err := kubeclientset.CoreV1().ResourceQuotas(tenantResourceQuota.GetName()).Get(context.TODO(), "core-quota", metav1.GetOptions{})
	util.OK(t, err)
	assignedQuota := tenantResourceQuota.Fetch()
	for key, value := range coreResourceQuota.Spec.Hard {
		if _, elementExists := assignedQuota[key]; elementExists {
			util.Equals(t, true, assignedQuota[key].Equal(value))
		}
	}

	subnamespace := g.subNamespaceObj
	subnamespace.SetNamespace(randomString)
	edgenetclientset.CoreV1alpha1().SubNamespaces(tenantResourceQuota.GetName()).Create(context.TODO(), subnamespace.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: subnamespace.GenerateChildName("")}}
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	resourceQuota := corev1.ResourceQuota{}
	resourceQuota.Name = "sub-quota"
	resourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              subnamespace.Spec.Workspace.ResourceAllocation["cpu"],
			"memory":           subnamespace.Spec.Workspace.ResourceAllocation["memory"],
			"requests.storage": resource.MustParse("8Gi"),
		},
	}
	subResourceQuota, err := kubeclientset.CoreV1().ResourceQuotas(namespace.GetName()).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{})
	util.OK(t, err)
	time.Sleep(time.Millisecond * 250)
	tenantResourceQuota, err = edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, 0, len(tenantResourceQuota.Spec.Drop))
	coreResourceQuota, err = kubeclientset.CoreV1().ResourceQuotas(tenantResourceQuota.GetName()).Get(context.TODO(), "core-quota", metav1.GetOptions{})
	util.OK(t, err)
	assignedQuota = tenantResourceQuota.Fetch()

	for key, value := range coreResourceQuota.Spec.Hard {
		if assignedQuantity, elementExists := assignedQuota[key]; elementExists {
			subQuotaValue := subResourceQuota.Spec.Hard[key]
			value.Add(subQuotaValue)

			util.Equals(t, true, assignedQuantity.Equal(value))
		}
	}

	edgenetclientset.CoreV1alpha1().SubNamespaces(tenantResourceQuota.GetName()).Delete(context.TODO(), subnamespace.GetName(), metav1.DeleteOptions{})
	kubeclientset.CoreV1().Namespaces().Delete(context.TODO(), subnamespace.GenerateChildName(""), metav1.DeleteOptions{})

	/*
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
		coreResourceQuota, err = kubeclientset.CoreV1().ResourceQuotas(tenantResourceQuota.GetName()).Get(context.TODO(), "core-quota", metav1.GetOptions{})
		util.OK(t, err)

		assignedQuota = tenantResourceQuota.Fetch()
		util.Equals(t, assignedQuota["cpu"], *coreResourceQuota.Spec.Hard.Cpu())
		util.Equals(t, assignedQuota["memory"], *coreResourceQuota.Spec.Hard.Memory())

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
		util.Equals(t, expectedCPU, cpuQuota)*/
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()

	cases := map[string]struct {
		input    []time.Duration
		sleep    time.Duration
		expected int
	}{
		"without expiry date": {nil, 110, 2},
		"expiries soon":       {[]time.Duration{100}, 400, 0},
		"expired":             {[]time.Duration{-1000}, 400, 0},
		"mix/1":               {[]time.Duration{1900, 2200, -100}, 400, 4},
		"mix/2":               {[]time.Duration{90, 2500, -100}, 400, 2},
		"mix/3":               {[]time.Duration{1750, 2600, 1800, 1900, -10, -100}, 400, 8},
		"mix/4":               {[]time.Duration{290, 50, 2500, 3400, -10, -100}, 400, 4},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			randomString := util.GenerateRandomString(6)
			g.CreateTenant(randomString)
			tenantResourceQuota := g.tenantResourceQuotaObj.DeepCopy()
			tenantResourceQuota.SetUID(types.UID(k))
			tenantResourceQuota.SetName(randomString)
			tenantResourceQuota.Spec.Claim = make(map[string]corev1alpha.ResourceTuning)
			tenantResourceQuota.Spec.Drop = make(map[string]corev1alpha.ResourceTuning)

			claim := g.claimObj
			drop := g.dropObj
			if tc.input != nil {
				for _, input := range tc.input {
					claim.Expiry = &metav1.Time{
						Time: time.Now().Add(input * time.Millisecond),
					}
					tenantResourceQuota.Spec.Claim[util.GenerateRandomString(6)] = claim
					drop.Expiry = &metav1.Time{
						Time: time.Now().Add(input * time.Millisecond),
					}
					tenantResourceQuota.Spec.Drop[util.GenerateRandomString(6)] = drop
				}
			} else {
				tenantResourceQuota.Spec.Claim[util.GenerateRandomString(6)] = claim
				tenantResourceQuota.Spec.Drop[util.GenerateRandomString(6)] = drop
			}
			edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota, metav1.CreateOptions{})
			time.Sleep(tc.sleep * time.Millisecond)
			tenantResourceQuotaCopy, err := edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, tc.expected, (len(tenantResourceQuotaCopy.Spec.Claim) + len(tenantResourceQuotaCopy.Spec.Drop)))
			edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Delete(context.TODO(), tenantResourceQuotaCopy.GetName(), metav1.DeleteOptions{})
		})
	}
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	randomString := util.GenerateRandomString(6)
	g.CreateTenant(randomString)
	tenantResourceQuota := g.tenantResourceQuotaObj.DeepCopy()
	tenantResourceQuota.SetName(randomString)
	_, err := edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.CreateOptions{})
	util.OK(t, err)
	time.Sleep(250 * time.Millisecond)
	defer edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Delete(context.TODO(), tenantResourceQuota.GetName(), metav1.DeleteOptions{})

	cases := map[string]struct {
		input    []time.Duration
		sleep    time.Duration
		expected int
	}{
		"without expiry date": {nil, 30, 2},
		"expiries soon":       {[]time.Duration{30}, 400, 0},
		"expired":             {[]time.Duration{-100}, 400, 0},
		"mix/1":               {[]time.Duration{1700, 1850, -100}, 400, 4},
		"mix/2":               {[]time.Duration{30, 2700, -100}, 400, 2},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			tenantResourceQuotaCopy, err := edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			tenantResourceQuotaCopy.Spec.Claim = make(map[string]corev1alpha.ResourceTuning)
			tenantResourceQuotaCopy.Spec.Drop = make(map[string]corev1alpha.ResourceTuning)

			claim := g.claimObj
			drop := g.dropObj
			if tc.input != nil {
				for _, expiry := range tc.input {
					claim.Expiry = &metav1.Time{
						Time: time.Now().Add(expiry * time.Millisecond),
					}
					tenantResourceQuotaCopy.Spec.Claim[util.GenerateRandomString(6)] = claim
					drop.Expiry = &metav1.Time{
						Time: time.Now().Add(expiry * time.Millisecond),
					}
					tenantResourceQuotaCopy.Spec.Drop[util.GenerateRandomString(6)] = drop
				}
			} else {
				tenantResourceQuotaCopy.Spec.Claim[util.GenerateRandomString(6)] = claim
				tenantResourceQuotaCopy.Spec.Drop[util.GenerateRandomString(6)] = drop
			}
			_, err = edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuotaCopy.DeepCopy(), metav1.UpdateOptions{})
			util.OK(t, err)
			time.Sleep(tc.sleep * time.Millisecond)
			tenantResourceQuotaCopy, err = edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuotaCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, tc.expected, (len(tenantResourceQuotaCopy.Spec.Claim) + len(tenantResourceQuotaCopy.Spec.Drop)))
		})
	}
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
