package clusterrolerequest

import (
	"context"
	"testing"
	"time"

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var edgenetInformerFactory = informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

var c = NewController(kubeclientset,
	edgenetclientset,
	edgenetInformerFactory.Registration().V1alpha().ClusterRoleRequests())

func getTestResource(name string) *registrationv1alpha.ClusterRoleRequest {
	g := TestGroup{}
	g.Init()
	clusterRoleRequestTest := g.roleRequestObj.DeepCopy()
	clusterRoleRequestTest.SetName(name)
	edgenetclientset.RegistrationV1alpha().ClusterRoleRequests().Create(context.TODO(), clusterRoleRequestTest, metav1.CreateOptions{})
	clusterRoleRequest, _ := edgenetclientset.RegistrationV1alpha().ClusterRoleRequests().Get(context.TODO(), clusterRoleRequestTest.GetName(), metav1.GetOptions{})
	return clusterRoleRequest
}

// TODO: case for return false
func TestCheckForRequestedRole(t *testing.T) {
	clusterRoleRequest := getTestResource("cluster-role-request-controller-test")
	ret := c.checkForRequestedRole(clusterRoleRequest)
	util.Equals(t, ret, true)
}

// TODO: more testcases
func TestProcessClusterRoleRequest(t *testing.T) {
	clusterRoleRequest := getTestResource("cluster-role-request-controller-test")
	clusterRoleRequest.Status.Expiry = nil
	c.processClusterRoleRequest(clusterRoleRequest)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), clusterRoleRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), clusterRoleRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), clusterRoleRequest.Status.Expiry.Year())

	clusterRoleRequest.Status.Expiry = &metav1.Time{
		Time: time.Now(),
	}
	c.processClusterRoleRequest(clusterRoleRequest)
	_, err := c.edgenetclientset.RegistrationV1alpha().RoleRequests(clusterRoleRequest.GetNamespace()).Get(context.TODO(), clusterRoleRequest.GetName(), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))

	clusterRoleRequest.Spec.Approved = false
	c.processClusterRoleRequest(clusterRoleRequest)
	util.Equals(t, pending, clusterRoleRequest.Status.State)
	util.Equals(t, messageRoleNotApproved, clusterRoleRequest.Status.Message)

	clusterRoleRequest.Spec.Approved = true
	c.processClusterRoleRequest(clusterRoleRequest)
	util.Equals(t, approved, clusterRoleRequest.Status.State)
	util.Equals(t, messageRoleApproved, clusterRoleRequest.Status.Message)

}

func TestEnqueueClusterRoleRequestAfter(t *testing.T) {
	clusterRoleRequest := getTestResource("cluster-role-request-controller-test")
	c.enqueueClusterRoleRequestAfter(clusterRoleRequest, 10*time.Millisecond)
	util.Equals(t, 1, c.workqueue.Len())
	time.Sleep(250 * time.Millisecond)
	util.Equals(t, 0, c.workqueue.Len())
}

func TestEnqueueClusterRoleRequest(t *testing.T) {
	clusterRoleRequest_1 := getTestResource("cluster-role-request-controller-test-1")
	clusterRoleRequest_2 := getTestResource("cluster-role-request-controller-test-2")

	c.enqueueClusterRoleRequest(clusterRoleRequest_1)
	util.Equals(t, 1, c.workqueue.Len())
	c.enqueueClusterRoleRequest(clusterRoleRequest_2)
	util.Equals(t, 2, c.workqueue.Len())
}

func TestSyncHandler(t *testing.T) {
	key := "default/cluster-role-request-controller-test"
	clusterRoleRequest := getTestResource("cluster-role-request-controller-test")
	clusterRoleRequest.Status.State = pending
	err := c.syncHandler(key)
	util.OK(t, err)
	util.Equals(t, approved, clusterRoleRequest.Status.State)
}

// TODO: More test cases
func TestProcessNextWorkItem(t *testing.T) {
	clusterRoleRequest := getTestResource("cluster-role-request-controller-test")
	c.enqueueClusterRoleRequest(clusterRoleRequest)
	util.Equals(t, 1, c.workqueue.Len())
	c.processNextWorkItem()
	util.Equals(t, 0, c.workqueue.Len())
}
