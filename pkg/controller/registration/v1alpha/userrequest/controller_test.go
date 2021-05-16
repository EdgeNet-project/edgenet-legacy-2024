package userrequest

import (
	"context"
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
	g.edgenetClient.RegistrationV1alpha().UserRequests().Create(context.TODO(), g.userRequestObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	userRequest, err := g.edgenetClient.RegistrationV1alpha().UserRequests().Get(context.TODO(), g.userRequestObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), userRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), userRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), userRequest.Status.Expiry.Year())
	util.EqualsMultipleExp(t, []string{statusDict["email-ok"], statusDict["email-fail"]}, userRequest.Status.Message[0])
	// Update a Tenant request
	userRequest.Spec.Approved = true
	g.edgenetClient.RegistrationV1alpha().UserRequests().Update(context.TODO(), userRequest, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Checking if user registration transitioned to user after update
	//_, err := g.edgenetClient.AppsV1alpha().Users().Get(context.TODO(), userRequest.GetName(), metav1.GetOptions{})
	//util.OK(t, err)
}
