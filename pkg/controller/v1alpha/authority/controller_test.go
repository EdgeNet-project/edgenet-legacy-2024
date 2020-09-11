package authority

import (
	"context"
	"fmt"
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
	g.edgenetclient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of the created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	if user == nil {
		t.Error(errorDict["add-func"])
	}
	// Update an authority
	g.authorityObj.Spec.FullName = "test"
	g.edgenetclient.AppsV1alpha().Authorities().Update(context.TODO(), g.authorityObj.DeepCopy(), metav1.UpdateOptions{})
	authority, _ := g.edgenetclient.AppsV1alpha().Authorities().Get(context.TODO(), g.authorityObj.GetName(), metav1.GetOptions{})
	if authority.Spec.FullName != "test" {
		t.Error(errorDict["upd-func"])
	}
	// Delete an authority
	g.edgenetclient.AppsV1alpha().Authorities().Delete(context.TODO(), g.authorityObj.GetName(), metav1.DeleteOptions{})
	authority, _ = g.edgenetclient.AppsV1alpha().Authorities().Get(context.TODO(), g.authorityObj.GetName(), metav1.GetOptions{})
	if authority != nil {
		t.Error(errorDict["del-func"])
	}
}
