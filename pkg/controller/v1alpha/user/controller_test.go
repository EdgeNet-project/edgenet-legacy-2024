package user

import (
	"edgenet/pkg/controller/v1alpha/authority"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := UserTestGroup{}
	g.Init()
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	g.authorityObj.Status.Enabled = true
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())

	// Run the controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Create a user
	g.edgenetclient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(g.userObj.DeepCopy())
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.GetName(), metav1.GetOptions{})
	if !user.Status.Active {
		t.Errorf("Add func of event handler doesn't work properly")
	}
	// Update a user
	g.userObj.Spec.FirstName = "newName"
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(g.userObj.DeepCopy())
	user, _ = g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.GetName(), metav1.GetOptions{})
	if user.Spec.FirstName != "newName" {
		t.Error("Update func of event handler doesn't work properly")
	}
	// Delete a user
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(g.userObj.GetName(), &metav1.DeleteOptions{})
	user, _ = g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.userObj.GetName(), metav1.GetOptions{})
	if user != nil {
		t.Error("Delete func of event handler doesn't work properly")
	}

}
