package totalresourcequota

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Dictionary for status messages
var errorDict = map[string]string{
	"k8-sync":     "Kubernetes clientset sync problem",
	"edgnet-sync": "EdgeNet clientset sync problem",
	"TRQ-failed":  "Failed to create Total resource quota",
	"TRQ-update":  "Failed to update Total resource quota. Exceeded resource quota was not balanced.",
	"add-func":    "Add func of event handler doesn't work properly",
	"upd-func":    "Update func of event handler doesn't work properly",
	"del-func":    "Delete func of event handler doesn't work properly",
}

// The main structure of test group
type TRQTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	sliceObj      apps_v1alpha.Slice
	TRQObj        apps_v1alpha.TotalResourceQuota
	client        kubernetes.Interface
	edgenetclient versioned.Interface
	handler       Handler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func (g *TRQTestGroup) Init() {
	authorityObj := apps_v1alpha.Authority{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Authority",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: apps_v1alpha.AuthoritySpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: apps_v1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: apps_v1alpha.Contact{
				Email:     "unittest@edge-net.org",
				FirstName: "unit",
				LastName:  "testing",
				Phone:     "+33NUMBER",
				Username:  "unittesting",
			},
			Enabled: true,
		},
	}
	TRQObj := apps_v1alpha.TotalResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TotalResourceQuota",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
			// Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.TotalResourceQuotaSpec{
			Enabled: false,
		},
		Status: apps_v1alpha.TotalResourceQuotaStatus{
			Exceeded: false,
		},
	}
	sliceObj := apps_v1alpha.Slice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Slice",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "Slice1",
			Namespace:       "authority-edgenet",
			OwnerReferences: []metav1.OwnerReference{},
		},
		Spec: apps_v1alpha.SliceSpec{
			Profile:     "High",
			Users:       []apps_v1alpha.SliceUsers{},
			Description: "This is a test description",
			Renew:       true,
		},
		Status: apps_v1alpha.SliceStatus{
			Expires: nil,
		},
	}
	g.authorityObj = authorityObj
	g.sliceObj = sliceObj
	g.TRQObj = TRQObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	g.authorityObj.Status.State = success
	g.authorityObj.Spec.Enabled = true
	// Update Authority status
	g.edgenetclient.AppsV1alpha().Authorities().UpdateStatus(context.TODO(), g.authorityObj.DeepCopy(), metav1.UpdateOptions{})
	authorityChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", g.authorityObj.GetName())}}
	// Create Authority child namepace
	g.client.CoreV1().Namespaces().Create(context.TODO(), authorityChildNamespace, metav1.CreateOptions{})

}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TRQTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error(errorDict["k8-sync"])
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error(errorDict["edgenet-sync"])
	}
}

func TestTRQCreate(t *testing.T) {
	g := TRQTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creation of Total resource quota
	t.Run("creation of total resource quota", func(t *testing.T) {
		g.TRQObj.Spec.Enabled = true
		g.edgenetclient.AppsV1alpha().TotalResourceQuotas().Create(context.TODO(), g.TRQObj.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.TRQObj.DeepCopy())
		TRQ, _ := g.edgenetclient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), g.TRQObj.GetName(), metav1.GetOptions{})
		if TRQ.Status.State != success {
			t.Error(errorDict["TRQ-failed"])
		}
	})
	t.Run("creation of resource quota", func(t *testing.T) {
		g.handler.Create("testquota")
		TRQ, _ := g.edgenetclient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), "testquota", metav1.GetOptions{})
		if TRQ.Spec.Claim == nil {
			t.Error(errorDict["TRQ-failed"])
		}
	})

}

func TestTRQupdate(t *testing.T) {
	g := TRQTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creation of Total resource quota
	t.Run("Update of total resource quota", func(t *testing.T) {
		// Create a resource quota
		g.handler.resourceQuota = &corev1.ResourceQuota{}
		g.handler.resourceQuota.Name = "slice-high-quota"
		g.handler.resourceQuota.Spec = corev1.ResourceQuotaSpec{
			Hard: map[corev1.ResourceName]resource.Quantity{
				"cpu":              resource.MustParse("8000m"),
				"memory":           resource.MustParse("8192Mi"),
				"requests.storage": resource.MustParse("8Gi"),
			},
		}
		// Create a slice
		g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.sliceObj.DeepCopy(), metav1.CreateOptions{})
		// Create Slice child namespace with resource quota
		sliceChildNamespaceStr := fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName())
		g.client.CoreV1().ResourceQuotas(sliceChildNamespaceStr).Create(context.TODO(), g.handler.resourceQuota, metav1.CreateOptions{})
		// Set a tiny claim for TRQ so handler triggers exceeded mechanism
		g.TRQObj.Spec.Claim = []apps_v1alpha.TotalResourceDetails{
			{
				Name:    "slice-exceed-quota",
				CPU:     "2m",
				Memory:  "2Mi",
				Expires: nil,
			},
		}
		g.TRQObj.Spec.Enabled = true
		var field fields
		field.spec = true
		g.edgenetclient.AppsV1alpha().TotalResourceQuotas().Update(context.TODO(), g.TRQObj.DeepCopy(), metav1.UpdateOptions{})
		// Triggering quota exceeded
		g.handler.ObjectUpdated(g.TRQObj.DeepCopy(), field)
		slice, _ := g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.sliceObj.GetName(), metav1.GetOptions{})
		if slice != nil {
			t.Error(errorDict["TRQ-update"])
		}
	})
}
