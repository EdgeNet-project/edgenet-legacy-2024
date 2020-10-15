package user

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()

	// Run the controller in a goroutine
	go Start(g.client, g.edgenetClient)
	// Create a user
	g.edgenetClient.AppsV1alpha().Users(g.userObj.GetNamespace()).Create(context.TODO(), g.userObj.DeepCopy(), metav1.CreateOptions{})
	go g.mockSigner(g.authorityObj.GetName(), g.userObj.GetName())
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 21000)
	// Get the object and check the status
	user, _ := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userObj.GetName(), metav1.GetOptions{})
	util.Equals(t, true, user.Spec.Active)
	util.Equals(t, success, user.Status.State)

	// Update a user
	user.Spec.Email = "update@edge-net.org"
	g.edgenetClient.AppsV1alpha().Users(user.GetNamespace()).Update(context.TODO(), user.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	user, _ = g.edgenetClient.AppsV1alpha().Users(user.GetNamespace()).Get(context.TODO(), user.GetName(), metav1.GetOptions{})
	util.Equals(t, false, user.Spec.Active)

	// Delete a user
	g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Delete(context.TODO(), g.userObj.GetName(), metav1.DeleteOptions{})
	_, err := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userObj.GetName(), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
}
