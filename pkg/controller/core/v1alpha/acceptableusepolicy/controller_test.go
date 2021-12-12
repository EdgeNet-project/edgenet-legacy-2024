package acceptableusepolicy

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
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

// The main structure of test group
type TestGroup struct {
	tenantObj              corev1alpha.Tenant
	acceptableUsePolicyObj corev1alpha.AcceptableUsePolicy
}

var controller *Controller
var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()

func TestMain(m *testing.M) {
	//klog.SetOutput(ioutil.Discard)
	//log.SetOutput(ioutil.Discard)
	//logrus.SetOutput(ioutil.Discard)

	flag.String("dir", "../../../../..", "Override the directory.")
	flag.String("smtp-path", "../../../../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	go func() {
		edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

		newController := NewController(kubeclientset,
			edgenetclientset,
			edgenetInformerFactory.Core().V1alpha().AcceptableUsePolicies())

		edgenetInformerFactory.Start(stopCh)
		controller = newController
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	os.Exit(m.Run())
	<-stopCh
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
	// tenantHandler := tenant.Handler{}
	// tenantHandler.Init(g.client, g.edgenetClient)
	// Create Tenant
	edgenetclientset.CoreV1alpha().Tenants().Create(context.TODO(), g.tenantObj.DeepCopy(), metav1.CreateOptions{})
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: g.acceptableUsePolicyObj.GetNamespace()}}
	namespaceLabels := map[string]string{"owner": "tenant", "owner-name": g.tenantObj.GetName(), "tenant-name": g.tenantObj.GetName()}
	namespace.SetLabels(namespaceLabels)
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()

	controller := g.acceptableUsePolicyObj.DeepCopy()
	controller.SetUID("controller")
	controller.SetName("controller")

	defer edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Delete(context.TODO(), controller.GetName(), metav1.DeleteOptions{})
	// Create an AUP
	edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), controller, metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	acceptableUsePolicy, err := edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), controller.GetName(), metav1.GetOptions{})
	// Check state
	util.OK(t, err)
	util.Equals(t, success, acceptableUsePolicy.Status.State)
	// TODO: Problem here
	// exp: "Successful"
	// got: ""

	// Update an AUP
	acceptableUsePolicy.Spec.Accepted = true
	// Requesting server to Update internal representation of AUP
	edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	acceptableUsePolicy, err = edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), acceptableUsePolicy.GetName(), metav1.GetOptions{})
	// Check state
	util.OK(t, err)
	util.Equals(t, success, acceptableUsePolicy.Status.State)
	expected := metav1.Time{
		Time: time.Now().Add(4382 * time.Hour),
	}
	util.Equals(t, expected.Day(), acceptableUsePolicy.Status.Expiry.Day())
	util.Equals(t, expected.Month(), acceptableUsePolicy.Status.Expiry.Month())
	util.Equals(t, expected.Year(), acceptableUsePolicy.Status.Expiry.Year())
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()

	regular := g.acceptableUsePolicyObj.DeepCopy()
	regular.SetUID("regular")
	regular.SetName("regular")
	accepted := g.acceptableUsePolicyObj.DeepCopy()
	accepted.SetUID("accepted")
	accepted.SetName("accepted")
	accepted.Spec.Accepted = true
	recreation := g.acceptableUsePolicyObj.DeepCopy()
	recreation.SetUID("recreation")
	recreation.SetName("recreation")
	recreation.Spec.Accepted = true
	recreation.Status.Expiry = &metav1.Time{
		Time: time.Now().Add(1000 * time.Hour),
	}
	recreationExpired := g.acceptableUsePolicyObj.DeepCopy()
	recreationExpired.SetUID("recreationExpired")
	recreationExpired.SetName("recreationExpired")
	recreationExpired.Spec.Accepted = true
	recreationExpired.Status.Expiry = &metav1.Time{
		Time: time.Now().Add(-1000 * time.Hour),
	}
	t.Run("regular", func(t *testing.T) {
		edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), regular.DeepCopy(), metav1.CreateOptions{})
		defer edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Delete(context.TODO(), regular.GetName(), metav1.DeleteOptions{})
		time.Sleep(time.Millisecond * 500)
		acceptableUsePolicy, err := edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), regular.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, success, acceptableUsePolicy.Status.State)
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "false", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
	t.Run("accepted already", func(t *testing.T) {
		edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), accepted.DeepCopy(), metav1.CreateOptions{})
		defer edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Delete(context.TODO(), accepted.GetName(), metav1.DeleteOptions{})
		time.Sleep(time.Millisecond * 500)
		acceptableUsePolicy, err := edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), accepted.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, success, acceptableUsePolicy.Status.State)
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "true", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
	t.Run("recreation", func(t *testing.T) {
		edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), recreation.DeepCopy(), metav1.CreateOptions{})
		defer edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Delete(context.TODO(), recreation.GetName(), metav1.DeleteOptions{})
		time.Sleep(time.Millisecond * 500)
		acceptableUsePolicy, err := edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), recreation.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, "Successful", acceptableUsePolicy.Status.State)
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "true", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
	t.Run("recreation of expired one", func(t *testing.T) {
		edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), recreationExpired.DeepCopy(), metav1.CreateOptions{})
		defer edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Delete(context.TODO(), recreationExpired.GetName(), metav1.DeleteOptions{})
		time.Sleep(time.Millisecond * 1500)
		acceptableUsePolicy, err := edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), recreationExpired.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, failure, acceptableUsePolicy.Status.State)
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "false", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
}

func TestAccept(t *testing.T) {
	g := TestGroup{}
	g.Init()

	accept := g.acceptableUsePolicyObj.DeepCopy()
	accept.SetUID("accept")
	accept.SetName("accept")

	// Create acceptableUsePolicy to update later
	edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Create(context.TODO(), accept, metav1.CreateOptions{})
	// Invoke ObjectCreated func to create a acceptableUsePolicy
	time.Sleep(time.Millisecond * 500)
	acceptableUsePolicy, err := edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), accept.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	// Update of acceptableUsePolicy status
	// Building field parameter
	acceptableUsePolicy.Spec.Accepted = true
	acceptableUsePolicy, err = edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy, metav1.UpdateOptions{})
	util.OK(t, err)
	time.Sleep(time.Millisecond * 500)

	acceptableUsePolicy, err = edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), acceptableUsePolicy.GetName(), metav1.GetOptions{})
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
		tenant, err := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
		util.OK(t, err)

		tenantLabels := tenant.GetLabels()
		util.Equals(t, "true", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
	})
	t.Run("timeout", func(t *testing.T) {
		acceptableUsePolicy.Status.Expiry = &metav1.Time{
			Time: time.Now().Add(10 * time.Millisecond),
		}
		_, err := edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Update(context.TODO(), acceptableUsePolicy.DeepCopy(), metav1.UpdateOptions{})
		util.OK(t, err)
		time.Sleep(time.Millisecond * 500)
		t.Run("expired", func(t *testing.T) {
			acceptableUsePolicy, err = edgenetclientset.CoreV1alpha().AcceptableUsePolicies().Get(context.TODO(), acceptableUsePolicy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, false, acceptableUsePolicy.Spec.Accepted)
		})
		t.Run("user status", func(t *testing.T) {
			aupLabels := acceptableUsePolicy.GetLabels()
			tenantName := aupLabels["edge-net.io/tenant"]
			tenant, err := edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			util.OK(t, err)

			tenantLabels := tenant.GetLabels()
			util.Equals(t, "false", tenantLabels[fmt.Sprintf("edge-net.io/aup-accepted-%s", acceptableUsePolicy.GetName())])
		})
	})
}
