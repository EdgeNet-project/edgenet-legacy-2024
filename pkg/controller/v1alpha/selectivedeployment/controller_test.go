package selectivedeployment

import (
	"context"
	"testing"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := SDTestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Creating four nodes
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeFRObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSSecondObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSThirdObj.DeepCopy(), metav1.CreateOptions{})
	// Invoking the Create function of SD
	g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDeployment.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
	// Get the selectiveDeployment
	if sd.Status.State != success {
		t.Errorf("Add func of event handler doesn't work properly")
	}
	// Invoking the Create function of SD
	g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDaemonset.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Get the selectiveDeployment
	sd, _ = g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDaemonset.GetName(), metav1.GetOptions{})
	if sd.Status.State != success {
		t.Errorf("Add func of event handler doesn't work properly")
	}
	// Creating a service for StatefulSet
	g.client.CoreV1().Services("").Create(context.TODO(), g.statefulsetService.DeepCopy(), metav1.CreateOptions{})
	// Invoking the Create function of SD
	g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjStatefulset.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdStatefulSet, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjStatefulset.GetName(), metav1.GetOptions{})
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
	g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Update(context.TODO(), g.sdObjDeployment.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	sd, _ = g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
	// Get the selectiveDeployment
	if sd.Spec.Controllers.Deployment[0].Spec.Template.Spec.Containers[0].Image != "nginx:1.8.0" {
		t.Errorf("update func of event handler doesn't work properly")
	}
	// Checking the node name
	deploymentOne, _ := g.client.AppsV1().Deployments("").Get(context.TODO(), g.sdObjDeployment.Spec.Controllers.Deployment[0].GetName(), metav1.GetOptions{})
	deploymentNodeNameOne := deploymentOne.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
	deploymentTwo, _ := g.client.AppsV1().Deployments("").Get(context.TODO(), g.sdObjDeployment.Spec.Controllers.Deployment[1].GetName(), metav1.GetOptions{})
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
