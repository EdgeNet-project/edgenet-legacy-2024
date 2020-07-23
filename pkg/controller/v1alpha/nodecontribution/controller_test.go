package nodecontribution

import (
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := NodecontributionTestGroup{}
	g.Init()

	// Run controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Create a Nodecontribution
	g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.nodecontributionObj.DeepCopy())
	// Wait for the status update of the created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	node, _ := g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.nodecontributionObj.GetName(), metav1.GetOptions{})
	if node == nil {
		t.Error("Add func of event handler doesn't work properly")
	}
	// Update a nodecontribution
	g.nodecontributionObj.Spec.Host = "testHost"
	g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(g.nodecontributionObj.DeepCopy())
	node, _ = g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.nodecontributionObj.GetName(), metav1.GetOptions{})
	if node.Spec.Host != "testHost" {
		t.Error("Update func of event handler doesn't work properly")
	}
}
