package subnamespace

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	// Run controller in a goroutine
	go Start(g.client, g.edgenetClient)

	coreResourceQuota, err := g.client.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("core-quota"), metav1.GetOptions{})
	util.OK(t, err)
	coreQuotaCPU := coreResourceQuota.Spec.Hard.Cpu().Value()
	coreQuotaMemory := coreResourceQuota.Spec.Hard.Memory().Value()

	// Create a subnamespace
	subNamespaceControllerTest := g.subNamespaceObj.DeepCopy()
	subNamespaceControllerTest.SetName("subnamespace-controller")
	_, err = g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subNamespaceControllerTest, metav1.CreateOptions{})
	util.OK(t, err)
	// Wait for the status update of the created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	_, err = g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subNamespaceControllerTest.GetName()), metav1.GetOptions{})
	util.OK(t, err)
	tunedCoreResourceQuota, err := g.client.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("core-quota"), metav1.GetOptions{})
	util.OK(t, err)
	tunedCoreQuotaCPU := tunedCoreResourceQuota.Spec.Hard.Cpu().Value()
	tunedCoreQuotaMemory := tunedCoreResourceQuota.Spec.Hard.Memory().Value()

	cpuResource := resource.MustParse(subNamespaceControllerTest.Spec.Resources.CPU)
	cpuDemand := cpuResource.Value()
	memoryResource := resource.MustParse(subNamespaceControllerTest.Spec.Resources.Memory)
	memoryDemand := memoryResource.Value()

	util.Equals(t, coreQuotaCPU-cpuDemand, tunedCoreQuotaCPU)
	util.Equals(t, coreQuotaMemory-memoryDemand, tunedCoreQuotaMemory)

	subResourceQuota, err := g.client.CoreV1().ResourceQuotas(fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subNamespaceControllerTest.GetName())).Get(context.TODO(), fmt.Sprintf("sub-quota"), metav1.GetOptions{})
	util.OK(t, err)
	subQuotaCPU := subResourceQuota.Spec.Hard.Cpu().Value()
	subQuotaMemory := subResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, int64(6), subQuotaCPU)
	util.Equals(t, int64(6442450944), subQuotaMemory)

	subNamespaceControllerNestedTest := g.subNamespaceObj.DeepCopy()
	subNamespaceControllerNestedTest.Spec.Resources.CPU = "1000m"
	subNamespaceControllerNestedTest.Spec.Resources.Memory = "1Gi"
	subNamespaceControllerNestedTest.SetName("subnamespace-controller-nested")
	subNamespaceControllerNestedTest.SetNamespace(fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subNamespaceControllerTest.GetName()))
	_, err = g.edgenetClient.CoreV1alpha().SubNamespaces(subNamespaceControllerNestedTest.GetNamespace()).Create(context.TODO(), subNamespaceControllerNestedTest, metav1.CreateOptions{})
	util.OK(t, err)
	// Wait for the status update of the created object
	time.Sleep(time.Millisecond * 500)

	subResourceQuota, err = g.client.CoreV1().ResourceQuotas(fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subNamespaceControllerTest.GetName())).Get(context.TODO(), fmt.Sprintf("sub-quota"), metav1.GetOptions{})
	util.OK(t, err)
	subQuotaCPU = subResourceQuota.Spec.Hard.Cpu().Value()
	subQuotaMemory = subResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, int64(5), subQuotaCPU)
	util.Equals(t, int64(5368709120), subQuotaMemory)

	tunedCoreResourceQuota, err = g.client.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("core-quota"), metav1.GetOptions{})
	util.OK(t, err)
	tunedCoreQuotaCPU = tunedCoreResourceQuota.Spec.Hard.Cpu().Value()
	tunedCoreQuotaMemory = tunedCoreResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, int64(2), tunedCoreQuotaCPU)
	util.Equals(t, int64(2147483648), tunedCoreQuotaMemory)

	nestedSubResourceQuota, err := g.client.CoreV1().ResourceQuotas(fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subNamespaceControllerNestedTest.GetName())).Get(context.TODO(), fmt.Sprintf("sub-quota"), metav1.GetOptions{})
	util.OK(t, err)
	nestedSubQuotaCPU := nestedSubResourceQuota.Spec.Hard.Cpu().Value()
	nestedSubQuotaMemory := nestedSubResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, int64(1), nestedSubQuotaCPU)
	util.Equals(t, int64(1073741824), nestedSubQuotaMemory)

	err = g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Delete(context.TODO(), subNamespaceControllerTest.GetName(), metav1.DeleteOptions{})
	util.OK(t, err)
	time.Sleep(time.Millisecond * 500)
	_, err = g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subNamespaceControllerTest.GetName()), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
	latestParentResourceQuota, err := g.client.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("core-quota"), metav1.GetOptions{})
	util.OK(t, err)
	latestParentQuotaCPU := latestParentResourceQuota.Spec.Hard.Cpu().Value()
	latestParentQuotaMemory := latestParentResourceQuota.Spec.Hard.Memory().Value()
	util.Equals(t, coreQuotaCPU, latestParentQuotaCPU)
	util.Equals(t, coreQuotaMemory, latestParentQuotaMemory)
}
