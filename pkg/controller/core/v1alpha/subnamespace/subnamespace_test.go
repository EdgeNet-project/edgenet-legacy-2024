package subnamespace

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type TestGroup struct {
	tenantObj        corev1alpha.Tenant
	trqObj           corev1alpha.TenantResourceQuota
	resourceQuotaObj corev1.ResourceQuota
	subNamespaceObj  corev1alpha.SubNamespace
	client           kubernetes.Interface
	edgenetClient    versioned.Interface
	handler          Handler
}

func TestMain(m *testing.M) {
	flag.String("dir", "../../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	//log.SetOutput(ioutil.Discard)
	//logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *TestGroup) Init() {
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "core.edgenet.io/v1alpha",
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
	trqObj := corev1alpha.TenantResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TenantResourceQuota",
			APIVersion: "core.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: corev1alpha.TenantResourceQuotaSpec{
			Claim: []corev1alpha.TenantResourceDetails{
				corev1alpha.TenantResourceDetails{
					Name:   "Default",
					CPU:    "8000m",
					Memory: "8192Mi",
				},
			},
		},
	}
	resourceQuotaObj := corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: "core-quota",
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: map[corev1.ResourceName]resource.Quantity{
				"cpu":              resource.MustParse("8000m"),
				"memory":           resource.MustParse("8192Mi"),
				"requests.storage": resource.MustParse("8Gi"),
			},
		},
	}
	subNamespaceObj := corev1alpha.SubNamespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SubNamespace",
			APIVersion: "core.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenet-sub",
			Namespace: "edgenet",
		},
		Spec: corev1alpha.SubNamespaceSpec{
			Resources: corev1alpha.Resources{
				CPU:    "6000m",
				Memory: "6Gi",
			},
			Inheritance: corev1alpha.Inheritance{
				RBAC:          true,
				NetworkPolicy: true,
			},
		},
	}

	g.tenantObj = tenantObj
	g.trqObj = trqObj
	g.resourceQuotaObj = resourceQuotaObj
	g.subNamespaceObj = subNamespaceObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()

	// Imitate tenant creation processes
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.tenantObj.GetName()}}
	namespaceLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/tenant": g.tenantObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	g.edgenetClient.CoreV1alpha().TenantResourceQuotas().Create(context.TODO(), g.trqObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().ResourceQuotas(namespace.GetName()).Create(context.TODO(), g.resourceQuotaObj.DeepCopy(), metav1.CreateOptions{})
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

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	go g.handler.RunExpiryController()

	subnamespace1 := g.subNamespaceObj.DeepCopy()
	subnamespace1.SetName("all")
	subnamespace2 := g.subNamespaceObj.DeepCopy()
	subnamespace2.SetName("rbac")
	subnamespace3 := g.subNamespaceObj.DeepCopy()
	subnamespace3.SetName("networkpolicy")
	subnamespace4 := g.subNamespaceObj.DeepCopy()
	subnamespace4.SetName("expiry")

	t.Run("inherit all without expiry date", func(t *testing.T) {
		defer time.Sleep(100 * time.Millisecond)
		defer g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Delete(context.TODO(), subnamespace1.GetName(), metav1.DeleteOptions{})

		_, err := g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace1, metav1.CreateOptions{})
		util.OK(t, err)
		g.handler.ObjectCreatedOrUpdated(subnamespace1)
		childNamespace, err := g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subnamespace1.GetName()), metav1.GetOptions{})
		util.OK(t, err)

		coreResourceQuota, _ := g.client.CoreV1().ResourceQuotas(g.tenantObj.GetName()).Get(context.TODO(), "core-quota", metav1.GetOptions{})
		util.Equals(t, int64(2), coreResourceQuota.Spec.Hard.Cpu().Value())
		util.Equals(t, int64(2147483648), coreResourceQuota.Spec.Hard.Memory().Value())

		if roleRaw, err := g.client.RbacV1().Roles(childNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace1.Spec.Inheritance.RBAC {
			// TODO: Provide err information at the status
			for _, roleRow := range roleRaw.Items {
				_, err := g.client.RbacV1().Roles(childNamespace.GetNamespace()).Get(context.TODO(), roleRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
		if roleBindingRaw, err := g.client.RbacV1().RoleBindings(childNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace1.Spec.Inheritance.RBAC {
			// TODO: Provide err information at the status
			for _, roleBindingRow := range roleBindingRaw.Items {
				_, err := g.client.RbacV1().RoleBindings(childNamespace.GetNamespace()).Get(context.TODO(), roleBindingRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
		if networkPolicyRaw, err := g.client.NetworkingV1().NetworkPolicies(childNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace1.Spec.Inheritance.NetworkPolicy {
			// TODO: Provide err information at the status
			for _, networkPolicyRow := range networkPolicyRaw.Items {
				_, err := g.client.NetworkingV1().NetworkPolicies(childNamespace.GetNamespace()).Get(context.TODO(), networkPolicyRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
	})
	t.Run("inherit rbac without expiry date", func(t *testing.T) {
		defer time.Sleep(100 * time.Millisecond)
		defer g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Delete(context.TODO(), subnamespace2.GetName(), metav1.DeleteOptions{})

		_, err := g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace2, metav1.CreateOptions{})
		util.OK(t, err)
		g.handler.ObjectCreatedOrUpdated(subnamespace2)
		childNamespace, err := g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subnamespace2.GetName()), metav1.GetOptions{})
		util.OK(t, err)
		if roleRaw, err := g.client.RbacV1().Roles(childNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace2.Spec.Inheritance.RBAC {
			// TODO: Provide err information at the status
			for _, roleRow := range roleRaw.Items {
				_, err := g.client.RbacV1().Roles(childNamespace.GetNamespace()).Get(context.TODO(), roleRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
		if roleBindingRaw, err := g.client.RbacV1().RoleBindings(childNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace2.Spec.Inheritance.RBAC {
			// TODO: Provide err information at the status
			for _, roleBindingRow := range roleBindingRaw.Items {
				_, err := g.client.RbacV1().RoleBindings(childNamespace.GetNamespace()).Get(context.TODO(), roleBindingRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
		if networkPolicyRaw, err := g.client.NetworkingV1().NetworkPolicies(childNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace2.Spec.Inheritance.NetworkPolicy {
			// TODO: Provide err information at the status
			for _, networkPolicyRow := range networkPolicyRaw.Items {
				_, err := g.client.NetworkingV1().NetworkPolicies(childNamespace.GetNamespace()).Get(context.TODO(), networkPolicyRow.GetName(), metav1.GetOptions{})
				util.Equals(t, true, errors.IsNotFound(err))
			}
		}
	})
	t.Run("inherit networkpolicy without expiry date", func(t *testing.T) {
		defer time.Sleep(100 * time.Millisecond)
		defer g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Delete(context.TODO(), subnamespace3.GetName(), metav1.DeleteOptions{})

		_, err := g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace3, metav1.CreateOptions{})
		util.OK(t, err)
		g.handler.ObjectCreatedOrUpdated(subnamespace3)
		childNamespace, err := g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subnamespace3.GetName()), metav1.GetOptions{})
		util.OK(t, err)
		if roleRaw, err := g.client.RbacV1().Roles(childNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace3.Spec.Inheritance.RBAC {
			// TODO: Provide err information at the status
			for _, roleRow := range roleRaw.Items {
				_, err := g.client.RbacV1().Roles(childNamespace.GetNamespace()).Get(context.TODO(), roleRow.GetName(), metav1.GetOptions{})
				util.Equals(t, true, errors.IsNotFound(err))

			}
		}
		if roleBindingRaw, err := g.client.RbacV1().RoleBindings(childNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace3.Spec.Inheritance.RBAC {
			// TODO: Provide err information at the status
			for _, roleBindingRow := range roleBindingRaw.Items {
				_, err := g.client.RbacV1().RoleBindings(childNamespace.GetNamespace()).Get(context.TODO(), roleBindingRow.GetName(), metav1.GetOptions{})
				util.Equals(t, true, errors.IsNotFound(err))
			}
		}
		if networkPolicyRaw, err := g.client.NetworkingV1().NetworkPolicies(childNamespace.GetNamespace()).List(context.TODO(), metav1.ListOptions{}); err == nil && subnamespace3.Spec.Inheritance.NetworkPolicy {
			// TODO: Provide err information at the status
			for _, networkPolicyRow := range networkPolicyRaw.Items {
				_, err := g.client.NetworkingV1().NetworkPolicies(childNamespace.GetNamespace()).Get(context.TODO(), networkPolicyRow.GetName(), metav1.GetOptions{})
				util.OK(t, err)
			}
		}
	})
	t.Run("inherit all with expiry date", func(t *testing.T) {
		subnamespace4.Spec.Expiry = &metav1.Time{
			Time: time.Now().Add(200 * time.Millisecond),
		}
		_, err := g.edgenetClient.CoreV1alpha().SubNamespaces(g.tenantObj.GetName()).Create(context.TODO(), subnamespace4, metav1.CreateOptions{})
		util.OK(t, err)
		g.handler.ObjectCreatedOrUpdated(subnamespace4)
		_, err = g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subnamespace4.GetName()), metav1.GetOptions{})
		util.OK(t, err)
		time.Sleep(500 * time.Millisecond)
		_, err = g.client.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("%s-%s", g.tenantObj.GetName(), subnamespace4.GetName()), metav1.GetOptions{})
		util.Equals(t, true, errors.IsNotFound(err))
	})
}
