package userregistrationrequest

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
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetClient)
	// Create a user registration object
	g.edgenetClient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.userRegistrationObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	URRCopy, _ := g.edgenetClient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.userRegistrationObj.GetName(), metav1.GetOptions{})
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), URRCopy.Status.Expiry.Day())
	util.Equals(t, expected.Month(), URRCopy.Status.Expiry.Month())
	util.Equals(t, expected.Year(), URRCopy.Status.Expiry.Year())
	util.EqualsMultipleExp(t, []string{statusDict["email-ok"], statusDict["email-fail"]}, URRCopy.Status.Message[0])
	// Update a Authority request
	URRCopy.Spec.Approved = true
	g.edgenetClient.AppsV1alpha().UserRegistrationRequests(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(context.TODO(), URRCopy, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Checking if user registration transitioned to user after update
	_, err := g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), URRCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
}
