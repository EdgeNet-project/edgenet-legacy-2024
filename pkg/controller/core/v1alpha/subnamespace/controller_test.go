package subnamespace

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
	// Create a subnamespace
	subNamespaceControllerTest := g.subNamespaceObj.DeepCopy()
	subNamespaceControllerTest.SetName("subnamespace-controller")
	_, err := g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subNamespaceControllerTest, metav1.CreateOptions{})
	util.OK(t, err)
	// Wait for the status update of the created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	_, err = g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subNamespaceControllerTest.GetName()), metav1.GetOptions{})
	util.OK(t, err)
	// util.Equals(t, tenant.Spec.Contact.Username, tenant.Spec.User[0].Username)
	// Update the tenant
	// tenant.Spec.Enabled = false
	//g.edgenetClient.CoreV1alpha().Tenants().Update(context.TODO(), tenant, metav1.UpdateOptions{})
	//time.Sleep(time.Millisecond * 500)
	//_, err = g.client.RbacV1().Roles(tenant.GetName()).Get(context.TODO(), fmt.Sprintf("edgenet:tenant-owner-%s", tenant.Spec.Contact.Username), metav1.GetOptions{})
	//util.Equals(t, "roles.rbac.authorization.k8s.io \"edgenet:tenant-owner-johndoe\" not found", err.Error())
}
