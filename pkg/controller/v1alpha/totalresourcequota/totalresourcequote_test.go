package totalresourcequota

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"
	"edgenet/pkg/controller/v1alpha/authority"
	"io/ioutil"
	"os"
	"testing"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type TRQTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	TRQObj        apps_v1alpha.TotalResourceQuota
	client        kubernetes.Interface
	edgenetclient versioned.Interface
	handler       Handler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func (g *TRQTestGroup) Init() {
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
			Enabled: true,
		},
	}
	TRQObj := apps_v1alpha.TotalResourceQuota{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TotalResourceQuota",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "TRQName",
			// Namespace: "authority-edgenet",
		},
		Spec: apps_v1alpha.TotalResourceQuotaSpec{
			Enabled: true,
		},
		Status: apps_v1alpha.TotalResourceQuotaStatus{
			Exceeded: false,
		},
	}
	g.authorityObj = authorityObj
	g.TRQObj = TRQObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	// invoke ObjectCreated to create namespace
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	// Create Authority
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
}

func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TRQTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error(errorDict["k8-sync"])
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error(errorDict["edgenet-sync"])
	}
}

func TestTRQCreate(t *testing.T) {
	g := TRQTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creation of Total resource quota
	t.Run("creation of total resource quota", func(t *testing.T) {
		g.edgenetclient.AppsV1alpha().TotalResourceQuotas().Create(g.TRQObj.DeepCopy())
		g.handler.ObjectCreated(g.TRQObj.DeepCopy())
		TRQ, err := g.edgenetclient.AppsV1alpha().TotalResourceQuotas().Get(g.TRQObj.GetName(), metav1.GetOptions{})
		if TRQ == nil {
			t.Error(errorDict["TRQ-failed"])
		}
	})
}
