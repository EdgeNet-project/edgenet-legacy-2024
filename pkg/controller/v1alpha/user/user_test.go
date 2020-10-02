package user

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

//The main structure of test group
type TestGroup struct {
	authorityObj  apps_v1alpha.Authority
	teamList      apps_v1alpha.TeamList
	sliceList     apps_v1alpha.SliceList
	userObj       apps_v1alpha.User
	urrObj        apps_v1alpha.UserRegistrationRequest
	client        kubernetes.Interface
	edgenetClient versioned.Interface
	handler       Handler
}

func TestMain(m *testing.M) {
	flag.String("dir", "../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

//Init syncs the test group
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
				Phone:     "+33NUMBER",
				Username:  "johndoe",
			},
			Enabled: true,
		},
	}
	teamList := apps_v1alpha.TeamList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TeamList",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ListMeta: metav1.ListMeta{
			SelfLink:        "teamSelfLink",
			ResourceVersion: "1",
		},
		Items: []apps_v1alpha.Team{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Team",
					APIVersion: "apps.edgenet.io/v1alpha",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "team",
				},
				Spec: apps_v1alpha.TeamSpec{
					Users: []apps_v1alpha.TeamUsers{
						{
							Authority: "authority-edgenet",
							Username:  "johnsmith",
						},
					},
					Description: "This is a team description",
					Enabled:     true,
				},
				Status: apps_v1alpha.TeamStatus{
					State: success,
				},
			},
		},
	}
	sliceList := apps_v1alpha.SliceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SliceList",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ListMeta: metav1.ListMeta{
			SelfLink:        "sliceSelfLink",
			ResourceVersion: "1",
		},
		Items: []apps_v1alpha.Slice{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Slice",
					APIVersion: "apps.edgenet.io/v1alpha",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "slice",
				},
				Spec: apps_v1alpha.SliceSpec{
					Users: []apps_v1alpha.SliceUsers{
						{
							Authority: "authority-edgenet",
							Username:  "johnsmith",
						},
					},
					Description: "This is a slice description",
				},
			},
		},
	}
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "johnsmith",
			Namespace:  "authority-edgenet",
			UID:        "TestUID",
			Generation: 1,
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			State: success,
			Type:  "user",
		},
	}
	urrObj := apps_v1alpha.UserRegistrationRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRegistrationRequest",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "joepublic",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.UserRegistrationRequestSpec{
			Email:     "joe.public@edge-net.org",
			FirstName: "Joe",
			LastName:  "Public",
		},
		Status: apps_v1alpha.UserRegistrationRequestStatus{
			EmailVerified: false,
		},
	}
	g.authorityObj = authorityObj
	g.teamList = teamList
	g.sliceList = sliceList
	g.userObj = userObj
	g.urrObj = urrObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// Invoke authority ObjectCreated to create namespace
	// authorityHandler := authority.Handler{}
	// authorityHandler.Init(g.client, g.edgenetClient)
	g.edgenetClient.AppsV1alpha().Authorities().Create(context.TODO(), g.authorityObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("authority-%s", g.authorityObj.GetName())}}
	namespaceLabels := map[string]string{"owner": "authority", "owner-name": g.authorityObj.GetName(), "authority-name": g.authorityObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	// authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
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

func TestCollision(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)

	urr1 := g.urrObj
	urr1.Spec.Email = g.userObj.Spec.Email
	urr2 := g.urrObj
	urr2.SetName(g.userObj.GetName())
	urr3 := g.urrObj
	urr3.SetNamespace("different")
	urr3.Spec.Email = g.userObj.Spec.Email
	urr4 := g.urrObj
	urr4.SetNamespace("different")
	urr5 := g.urrObj

	cases := map[string]struct {
		request  apps_v1alpha.UserRegistrationRequest
		expected bool
	}{
		"urr/email/same-namespace":         {urr1, true},
		"urr/username/same-namespace":      {urr2, true},
		"urr/email/different-namespace":    {urr3, true},
		"urr/username/different-namespace": {urr4, false},
		"urr/none/same-namespace":          {urr5, false},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			_, err := g.edgenetClient.AppsV1alpha().UserRegistrationRequests(tc.request.GetNamespace()).Create(context.TODO(), tc.request.DeepCopy(), metav1.CreateOptions{})
			util.OK(t, err)
			g.handler.checkDuplicateObject(g.userObj.DeepCopy(), g.authorityObj.GetName())
			_, err = g.edgenetClient.AppsV1alpha().UserRegistrationRequests(tc.request.GetNamespace()).Get(context.TODO(), tc.request.GetName(), metav1.GetOptions{})
			util.Equals(t, tc.expected, errors.IsNotFound(err))
			g.edgenetClient.AppsV1alpha().UserRegistrationRequests(tc.request.GetNamespace()).Delete(context.TODO(), tc.request.GetName(), metav1.DeleteOptions{})
		})
	}

	user := g.userObj
	user.SetNamespace("different")
	user.SetUID("UID")
	t.Run("user/email/different-namespace", func(t *testing.T) {
		g.edgenetClient.AppsV1alpha().Users(user.GetNamespace()).Create(context.TODO(), user.DeepCopy(), metav1.CreateOptions{})

		emailExists, _ := g.handler.checkDuplicateObject(g.userObj.DeepCopy(), g.authorityObj.GetName())
		util.Equals(t, true, emailExists)
	})
}

func TestCreateUser(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	failed := g.handler.Create(g.urrObj.DeepCopy())
	util.Equals(t, false, failed)
}

func TestCreateRoleBindings(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Invoking createRoleBindings
	g.handler.createRoleBindings(g.userObj.DeepCopy(), g.sliceList.DeepCopy(), g.teamList.DeepCopy(), fmt.Sprintf("authority-%s", g.authorityObj.GetName()))
	// Check the creation of use role Binding
	_, err := g.handler.clientset.RbacV1().RoleBindings(g.userObj.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-user-%s", g.userObj.GetNamespace(), g.userObj.GetName()), metav1.GetOptions{})
	util.OK(t, err)
}

func TestDeleteRoleBindings(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Invoking createRoleBindings
	g.handler.createRoleBindings(g.userObj.DeepCopy(), g.sliceList.DeepCopy(), g.teamList.DeepCopy(), fmt.Sprintf("authority-%s", g.authorityObj.GetName()))
	// Check the creation of use role Binding
	_, err := g.handler.clientset.RbacV1().RoleBindings(g.userObj.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-user-%s", g.userObj.GetNamespace(), g.userObj.GetName()), metav1.GetOptions{})
	util.OK(t, err)

	g.handler.deleteRoleBindings(g.userObj.DeepCopy(), g.sliceList.DeepCopy(), g.teamList.DeepCopy())
	_, err = g.handler.clientset.RbacV1().RoleBindings(g.userObj.GetNamespace()).Get(context.TODO(), fmt.Sprintf("%s-user-aup-%s", g.userObj.GetNamespace(), g.userObj.GetName()), metav1.GetOptions{})
	util.Equals(t, true, errors.IsNotFound(err))
}

func (g *TestGroup) mockSigner(authority, user string) {
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
				CSRObj, err := g.client.CertificatesV1().CertificateSigningRequests().Get(context.TODO(), fmt.Sprintf("%s-%s", authority, user), metav1.GetOptions{})
				if err == nil {
					CSRObj.Status.Certificate = CSRObj.Spec.Request
					_, err = g.client.CertificatesV1().CertificateSigningRequests().UpdateStatus(context.TODO(), CSRObj, metav1.UpdateOptions{})
					if err == nil {
						break check
					}
				}
			}
		}
	}()
}
