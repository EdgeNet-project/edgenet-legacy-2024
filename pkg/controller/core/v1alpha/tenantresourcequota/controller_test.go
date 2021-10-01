package tenantresourcequota

import (
	"context"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"

	"github.com/EdgeNet-project/edgenet/pkg/util"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetClient)
	// Create a resource request
	tenantResourceQuotaObj := g.tenantResourceQuotaObj
	tenantResourceQuotaObj.Spec.Claim = append(tenantResourceQuotaObj.Spec.Claim, g.claimObj)
	g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), tenantResourceQuotaObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	tenantResourceQuota, err := g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, tenantResourceQuota.Status.State)
	// Update the tenant resource quota
	drop := g.dropObj
	drop.Expiry = &metav1.Time{
		Time: time.Now().Add(1300 * time.Millisecond),
	}
	tenantResourceQuota.Spec.Drop = append(tenantResourceQuota.Spec.Drop, drop)
	g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 200)
	tenantResourceQuota, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, 1, len(tenantResourceQuota.Spec.Drop))
	coreResourceQuota, err := g.client.CoreV1().ResourceQuotas(tenantResourceQuota.GetName()).Get(context.TODO(), "core-quota", metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota := g.handler.calculateTenantQuota(tenantResourceQuota)
	util.Equals(t, cpuQuota, coreResourceQuota.Spec.Hard.Cpu().Value())
	util.Equals(t, memoryQuota, coreResourceQuota.Spec.Hard.Memory().Value())

	time.Sleep(time.Millisecond * 1200)
	tenantResourceQuota, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, 0, len(tenantResourceQuota.Spec.Drop))
	coreResourceQuota, err = g.client.CoreV1().ResourceQuotas(tenantResourceQuota.GetName()).Get(context.TODO(), "core-quota", metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = g.handler.calculateTenantQuota(tenantResourceQuota)
	util.Equals(t, cpuQuota, coreResourceQuota.Spec.Hard.Cpu().Value())
	util.Equals(t, memoryQuota, coreResourceQuota.Spec.Hard.Memory().Value())

	expectedMemoryRes := resource.MustParse(g.claimObj.Memory)
	expectedMemory := expectedMemoryRes.Value()
	expectedMemoryRew := expectedMemory + int64(float64(g.nodeObj.Status.Capacity.Memory().Value())*1.3)
	expectedCPURes := resource.MustParse(g.claimObj.CPU)
	expectedCPU := expectedCPURes.Value()
	expectedCPURew := expectedCPU + int64(float64(g.nodeObj.Status.Capacity.Cpu().Value())*1.5)

	node := g.nodeObj
	nodeCopy, _ := g.client.CoreV1().Nodes().Create(context.TODO(), node.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	tenantResourceQuota, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	reward := false
	for _, claim := range tenantResourceQuota.Spec.Claim {
		if claim.Name == nodeCopy.GetName() {
			reward = true
		}
	}
	util.Equals(t, true, reward)
	cpuQuota, memoryQuota = getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemoryRew, memoryQuota)
	util.Equals(t, expectedCPURew, cpuQuota)
	coreResourceQuota, err = g.client.CoreV1().ResourceQuotas(tenantResourceQuota.GetName()).Get(context.TODO(), "core-quota", metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = g.handler.calculateTenantQuota(tenantResourceQuota)
	util.Equals(t, cpuQuota, coreResourceQuota.Spec.Hard.Cpu().Value())
	util.Equals(t, memoryQuota, coreResourceQuota.Spec.Hard.Memory().Value())

	nodeCopy.Status.Conditions[0].Status = "False"
	g.client.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	tenantResourceQuota, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemory, memoryQuota)
	util.Equals(t, expectedCPU, cpuQuota)

	nodeCopy.Status.Conditions[0].Status = "True"
	g.client.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	tenantResourceQuota, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemoryRew, memoryQuota)
	util.Equals(t, expectedCPURew, cpuQuota)

	nodeCopy.Status.Conditions[0].Status = "Unknown"
	g.client.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	tenantResourceQuota, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemory, memoryQuota)
	util.Equals(t, expectedCPU, cpuQuota)

	g.client.CoreV1().Nodes().Delete(context.TODO(), nodeCopy.GetName(), metav1.DeleteOptions{})
	time.Sleep(time.Millisecond * 500)
	tenantResourceQuota, err = g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), tenantResourceQuota.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	cpuQuota, memoryQuota = getQuotas(tenantResourceQuota.Spec.Claim)
	util.Equals(t, expectedMemory, memoryQuota)
	util.Equals(t, expectedCPU, cpuQuota)
}

func getQuotas(claimRaw []corev1alpha.TenantResourceDetails) (int64, int64) {
	var cpuQuota int64
	var memoryQuota int64
	for _, claimRow := range claimRaw {
		CPUResource := resource.MustParse(claimRow.CPU)
		cpuQuota += CPUResource.Value()
		memoryResource := resource.MustParse(claimRow.Memory)
		memoryQuota += memoryResource.Value()
	}
	return cpuQuota, memoryQuota
}
