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
	g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), g.acceptableUsePolicyObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	acceptableUsePolicy, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), g.acceptableUsePolicyObj.GetName(), metav1.GetOptions{})
	// Check state
	util.OK(t, err)
	util.Equals(t, success, acceptableUsePolicy.Status.State)

	// Update an AUP
	acceptableUsePolicy.Spec.Accepted = true
	// Requesting server to Update internal representation of AUP
	g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	acceptableUsePolicy, err = g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), acceptableUsePolicy.GetName(), metav1.GetOptions{})
	// Check state
	util.OK(t, err)
	util.Equals(t, success, acceptableUsePolicy.Status.State)
	expected := metav1.Time{
		Time: time.Now().Add(4382 * time.Hour),
	}
	util.Equals(t, expected.Day(), acceptableUsePolicy.Status.Expiry.Day())
	util.Equals(t, expected.Month(), acceptableUsePolicy.Status.Expiry.Month())
	util.Equals(t, expected.Year(), acceptableUsePolicy.Status.Expiry.Year())
}
