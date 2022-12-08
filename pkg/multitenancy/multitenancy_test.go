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

package multitenancy

import (
	"context"
	"fmt"
	"testing"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	registrationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha1"
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
	tenant                 corev1alpha1.Tenant
	namespace              corev1.Namespace
	tenantObj              corev1alpha1.Tenant
	tenantRequest          registrationv1alpha1.TenantRequest
	tenantResourceQuotaObj corev1alpha1.TenantResourceQuota
	client                 kubernetes.Interface
	edgenetclient          versioned.Interface
	multitenancyManager    *Manager
}

func (g *TestGroup) Init() {
	tenantObj := corev1alpha1.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "core.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: corev1alpha1.TenantSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: corev1alpha1.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha1.Contact{
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+333333333",
			},
			Enabled: false,
		},
	}
	tenantRequestObj := registrationv1alpha1.TenantRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "tenantRequest",
			APIVersion: "apps.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet-request",
		},
		Spec: registrationv1alpha1.TenantRequestSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: corev1alpha1.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha1.Contact{
				Email:     "tom.public@edge-net.org",
				FirstName: "Tom",
				LastName:  "Public",
				Phone:     "+33NUMBER",
			},
		},
	}
	tenantResourceQuotaObj := corev1alpha1.TenantResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "tenantResourceQuota",
			APIVersion: "apps.edgenet.io/v1alpha1",
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
	g.namespace = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.tenant.GetName()}}
	g.client.CoreV1().Namespaces().Create(context.TODO(), &g.namespace, metav1.CreateOptions{})
	multitenancyManager := NewManager(g.client, g.edgenetclient)
	g.multitenancyManager = multitenancyManager
}

func TestCreateObjectSpecificClusterRole(t *testing.T) {
	g := TestGroup{}
	g.Init()

	tenant1 := g.tenant
	tenant2 := g.tenant
	tenant2.SetName("lip6")

	cases := map[string]struct {
		tenant       corev1alpha1.Tenant
		apiGroup     string
		resource     string
		resourceName string
		verbs        []string
		expected     string
	}{
		"tenant":                {tenant1, "core.edgenet.io", "tenants", tenant1.GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("edgenet:tenants:%s-name", tenant1.GetName())},
		"tenant resource quota": {tenant1, "core.edgenet.io", "tenantresourcequotas", tenant1.GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("edgenet:tenantresourcequotas:%s-name", tenant1.GetName())},
		"node contribution":     {tenant2, "core.edgenet.io", "nodecontributions", "ple", []string{"get", "update", "patch", "delete"}, fmt.Sprintf("edgenet:nodecontributions:%s-name", "ple")},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			g.multitenancyManager.createObjectSpecificClusterRole(tc.apiGroup, tc.resource, tc.resourceName, "name", tc.verbs, []metav1.OwnerReference{})
			clusterRole, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
			if err == nil {
				util.Equals(t, tc.verbs, clusterRole.Rules[0].Verbs)
			}
			_, err = g.multitenancyManager.createObjectSpecificClusterRole(tc.apiGroup, tc.resource, tc.resourceName, "name", tc.verbs, []metav1.OwnerReference{})
			util.OK(t, err)
		})
	}

	t.Run("cluster role binding", func(t *testing.T) {
		cases := map[string]struct {
			roleName string
			email    string
			expected string
		}{
			"tenant":                {fmt.Sprintf("%s-tenants-%s", tenant1.GetName(), tenant1.GetName()), tenant1.Spec.Contact.Email, fmt.Sprintf("%s-tenants-%s", tenant1.GetName(), tenant1.GetName())},
			"tenant resource quota": {fmt.Sprintf("%s-tenantresourcequotas-%s", tenant1.GetName(), tenant1.GetName()), tenant1.Spec.Contact.Email, fmt.Sprintf("%s-tenantresourcequotas-%s", tenant1.GetName(), tenant1.GetName())},
			"node contribution":     {fmt.Sprintf("%s-nodecontributions-%s", tenant1.GetName(), "ple"), tenant1.Spec.Contact.Email, fmt.Sprintf("%s-nodecontributions-ple", tenant1.GetName())},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				g.multitenancyManager.createObjectSpecificClusterRoleBinding(tc.roleName, tc.email, []metav1.OwnerReference{})
				_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), tc.roleName, metav1.GetOptions{})
				util.OK(t, err)
				err = g.multitenancyManager.createObjectSpecificClusterRoleBinding(tc.roleName, tc.email, []metav1.OwnerReference{})
				util.OK(t, err)
			})
		}
	})
}

func TestApplyTenantResourceQuota(t *testing.T) {
	g := TestGroup{}
	g.Init()

	_, err := g.multitenancyManager.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), g.tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
	claim := corev1alpha1.ResourceTuning{
		ResourceList: map[corev1.ResourceName]resource.Quantity{
			"cpu":    resource.MustParse("6000m"),
			"memory": resource.MustParse("6Gi"),
		},
	}
	applied := make(chan error)
	g.multitenancyManager.ApplyTenantResourceQuota(g.tenantResourceQuotaObj.GetName(), nil, claim, applied)
	util.OK(t, <-applied)
	_, err = g.multitenancyManager.edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), g.tenantResourceQuotaObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
}
