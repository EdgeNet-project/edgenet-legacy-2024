package permission

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type TestGroup struct {
	authorityObj   apps_v1alpha.Authority
	userObj        apps_v1alpha.User
	client         kubernetes.Interface
	edgenetclient  versioned.Interface
	namespace      corev1.Namespace
	teamNamespace  corev1.Namespace
	sliceNamespace corev1.Namespace
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func (g *TestGroup) Init() {
	authorityObj := apps_v1alpha.Authority{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Authority",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: apps_v1alpha.AuthoritySpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: apps_v1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: apps_v1alpha.Contact{
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+333333333",
				Username:  "johndoe",
			},
			Enabled: false,
		},
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "johndoe",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "EdgeNet",
			LastName:  "EdgeNet",
			Email:     "john.doe@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			Type: "admin",
		},
	}
	g.authorityObj = authorityObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	// Sync Clientset with fake client
	Clientset = g.client
	// Create namespaces
	g.namespace = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", g.authorityObj.GetName())}}
	g.client.CoreV1().Namespaces().Create(context.TODO(), &g.namespace, metav1.CreateOptions{})
	g.sliceNamespace = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s-slice-1", g.authorityObj.GetName())}}
	g.client.CoreV1().Namespaces().Create(context.TODO(), &g.sliceNamespace, metav1.CreateOptions{})
	g.teamNamespace = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s-team-1", g.authorityObj.GetName())}}
	g.client.CoreV1().Namespaces().Create(context.TODO(), &g.teamNamespace, metav1.CreateOptions{})
}

func TestCreateClusterRoles(t *testing.T) {
	g := TestGroup{}
	g.Init()

	authority1 := g.authorityObj
	authority2 := g.authorityObj
	authority2.SetName("edgenet-2")
	authority3 := g.authorityObj
	authority2.SetName("edgenet-3")

	cases := map[string]struct {
		authority apps_v1alpha.Authority
		expected  string
	}{
		"create cluster role 1":        {authority1, fmt.Sprintf("authority-%s", authority1.GetName())},
		"update existing cluster role": {authority1, fmt.Sprintf("authority-%s", authority1.GetName())},
		"create cluster role 2":        {authority2, fmt.Sprintf("authority-%s", authority2.GetName())},
		"create cluster role 3":        {authority3, fmt.Sprintf("authority-%s", authority3.GetName())},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			err := CreateClusterRoles(tc.authority.DeepCopy())
			util.OK(t, err)
			_, err = g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
		})
	}
}

func TestEstablishPrivateRoleBindings(t *testing.T) {
	g := TestGroup{}
	g.Init()

	user1 := g.userObj
	user2 := g.userObj
	user2.SetName("johndoe-2")
	user3 := g.userObj
	user3.SetName("johndoe-3")

	cases := map[string]struct {
		user        apps_v1alpha.User
		bindingType []string
		expected    []string
	}{
		"create private role bindings 1": {
			user1,
			[]string{"Role", "ClusterRole"},
			[]string{
				fmt.Sprintf("%s-user-%s", user1.GetNamespace(), user1.GetName()),
				fmt.Sprintf("%s-%s-for-authority", user1.GetNamespace(), user1.GetName()),
			},
		},
		"update existing private role bindings": {
			user1,
			[]string{"Role", "ClusterRole"},
			[]string{
				fmt.Sprintf("%s-user-%s", user1.GetNamespace(), user1.GetName()),
				fmt.Sprintf("%s-%s-for-authority", user1.GetNamespace(), user1.GetName()),
			},
		},
		"create private role bindings 2": {
			user2,
			[]string{"Role", "ClusterRole"},
			[]string{
				fmt.Sprintf("%s-user-%s", user2.GetNamespace(), user2.GetName()),
				fmt.Sprintf("%s-%s-for-authority", user2.GetNamespace(), user2.GetName()),
			},
		},
		"create private role bindings 3": {
			user3,
			[]string{"Role", "ClusterRole"},
			[]string{
				fmt.Sprintf("%s-user-%s", user3.GetNamespace(), user3.GetName()),
				fmt.Sprintf("%s-%s-for-authority", user3.GetNamespace(), user3.GetName()),
			},
		},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			err := EstablishPrivateRoleBindings(tc.user.DeepCopy())
			util.OK(t, err)
			for i, bindingName := range tc.expected {
				if tc.bindingType[i] == "Role" {
					_, err = g.client.RbacV1().RoleBindings(tc.user.GetNamespace()).Get(context.TODO(), bindingName, metav1.GetOptions{})
					util.OK(t, err)
				} else if tc.bindingType[i] == "ClusterRole" {
					_, err = g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), bindingName, metav1.GetOptions{})
					util.OK(t, err)
				}
			}
		})
	}
}

func TestEstablishRoleBindings(t *testing.T) {
	g := TestGroup{}
	g.Init()

	user1 := g.userObj
	user2 := g.userObj
	user2.SetName("johndoe-2")
	user3 := g.userObj
	user3.SetName("johndoe-3")
	user3.Status.Type = "user"

	cases := map[string]struct {
		user          apps_v1alpha.User
		namespaceType string
		expected      string
	}{
		"create role 1": {
			user1,
			"Authority",
			fmt.Sprintf("%s-%s-%s-%s", user1.GetNamespace(), user1.GetName(), strings.ToLower("Authority"), strings.ToLower(user1.Status.Type)),
		},
		"update existing role": {
			user1,
			"Authority",
			fmt.Sprintf("%s-%s-%s-%s", user1.GetNamespace(), user1.GetName(), strings.ToLower("Authority"), strings.ToLower(user1.Status.Type)),
		},
		"create role 2": {
			user2,
			"Slice",
			fmt.Sprintf("%s-%s-%s-%s", user2.GetNamespace(), user2.GetName(), strings.ToLower("Slice"), strings.ToLower(user2.Status.Type)),
		},
		"create role 3": {
			user3,
			"Team",
			fmt.Sprintf("%s-%s-%s-%s", user3.GetNamespace(), user3.GetName(), strings.ToLower("Team"), strings.ToLower(user3.Status.Type)),
		},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			err := EstablishRoleBindings(tc.user.DeepCopy(), tc.user.GetNamespace(), tc.namespaceType)
			util.OK(t, err)
			_, err = g.client.RbacV1().RoleBindings(tc.user.GetNamespace()).Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
		})
	}
}

func TestPermissionSystem(t *testing.T) {
	g := TestGroup{}
	g.Init()

	user1 := g.userObj
	user2 := g.userObj
	user2.SetName("joepublic")
	user2.Spec.Email = "joe.public@edge-net.org"
	user2.Status.Type = "user"
	user3 := g.userObj
	user3.SetName("johnsmith")
	user3.Spec.Email = "john.smith@edge-net.org"
	user3.Status.Type = "user"

	t.Run("create authority admin role", func(t *testing.T) {
		err := CreateAuthorityAdminRole()
		util.OK(t, err)
	})
	t.Run("update existing authority admin role", func(t *testing.T) {
		err := CreateAuthorityAdminRole()
		util.OK(t, err)
	})
	t.Run("create authority user role", func(t *testing.T) {
		err := CreateAuthorityUserRole()
		util.OK(t, err)
	})
	t.Run("update existing authority user role", func(t *testing.T) {
		err := CreateAuthorityUserRole()
		util.OK(t, err)
	})
	t.Run("create slice roles", func(t *testing.T) {
		err := CreateSliceRoles()
		util.OK(t, err)
	})
	t.Run("update existing slice roles", func(t *testing.T) {
		err := CreateSliceRoles()
		util.OK(t, err)
	})
	t.Run("create team roles", func(t *testing.T) {
		err := CreateTeamRoles()
		util.OK(t, err)
	})
	t.Run("update existing team roles", func(t *testing.T) {
		err := CreateTeamRoles()
		util.OK(t, err)
	})

	err := CreateClusterRoles(g.authorityObj.DeepCopy())
	util.OK(t, err)

	t.Run("create user specific role", func(t *testing.T) {
		err := CreateUserSpecificRole(user1.DeepCopy(), &g.namespace, []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateUserSpecificRole(user2.DeepCopy(), &g.namespace, []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateUserSpecificRole(user3.DeepCopy(), &g.namespace, []metav1.OwnerReference{})
		util.OK(t, err)
	})
	t.Run("update existing user specific role", func(t *testing.T) {
		err := CreateUserSpecificRole(user1.DeepCopy(), &g.namespace, []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateUserSpecificRole(user2.DeepCopy(), &g.namespace, []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateUserSpecificRole(user3.DeepCopy(), &g.namespace, []metav1.OwnerReference{})
		util.OK(t, err)
	})
	t.Run("create acceptable use policy role", func(t *testing.T) {
		err := CreateUserAUPRole(user1.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateUserAUPRole(user2.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateUserAUPRole(user3.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
	})
	t.Run("update acceptable use policy role", func(t *testing.T) {
		err := CreateUserAUPRole(user1.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateUserAUPRole(user2.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateUserAUPRole(user3.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
	})
	t.Run("create acceptable use policy role binding", func(t *testing.T) {
		err := CreateAUPRoleBinding(user1.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateAUPRoleBinding(user2.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateAUPRoleBinding(user3.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
	})
	t.Run("update acceptable use policy role binding", func(t *testing.T) {
		err := CreateAUPRoleBinding(user1.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateAUPRoleBinding(user2.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
		err = CreateAUPRoleBinding(user3.DeepCopy(), []metav1.OwnerReference{})
		util.OK(t, err)
	})

	err = EstablishPrivateRoleBindings(user1.DeepCopy())
	util.OK(t, err)
	err = EstablishRoleBindings(user1.DeepCopy(), g.namespace.GetName(), "Authority")
	util.OK(t, err)
	err = EstablishRoleBindings(user1.DeepCopy(), g.sliceNamespace.GetName(), "Slice")
	util.OK(t, err)
	err = EstablishRoleBindings(user1.DeepCopy(), g.teamNamespace.GetName(), "Team")
	util.OK(t, err)

	err = EstablishPrivateRoleBindings(user2.DeepCopy())
	util.OK(t, err)
	err = EstablishRoleBindings(user2.DeepCopy(), g.namespace.GetName(), "Authority")
	util.OK(t, err)
	err = EstablishRoleBindings(user2.DeepCopy(), g.sliceNamespace.GetName(), "Slice")
	util.OK(t, err)
	err = EstablishRoleBindings(user2.DeepCopy(), g.teamNamespace.GetName(), "Team")
	util.OK(t, err)
	roleName := "workload-manager"
	policyRule := []rbacv1.PolicyRule{{APIGroups: []string{"apps.edgenet.io"}, Resources: []string{"slices", "teams"}, Verbs: []string{"create", "update", "patch"}}}
	userRole := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: roleName}, Rules: policyRule}
	_, err = g.client.RbacV1().Roles(g.namespace.GetName()).Create(context.TODO(), userRole, metav1.CreateOptions{})
	util.OK(t, err)
	rbSubjects := []rbacv1.Subject{{Kind: "User", Name: user2.Spec.Email, APIGroup: "rbac.authorization.k8s.io"}}
	roleRef := rbacv1.RoleRef{Kind: "Role", Name: roleName}
	roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: g.namespace.GetName(), Name: fmt.Sprintf("%s-%s", roleName, user2.GetName())},
		Subjects: rbSubjects, RoleRef: roleRef}
	_, err = g.client.RbacV1().RoleBindings(g.namespace.GetName()).Create(context.TODO(), roleBind, metav1.CreateOptions{})
	util.OK(t, err)

	err = EstablishPrivateRoleBindings(user3.DeepCopy())
	util.OK(t, err)
	err = EstablishRoleBindings(user3.DeepCopy(), g.namespace.GetName(), "Authority")
	util.OK(t, err)

	cases := map[string]struct {
		user         apps_v1alpha.User
		namespace    string
		resource     string
		resourceName string
		expected     bool
	}{
		"admin/authorized for slices/authority":                    {user1, g.namespace.GetName(), "slices", "", true},
		"admin/authorized for users/authority":                     {user1, g.namespace.GetName(), "users", "", true},
		"admin/authorized for roles/authority":                     {user1, g.namespace.GetName(), "roles", "", true},
		"admin/authorized for role bindings/authority":             {user1, g.namespace.GetName(), "rolebindings", "", true},
		"admin/unauthorized for cluster role bindings/authority":   {user1, g.namespace.GetName(), "clusterrolebindings", "", false},
		"authorized for acceptable use policies/authority":         {user2, g.namespace.GetName(), "acceptableusepolicies", user2.GetName(), true},
		"authorized for user object/authority":                     {user2, g.namespace.GetName(), "users", user2.GetName(), true},
		"user/authorized for acceptable use policies/authority":    {user3, g.namespace.GetName(), "acceptableusepolicies", user3.GetName(), true},
		"user/authorized for user object/authority":                {user3, g.namespace.GetName(), "users", user3.GetName(), true},
		"unauthorized for other user objects/authority":            {user2, g.namespace.GetName(), "users", user1.GetName(), false},
		"unauthorized for other acceptable use policies/authority": {user2, g.namespace.GetName(), "acceptableusepolicies", user1.GetName(), false},
		"unauthorized for roles/authority":                         {user2, g.namespace.GetName(), "roles", "", false},
		"admin/authorized for pods/slice":                          {user1, g.sliceNamespace.GetName(), "pods", "", true},
		"admin/authorized for pods log/slice":                      {user1, g.sliceNamespace.GetName(), "pods/log", "", true},
		"admin/authorized for pods exec/slice":                     {user1, g.sliceNamespace.GetName(), "pods/exec", "", true},
		"admin/authorized for daemonsets/slice":                    {user1, g.sliceNamespace.GetName(), "daemonsets", "", true},
		"admin/unauthorized for slices/slice":                      {user1, g.sliceNamespace.GetName(), "slices", "", false},
		"admin/unauthorized for users/slice":                       {user1, g.sliceNamespace.GetName(), "users", "", false},
		"authorized for pods/slice":                                {user2, g.sliceNamespace.GetName(), "pods", "", true},
		"authorized for pods log/slice":                            {user2, g.sliceNamespace.GetName(), "pods/log", "", true},
		"admin/authorized for slices/team":                         {user1, g.teamNamespace.GetName(), "slices", "", true},
		"admin/unauthorized for users/team":                        {user1, g.teamNamespace.GetName(), "users", "", false},
		"admin/unauthorized for pods/team":                         {user1, g.teamNamespace.GetName(), "pods", "", false},
		"authorized for slices/team":                               {user2, g.teamNamespace.GetName(), "slices", "", true},
		"unauthorized for deployments/team":                        {user2, g.teamNamespace.GetName(), "deployments", "", false},
		"authorized for slices/authority":                          {user2, g.namespace.GetName(), "slices", "", true},
		"authorized for teams/authority":                           {user2, g.namespace.GetName(), "teams", "", true},
		"user/unauthorized for pods/slice":                         {user3, g.sliceNamespace.GetName(), "pods", "", false},
		"user/unauthorized for pods log/slice":                     {user3, g.sliceNamespace.GetName(), "pods/log", "", false},
		"user/unauthorized for slices/team":                        {user3, g.teamNamespace.GetName(), "slices", "", false},
		"user/unauthorized for slices/authority":                   {user3, g.namespace.GetName(), "slices", "", false},
		"user/unauthorized for teams/authority":                    {user3, g.namespace.GetName(), "teams", "", false},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			authorized := CheckAuthorization(tc.namespace, tc.user.Spec.Email, tc.resource, tc.resourceName)
			util.Equals(t, tc.expected, authorized)
		})
	}
}
