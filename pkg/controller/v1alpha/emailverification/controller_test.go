package emailverification

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartController(t *testing.T) {
	g := EVTestGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client, g.edgenetclient)
	// Create a EV object
	g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(context.TODO(), g.EVObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	EV, _ := g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.EVObj.GetName(), metav1.GetOptions{})
	// Handler will delete EV if verified
	if EV.Status.Expires == nil {
		t.Error(errorDict["add-func"])
	}
	// Update an EV
	g.EVObj.Spec.Verified = true
	g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(context.TODO(), g.EVObj.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Checking if handler created user from user registration and deleted user registration
	EV, _ = g.edgenetclient.AppsV1alpha().EmailVerifications(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(context.TODO(), g.EVObj.GetName(), metav1.GetOptions{})
	// Handler will delete EV if verified
	if EV != nil {
		t.Error(errorDict["upd-func"])
	}
}
