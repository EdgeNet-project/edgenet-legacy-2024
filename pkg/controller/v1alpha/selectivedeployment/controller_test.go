package selectivedeployment

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := SDTestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Creating four nodes
	g.client.CoreV1().Nodes().Create(g.nodeFRObj.DeepCopy())
	g.client.CoreV1().Nodes().Create(g.nodeUSObj.DeepCopy())
	g.client.CoreV1().Nodes().Create(g.nodeUSSecondObj.DeepCopy())
	g.client.CoreV1().Nodes().Create(g.nodeUSThirdObj.DeepCopy())
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
	g.sdObjDeployment.Spec.Selector = []apps_v1alpha.Selector{
		{
			Value:    []string{"US"},
			Operator: "In",
			Quantity: 2,
			Name:     "Country",
		},
	}
	// Apending the second deployment object
	g.deploymentObj.Name = "deployment2"
	g.sdObjDeployment.Spec.Controllers.Deployment = append(g.sdObjDeployment.Spec.Controllers.Deployment, g.deploymentObj)
	g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Update(g.sdObjDeployment.DeepCopy())
	time.Sleep(time.Millisecond * 500)
	sd, _ = g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDeployment.GetName(), metav1.GetOptions{})
	// Get the selectiveDeployment
	if sd.Spec.Controllers.Deployment[0].Spec.Template.Spec.Containers[0].Image != "nginx:1.8.0" {
		t.Errorf("update func of event handler doesn't work properly")
	}
	// Checking the node name
	deploymentOne, _ := g.client.AppsV1().Deployments("").Get(g.sdObjDeployment.Spec.Controllers.Deployment[0].GetName(), metav1.GetOptions{})
	deploymentNodeNameOne := deploymentOne.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
	deploymentTwo, _ := g.client.AppsV1().Deployments("").Get(g.sdObjDeployment.Spec.Controllers.Deployment[1].GetName(), metav1.GetOptions{})
	deploymentNodeNameTwo := deploymentTwo.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[1]
	usNodesNames := map[string]bool{
		g.nodeUSObj.GetName():       true,
		g.nodeUSSecondObj.GetName(): true,
		g.nodeUSThirdObj.GetName():  true,
	}
	if !usNodesNames[deploymentNodeNameOne] || !usNodesNames[deploymentNodeNameTwo] {
		t.Errorf("update func of event handler doesn't work properly")
	}
}
