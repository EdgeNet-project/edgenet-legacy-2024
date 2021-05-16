package tenantrequest

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
	// Create an tenant request
	g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), g.tenantRequestObj.DeepCopy(), metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	tenantRequst, _ := g.edgenetClient.RegistrationV1alpha().TenantRequests().Get(context.TODO(), g.tenantRequestObj.GetName(), metav1.GetOptions{})
	util.NotEquals(t, nil, tenantRequst.Status.Expiry)
	// Update an tenant request
	g.tenantRequestObj.Spec.Approved = true
	g.edgenetClient.RegistrationV1alpha().TenantRequests().Update(context.TODO(), g.tenantRequestObj.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Checking if Tenant Request transitioned to tenant after the approval
	_, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), g.tenantRequestObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
}
