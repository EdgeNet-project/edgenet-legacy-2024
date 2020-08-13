package selectivedeployment

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := SDTestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Creating two nodes
	g.client.CoreV1().Nodes().Create(g.nodeFRObj.DeepCopy())
	g.client.CoreV1().Nodes().Create(g.nodeUSObj.DeepCopy())
	// Invoking the Create function of SD
	g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjDeployment.DeepCopy())
	time.Sleep(time.Millisecond * 500)
	sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDeployment.GetName(), metav1.GetOptions{})
	// Get the selectiveDeployment
	if sd.Status.State != success {
		t.Errorf("Add func of event handler doesn't work properly")
	}
	// Invoking the Create function of SD
	g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjDaemonset.DeepCopy())
	time.Sleep(time.Millisecond * 500)
	// Get the selectiveDeployment
	sd, _ = g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDaemonset.GetName(), metav1.GetOptions{})
	if sd.Status.State != success {
		t.Errorf("Add func of event handler doesn't work properly")
	}
	// Creating a service for StatefulSet
	g.client.CoreV1().Services("").Create(g.statefulsetService.DeepCopy())
	// Invoking the Create function of SD
	g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjStatefulset.DeepCopy())
	time.Sleep(time.Millisecond * 500)
	sdStatefulSet, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjStatefulset.GetName(), metav1.GetOptions{})
	if sdStatefulSet.Status.State != success {
		t.Errorf("Add func of event handler doesn't work properly")
	}
	g.sdObjDeployment.Spec.Controllers.Deployment[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.0"
	g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Update(g.sdObjDeployment.DeepCopy())
	time.Sleep(time.Millisecond * 500)
	sd, _ = g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDeployment.GetName(), metav1.GetOptions{})
	// Get the selectiveDeployment
	if sd.Spec.Controllers.Deployment[0].Spec.Template.Spec.Containers[0].Image != "nginx:1.8.0" {
		t.Errorf("update func of event handler doesn't work properly")
	}
}
