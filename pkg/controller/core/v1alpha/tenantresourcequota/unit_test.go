package tenantresourcequota

import (
	"context"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
)

var kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)
var edgenetInformerFactory = informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

var c = NewController(kubeclientset,
	edgenetclientset,
	kubeInformerFactory.Core().V1().Nodes(),
	edgenetInformerFactory.Core().V1alpha().TenantResourceQuotas())

func getTenantResourceQuota() *corev1alpha.TenantResourceQuota {
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
	edgenetclientset.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuotaObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(250 * time.Millisecond)
	// Get the object and check the status
	tenantResourceQuota, _ := edgenetclientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	return tenantResourceQuota
}

func TestEnqueueTenantResourceQuotaAfter(t *testing.T) {
	tenantResourceQuota := getTenantResourceQuota()
	c.enqueueTenantResourceQuotaAfter(tenantResourceQuota, 10*time.Millisecond)
	util.Equals(t, 1, c.workqueue.Len())
	time.Sleep(250 * time.Millisecond)
	util.Equals(t, 0, c.workqueue.Len())
}

// TODO: test failed
/*
unit_test.go:58:
        exp: 1
        got: 2
unit_test.go:60:
        exp: 2
        got: 3
--- FAIL: TestEnqueueTenantResourceQuota (0.52s)
*/
func TestEnqueueTenantResourceQuota(t *testing.T) {
	tenantResourceQuota_1 := getTenantResourceQuota()
	tenantResourceQuota_2 := getTenantResourceQuota()

	c.enqueueTenantResourceQuota(tenantResourceQuota_1)
	util.Equals(t, 1, c.workqueue.Len())
	c.enqueueTenantResourceQuota(tenantResourceQuota_2)
	util.Equals(t, 2, c.workqueue.Len())
}

func TestProcessNextWorkItem(t *testing.T) {
	tenantResourceQuota := getTenantResourceQuota()
	c.enqueueTenantResourceQuota(tenantResourceQuota)
	c.processNextWorkItem()
	util.Equals(t, 0, c.workqueue.Len())
}

func TestProcessTenantResourceQuota(t *testing.T) {
	tenantResourceQuota := getTenantResourceQuota()
	c.processTenantResourceQuota(tenantResourceQuota)
	util.Equals(t, success, tenantResourceQuota.Status.State)
	util.Equals(t, messageApplied, tenantResourceQuota.Status.Message)
}

// TODO:
// func TestAccumulateQuota(t *testing.T) {}
// func TestSyncHandler(t *testing.T) {}
// func TestTuneResourceQuotaAcrossNamespaces(t *testing.T){}
// func TestNamespaceTraversal(t *testing.T){}
// func TestTraverse(t *testing.T){}
