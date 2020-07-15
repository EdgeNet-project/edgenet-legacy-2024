package nodecontribution

import (
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/controller/v1alpha/authority"
	"edgenet/pkg/controller/v1alpha/user"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
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
	nodecontributionObj apps_v1alpha.NodeContribution
	authorityObj        apps_v1alpha.Authority
	userObj             apps_v1alpha.User
	client              kubernetes.Interface
	edgenetclient       versioned.Interface
	handler             Handler
}

func TestMain(m *testing.M) {

	// Patch for fixing relative path issue while implementing unit tests
	flag.Parse()
	os.Args = []string{"-ssh-path", "../../../../config/.ssh/id_rsa",
		"-fakenamecheap-path", "../../../../config/namecheap.yaml",
		"-smtp-path", "../../../../config/smtp.yaml",
		"-authoritycreationtemplate-path", "../../../../assets/templates/email/authority-creation.html"}

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func (g *NodecontributionTestGroup) Init() {
	nodecontributionObj := apps_v1alpha.NodeContribution{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NodeContribution",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edgenetNode",
			Namespace: "authority-edgenetUnitTest",
		},
		Spec: apps_v1alpha.NodeContributionSpec{
			Host:    "143.197.162.100",
			Port:    525,
			User:    "edgenetNodeUser",
			Enabled: true,
		},
	}

	authorityObj := apps_v1alpha.Authority{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Authority",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenetUnitTest",
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
				Email:     "unitTest@edge-net.org",
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
	userObj := apps_v1alpha.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "unittestingObj",
			Namespace:  "authority-edgenetUnitTest",
			UID:        "TestUID",
			Generation: 1,
		},
		Spec: apps_v1alpha.UserSpec{
			FirstName: "EdgeNetFirstName",
			LastName:  "EdgeNetLastName",
			Roles:     []string{"Manager"},
			Email:     "userObj@email.com",
		},
		Status: apps_v1alpha.UserStatus{
			State:  success,
			Active: true,
			AUP:    true,
		},
	}
	g.authorityObj = authorityObj
	g.userObj = userObj
	g.nodecontributionObj = nodecontributionObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	// Invoke authority ObjectCreated to create namespace
	authorityHandler := authority.Handler{}
	authorityHandler.Init(g.client, g.edgenetclient)
	g.edgenetclient.AppsV1alpha().Authorities().Create(g.authorityObj.DeepCopy())
	authorityHandler.ObjectCreated(g.authorityObj.DeepCopy())
	// Invoke user ObjectCreated to create a user with role manager
	userHandler := user.Handler{}
	userHandler.Init(g.client, g.edgenetclient)
	g.edgenetclient.AppsV1alpha().Users(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.userObj.DeepCopy())
	userHandler.ObjectCreated(g.userObj.DeepCopy())
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

func TestNodeContributionCreate(t *testing.T) {
	g := NodecontributionTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	t.Run("Creation of Node", func(t *testing.T) {
		// Create a node
		g.edgenetclient.AppsV1alpha().NodeContributions(g.nodecontributionObj.GetNamespace()).Create(g.nodecontributionObj.DeepCopy())
		g.handler.ObjectCreated(g.nodecontributionObj.DeepCopy())
		node, _ := g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.nodecontributionObj.Name, metav1.GetOptions{})
		if node != nil {
			t.Log(node)
			t.Error("Fake Error!")
		}
	})

	t.Run("Creation of Node with Null Host", func(t *testing.T) {
		// Create a node with null Host field
		g.nodecontributionObj.Spec.Host = ""
		g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.nodecontributionObj.DeepCopy())
		g.handler.ObjectCreated(g.nodecontributionObj.DeepCopy())
		node, _ := g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.nodecontributionObj.Name, metav1.GetOptions{})
		if !reflect.DeepEqual(node.Status.Message, []string{"Host field must be an IP Address"}) {
			t.Error("Empty Host field get not detected!")
		}
	})

	t.Run("Creation of Node while Authority is not enabled", func(t *testing.T) {
		g.authorityObj.Status.Enabled = false
		g.edgenetclient.AppsV1alpha().Authorities().Update(g.authorityObj.DeepCopy())
		//create a node
		g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.nodecontributionObj.DeepCopy())
		g.handler.ObjectCreated(g.nodecontributionObj.DeepCopy())
		node, _ := g.handler.edgenetClientset.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.nodecontributionObj.Name, metav1.GetOptions{})
		if !reflect.DeepEqual(node.Status.Message, []string{"Authority disabled"}) {
			t.Error("Authority enabled field check failed!")
		}
	})
}

func TestNodeContributionUpdate(t *testing.T) {
	g := NodecontributionTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	t.Run("Update of NodeContribution with empty host, Authority enabled", func(t *testing.T) {
		//create a node
		g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.nodecontributionObj.DeepCopy())
		g.handler.ObjectCreated(g.nodecontributionObj.DeepCopy())
		node, _ := g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(g.nodecontributionObj.Name, metav1.GetOptions{})
		node.Spec.Host = ""
		g.handler.ObjectUpdated(node.DeepCopy())
		node, _ = g.handler.edgenetClientset.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Get(node.Name, metav1.GetOptions{})
		if !reflect.DeepEqual(node.Status.Message, []string{"Host field must be an IP Address"}) {
			t.Log(node)
			t.Errorf("Host field detection failed")
		}
	})
}

func TestSendEmail(t *testing.T) {
	g := NodecontributionTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	//create a node
	g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.nodecontributionObj.DeepCopy())
	g.handler.ObjectCreated(g.nodecontributionObj.DeepCopy())
	err := g.handler.sendEmail(g.nodecontributionObj.DeepCopy())
	if err != nil {
		t.Errorf("Send Email failed")
	}
}

func TestSetAuthorityAsOwnerRefrence(t *testing.T) {
	g := NodecontributionTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	//create a node
	g.edgenetclient.AppsV1alpha().NodeContributions(fmt.Sprintf("authority-%s", g.authorityObj.GetName())).Create(g.nodecontributionObj.DeepCopy())
	g.handler.ObjectCreated(g.nodecontributionObj.DeepCopy())
	err := g.handler.setAuthorityAsOwnerReference(g.authorityObj.Name, g.nodecontributionObj.Name)
	if err != nil {
		t.Errorf("SetAuthority as Owner Refrence Failed")
	}
}

func TestGetInstallCommands(t *testing.T) {
	g := NodecontributionTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	var fakeOSDebian []byte = []byte("NAME=UbuntuID=ubuntuID_LIKE=debianPRETTY_NAME=Ubuntu")
	var fakeOSCentos []byte = []byte("NAME=CentosID=centosID_LIKE=centosPRETTY_NAME=Centos")
	_, err := getInstallCommands(g.client, nil, g.nodecontributionObj.GetName(), "1.12", fakeOSDebian)
	if err != nil {
		t.Errorf("Get Install Debian Commands Failed")
	}
	_, err = getInstallCommands(g.client, nil, g.nodecontributionObj.GetName(), "1.15", fakeOSCentos)
	if err != nil {
		t.Errorf("Get Install Centos Commands Failed")
	}
}

func TestGetUnistallCommands(t *testing.T) {
	var fakeOSDebian []byte = []byte("NAME=UbuntuID=ubuntuID_LIKE=debianPRETTY_NAME=Ubuntu")
	var fakeOSCentos []byte = []byte("NAME=CentosID=centosID_LIKE=centosPRETTY_NAME=Centos")
	_, err := getUninstallCommands(nil, fakeOSDebian)
	if err != nil {
		t.Errorf("Get Unistall Debian Commands Failed")
	}
	_, err = getUninstallCommands(nil, fakeOSCentos)
	if err != nil {
		t.Errorf("Get Unistall Centos Commands Failed")
	}
}

func TestGetReconfigurationCommands(t *testing.T) {
	fakeHostName := "TestHost"
	var fakeOSDebian []byte = []byte("NAME=UbuntuID=ubuntuID_LIKE=debianPRETTY_NAME=Ubuntu")
	var fakeOSCentos []byte = []byte("NAME=CentosID=centosID_LIKE=centosPRETTY_NAME=Centos")
	_, err := getReconfigurationCommands(nil, fakeHostName, fakeOSDebian)
	if err != nil {
		t.Errorf("Get Reconfiguration Debian Commands Failed")
	}
	_, err = getReconfigurationCommands(nil, fakeHostName, fakeOSCentos)
	if err != nil {
		t.Errorf("Get Reconfiguration Centos Commands Failed")
	}
}

func TestGetRecordType(t *testing.T) {
	ipV4 := "98.139.180.149"
	ipV6 := "2607:f0d0:1002:51::4"
	if (getRecordType(ipV4) != "A") || (getRecordType(ipV6) != "AAAA") {
		t.Errorf("IP type detection failed")
	}
}
