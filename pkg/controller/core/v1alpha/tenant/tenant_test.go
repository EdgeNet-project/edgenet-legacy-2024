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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type TestGroup struct {
	tenantObj        corev1alpha.Tenant
	tenantRequestObj registrationv1alpha.TenantRequest
	userObj          registrationv1alpha.UserRequest
	client           kubernetes.Interface
	edgenetClient    versioned.Interface
	handler          Handler
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
			Contact: corev1alpha.Contact{
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+33NUMBER",
				Username:  "johndoe",
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
			Contact: corev1alpha.Contact{
				Email:     "tom.public@edge-net.org",
				FirstName: "Tom",
				LastName:  "Public",
				Phone:     "+33NUMBER",
				Username:  "tompublic",
			},
		},
		Status: registrationv1alpha.TenantRequestStatus{
			State: established,
		},
	}
	userObj := registrationv1alpha.UserRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRequest",
			APIVersion: "registration.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "johndoe",
			Labels: map[string]string{"edge-net.io/user-template-hash": "1a2b3c"},
		},
		Spec: registrationv1alpha.UserRequestSpec{
			Tenant:    "edgenet",
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@edge-net.org",
			Role:      "Owner",
		},
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
	g.handler.ObjectCreatedOrUpdated(g.tenantObj.DeepCopy())
	g.mockSigner(g.tenantObj.GetName())
	g.handler.ConfigurePermissions(g.tenantObj.DeepCopy(), g.userObj.DeepCopy(), []metav1.OwnerReference{})
	labels := g.userObj.GetLabels()
	aupName := fmt.Sprintf("%s-%s", g.userObj.GetName(), labels["edge-net.io/user-template-hash"])
	t.Run("user configuration", func(t *testing.T) {
		tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), g.tenantObj.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		time.Sleep(500 * time.Millisecond)
		aup, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), aupName, metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, false, aup.Spec.Accepted)

		aup.Spec.Accepted = true
		g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), aup, metav1.UpdateOptions{})
		g.handler.ObjectCreatedOrUpdated(tenant)
		g.handler.ConfigurePermissions(g.tenantObj.DeepCopy(), g.userObj.DeepCopy(), []metav1.OwnerReference{})

		t.Run("cluster role binding", func(t *testing.T) {
			_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-owner-%s-%s", g.tenantObj.GetName(), g.tenantObj.GetName(), g.tenantObj.Spec.Contact.Username, labels["edge-net.io/user-template-hash"]), metav1.GetOptions{})
			util.OK(t, err)
		})
		t.Run("role binding", func(t *testing.T) {
			_, err := g.client.RbacV1().RoleBindings(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("edgenet:tenant-owner-%s-%s", g.tenantObj.Spec.Contact.Username, labels["edge-net.io/user-template-hash"]), metav1.GetOptions{})
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

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Create a tenant to update later
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreatedOrUpdated func to create a user
	g.handler.ObjectCreatedOrUpdated(g.tenantObj.DeepCopy())
	g.mockSigner(g.tenantObj.GetName())
	g.handler.ConfigurePermissions(g.tenantObj.DeepCopy(), g.userObj.DeepCopy(), []metav1.OwnerReference{})
	labels := g.userObj.GetLabels()
	aupName := fmt.Sprintf("%s-%s", g.userObj.GetName(), labels["edge-net.io/user-template-hash"])
	tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), g.tenantObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	// util.Equals(t, tenant.Spec.Contact.Username, tenant.Spec.User[0].Username)
	g.mockSigner(tenant.GetName())
	g.handler.ObjectCreatedOrUpdated(tenant)
	time.Sleep(500 * time.Millisecond)

	aup, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), aupName, metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, false, aup.Spec.Accepted)
	aup.Spec.Accepted = true
	g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), aup, metav1.UpdateOptions{})
	g.handler.ObjectCreatedOrUpdated(tenant)
	g.handler.ConfigurePermissions(g.tenantObj.DeepCopy(), g.userObj.DeepCopy(), []metav1.OwnerReference{})

	_, err = g.client.RbacV1().RoleBindings(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("edgenet:tenant-owner-%s-%s", g.tenantObj.Spec.Contact.Username, labels["edge-net.io/user-template-hash"]), metav1.GetOptions{})
	util.OK(t, err)
	tenant.Spec.Enabled = false
	g.handler.ObjectCreatedOrUpdated(tenant)

	_, err = g.client.RbacV1().Roles(g.tenantObj.GetName()).Get(context.TODO(), fmt.Sprintf("edgenet:tenant-owner-%s", g.tenantObj.Spec.Contact.Username), metav1.GetOptions{})
	util.Equals(t, "roles.rbac.authorization.k8s.io \"edgenet:tenant-owner-johndoe\" not found", err.Error())

	t.Run("tenant status update", func(t *testing.T) {
		tenantStatusTest := g.tenantObj
		tenantStatusTest.SetName("status-test")
		tenantStatusTest.Status.State = failure
		tenantStatusTest.Status.Message = append(tenantStatusTest.Status.Message, statusDict["namespace-failure"])
		g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), tenantStatusTest.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreatedOrUpdated(tenantStatusTest.DeepCopy())
		tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), tenantStatusTest.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, statusDict["tenant-established"], tenant.Status.Message[0])
	})
}

func (g *TestGroup) mockSigner(tenant string) {
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
				if acceptableUsePolicyRaw, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("edge-net.io/generated=true,edge-net.io/tenant=%s,edge-net.io/identity=true", tenant)}); err == nil {
				users:
					for _, acceptableUsePolicyRow := range acceptableUsePolicyRaw.Items {
						aupLabels := acceptableUsePolicyRow.GetLabels()
						if aupLabels != nil && aupLabels["edge-net.io/username"] != "" && aupLabels["edge-net.io/user-template-hash"] != "" {
							if aupLabels["edge-net.io/role"] == "Owner" || aupLabels["edge-net.io/role"] == "Admin" {
								_, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), fmt.Sprintf("%s-%s", aupLabels["edge-net.io/username"], aupLabels["edge-net.io/user-template-hash"]), metav1.GetOptions{})
								if err != nil {
									continue
								}
								csrObj, err := g.client.CertificatesV1().CertificateSigningRequests().Get(context.TODO(), fmt.Sprintf("%s-%s-%s", tenant, aupLabels["edge-net.io/username"], aupLabels["edge-net.io/user-template-hash"]), metav1.GetOptions{})
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
						}
					}
				} else {
					log.Println(err)
				}

				if allDone {
					break check
				}
			}
		}
	}()
}
