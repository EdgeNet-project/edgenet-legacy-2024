package permission

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/Sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type TestGroup struct {
	tenant        corev1alpha.Tenant
	user          corev1alpha.User
	client        kubernetes.Interface
	edgenetclient versioned.Interface
	namespace     corev1.Namespace
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

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
				Phone:     "+333333333",
				Username:  "johndoe",
			},
			User: []corev1alpha.User{
				corev1alpha.User{
					Username:  "johndoe",
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john.doe@edge-net.org",
					Role:      "Owner",
				},
			},
			Enabled: false,
		},
	}
	userObj := corev1alpha.User{
		Username:  "johnsmith",
		FirstName: "John",
		LastName:  "Smith",
		Email:     "john.smith@edge-net.org",
		Role:      "Collaborator",
	}
	g.tenant = tenantObj
	g.user = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	Clientset = g.client
	g.namespace = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s", g.tenant.GetName())}}
	g.client.CoreV1().Namespaces().Create(context.TODO(), &g.namespace, metav1.CreateOptions{})
}

func TestCreateClusterRoles(t *testing.T) {
	g := TestGroup{}
	g.Init()

	err := CreateClusterRoles()
	util.OK(t, err)

	cases := map[string]struct {
		expected string
	}{
		"default tenant owner": {"tenant-owner"},
		"default tenant admin": {"tenant-admin"},
		"default collaborator": {"tenant-collaborator"},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			_, err = g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
		})
	}

	tenant := g.tenant
	user1 := g.user
	user2 := g.user
	user2.Username = "joepublic"
	user2.FirstName = "Joe"
	user2.LastName = "Public"
	user2.Email = "joe.public@edge-net.org"
	user2.Role = "Admin"

	t.Run("role binding", func(t *testing.T) {
		cases := map[string]struct {
			tenant    string
			namespace string
			roleName  string
			user      corev1alpha.User
			expected  string
		}{
			"owner":        {tenant.GetName(), tenant.GetName(), "tenant-owner", tenant.Spec.User[0], fmt.Sprintf("tenant-owner-%s", tenant.Spec.User[0].GetName())},
			"collaborator": {tenant.GetName(), tenant.GetName(), "tenant-collaborator", user1, fmt.Sprintf("tenant-collaborator-%s", user1.GetName())},
			"admin":        {tenant.GetName(), tenant.GetName(), "tenant-admin", user2, fmt.Sprintf("tenant-admin-%s", user2.GetName())},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				CreateObjectSpecificRoleBinding(tc.tenant, tc.namespace, tc.roleName, tc.user)
				_, err := g.client.RbacV1().RoleBindings(tenant.GetName()).Get(context.TODO(), tc.expected, metav1.GetOptions{})
				util.OK(t, err)
				err = CreateObjectSpecificRoleBinding(tc.tenant, tc.namespace, tc.roleName, tc.user)
				util.OK(t, err)
			})
		}
	})
	err = CreateClusterRoles()
	util.OK(t, err)
}

func TestCreateObjectSpecificClusterRole(t *testing.T) {
	g := TestGroup{}
	g.Init()

	tenant1 := g.tenant
	tenant2 := g.tenant
	tenant2.SetName("lip6")

	cases := map[string]struct {
		tenant       corev1alpha.Tenant
		resource     string
		resourceName string
		verbs        []string
		expected     string
	}{
		"tenant":                    {tenant1, "tenants", tenant1.GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("%s-tenants-%s-name", tenant1.GetName(), tenant1.GetName())},
		"tenant resource quota":     {tenant1, "tenantresourcequotas", tenant1.GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("%s-tenantresourcequotas-%s-name", tenant1.GetName(), tenant1.GetName())},
		"node contribution":         {tenant2, "nodecontributions", "ple", []string{"get", "update", "patch", "delete"}, fmt.Sprintf("%s-nodecontributions-%s-name", tenant2.GetName(), "ple")},
		"user registration request": {tenant1, "userregistrationrequests", g.user.GetName(), []string{"get", "update", "patch", "delete"}, fmt.Sprintf("%s-userregistrationrequests-%s-name", tenant1.GetName(), g.user.GetName())},
		"email verification":        {tenant2, "emailverifications", "abcdefghi", []string{"get", "update", "patch", "delete"}, fmt.Sprintf("%s-emailverifications-%s-name", tenant2.GetName(), "abcdefghi")},
		"acceptable use policy":     {tenant1, "acceptableusepolicies", tenant1.Spec.User[0].GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("%s-acceptableusepolicies-%s-name", tenant1.GetName(), tenant1.Spec.User[0].GetName())},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			CreateObjectSpecificClusterRole(tc.tenant.GetName(), tc.resource, tc.resourceName, "name", tc.verbs, []metav1.OwnerReference{})
			clusterRole, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
			if err == nil {
				util.Equals(t, tc.verbs, clusterRole.Rules[0].Verbs)
			}
			err = CreateObjectSpecificClusterRole(tc.tenant.GetName(), tc.resource, tc.resourceName, "name", tc.verbs, []metav1.OwnerReference{})
			util.OK(t, err)
		})
	}

	t.Run("cluster role binding", func(t *testing.T) {
		cases := map[string]struct {
			tenant   string
			roleName string
			user     corev1alpha.User
			expected string
		}{
			"tenant":                    {tenant1.GetName(), fmt.Sprintf("%s-tenants-%s", tenant1.GetName(), tenant1.GetName()), tenant1.Spec.User[0], fmt.Sprintf("%s-tenants-%s-%s", tenant1.GetName(), tenant1.GetName(), tenant1.Spec.User[0].GetName())},
			"tenant resource quota":     {tenant1.GetName(), fmt.Sprintf("%s-tenantresourcequotas-%s", tenant1.GetName(), tenant1.GetName()), tenant1.Spec.User[0], fmt.Sprintf("%s-tenantresourcequotas-%s-%s", tenant1.GetName(), tenant1.GetName(), tenant1.Spec.User[0].GetName())},
			"node contribution":         {tenant1.GetName(), fmt.Sprintf("%s-nodecontributions-%s", tenant1.GetName(), "ple"), tenant1.Spec.User[0], fmt.Sprintf("%s-nodecontributions-%s-%s", tenant1.GetName(), "ple", tenant1.Spec.User[0].GetName())},
			"user registration request": {tenant1.GetName(), fmt.Sprintf("%s-userregistrationrequests-%s", tenant1.GetName(), g.user.GetName()), tenant1.Spec.User[0], fmt.Sprintf("%s-userregistrationrequests-%s-%s", tenant1.GetName(), g.user.GetName(), tenant1.Spec.User[0].GetName())},
			"email verification":        {tenant1.GetName(), fmt.Sprintf("%s-emailverifications-%s", tenant1.GetName(), "abcdefghi"), g.user, fmt.Sprintf("%s-emailverifications-%s-%s", tenant1.GetName(), "abcdefghi", g.user.GetName())},
			"acceptable use policy":     {tenant1.GetName(), fmt.Sprintf("%s-acceptableusepolicies-%s", tenant1.GetName(), tenant1.Spec.User[0].GetName()), tenant1.Spec.User[0], fmt.Sprintf("%s-acceptableusepolicies-%s-%s", tenant1.GetName(), tenant1.Spec.User[0].GetName(), tenant1.Spec.User[0].GetName())},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				CreateObjectSpecificClusterRoleBinding(tc.tenant, tc.roleName, tc.user, []metav1.OwnerReference{})
				_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), tc.expected, metav1.GetOptions{})
				util.OK(t, err)
				err = CreateObjectSpecificClusterRoleBinding(tc.tenant, tc.roleName, tc.user, []metav1.OwnerReference{})
				util.OK(t, err)
			})
		}
	})
}

func TestPermissionSystem(t *testing.T) {
	g := TestGroup{}
	g.Init()

	tenant := g.tenant
	user1 := tenant.Spec.User[0]
	user2 := g.user
	user3 := g.user
	user3.Username = "joepublic"
	user3.FirstName = "Joe"
	user3.LastName = "Public"
	user3.Email = "joe.public@edge-net.org"
	user3.Role = "Admin"

	err := CreateClusterRoles()
	util.OK(t, err)
	cases := map[string]struct {
		expected string
	}{
		"create cluster role for tenant owner":         {"tenant-owner"},
		"create cluster role for default collaborator": {"tenant-collaborator"},
		"create cluster role for default tenant admin": {"tenant-admin"},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
		})
	}
	t.Run("bind cluster role for tenant owner", func(t *testing.T) {
		CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), "tenant-owner", user1)
	})
	t.Run("bind cluster role for tenant collaborator", func(t *testing.T) {
		CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), "tenant-collaborator", user2)
	})
	t.Run("bind cluster role for tenant admin", func(t *testing.T) {
		CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), "tenant-admin", user3)
	})

	t.Run("create owner specific tenant role", func(t *testing.T) {
		CreateObjectSpecificClusterRole(tenant.GetName(), "tenants", tenant.GetName(), "owner", []string{"get", "update", "patch"}, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("%s-tenants-%s-owner", tenant.GetName(), tenant.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("create admin specific tenant role", func(t *testing.T) {
		CreateObjectSpecificClusterRole(tenant.GetName(), "tenants", tenant.GetName(), "admin", []string{"get"}, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("%s-tenants-%s-admin", tenant.GetName(), tenant.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("create owner role binding", func(t *testing.T) {
		CreateObjectSpecificClusterRoleBinding(tenant.GetName(), fmt.Sprintf("%s-tenants-%s-owner", tenant.GetName(), tenant.GetName()), user1, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("%s-tenants-%s-owner-%s", tenant.GetName(), tenant.GetName(), user1.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("create admin role binding", func(t *testing.T) {
		CreateObjectSpecificClusterRoleBinding(tenant.GetName(), fmt.Sprintf("%s-tenants-%s-admin", tenant.GetName(), tenant.GetName()), user3, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("%s-tenants-%s-admin-%s", tenant.GetName(), tenant.GetName(), user3.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})

	permissionCases := map[string]struct {
		user         corev1alpha.User
		namespace    string
		resource     string
		resourceName string
		scope        string
		expected     bool
	}{
		"owner/authorized for subnamespace":          {user1, g.namespace.GetName(), "subnamespaces", "", "namespace", true},
		"collaborator/authorized for subnamespace":   {user2, g.namespace.GetName(), "subnamespaces", "", "namespace", false},
		"owner/authorized for roles":                 {user1, g.namespace.GetName(), "roles", "", "namespace", true},
		"collaborator/authorized for roles":          {user2, g.namespace.GetName(), "roles", "", "namespace", false},
		"owner/authorized for role bindings":         {user1, g.namespace.GetName(), "rolebindings", "", "namespace", true},
		"admin/authorized for role bindings":         {user3, g.namespace.GetName(), "rolebindings", "", "namespace", true},
		"owner/authorized for cluster role bindings": {user1, "", "clusterrolebindings", "", "cluster", false},
		"owner/authorized for tenant object":         {user1, "", "tenants", tenant.GetName(), "cluster", true},
		"collaborator/authorized for tenant object":  {user2, "", "tenants", tenant.GetName(), "cluster", false},
		"admin/authorized for tenant object":         {user3, "", "tenants", tenant.GetName(), "cluster", false},
	}
	for k, tc := range permissionCases {
		t.Run(k, func(t *testing.T) {
			authorized := CheckAuthorization(tc.namespace, tc.user.Email, tc.resource, tc.resourceName, tc.scope)
			util.Equals(t, tc.expected, authorized)
		})
	}
}
