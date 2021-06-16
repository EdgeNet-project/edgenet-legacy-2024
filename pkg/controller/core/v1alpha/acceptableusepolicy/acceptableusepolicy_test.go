package acceptableusepolicy

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
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type TestGroup struct {
	tenantObj              corev1alpha.Tenant
	acceptableUsePolicyObj corev1alpha.AcceptableUsePolicy
	client                 kubernetes.Interface
	edgenetClient          versioned.Interface
	handler                Handler
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
			Labels: map[string]string{
				"edge-net.io/generated": "true",
			},
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
				Email:     "joe.public@edge-net.org",
				FirstName: "Joe",
				LastName:  "Public",
				Phone:     "+33NUMBER",
				Username:  "joepublic",
			},
			Enabled: true,
		},
	}
	acceptableUsePolicyObj := corev1alpha.AcceptableUsePolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AcceptableUsePolicy",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet-joepublic",
			Labels: map[string]string{
				"edge-net.io/generated": "true",
				"edge-net.io/tenant":    "edgenet",
				"edge-net.io/username":  "joepublic",
			},
		},
		Spec: corev1alpha.AcceptableUsePolicySpec{
			Accepted: false,
		},
	}
	g.tenantObj = tenantObj
	g.acceptableUsePolicyObj = acceptableUsePolicyObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
	// tenantHandler := tenant.Handler{}
	// tenantHandler.Init(g.client, g.edgenetClient)
	// Create Tenant
	g.edgenetClient.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.acceptableUsePolicyObj.GetNamespace()}}
	namespaceLabels := map[string]string{"owner": "tenant", "owner-name": g.tenantObj.GetName(), "tenant-name": g.tenantObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	g.client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{}) // Invoke ObjectCreated to create namespace
	// Create a user as admin on tenant
	/*
		user := apps_v1alpha.User{}
		user.SetName(strings.ToLower(g.tenantObj.Spec.Contact.Username))
		user.Spec.Email = g.tenantObj.Spec.Contact.Email
		user.Spec.FirstName = g.tenantObj.Spec.Contact.FirstName
		user.Spec.LastName = g.tenantObj.Spec.Contact.LastName
		user.Spec.Active = true
		user.Status.acceptableUsePolicy = false
		user.Status.Type = "admin"
		g.edgenetClient.AppsV1alpha().Users(fmt.Sprintf("tenant-%s", g.tenantObj.GetName())).Create(context.TODO(), user.DeepCopy(), metav1.CreateOptions{})
	*/
	// tenantHandler.ObjectCreated(g.tenantObj.DeepCopy())
}

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
	regular := g.acceptableUsePolicyObj.DeepCopy()
	regular.SetUID("regular")
	accepted := g.acceptableUsePolicyObj.DeepCopy()
	accepted.SetUID("accepted")
	accepted.Spec.Accepted = true
	recreation := g.acceptableUsePolicyObj.DeepCopy()
	recreation.SetUID("recreation")
	recreation.Spec.Accepted = true
	recreation.Status.Expiry = &metav1.Time{
		Time: time.Now().Add(1000 * time.Hour),
	}
	recreationExpired := g.acceptableUsePolicyObj.DeepCopy()
	recreationExpired.SetUID("recreationExpired")
	recreationExpired.Spec.Accepted = true
	recreationExpired.Status.Expiry = &metav1.Time{
		Time: time.Now().Add(-1000 * time.Hour),
	}
	t.Run("regular", func(t *testing.T) {
		g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), regular.DeepCopy(), metav1.CreateOptions{})
		defer g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Delete(context.TODO(), regular.GetName(), metav1.DeleteOptions{})
		g.handler.ObjectCreatedOrUpdated(regular.DeepCopy())
		acceptableUsePolicy, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), regular.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, success, acceptableUsePolicy.Status.State)
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "false", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
	t.Run("accepted already", func(t *testing.T) {
		g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), accepted.DeepCopy(), metav1.CreateOptions{})
		defer g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Delete(context.TODO(), accepted.GetName(), metav1.DeleteOptions{})
		g.handler.ObjectCreatedOrUpdated(accepted.DeepCopy())
		acceptableUsePolicy, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), accepted.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, success, acceptableUsePolicy.Status.State)
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "true", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
	t.Run("recreation", func(t *testing.T) {
		g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), recreation.DeepCopy(), metav1.CreateOptions{})
		defer g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Delete(context.TODO(), recreation.GetName(), metav1.DeleteOptions{})
		g.handler.ObjectCreatedOrUpdated(recreation.DeepCopy())
		acceptableUsePolicy, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), recreation.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, "", acceptableUsePolicy.Status.State)
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "true", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
	t.Run("recreation of expired one", func(t *testing.T) {
		g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), recreationExpired.DeepCopy(), metav1.CreateOptions{})
		defer g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Delete(context.TODO(), recreationExpired.GetName(), metav1.DeleteOptions{})
		g.handler.ObjectCreatedOrUpdated(recreationExpired.DeepCopy())
		time.Sleep(time.Millisecond * 100)
		acceptableUsePolicy, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), recreationExpired.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		g.handler.ObjectCreatedOrUpdated(acceptableUsePolicy)
		acceptableUsePolicy, err = g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), recreationExpired.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, failure, acceptableUsePolicy.Status.State)
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "false", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
}

func TestAccept(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	go g.handler.RunExpiryController()
	// Create acceptableUsePolicy to update later
	g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), g.acceptableUsePolicyObj.DeepCopy(), metav1.CreateOptions{})
	// Invoke ObjectCreated func to create a acceptableUsePolicy
	g.handler.ObjectCreatedOrUpdated(g.acceptableUsePolicyObj.DeepCopy())
	acceptableUsePolicy, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), g.acceptableUsePolicyObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	// Update of acceptableUsePolicy status
	// Building field parameter
	acceptableUsePolicy.Spec.Accepted = true
	g.handler.ObjectCreatedOrUpdated(acceptableUsePolicy.DeepCopy())
	time.Sleep(time.Millisecond * 100)

	acceptableUsePolicy, err = g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), g.acceptableUsePolicyObj.GetName(), metav1.GetOptions{})
	t.Run("update", func(t *testing.T) {
		util.OK(t, err)
		util.Equals(t, success, acceptableUsePolicy.Status.State)
	})
	t.Run("set expiry date", func(t *testing.T) {
		expected := metav1.Time{
			Time: time.Now().Add(4382 * time.Hour),
		}
		util.Equals(t, expected.Day(), acceptableUsePolicy.Status.Expiry.Day())
		util.Equals(t, expected.Month(), acceptableUsePolicy.Status.Expiry.Month())
		util.Equals(t, expected.Year(), acceptableUsePolicy.Status.Expiry.Year())
	})
	t.Run("user status", func(t *testing.T) {
		aupLabels := acceptableUsePolicy.GetLabels()
		tenantName := aupLabels["edge-net.io/tenant"]
		tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
		util.OK(t, err)

		tenantLabels := tenant.GetLabels()
		util.Equals(t, "true", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
	})
	t.Run("timeout", func(t *testing.T) {
		acceptableUsePolicy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		_, err := g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy.DeepCopy(), metav1.UpdateOptions{})
		util.OK(t, err)
		time.Sleep(100 * time.Millisecond)
		t.Run("expired", func(t *testing.T) {
			acceptableUsePolicy, err = g.edgenetClient.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), acceptableUsePolicy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, false, acceptableUsePolicy.Spec.Accepted)
			g.handler.ObjectCreatedOrUpdated(acceptableUsePolicy)
		})
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := g.edgenetClient.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "false", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
}
