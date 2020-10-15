package totalresourcequota

import (
	"context"
	"testing"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
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
	TRQObj := g.TRQObj
	TRQObj.Spec.Claim = append(TRQObj.Spec.Claim, g.claimObj)
	g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Create(context.TODO(), TRQObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	TRQCopy, err := g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, TRQCopy.Status.State)
	// Update the TRQ
	drop := g.dropObj
	drop.Expires = &metav1.Time{
		Time: time.Now().Add(400 * time.Millisecond),
	}
	TRQCopy.Spec.Drop = append(TRQCopy.Spec.Drop, drop)
	g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Update(context.TODO(), TRQCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 200)
	TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, 1, len(TRQCopy.Spec.Drop))
	time.Sleep(time.Millisecond * 500)
	TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, 0, len(TRQCopy.Spec.Drop))

	expectedMemoryRes := resource.MustParse(g.claimObj.Memory)
	expectedMemory := expectedMemoryRes.Value()
	expectedMemoryRew := expectedMemory + int64(float64(g.nodeObj.Status.Capacity.Memory().Value())*1.3)
	expectedCPURes := resource.MustParse(g.claimObj.CPU)
	expectedCPU := expectedCPURes.Value()
	expectedCPURew := expectedCPU + int64(float64(g.nodeObj.Status.Capacity.Cpu().Value())*1.5)

	node := g.nodeObj
	nodeCopy, _ := g.client.CoreV1().Nodes().Create(context.TODO(), node.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	reward := false
	for _, claim := range TRQCopy.Spec.Claim {
		if claim.Name == "Reward" {
			reward = true
		}
	}
	util.Equals(t, true, reward)
	CPUQuota, memoryQuota := getQuotas(TRQCopy.Spec.Claim)
	util.Equals(t, expectedMemoryRew, memoryQuota)
	util.Equals(t, expectedCPURew, CPUQuota)

	nodeCopy.Status.Conditions[0].Status = "False"
	g.client.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	CPUQuota, memoryQuota = getQuotas(TRQCopy.Spec.Claim)
	util.Equals(t, expectedMemory, memoryQuota)
	util.Equals(t, expectedCPU, CPUQuota)

	nodeCopy.Status.Conditions[0].Status = "True"
	g.client.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	CPUQuota, memoryQuota = getQuotas(TRQCopy.Spec.Claim)
	util.Equals(t, expectedMemoryRew, memoryQuota)
	util.Equals(t, expectedCPURew, CPUQuota)

	nodeCopy.Status.Conditions[0].Status = "Unknown"
	g.client.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	CPUQuota, memoryQuota = getQuotas(TRQCopy.Spec.Claim)
	util.Equals(t, expectedMemory, memoryQuota)
	util.Equals(t, expectedCPU, CPUQuota)

	g.client.CoreV1().Nodes().Delete(context.TODO(), nodeCopy.GetName(), metav1.DeleteOptions{})
	time.Sleep(time.Millisecond * 500)
	TRQCopy, err = g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Get(context.TODO(), TRQCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	CPUQuota, memoryQuota = getQuotas(TRQCopy.Spec.Claim)
	util.Equals(t, expectedMemory, memoryQuota)
	util.Equals(t, expectedCPU, CPUQuota)
}

func getQuotas(claimRaw []apps_v1alpha.TotalResourceDetails) (int64, int64) {
	var CPUQuota int64
	var memoryQuota int64
	for _, claimRow := range claimRaw {
		CPUResource := resource.MustParse(claimRow.CPU)
		CPUQuota += CPUResource.Value()
		memoryResource := resource.MustParse(claimRow.Memory)
		memoryQuota += memoryResource.Value()
	}
	return CPUQuota, memoryQuota
}
