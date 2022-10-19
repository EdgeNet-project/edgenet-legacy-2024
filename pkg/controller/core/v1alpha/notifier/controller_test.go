package notifier

import (
	"context"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()

// var kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeclientset, 0)

var edgenetInformerFactory = informers.NewSharedInformerFactory(edgenetclientset, 0)

var tenantrequestInformer = edgenetInformerFactory.Registration().V1alpha().TenantRequests()
var rolerequestInformer = edgenetInformerFactory.Registration().V1alpha().RoleRequests()

var c = NewController(kubeclientset, edgenetclientset,
	tenantrequestInformer, rolerequestInformer)

// func TestProcessNextWorkItem(t *testing.T) {
// 		subNamespaceObj := getSubNameSpaceTestObj()
// 		c.enqueueSubNamespace(subNamespaceObj)
// 		c.processNextWorkItem()
// 		util.Equals(t, 0, c.workqueue.Len())
// }

func TestProcessNextWorkItem(t *testing.T) {
	tenantRequestObj := getTenantRequestObj()
	c.enqueueNotifier(tenantRequestObj)
	roleRequestObj := getRoleRequestObj()
	c.enqueueNotifier(roleRequestObj)
	util.Equals(t, 2, c.workqueue.Len())
	err := c.processNextWorkItem()
	util.Equals(t, true, err)
	util.Equals(t, 1, c.workqueue.Len())
	err = c.processNextWorkItem()
	util.Equals(t, true, err)
	util.Equals(t, 0, c.workqueue.Len())
}

func TestSyncTenantRequestHandler(t *testing.T) {
	tenantRequestObj := getTenantRequestObj()
	key := tenantRequestObj.GetNamespace() + "/" + tenantRequestObj.GetName()
	err := c.syncTenantRequestHandler(key)
	util.OK(t, err)
}
func TestSyncRoleRequestHandler(t *testing.T) {
	roleRequestObj := getRoleRequestObj()
	key := roleRequestObj.GetNamespace() + "/" + roleRequestObj.GetName()
	err := c.syncTenantRequestHandler(key)
	util.OK(t, err)
}

func TestEnqueueNotifier(t *testing.T) {
	tenantRequestObj := getTenantRequestObj()
	c.enqueueNotifier(tenantRequestObj)
	util.Equals(t, 1, c.workqueue.Len())
	roleRequestObj := getRoleRequestObj()
	c.enqueueNotifier(roleRequestObj)
	util.Equals(t, 2, c.workqueue.Len())
}

//TODO: How to check the result
func TestProcessTenantRequest(t *testing.T) {
	tenantRequestObj := getTenantRequestObj()
	tenantRequestObj.Status.State = pending
	c.processTenantRequest(tenantRequestObj)
}

//TODO: How to check the result
func TestProcessRoleRequest(t *testing.T) {
	roleRequestObj := getRoleRequestObj()
	roleRequestObj.Status.State = pending
	c.processRoleRequest(roleRequestObj)
}

func getRoleRequestObj() *registrationv1alpha.RoleRequest {
	roleRequest := registrationv1alpha.RoleRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "johnsmith",
			Namespace: "edgenet",
		},
		Spec: registrationv1alpha.RoleRequestSpec{
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
			RoleRef: registrationv1alpha.RoleRefSpec{
				Kind: "ClusterRole",
				Name: "edgenet:tenant-admin",
			},
		},
	}
	edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequest.GetNamespace()).Create(context.TODO(), &roleRequest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(time.Millisecond * 500)
	// Get the object and check the status
	roleRequestObj, _ := edgenetclientset.RegistrationV1alpha().RoleRequests(roleRequest.GetNamespace()).Get(context.TODO(), roleRequest.GetName(), metav1.GetOptions{})
	return roleRequestObj
}

func getTenantRequestObj() *registrationv1alpha.TenantRequest {
	tenantRequest := registrationv1alpha.TenantRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "tenantRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet-request",
		},
		Spec: registrationv1alpha.TenantRequestSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: corev1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha.Contact{
				Email:     "tom.public@edge-net.org",
				FirstName: "Tom",
				LastName:  "Public",
				Phone:     "+33NUMBER",
			},
		},
	}
	edgenetclientset.RegistrationV1alpha().TenantRequests().Create(context.TODO(), &tenantRequest, metav1.CreateOptions{})
	// Wait for the status update of created object
	time.Sleep(250 * time.Millisecond)
	// Get the object and check the status
	tenantRequestObj, _ := edgenetclientset.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tenantRequest.GetName(), metav1.GetOptions{})
	return tenantRequestObj
}
