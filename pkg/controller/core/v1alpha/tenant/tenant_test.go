package tenant

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type TestGroup struct {
	tenantObj           corev1alpha.Tenant
	tenantRequestObj    registrationv1alpha.TenantRequest
	userObj             corev1alpha.User
	userRegistrationObj registrationv1alpha.UserRequest
	client              kubernetes.Interface
	edgenetClient       versioned.Interface
	handler             Handler
}

func TestMain(m *testing.M) {
	flag.String("dir", "../../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *TestGroup) Init() {
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: corev1alpha.TenantSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: corev1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha.User{
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+33NUMBER",
				Username:  "johndoe",
			},
			User: []corev1alpha.User{
				corev1alpha.User{
					Email:     "john.doe@edge-net.org",
					FirstName: "John",
					LastName:  "Doe",
					Phone:     "+33NUMBER",
					Username:  "johndoe",
					Role:      "Owner",
				},
			},
			Enabled: true,
		},
	}
	tenantRequestObj := registrationv1alpha.TenantRequest{
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
			Contact: corev1alpha.User{
				Email:     "tom.public@edge-net.org",
				FirstName: "Tom",
				LastName:  "Public",
				Phone:     "+33NUMBER",
				Username:  "tompublic",
			},
		},
		Status: registrationv1alpha.TenantRequestStatus{
			State: success,
		},
	}
	userObj := corev1alpha.User{
		Username:  "joepublic",
		FirstName: "Joe",
		LastName:  "Public",
		Email:     "joe.public@edge-net.org",
		Role:      "Admin",
	}
	g.tenantObj = tenantObj
	g.tenantRequestObj = tenantRequestObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
}

// TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetClient)
	util.Equals(t, g.client, g.handler.clientset)
	util.Equals(t, g.edgenetClient, g.handler.edgenetClientset)
}

func TestCreateTenant(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	tenantRequest := g.tenantRequestObj.DeepCopy()
	tenantRequest.SetName("request-approval-test")
	created := g.handler.Create(tenantRequest)
	util.Equals(t, true, created)
	created = g.handler.Create(tenantRequest)
	util.Equals(t, false, created)
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	g.mockSigner(g.tenantObj.GetName(), g.tenantObj.Spec.User)
	g.handler.ObjectCreatedOrUpdated(g.tenantObj.DeepCopy())

	t.Run("user configuration", func(t *testing.T) {
		tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), g.tenantObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		time.Sleep(500 * time.Millisecond)
		aup, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), tenant.Spec.Contact.Username, metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, false, aup.Spec.Accepted)

		aup.Spec.Accepted = true
		g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), aup, metav1.UpdateOptions{})
		g.mockSigner(tenant.GetName(), tenant.Spec.User)
		g.handler.ObjectCreatedOrUpdated(tenant)

		t.Run("cluster role binding", func(t *testing.T) {
			_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-owner-%s", g.tenantObj.GetName(), g.tenantObj.GetName(), g.tenantObj.Spec.Contact.Username), metav1.GetOptions{})
			util.OK(t, err)
		})
		t.Run("role binding", func(t *testing.T) {
			_, err := g.client.RbacV1().RoleBindings(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("edgenet:tenant-owner-%s", g.tenantObj.Spec.Contact.Username), metav1.GetOptions{})
			util.OK(t, err)
		})
	})
	t.Run("tenant resource quota", func(t *testing.T) {
		_, err := g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), g.tenantObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("cluster roles", func(t *testing.T) {
		_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-owner", g.tenantObj.GetName(), g.tenantObj.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
}

func TestCollision(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)

	tenantRequest1 := g.tenantRequestObj.DeepCopy()
	tenantRequest1.Spec.Contact.Email = g.tenantObj.Spec.Contact.Email
	tenantRequest2 := g.tenantRequestObj.DeepCopy()
	tenantRequest2.SetName(g.tenantObj.GetName())
	tenantRequest3 := g.tenantRequestObj.DeepCopy()

	user1 := g.userObj
	user1.Email = g.tenantObj.Spec.Contact.Email
	user2 := g.userObj

	tenant1 := g.tenantObj.DeepCopy()
	tenant1.SetName("tenant-1-diff")
	tenant1.Spec.Contact.Email = user1.Email
	tenant1.Spec.User[0].Email = user1.Email

	tenant2 := g.tenantObj.DeepCopy()
	tenant2.SetName("tenant-2-diff")
	tenant2.Spec.Contact.Email = user2.Email
	tenant2.Spec.User[0].Email = user2.Email

	cases := map[string]struct {
		request  interface{}
		kind     string
		expected bool
	}{
		"ar/email":   {tenantRequest1, "TenantRequest", true},
		"ar/name":    {tenantRequest2, "TenantRequest", true},
		"ar/none":    {tenantRequest3, "TenantRequest", false},
		"user/email": {tenant1, "User", true},
		"user/none":  {tenant2, "User", false},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			if tc.kind == "TenantRequest" {
				_, err := g.edgenetClient.RegistrationV1alpha().TenantRequests().Create(context.TODO(), tc.request.(*registrationv1alpha.TenantRequest), metav1.CreateOptions{})
				util.OK(t, err)
				defer g.edgenetClient.RegistrationV1alpha().TenantRequests().Delete(context.TODO(), tc.request.(*registrationv1alpha.TenantRequest).GetName(), metav1.DeleteOptions{})
				g.handler.checkDuplicateObject(g.tenantObj.DeepCopy())
				_, err = g.edgenetClient.RegistrationV1alpha().TenantRequests().Get(context.TODO(), tc.request.(*registrationv1alpha.TenantRequest).GetName(), metav1.GetOptions{})
				util.Equals(t, tc.expected, errors.IsNotFound(err))
			} else if tc.kind == "User" {
				_, err := g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), tc.request.(*corev1alpha.Tenant), metav1.CreateOptions{})
				util.OK(t, err)
				defer g.edgenetClient.CoreV1alpha().Tenants().Delete(context.TODO(), tc.request.(*corev1alpha.Tenant).GetName(), metav1.DeleteOptions{})
				exists, _ := g.handler.checkDuplicateObject(g.tenantObj.DeepCopy())
				util.Equals(t, tc.expected, exists)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Create a tenant to update later
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreatedOrUpdated func to create a user
	g.mockSigner(g.tenantObj.GetName(), g.tenantObj.Spec.User)
	g.handler.ObjectCreatedOrUpdated(g.tenantObj.DeepCopy())
	tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), g.tenantObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, tenant.Spec.Contact.Username, tenant.Spec.User[0].Username)
	tenant.Spec.User = append(tenant.Spec.User, g.userObj)
	g.mockSigner(tenant.GetName(), tenant.Spec.User)
	g.handler.ObjectCreatedOrUpdated(tenant)
	time.Sleep(500 * time.Millisecond)
	aup, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), tenant.Spec.Contact.Username, metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, false, aup.Spec.Accepted)
	aup.Spec.Accepted = true
	g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), aup, metav1.UpdateOptions{})
	aup, err = g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), g.userObj.Username, metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, false, aup.Spec.Accepted)
	aup.Spec.Accepted = true
	g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), aup, metav1.UpdateOptions{})
	g.mockSigner(tenant.GetName(), tenant.Spec.User)
	g.handler.ObjectCreatedOrUpdated(tenant)

	_, err = g.client.RbacV1().RoleBindings(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("edgenet:tenant-owner-%s", g.tenantObj.Spec.Contact.Username), metav1.GetOptions{})
	util.OK(t, err)
	_, err = g.client.RbacV1().RoleBindings(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("edgenet:tenant-admin-%s", g.userObj.Username), metav1.GetOptions{})
	util.OK(t, err)
	tenant.Spec.Enabled = false
	g.mockSigner(tenant.GetName(), tenant.Spec.User)
	g.handler.ObjectCreatedOrUpdated(tenant)

	_, err = g.client.RbacV1().Roles(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("edgenet:tenant-owner-%s", g.tenantObj.Spec.Contact.Username), metav1.GetOptions{})
	util.Equals(t, "roles.rbac.authorization.k8s.io \"edgenet:tenant-owner-johndoe\" not found", err.Error())
	_, err = g.client.RbacV1().Roles(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("edgenet:tenant-admin-%s", g.userObj.Username), metav1.GetOptions{})
	util.Equals(t, "roles.rbac.authorization.k8s.io \"edgenet:tenant-admin-joepublic\" not found", err.Error())
}

func (g *TestGroup) mockSigner(tenant string, userRaw []corev1alpha.User) {
	// Mock the signer
	go func() {
		timeout := time.After(10 * time.Second)
		ticker := time.Tick(1 * time.Second)
	check:
		for {
			select {
			case <-timeout:
				break check
			case <-ticker:
				allDone := true
			users:
				for _, userRow := range userRaw {
					_, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), userRow.GetName(), metav1.GetOptions{})
					if err == nil {
						continue
					}
					csrObj, err := g.client.CertificatesV1().CertificateSigningRequests().Get(context.TODO(), fmt.Sprintf("%s-%s", tenant, userRow.GetName()), metav1.GetOptions{})
					if err != nil {
						allDone = false
						break users
					}
					csrObj.Status.Certificate = csrObj.Spec.Request
					if _, err := g.client.CertificatesV1().CertificateSigningRequests().UpdateStatus(context.TODO(), csrObj, metav1.UpdateOptions{}); err != nil {
						allDone = false
						break users
					}
				}
				if allDone {
					break check
				}
			}
		}
	}()
}
