/*
Copyright 2021 Contributors to the EdgeNet project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package access

import (
	"context"
	"fmt"
	"testing"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
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

type TestGroup struct {
	tenant                 corev1alpha.Tenant
	namespace              corev1.Namespace
	tenantObj              corev1alpha.Tenant
	tenantRequest          registrationv1alpha.TenantRequest
	tenantResourceQuotaObj corev1alpha.TenantResourceQuota
	client                 kubernetes.Interface
	edgenetclient          versioned.Interface
}

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
			Contact: corev1alpha.Contact{
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+333333333",
				Handle:    "johndoe",
			},
			Enabled: false,
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
				Handle:    "tompublic",
			},
		},
	}
	tenantResourceQuotaObj := corev1alpha.TenantResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "tenantResourceQuota",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
			UID:  "trq",
		},
	}
	g.tenantResourceQuotaObj = tenantResourceQuotaObj
	g.tenantRequest = tenantRequestObj
	g.tenantObj = tenantObj
	g.tenantObj.Spec.Enabled = true
	g.tenant = tenantObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	Clientset = g.client
	EdgenetClientset = g.edgenetclient
	g.namespace = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.tenant.GetName()}}
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
		"default tenant owner": {"edgenet:tenant-owner"},
		"default tenant admin": {"edgenet:tenant-admin"},
		"default collaborator": {"edgenet:tenant-collaborator"},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			_, err = g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
		})
	}

	tenant := g.tenant
	t.Run("role binding", func(t *testing.T) {
		cases := map[string]struct {
			tenant        string
			namespace     string
			roleName      string
			initialHandle string
			email         string
			expected      string
		}{
			"owner":        {tenant.GetName(), tenant.GetName(), "edgenet:tenant-owner", g.tenant.Spec.Contact.Handle, g.tenant.Spec.Contact.Email, fmt.Sprintf("edgenet:tenant-owner-%s", g.tenant.Spec.Contact.Handle)},
			"collaborator": {tenant.GetName(), tenant.GetName(), "edgenet:tenant-collaborator", g.tenant.Spec.Contact.Handle, g.tenant.Spec.Contact.Email, fmt.Sprintf("edgenet:tenant-collaborator-%s", g.tenant.Spec.Contact.Handle)},
			"admin":        {tenant.GetName(), tenant.GetName(), "edgenet:tenant-admin", g.tenant.Spec.Contact.Handle, g.tenant.Spec.Contact.Email, fmt.Sprintf("edgenet:tenant-admin-%s", g.tenant.Spec.Contact.Handle)},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				CreateObjectSpecificRoleBinding(tc.tenant, tc.namespace, tc.roleName, tc.initialHandle, tc.email)
				_, err := g.client.RbacV1().RoleBindings(tenant.GetName()).Get(context.TODO(), tc.expected, metav1.GetOptions{})
				util.OK(t, err)
				err = CreateObjectSpecificRoleBinding(tc.tenant, tc.namespace, tc.roleName, tc.initialHandle, tc.email)
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
		apiGroup     string
		resource     string
		resourceName string
		verbs        []string
		expected     string
	}{
		"tenant":                {tenant1, "core.edgenet.io", "tenants", tenant1.GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("edgenet:%s:tenants:%s-name", tenant1.GetName(), tenant1.GetName())},
		"tenant resource quota": {tenant1, "core.edgenet.io", "tenantresourcequotas", tenant1.GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("edgenet:%s:tenantresourcequotas:%s-name", tenant1.GetName(), tenant1.GetName())},
		"node contribution":     {tenant2, "core.edgenet.io", "nodecontributions", "ple", []string{"get", "update", "patch", "delete"}, fmt.Sprintf("edgenet:%s:nodecontributions:%s-name", tenant2.GetName(), "ple")},
		"acceptable use policy": {tenant1, "core.edgenet.io", "acceptableusepolicies", tenant1.Spec.Contact.Handle, []string{"get", "update", "patch"}, fmt.Sprintf("edgenet:%s:acceptableusepolicies:%s-name", tenant1.GetName(), tenant1.Spec.Contact.Handle)},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			CreateObjectSpecificClusterRole(tc.tenant.GetName(), tc.apiGroup, tc.resource, tc.resourceName, "name", tc.verbs, []metav1.OwnerReference{})
			clusterRole, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
			if err == nil {
				util.Equals(t, tc.verbs, clusterRole.Rules[0].Verbs)
			}
			_, err = CreateObjectSpecificClusterRole(tc.tenant.GetName(), tc.apiGroup, tc.resource, tc.resourceName, "name", tc.verbs, []metav1.OwnerReference{})
			util.OK(t, err)
		})
	}

	t.Run("cluster role binding", func(t *testing.T) {
		cases := map[string]struct {
			roleName      string
			initialHandle string
			email         string
			expected      string
		}{
			"tenant":                {fmt.Sprintf("%s-tenants-%s", tenant1.GetName(), tenant1.GetName()), tenant1.Spec.Contact.Handle, tenant1.Spec.Contact.Email, fmt.Sprintf("%s-tenants-%s-%s", tenant1.GetName(), tenant1.GetName(), tenant1.Spec.Contact.Handle)},
			"tenant resource quota": {fmt.Sprintf("%s-tenantresourcequotas-%s", tenant1.GetName(), tenant1.GetName()), tenant1.Spec.Contact.Handle, tenant1.Spec.Contact.Email, fmt.Sprintf("%s-tenantresourcequotas-%s-%s", tenant1.GetName(), tenant1.GetName(), tenant1.Spec.Contact.Handle)},
			"node contribution":     {fmt.Sprintf("%s-nodecontributions-%s", tenant1.GetName(), "ple"), tenant1.Spec.Contact.Handle, tenant1.Spec.Contact.Email, fmt.Sprintf("%s-nodecontributions-%s-%s", tenant1.GetName(), "ple", tenant1.Spec.Contact.Handle)},
			"acceptable use policy": {fmt.Sprintf("%s-acceptableusepolicies-%s", tenant1.GetName(), tenant1.Spec.Contact.Handle), tenant1.Spec.Contact.Handle, tenant1.Spec.Contact.Email, fmt.Sprintf("%s-acceptableusepolicies-%s-%s", tenant1.GetName(), tenant1.Spec.Contact.Handle, tenant1.Spec.Contact.Handle)},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				roleBindLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/identity": "true"}
				CreateObjectSpecificClusterRoleBinding(tc.roleName, tc.initialHandle, tc.email, roleBindLabels, []metav1.OwnerReference{})
				_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("%s-%s", tc.roleName, tc.initialHandle), metav1.GetOptions{})
				util.OK(t, err)
				err = CreateObjectSpecificClusterRoleBinding(tc.roleName, tc.initialHandle, tc.email, roleBindLabels, []metav1.OwnerReference{})
				util.OK(t, err)
			})
		}
	})
}

func TestPermissionSystem(t *testing.T) {
	g := TestGroup{}
	g.Init()

	tenant := g.tenant

	owner := map[string]string{"InitialHandle": "johndoe", "Email": "john.doe@edge-net.org"}
	collaborator := map[string]string{"InitialHandle": "tompublic", "Email": "tom.public@edge-net.org"}
	admin := map[string]string{"InitialHandle": "joedoe", "Email": "joe.doe@edge-net.org"}

	err := CreateClusterRoles()
	util.OK(t, err)
	cases := map[string]struct {
		expected string
	}{
		"create cluster role for tenant owner":         {"edgenet:tenant-owner"},
		"create cluster role for default collaborator": {"edgenet:tenant-collaborator"},
		"create cluster role for default tenant admin": {"edgenet:tenant-admin"},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
		})
	}
	t.Run("bind cluster role for tenant owner", func(t *testing.T) {
		err = CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), "edgenet:tenant-owner", owner["InitialHandle"], owner["Email"])
		util.OK(t, err)
	})
	t.Run("bind cluster role for tenant collaborator", func(t *testing.T) {
		err = CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), "edgenet:tenant-collaborator", collaborator["InitialHandle"], collaborator["Email"])
		util.OK(t, err)
	})
	t.Run("bind cluster role for tenant admin", func(t *testing.T) {
		err = CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), "edgenet:tenant-admin", admin["InitialHandle"], admin["Email"])
		util.OK(t, err)
	})

	t.Run("create owner specific tenant role", func(t *testing.T) {
		CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "tenants", tenant.GetName(), "owner", []string{"get", "update", "patch"}, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-owner", tenant.GetName(), tenant.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("create admin specific tenant role", func(t *testing.T) {
		CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "tenants", tenant.GetName(), "admin", []string{"get"}, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-admin", tenant.GetName(), tenant.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("create owner role binding", func(t *testing.T) {
		roleBindLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/identity": "true"}

		CreateObjectSpecificClusterRoleBinding(fmt.Sprintf("edgenet:%s:tenants:%s-owner", tenant.GetName(), tenant.GetName()), owner["InitialHandle"], owner["Email"], roleBindLabels, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-owner-%s", tenant.GetName(), tenant.GetName(), owner["InitialHandle"]), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("create admin role binding", func(t *testing.T) {
		roleBindLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/identity": "true"}

		CreateObjectSpecificClusterRoleBinding(fmt.Sprintf("edgenet:%s:tenants:%s-admin", tenant.GetName(), tenant.GetName()), admin["InitialHandle"], admin["Email"], roleBindLabels, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-admin-%s", tenant.GetName(), tenant.GetName(), admin["InitialHandle"]), metav1.GetOptions{})
		util.OK(t, err)
	})

	permissionCases := map[string]struct {
		user         map[string]string
		namespace    string
		resource     string
		resourceName string
		scope        string
		expected     bool
	}{
		"owner/authorized for subnamespace":          {owner, g.namespace.GetName(), "subnamespaces", "", "namespace", true},
		"collaborator/authorized for subnamespace":   {collaborator, g.namespace.GetName(), "subnamespaces", "", "namespace", false},
		"owner/authorized for roles":                 {owner, g.namespace.GetName(), "roles", "", "namespace", true},
		"collaborator/authorized for roles":          {collaborator, g.namespace.GetName(), "roles", "", "namespace", false},
		"owner/authorized for role bindings":         {owner, g.namespace.GetName(), "rolebindings", "", "namespace", true},
		"admin/authorized for role bindings":         {admin, g.namespace.GetName(), "rolebindings", "", "namespace", true},
		"owner/authorized for cluster role bindings": {owner, "", "clusterrolebindings", "", "cluster", false},
		"owner/authorized for tenant object":         {owner, "", "tenants", tenant.GetName(), "cluster", true},
		"collaborator/authorized for tenant object":  {collaborator, "", "tenants", tenant.GetName(), "cluster", false},
		"admin/authorized for tenant object":         {admin, "", "tenants", tenant.GetName(), "cluster", false},
	}
	for k, tc := range permissionCases {
		t.Run(k, func(t *testing.T) {
			authorized := CheckAuthorization(tc.namespace, tc.user["Email"], tc.resource, tc.resourceName, tc.scope)
			util.Equals(t, tc.expected, authorized)
		})
	}
}

func TestApplyTenantResourceQuota(t *testing.T) {
	g := TestGroup{}
	g.Init()

	_, err := EdgenetClientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), g.tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
	claim := corev1alpha.ResourceTuning{
		ResourceList: map[corev1.ResourceName]resource.Quantity{
			"cpu":    resource.MustParse("6000m"),
			"memory": resource.MustParse("6Gi"),
		},
	}
	ApplyTenantResourceQuota(g.tenantResourceQuotaObj.GetName(), nil, claim)
	_, err = EdgenetClientset.CoreV1alpha().TenantResourceQuotas().Get(context.TODO(), g.tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
}
