package nodecontribution

import (
	"edgenet/pkg/client/clientset/versioned"
	"io/ioutil"
	"os"
	"testing"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"

	testclient "k8s.io/client-go/kubernetes/fake"
)

type NodecontributionTestGroup struct {
	authorityObj  apps_v1alpha.Authority
	client        kubernetes.Interface
	edgenetclient versioned.Interface
	handler       Handler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func (g *NodecontributionTestGroup) Init() {
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
		},
		Status: apps_v1alpha.AuthorityStatus{
			Enabled: false,
		},
	}
	g.authorityObj = authorityObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
}

//TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	//Sync the test group
	g := NodecontributionTestGroup{}
	g.Init()
	//Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error("Kubernetes clientset sync problem")
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error("EdgeNet clientset sync problem")
	}

}
