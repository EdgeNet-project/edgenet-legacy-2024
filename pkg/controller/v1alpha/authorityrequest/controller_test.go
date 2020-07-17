package authorityrequest

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := ARTestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Create an authority request
	g.edgenetclient.AppsV1alpha().AuthorityRequests().Create(g.authorityRequestObj.DeepCopy())
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	AR, _ := g.edgenetclient.AppsV1alpha().AuthorityRequests().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
	if AR.Status.Expires == nil || AR.Status.Message == nil {
		t.Error("Add func of event handler authority request doesn't work properly")
	}
	// Update a Authority request
	// Update contact email
	g.authorityRequestObj.Spec.Contact.Email = "JohnDoe1@edge-net.org"
	g.authorityRequestObj.Status.Approved = true
	g.edgenetclient.AppsV1alpha().AuthorityRequests().Update(g.authorityRequestObj.DeepCopy())
	time.Sleep(time.Millisecond * 500)
	// Checking if Authority Request transitioned to Authority after update
	authority, _ := g.edgenetclient.AppsV1alpha().Authorities().Get(g.authorityRequestObj.GetName(), metav1.GetOptions{})
	if authority == nil {
		t.Error("Failed to create Authority from Authority Request after approval")
	}
}
