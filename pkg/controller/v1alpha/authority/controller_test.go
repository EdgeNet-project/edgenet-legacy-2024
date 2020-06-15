package authority

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := AuthorityTestGroup{}
	g.Init()
	// Run controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Create an authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	// Wait for the status update of the created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	authority, _ := g.edgenetclient.AppsV1alpha().Authorities().Get(g.authorityObj.GetName(), metav1.GetOptions{})
	if !authority.Status.Enabled {
		t.Error("Add func of event handler doesn't work properly")
	}
	// Update an authority
	g.authorityObj.Spec.FullName = "test"
	g.edgenetclient.AppsV1alpha().Authorities().Update(g.authorityObj.DeepCopy())
	authority, _ = g.edgenetclient.AppsV1alpha().Authorities().Get(g.authorityObj.GetName(), metav1.GetOptions{})
	if authority.Spec.FullName != "test" {
		t.Error("Update func of event handler doesn't work properly")
	}
	// Delete an authority
	g.edgenetclient.AppsV1alpha().Authorities().Delete(g.authorityObj.GetName(), &metav1.DeleteOptions{})
	authority, _ = g.edgenetclient.AppsV1alpha().Authorities().Get(g.authorityObj.GetName(), metav1.GetOptions{})
	if authority != nil {
		t.Error("Delete func of event handler doesn't work properly")
	}
}
