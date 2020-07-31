package permission

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"

	"github.com/Sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

type PermissionTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	userObj       apps_v1alpha.User
	client        kubernetes.Interface
	edgenetclient versioned.Interface
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func (g *PermissionTestGroup) Init() {
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
				Email:     "unittest@edge-net.org",
				FirstName: "unit",
				LastName:  "testing",
				Phone:     "+33NUMBER",
				Username:  "unittesting",
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
			Name:      "unittesting",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "EdgeNet",
			LastName:  "EdgeNet",
			Email:     "unittest@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			Type: "Admin",
		},
	}
	g.authorityObj = authorityObj
	g.userObj = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	// Sync Clientset with fake client
	Clientset = g.client

	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	g.authorityObj.Status.State = metav1.StatusSuccess
	g.authorityObj.Spec.Enabled = true
	g.edgenetclient.AppsV1alpha().Authorities().UpdateStatus(g.authorityObj.DeepCopy())
	authorityChildNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", g.authorityObj.GetName())}}
	g.client.CoreV1().Namespaces().Create(authorityChildNamespace)
}

func TestCreateClusterRoles(t *testing.T) {
	g := PermissionTestGroup{}
	g.Init()

	t.Run("Creation of Cluster Role", func(t *testing.T) {
		createErr := CreateClusterRoles(g.authorityObj.DeepCopy())
		_, err := g.client.RbacV1().ClusterRoles().Get(fmt.Sprintf("authority-%s", g.authorityObj.GetName()), metav1.GetOptions{})
		if err != nil || createErr != nil {
			t.Errorf("Create Cluster Roles Failed")
		}
	})
}

func TestEstablishPrivateRoleBindings(t *testing.T) {
	g := PermissionTestGroup{}
	g.Init()

	err := EstablishPrivateRoleBindings(g.userObj.DeepCopy())
	role, _ := g.client.RbacV1().ClusterRoleBindings().Get(fmt.Sprintf("%s-%s-for-authority", g.userObj.GetNamespace(), g.userObj.GetName()), metav1.GetOptions{})
	if err != nil && role == nil {
		t.Errorf("cluster rolebinding failed")
	}
}

func TestEstablishRoleBindings(t *testing.T) {
	g := PermissionTestGroup{}
	g.Init()

	t.Run("Establish Rolebindings", func(t *testing.T) {
		err := EstablishRoleBindings(g.userObj.DeepCopy(), g.userObj.GetNamespace(), "Authority")
		userRoleBind, errUserRoleBind := g.client.RbacV1().RoleBindings(g.userObj.GetNamespace()).Get(fmt.Sprintf("%s-%s-authority-admin", g.userObj.GetNamespace(), g.userObj.GetName()), metav1.GetOptions{})
		if err != nil || userRoleBind == nil || errUserRoleBind != nil {
			t.Errorf("Establish role Bindings Failed")
		}
	})
	t.Run("Establish Rolebindings", func(t *testing.T) {
		err := EstablishRoleBindings(g.userObj.DeepCopy(), g.userObj.GetNamespace(), "Authority")
		if err == nil {
			t.Error("Existed role not identified", err)
		}
	})
}

func TestCheckAuthorization(t *testing.T) {
	g := PermissionTestGroup{}
	g.Init()
	// Creating RoleBinding with Kind of ClusterRole
	EstablishRoleBindings(g.userObj.DeepCopy(), g.userObj.GetNamespace(), "Authority")
	authorized := CheckAuthorization(g.userObj.GetNamespace(), g.userObj.Spec.Email, "authority", g.userObj.GetName())
	if authorized {
		t.Errorf("failed to determine the Kind of role")
	}

}
