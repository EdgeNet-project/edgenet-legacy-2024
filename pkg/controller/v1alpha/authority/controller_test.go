package authority

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	// Run controller in a goroutine
	go Start(g.client, g.edgenetClient)
	// Create an authority
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of the created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	_, err := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	util.OK(t, err)
	user, _ := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	util.Equals(t, true, user.Spec.Active)
	// Update an authority
	g.authorityObj.Spec.Enabled = false
	g.edgenetClient.AppsV1alpha().Authorities().Update(context.TODO(), g.authorityObj.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	user, _ = g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.authorityObj.Spec.Contact.Username, metav1.GetOptions{})
	util.Equals(t, false, user.Spec.Active)
}
