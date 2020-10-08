package slice

import (
	"context"
	"fmt"
	"testing"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.edgenetClient.AppsV1alpha().TotalResourceQuotas().Create(context.TODO(), g.TRQObj.DeepCopy(), metav1.CreateOptions{})
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetClient)
	// Create a slice
	_, err := g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Create(context.TODO(), g.sliceObj.DeepCopy(), metav1.CreateOptions{})
	util.OK(t, err)
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	slice, err := g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Get(context.TODO(), g.sliceObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	expected := metav1.Time{
		Time: time.Now().Add(336 * time.Hour),
	}
	util.Equals(t, expected.Day(), slice.Status.Expires.Day())
	util.Equals(t, expected.Month(), slice.Status.Expires.Month())
	util.Equals(t, expected.Year(), slice.Status.Expires.Year())
	// Update a slice
	slice.Spec.Users = []apps_v1alpha.SliceUsers{
		apps_v1alpha.SliceUsers{
			Authority: g.authorityObj.GetName(),
			Username:  "joepublic",
		},
	}
	// Creating User before updating requesting server to update internal representation of slice
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	// Requesting server to update internal representation of slice
	g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Update(context.TODO(), slice, metav1.UpdateOptions{})
	// Check user rolebinding in slice child namespace
	user, _ := g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Get(context.TODO(), g.userObj.GetName(), metav1.GetOptions{})
	time.Sleep(time.Millisecond * 500)
	_, err = g.client.RbacV1().RoleBindings(fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName())).Get(context.TODO(), fmt.Sprintf("%s-%s-slice-%s", user.GetNamespace(), user.GetName(), user.Status.Type), metav1.GetOptions{})
	// Verifying server created rolebinding for new user in slice's child namespace
	util.OK(t, err)
	// Delete a user
	// Requesting server to delete internal representation of slice
	g.edgenetClient.AppsV1alpha().Slices(g.sliceObj.GetNamespace()).Delete(context.TODO(), g.sliceObj.Name, metav1.DeleteOptions{})
	time.Sleep(time.Millisecond * 500)
	_, err = g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-slice-%s", g.sliceObj.GetNamespace(), g.sliceObj.GetName()), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
}
