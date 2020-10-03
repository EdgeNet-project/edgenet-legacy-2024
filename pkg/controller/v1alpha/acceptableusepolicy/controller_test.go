package acceptableusepolicy

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
	// Create an AUP
	g.edgenetClient.AppsV1alpha().AcceptableUsePolicies(g.AUPObj.GetNamespace()).Create(context.TODO(), g.AUPObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	AUP, err := g.edgenetClient.AppsV1alpha().AcceptableUsePolicies(g.AUPObj.GetNamespace()).Get(context.TODO(), g.AUPObj.GetName(), metav1.GetOptions{})
	// Check state
	util.OK(t, err)
	util.Equals(t, success, AUP.Status.State)

	// Update an AUP
	AUP.Spec.Accepted = true
	// Requesting server to Update internal representation of AUP
	g.edgenetClient.AppsV1alpha().AcceptableUsePolicies(AUP.GetNamespace()).Update(context.TODO(), AUP.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	AUP, err = g.edgenetClient.AppsV1alpha().AcceptableUsePolicies(AUP.GetNamespace()).Get(context.TODO(), AUP.GetName(), metav1.GetOptions{})
	// Check state
	util.OK(t, err)
	util.Equals(t, success, AUP.Status.State)
	expected := metav1.Time{
		Time: time.Now().Add(4382 * time.Hour),
	}
	util.Equals(t, expected.Day(), AUP.Status.Expires.Day())
	util.Equals(t, expected.Month(), AUP.Status.Expires.Month())
	util.Equals(t, expected.Year(), AUP.Status.Expires.Year())
}
