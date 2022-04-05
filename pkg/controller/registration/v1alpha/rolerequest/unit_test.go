package rolerequest

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
	edgenetInformerFactory.Registration().V1alpha().RoleRequests())

func getTestResource(name string) *registrationv1alpha.RoleRequest {
	g := TestGroup{}
	g.Init()
	roleRequestTest := g.roleRequestObj.DeepCopy()
	roleRequestTest.SetName(name)

	// Create a role request object
	edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Create(context.TODO(), roleRequestTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	roleRequest, _ := edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequestTest.GetNamespace()).Get(context.TODO(), roleRequestTest.GetName(), metav1.GetOptions{})
	return roleRequest
}

//TODO: test failed
/*
I0405 00:07:56.057605    4792 event.go:291] "Event occurred" object="edgenet/role-request-controller-test" kind="RoleRequest" apiVersion="apps.edgenet.io/v1alpha" type="Normal" reason="Found" message="Requested Role / Cluster Role found successfully"
[31munit_test.go:42:

        exp: false

        got: true[39m

I0405 00:07:56.058915    4792 event.go:291] "Event occurred" object="edgenet/role-request-controller-test" kind="RoleRequest" apiVersion="apps.edgenet.io/v1alpha" type="Warning" reason="Not Found" message="Requested Role / Cluster Role does not exist"
--- FAIL: TestCheckForRequestedRole (0.51s)
FAIL
I0405 00:07:56.063319    4792 event.go:291] "Event occurred" object="edgenet/role-request-controller-test" kind="RoleRequest" apiVersion="apps.edgenet.io/v1alpha" type="Warning" reason="Not Found" message="Requested Role / Cluster Role does not exist"
exit status 1
FAIL    github.com/EdgeNet-project/edgenet/pkg/controller/registration/v1alpha/rolerequest      4.782s
*/
func TestCheckForRequestedRole(t *testing.T) {
	roleRequest := getTestResource("role-request-controller-test")
	ret := c.checkForRequestedRole(roleRequest)
	util.Equals(t, ret, true)
	roleRequest.Spec.RoleRef.Kind = "Role"
	ret = c.checkForRequestedRole(roleRequest)
	util.Equals(t, ret, true)
	roleRequest.Spec.RoleRef.Kind = "Illegal"
	ret = c.checkForRequestedRole(roleRequest)
	util.Equals(t, ret, false)
}

// TODO: more testcases, e.g. for variation situation of roleRequestCopy.Spec.Approved=true
func TestProcessRoleRequest(t *testing.T) {
	roleRequest := getTestResource("role-request-controller-test")
	roleRequest.Status.Expiry = nil
	c.processRoleRequest(roleRequest)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), roleRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), roleRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), roleRequest.Status.Expiry.Year())

	roleRequest.Status.Expiry = &metav1.Time{
		Time: time.Now(),
	}
	c.processRoleRequest(roleRequest)
	_, err := c.edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequest.GetNamespace()).Get(context.TODO(), roleRequest.GetName(), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))

	roleRequest = getTestResource("role-request-controller-test")
	roleRequest.Spec.Approved = false
	c.processRoleRequest(roleRequest)
	util.Equals(t, pending, roleRequest.Status.State)
	util.Equals(t, messageRoleNotApproved, roleRequest.Status.Message)

	roleRequest.Spec.Approved = true
	c.processRoleRequest(roleRequest)
	util.Equals(t, approved, roleRequest.Status.State)
	util.Equals(t, messageRoleApproved, roleRequest.Status.Message)
}

func TestEnqueueRoleRequestAfter(t *testing.T) {
	roleRequest := getTestResource("role-request-controller-test")
	c.enqueueRoleRequestAfter(roleRequest, 10*time.Millisecond)
	util.Equals(t, 1, c.workqueue.Len())
	time.Sleep(250 * time.Millisecond)
	util.Equals(t, 0, c.workqueue.Len())
}

//TODO: test failed
func TestEnqueueRoleRequest(t *testing.T) {
	roleRequest_1 := getTestResource("role-request-controller-test-1")
	roleRequest_2 := getTestResource("role-request-controller-test-2")

	c.enqueueRoleRequest(roleRequest_1)
	util.Equals(t, 1, c.workqueue.Len())
	c.enqueueRoleRequest(roleRequest_2)
	util.Equals(t, 2, c.workqueue.Len())
}

//TODO: test failed
/*
0405 00:01:08.279631   11372 controller.go:220] rolerequest 'default/role-request-controller-test' in work queue no longer exists
[31munit_test.go:92:

        exp: "Approved"

        got: "Pending"[39m

--- FAIL: TestSyncHandler (0.50s)
*/
func TestSyncHandler(t *testing.T) {
	key := "default/role-request-controller-test"
	roleRequest := getTestResource("role-request-controller-test")
	roleRequest.Status.State = pending
	err := c.syncHandler(key)
	util.OK(t, err)
	util.Equals(t, approved, roleRequest.Status.State)
}

// TODO: More test cases
func TestProcessNextWorkItem(t *testing.T) {
	roleRequest := getTestResource("role-request-controller-test")
	c.enqueueRoleRequest(roleRequest)
	util.Equals(t, 1, c.workqueue.Len())
	c.processNextWorkItem()
	util.Equals(t, 0, c.workqueue.Len())
}
