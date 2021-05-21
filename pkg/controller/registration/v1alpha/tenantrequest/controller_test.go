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
	tenantRequestTest := g.tenantRequestObj.DeepCopy()
	tenantRequestTest.SetName("tenant-request-controller-test")
	// Create an tenant request
	g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), tenantRequestTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	tenantRequest, _ := g.edgenetClient.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
	util.NotEquals(t, nil, tenantRequest.Status.Expiry)
	// Update an tenant request
	tenantRequest.Spec.Contact.Email = "different-email@edge-net.org"
	tenantRequest, _ = g.edgenetClient.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Update an tenant request
	tenantRequest.Spec.Approved = true
	g.edgenetClient.RegistrationV1alpha().TenantRequests().Update(context.TODO(), tenantRequest, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	// Checking if Tenant Request transitioned to tenant after the approval
	_, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), tenantRequest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
}
