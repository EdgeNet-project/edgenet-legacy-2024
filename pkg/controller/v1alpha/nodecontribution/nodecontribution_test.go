package nodecontribution

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"edgenet/pkg/client/clientset/versioned"
	"edgenet/pkg/controller/v1alpha/authority"
	"edgenet/pkg/controller/v1alpha/user"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
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

type cleanuper interface {
	Cleanup(f func())
}

func TestMain(m *testing.M) {
	// Creating ssh directory and put SSH keys in there for testing purposes
	dirSSH := "../../../../configs/.ssh/"
	checkExistence := exists(dirSSH)
	if !checkExistence {
		err := os.Mkdir(dirSSH, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
	// Generate private keys
	err := generatePairofKeys()
	if err != nil {
		log.Fatal(err)
	}

	// Patch for fixing relative path issue while implementing unit tests
	// We defined extra argument which can be passed to the program
	flag.Parse()
	os.Args = []string{"-ssh-path", "../../../../configs/.ssh/id_rsa",
		"-fakenamecheap-path", "../../../../configs/namecheap_template.yaml",
		"-smtp-path", "../../../../configs/smtp_template.yaml",
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
			Host:    "127.0.0.1",
			Port:    5535,
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
			Enabled: true,
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
			Email:     "userObj@email.com",
			Active:    true,
		},
		Status: apps_v1alpha.UserStatus{
			Type:  "Admin",
			State: success,
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
		g.authorityObj.Spec.Enabled = false
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

func TestGetInstallCommands(t *testing.T) {
	g := NodecontributionTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)

	var fakeOSDebian []byte = []byte("NAME=UbuntuID=ubuntuID_LIKE=debianPRETTY_NAME=Ubuntu")
	var fakeOSCentos []byte = []byte("NAME=CentosID=centosID_LIKE=centosPRETTY_NAME=Centos")
	_, err := getInstallCommands(nil, g.nodecontributionObj.GetName(), "1.12", fakeOSDebian)
	if err != nil {
		t.Errorf("Get Install Debian Commands Failed")
	}
	_, err = getInstallCommands(nil, g.nodecontributionObj.GetName(), "1.15", fakeOSCentos)
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

	t.Cleanup(func() {
		// Cleaning up the generated directory and Keys
		t.Log("CLENUP RUNNED")
		os.RemoveAll("../../../../configs/.ssh/")
	})
}

// Extra function for running the unit tests and prepare the requirements
// exists returns whether the given file or directory exists
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func generatePairofKeys() error {
	savePrivateFileTo := "../../../../configs/.ssh/id_rsa"
	savePublicFileTo := "../../../../configs/.ssh/id_rsa.pub"
	bitSize := 4096

	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		log.Fatal(err.Error())
	}

	publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Fatal(err.Error())
	}

	privateKeyBytes := encodePrivateKeyToPEM(privateKey)

	err = writeKeyToFile(privateKeyBytes, savePrivateFileTo)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = writeKeyToFile([]byte(publicKeyBytes), savePublicFileTo)
	if err != nil {
		log.Fatal(err.Error())
	}
	return err
}

// generatePrivateKey creates a RSA Private Key of specified byte size
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	log.Println("Private Key generated")
	return privateKey, nil
}

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}

// generatePublicKey take a rsa.PublicKey and return bytes suitable for writing to .pub file
// returns in the format "ssh-rsa ..."
func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	log.Println("Public key generated")
	return pubKeyBytes, nil
}

// writePemToFile writes keys to a file
func writeKeyToFile(keyBytes []byte, saveFileTo string) error {
	err := ioutil.WriteFile(saveFileTo, keyBytes, 0600)
	if err != nil {
		return err
	}

	log.Printf("Key saved to: %s", saveFileTo)
	return nil
}
