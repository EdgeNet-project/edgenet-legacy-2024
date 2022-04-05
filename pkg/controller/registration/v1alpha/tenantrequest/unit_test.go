package tenantrequest

import (
	"context"
	"testing"
	"time"

	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var edgenetInformerFactory = informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

var c = NewController(kubeclientset,
	edgenetclientset,
	edgenetInformerFactory.Registration().V1alpha().TenantRequests())

func getTestResource(name string) *registrationv1alpha.TenantRequest {
	g := TestGroup{}
	g.Init()
	tenantRequestTest := g.tenantRequestObj.DeepCopy()
	tenantRequestTest.SetName(name)
	tenantRequestTest.SetNamespace("default")
	// Create a tenant request
	edgenetclientset.RegistrationV1alpha().TenantRequests().Create(context.TODO(), tenantRequestTest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(250 * time.Millisecond)
	// Get the object and check the status
	// tenantRequest, _ := edgenetclientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequestTest.GetName(), metav1.GetOptions{})
	return tenantRequestTest
}

//TODO: running failed
/*
=== RUN   TestProcessTenantRequest
I0404 22:50:29.285138    5648 event.go:291] "Event occurred" object="default/tenant-request-controller-test" kind="tenantRequest" apiVersion="apps.edgenet.io/v1alpha" type="Warning" reason="Not Approved" message="Waiting for Requested Tenant to be approved"
E0404 22:50:29.285212    5648 event.go:264] Server rejected event '&v1.Event{TypeMeta:v1.TypeMeta{Kind:"", APIVersion:""}, ObjectMeta:v1.ObjectMeta{Name:"tenant-request-controller-test.16e2cca68dada878", GenerateName:"", Namespace:"default", SelfLink:"", UID:"", ResourceVersion:"", Generation:0, CreationTimestamp:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), DeletionTimestamp:<nil>, DeletionGracePeriodSeconds:(*int64)(nil), Labels:map[string]string(nil), Annotations:map[string]string(nil), OwnerReferences:[]v1.OwnerReference(nil), Finalizers:[]string(nil), ClusterName:"", ManagedFields:[]v1.ManagedFieldsEntry(nil)}, InvolvedObject:v1.ObjectReference{Kind:"tenantRequest", Namespace:"default", Name:"tenant-request-controller-test", UID:"", APIVersion:"apps.edgenet.io/v1alpha", ResourceVersion:"", FieldPath:""}, Reason:"Not Approved", Message:"Waiting for Requested Tenant to be approved", Source:v1.EventSource{Component:"tenantrequest-controller", Host:""}, FirstTimestamp:time.Date(2022, time.April, 4, 22, 50, 29, 284628600, time.Local), LastTimestamp:time.Date(2022, time.April, 4, 22, 50, 29, 284628600, time.Local), Count:1, Type:"Warning", EventTime:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), Series:(*v1.EventSeries)(nil), Action:"", Related:(*v1.ObjectReference)(nil), ReportingController:"", ReportingInstance:""}': 'request namespace does not match object namespace, request: "" object: "default"' (will not retry!)
I0404 22:50:29.302252    5648 event.go:291] "Event occurred" object="default/tenant-request-controller-test" kind="tenantRequest" apiVersion="apps.edgenet.io/v1alpha" type="Normal" reason="Approved" message="Requested Tenant approved successfully"
E0404 22:50:29.308696    5648 event.go:264] Server rejected event '&v1.Event{TypeMeta:v1.TypeMeta{Kind:"", APIVersion:""}, ObjectMeta:v1.ObjectMeta{Name:"tenant-request-controller-test.16e2cca68ea44098", GenerateName:"", Namespace:"default", SelfLink:"", UID:"", ResourceVersion:"", Generation:0, CreationTimestamp:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), DeletionTimestamp:<nil>, DeletionGracePeriodSeconds:(*int64)(nil), Labels:map[string]string(nil), Annotations:map[string]string(nil), OwnerReferences:[]v1.OwnerReference(nil), Finalizers:[]string(nil), ClusterName:"", ManagedFields:[]v1.ManagedFieldsEntry(nil)}, InvolvedObject:v1.ObjectReference{Kind:"tenantRequest", Namespace:"default", Name:"tenant-request-controller-test", UID:"", APIVersion:"apps.edgenet.io/v1alpha", ResourceVersion:"", FieldPath:""}, Reason:"Approved", Message:"Requested Tenant approved successfully", Source:v1.EventSource{Component:"tenantrequest-controller", Host:""}, FirstTimestamp:time.Date(2022, time.April, 4, 22, 50, 29, 300789400, time.Local), LastTimestamp:time.Date(2022, time.April, 4, 22, 50, 29, 300789400, time.Local), Count:1, Type:"Normal", EventTime:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), Series:(*v1.EventSeries)(nil), Action:"", Related:(*v1.ObjectReference)(nil), ReportingController:"", ReportingInstance:""}': 'request namespace does not match object namespace, request: "" object: "default"' (will not retry!)
*/
func TestProcessTenantRequest(t *testing.T) {
	tenantRequest := getTestResource("tenant-request-controller-test")
	//case 1: tenantRequestCopy.Status.Expiry = nil
	tenantRequest.Status.Expiry = nil
	c.processTenantRequest(tenantRequest)
	expected := metav1.Time{
		Time: time.Now().Add(72 * time.Hour),
	}
	util.Equals(t, expected.Day(), tenantRequest.Status.Expiry.Day())
	util.Equals(t, expected.Month(), tenantRequest.Status.Expiry.Month())
	util.Equals(t, expected.Year(), tenantRequest.Status.Expiry.Year())
	// case 2:
	tenantRequest.Spec.Approved = false
	c.processTenantRequest(tenantRequest)
	util.Equals(t, pending, tenantRequest.Status.State)
	util.Equals(t, messageNotApproved, tenantRequest.Status.Message)
	// case 3: not sure if node access.CreateTenant(tenantRequestCopy) will be successful or not
	tenantRequest.Spec.Approved = true
	c.processTenantRequest(tenantRequest)
	util.Equals(t, approved, tenantRequest.Status.State)
	util.Equals(t, messageRoleApproved, tenantRequest.Status.Message)
}

// TODO: test failed
/*
=== RUN   TestEnqueueTenantRequestAfter
[31munit_test.go:71:

        exp: 1

        got: 0[39m

[31munit_test.go:73:

        exp: 0

        got: 1[39m

--- FAIL: TestEnqueueTenantRequestAfter (0.51s)
*/
func TestEnqueueTenantRequestAfter(t *testing.T) {
	tenantRequest := getTestResource("tenant-request-controller-test")
	c.enqueueTenantRequestAfter(tenantRequest, 10*time.Millisecond)
	util.Equals(t, 1, c.workqueue.Len())
	time.Sleep(250 * time.Millisecond)
	util.Equals(t, 0, c.workqueue.Len())
}

// TODO: test failed
func TestEnqueueTenantRequest(t *testing.T) {
	tenantRequest_1 := getTestResource("tenant-request-controller-test")
	tenantRequest_2 := getTestResource("tenant-request-controller-test-2")

	c.enqueueTenantRequest(tenantRequest_1)
	util.Equals(t, 1, c.workqueue.Len())
	c.enqueueTenantRequest(tenantRequest_2)
	util.Equals(t, 1, c.workqueue.Len())
}

// TODO: test failed
/*
=== RUN   TestSyncHandler
E0404 23:33:34.092497    1708 controller.go:218] tenantrequest 'default/tenant-request-controller-test' in work queue no longer exists
[31munit_test.go:110:

        exp: "Approved"

        got: "Pending"[39m

--- FAIL: TestSyncHandler (0.27s)
FAIL
exit status 1
*/
func TestSyncHandler(t *testing.T) {
	key := "default/tenant-request-controller-test"
	tenantRequest := getTestResource("tenant-request-controller-test")
	tenantRequest.Status.State = pending
	err := c.syncHandler(key)
	util.OK(t, err)
	util.Equals(t, approved, tenantRequest.Status.State)
}

// TODO: More test cases
func TestProcessNextWorkItem(t *testing.T) {
	tenantRequest := getTestResource("tenant-request-controller-test")
	c.enqueueTenantRequest(tenantRequest)
	util.Equals(t, 1, c.workqueue.Len())
	c.processNextWorkItem()
	util.Equals(t, 0, c.workqueue.Len())
}
