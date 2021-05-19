package tenant

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
	// Create a tenant
	tenantControllerTest := g.tenantObj.DeepCopy()
	tenantControllerTest.SetName("tenant-controller")
	g.mockSigner(tenantControllerTest.GetName(), tenantControllerTest.Spec.User)
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), tenantControllerTest, metav1.CreateOptions{})
	// Wait for the status update of the created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), tenantControllerTest.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, tenant.Spec.Contact.Username, tenant.Spec.User[0].Username)
	// Update the tenant
	g.mockSigner(tenant.GetName(), tenant.Spec.User)
	tenant.Spec.Enabled = false
	g.edgenetClient.CoreV1alpha().Tenants().Update(context.TODO(), tenant, metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	_, err = g.client.RbacV1().Roles(tenant.GetName()).Get(context.TODO(), fmt.Sprintf("tenant-owner-%s", tenant.Spec.Contact.Username), metav1.GetOptions{})
	util.Equals(t, "roles.rbac.authorization.k8s.io \"tenant-owner-johndoe\" not found", err.Error())
}
