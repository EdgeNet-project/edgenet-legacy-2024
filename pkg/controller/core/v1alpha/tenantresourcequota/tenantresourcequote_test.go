package tenantresourcequota

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type TestGroup struct {
	tenantResourceQuotaObj corev1alpha.TenantResourceQuota
	claimObj               corev1alpha.TenantResourceDetails
	dropObj                corev1alpha.TenantResourceDetails
	tenantObj              corev1alpha.Tenant
	nodeObj                corev1.Node
	client                 kubernetes.Interface
	edgenetClient          versioned.Interface
	handler                Handler
}

func TestMain(m *testing.M) {
	flag.String("dir", "../../../../..", "Override the directory.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func (g *TestGroup) Init() {
	tenantResourceQuotaObj := corev1alpha.TenantResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "tenantResourceQuota",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
			UID:  "trq",
		},
	}
	claimObj := corev1alpha.TenantResourceDetails{
		Name:   "Default",
		CPU:    "12000m",
		Memory: "12Gi",
	}
	dropObj := corev1alpha.TenantResourceDetails{
		Name:   "Default",
		CPU:    "10000m",
		Memory: "10Gi",
	}
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "apps.edgenet.io/v1alpha",
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
				Username:  "johndoe",
			},
			Enabled: true,
		},
	}
	nodeObj := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fr-idf-0000.edge-net.io",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: "apps.edgenet.io/v1alpha",
					Kind:       "Tenant",
					Name:       "edgenet",
					UID:        "edgenet"},
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
				corev1.NodeCondition{
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
	g.nodeObj = nodeObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// Imitate tenant creation processes
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.tenantObj.GetName()}}
	namespaceLabels := map[string]string{"owner": "tenant", "owner-name": g.tenantObj.GetName(), "tenant-name": g.tenantObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	resourceQuota := corev1.ResourceQuota{}
	resourceQuota.Name = "core-quota"
	resourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: map[corev1.ResourceName]resource.Quantity{
			"cpu":              resource.MustParse("8000m"),
			"memory":           resource.MustParse("8192Mi"),
			"requests.storage": resource.MustParse("8Gi"),
		},
	}
	g.client.CoreV1().ResourceQuotas(namespace.GetName()).Create(context.TODO(), resourceQuota.DeepCopy(), metav1.CreateOptions{})
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetClient)
	util.Equals(t, g.client, g.handler.clientset)
	util.Equals(t, g.edgenetClient, g.handler.edgenetClientset)
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	go g.handler.RunExpiryController()

	cases := map[string]struct {
		input    []time.Duration
		sleep    time.Duration
		expected int
	}{
		"without expiry date": {nil, 110, 2},
		"expiries soon":       {[]time.Duration{100}, 300, 0},
		"expired":             {[]time.Duration{-1000}, 300, 0},
		"mix/1":               {[]time.Duration{1500, 2200, -100}, 300, 4},
		"mix/2":               {[]time.Duration{90, 2500, -100}, 300, 2},
		"mix/3":               {[]time.Duration{1450, 1600, 1800, 1900, -10, -100}, 250, 8},
		"mix/4":               {[]time.Duration{90, 50, 2500, 3400, -10, -100}, 300, 4},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			tenantResourceQuota := g.tenantResourceQuotaObj.DeepCopy()
			tenantResourceQuota.SetUID(types.UID(k))
			claim := g.claimObj
			drop := g.dropObj
			if tc.input != nil {
				for _, input := range tc.input {
					claim.Expiry = &metav1.Time{
						Time: time.Now().Add(input * time.Millisecond),
					}
					tenantResourceQuota.Spec.Claim = append(tenantResourceQuota.Spec.Claim, claim)
					drop.Expiry = &metav1.Time{
						Time: time.Now().Add(input * time.Millisecond),
					}
					tenantResourceQuota.Spec.Drop = append(tenantResourceQuota.Spec.Drop, drop)
				}
			} else {
				tenantResourceQuota.Spec.Claim = append(tenantResourceQuota.Spec.Claim, claim)
				tenantResourceQuota.Spec.Drop = append(tenantResourceQuota.Spec.Drop, drop)
			}
			g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota, metav1.CreateOptions{})
			g.handler.ObjectCreatedOrUpdated(tenantResourceQuota)
			time.Sleep(tc.sleep * time.Millisecond)
			tenantResourceQuotaCopy, err := g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, tc.expected, (len(tenantResourceQuotaCopy.Spec.Claim) + len(tenantResourceQuotaCopy.Spec.Drop)))
			g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Delete(context.TODO(), tenantResourceQuotaCopy.GetName(), metav1.DeleteOptions{})
		})
	}
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	go g.handler.RunExpiryController()
	tenantResourceQuota := g.tenantResourceQuotaObj
	_, err := g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.CreateOptions{})
	util.OK(t, err)
	g.handler.ObjectCreatedOrUpdated(tenantResourceQuota.DeepCopy())
	defer g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Delete(context.TODO(), tenantResourceQuota.GetName(), metav1.DeleteOptions{})

	cases := map[string]struct {
		input    []time.Duration
		sleep    time.Duration
		expected int
	}{
		"without expiry date": {nil, 30, 2},
		"expiries soon":       {[]time.Duration{30}, 300, 0},
		"expired":             {[]time.Duration{-100}, 150, 0},
		"mix/1":               {[]time.Duration{1700, 1850, -100}, 300, 4},
		"mix/2":               {[]time.Duration{30, 2700, -100}, 250, 2},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			tenantResourceQuotaCopy, err := g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			tenantResourceQuotaCopy.Spec.Claim = []corev1alpha.TenantResourceDetails{}
			tenantResourceQuotaCopy.Spec.Drop = []corev1alpha.TenantResourceDetails{}

			claim := g.claimObj
			drop := g.dropObj
			if tc.input != nil {
				for _, expiry := range tc.input {
					claim.Expiry = &metav1.Time{
						Time: time.Now().Add(expiry * time.Millisecond),
					}
					tenantResourceQuotaCopy.Spec.Claim = append(tenantResourceQuotaCopy.Spec.Claim, claim)
					drop.Expiry = &metav1.Time{
						Time: time.Now().Add(expiry * time.Millisecond),
					}
					tenantResourceQuotaCopy.Spec.Drop = append(tenantResourceQuotaCopy.Spec.Drop, drop)
				}
			} else {
				tenantResourceQuotaCopy.Spec.Claim = append(tenantResourceQuotaCopy.Spec.Claim, claim)
				tenantResourceQuotaCopy.Spec.Drop = append(tenantResourceQuotaCopy.Spec.Drop, drop)
			}
			_, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuotaCopy.DeepCopy(), metav1.UpdateOptions{})
			util.OK(t, err)
			g.handler.ObjectCreatedOrUpdated(tenantResourceQuotaCopy.DeepCopy())
			time.Sleep(tc.sleep * time.Millisecond)
			tenantResourceQuotaCopy, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuotaCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, tc.expected, (len(tenantResourceQuotaCopy.Spec.Claim) + len(tenantResourceQuotaCopy.Spec.Drop)))
		})
	}
}

func TestCreatetenantResourceQuota(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)

	_, err := g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), g.tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
	g.handler.Create(g.tenantResourceQuotaObj.GetName(), nil)
	_, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), g.tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
}
