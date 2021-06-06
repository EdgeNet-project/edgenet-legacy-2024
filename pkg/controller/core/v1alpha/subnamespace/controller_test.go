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
	err = g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Delete(context.TODO(), subNamespaceControllerTest.GetName(), metav1.DeleteOptions{})
	util.OK(t, err)
}
