package slice

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := SliceTestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Create a slice
	g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.sliceObj.DeepCopy())
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	slice, _ := g.edgenetclient.AppsV1alpha().Slices(g.authorityObj.GetNamespace()).Get(g.sliceObj.GetName(), metav1.GetOptions{})
	if slice.Status.Expires == nil {
		t.Error(errorDict["add-func"])
	}
	// Update a slice
	slice.Spec.Users = []apps_v1alpha.SliceUsers{
		apps_v1alpha.SliceUsers{
			Authority: g.authorityObj.GetName(),
			Username:  "user1",
		},
	}
	g.userObj.Spec.Active, g.userObj.Status.AUP = true, true
	// Creating User before updating requesting server to update internal representation of slice
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
	// Requesting server to update internal representation of slice
	g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(slice)
	// Check user rolebinding in slice child namespace
	user, _ := g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get("user1", metav1.GetOptions{})
	time.Sleep(time.Millisecond * 500)
	roleBindings, _ := g.client.RbacV1().RoleBindings(fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName())).Get(fmt.Sprintf("%s-%s-slice-%s", user.GetNamespace(), user.GetName(), "admin"), metav1.GetOptions{})
	// Verifying server created rolebinding for new user in slice's child namespace
	if roleBindings == nil {
		t.Error(errorDict["upd-func"])
	}
	// Delete a user
	// Requesting server to delete internal representation of slice
	g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(g.sliceObj.Name, &metav1.DeleteOptions{})
	slice, _ = g.edgenetclient.AppsV1alpha().Slices(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.sliceObj.GetName(), metav1.GetOptions{})
	if slice != nil {
		t.Error(errorDict["del-func"])
	}
	time.Sleep(time.Millisecond * 500)
	sliceChildNamespace, _ := g.client.CoreV1().Namespaces().Get(fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName()), metav1.GetOptions{})
	if sliceChildNamespace != nil {
		t.Error("Failed to delete slice child namespace")
	}
}
