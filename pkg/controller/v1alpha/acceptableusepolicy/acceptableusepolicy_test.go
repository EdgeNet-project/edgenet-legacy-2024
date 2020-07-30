package acceptableusepolicy

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"edgenet/pkg/controller/v1alpha/authority"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Dictionary for status messages
var errorDict = map[string]string{
	"k8-sync":     "Kubernetes clientset sync problem",
	"edgnet-sync": "EdgeNet clientset sync problem",
	"AUP-create":  "Failed to create Acceptable use policy",
	"AUP-update":  "Failed to update Acceptable use policy",
	"add-func":    "Add func of event handler doesn't work properly",
}

// The main structure of test group
type AUPTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	AUPObj        apps_v1alpha.AcceptableUsePolicy
	client        kubernetes.Interface
	edgenetclient versioned.Interface
	handler       Handler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *AUPTestGroup) Init() {
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
				Username:  "edgenetAUP",
			},
			Enabled: true,
		},
	}
	AUPObj := apps_v1alpha.AcceptableUsePolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AcceptableUsePolicy",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenetAUP",
			Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.AcceptableUsePolicySpec{
			Accepted: false,
			Renew:    false,
		},
	}
	g.authorityObj = authorityObj
	g.AUPObj = AUPObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	// Invoke ObjectCreated to create namespace
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := AUPTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error(errorDict["k8-sync"])
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error(errorDict["edgnet-sync"])
	}
}

func TestAUPCreate(t *testing.T) {
	g := AUPTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creation of AUP
	t.Run("creation of AUP", func(t *testing.T) {
		// Create AUP
		g.AUPObj.Spec.Accepted = true
		g.edgenetclient.AppsV1alpha().AcceptableUsePolicies(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.AUPObj.DeepCopy())
		g.handler.ObjectCreated(g.AUPObj.DeepCopy())
		AUP, _ := g.edgenetclient.AppsV1alpha().AcceptableUsePolicies(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.AUPObj.GetName(), metav1.GetOptions{})
		if AUP.Status.State != success && AUP.Status.Expires != nil {
			t.Errorf(errorDict["AUP-create"])
		}
	})
}

func TestAUPUpdate(t *testing.T) {
	g := AUPTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Create AUP to update later
	g.edgenetclient.AppsV1alpha().AcceptableUsePolicies(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.AUPObj.DeepCopy())
	// Invoke ObjectCreated func to create a AUP
	g.handler.ObjectCreated(g.AUPObj.DeepCopy())
	// Update of AUP status
	t.Run("Update existing AUP", func(t *testing.T) {
		g.AUPObj.Spec.Accepted, g.AUPObj.Spec.Renew = true, true
		// Building field parameter
		var field fields
		// Requesting server to Update internal representation of AUP
		g.edgenetclient.AppsV1alpha().AcceptableUsePolicies(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Update(g.AUPObj.DeepCopy())
		g.handler.ObjectUpdated(g.AUPObj.DeepCopy(), field)
		// Verifying server triggered changes
		AUP, _ := g.edgenetclient.AppsV1alpha().AcceptableUsePolicies(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.AUPObj.GetName(), metav1.GetOptions{})
		if AUP.Status.State != success && AUP.Status.Expires != nil && strings.Contains(AUP.Status.Message[0], "Agreed and Renewed") {
			t.Errorf(errorDict["AUP-update"])
		}
	})
}
