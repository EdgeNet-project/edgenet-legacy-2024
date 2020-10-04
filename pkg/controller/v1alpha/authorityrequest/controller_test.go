package authorityrequest

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
	// Create an authority request
	g.edgenetClient.AppsV1alpha().AuthorityRequests().Create(context.TODO(), g.authorityRequestObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	AR, _ := g.edgenetClient.AppsV1alpha().AuthorityRequests().Get(context.TODO(), g.authorityRequestObj.GetName(), metav1.GetOptions{})
	util.NotEquals(t, nil, AR.Status.Expires)
	// Update an authority request
	g.authorityRequestObj.Spec.Approved = true
	g.edgenetClient.AppsV1alpha().AuthorityRequests().Update(context.TODO(), g.authorityRequestObj.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Checking if Authority Request transitioned to authority after the approval
	_, err := g.edgenetClient.AppsV1alpha().Authorities().Get(context.TODO(), g.authorityRequestObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
}
