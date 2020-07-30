package nodecontribution

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
)

// Dictionary for error messages
var errorDict = map[string]string{
	"k8-sync":        "Kubernetes clientset sync problem",
	"edgenet-sync":   "EdgeNet clientset sync problem",
	"dupl-val":       "Duplicate value cannot be detected",
	"node-detect":    "Empty Host field get not detected",
	"auth-enabled":   "Authority enabled field check failed",
	"node-upd":       "Host field detection failed when updating",
	"email-sent":     "Send email failed in nodecontribution pkg",
	"getcm-install":  "Get install Debian/Centos Commands Failed",
	"getcm-unistall": "Get unistall Debian/Centos Commands Failed",
	"getcm-reconf":   "Get Reconfiguration Debian/Centos Commands Failed",
	"add-func":       "Add func of event handler doesn't work properly",
	"upd-func":       "Update func of event handler doesn't work properly",
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestGetUnistallCommands(t *testing.T) {
	var fakeOSDebian []byte = []byte("NAME=UbuntuID=ubuntuID_LIKE=debianPRETTY_NAME=Ubuntu")
	var fakeOSCentos []byte = []byte("NAME=CentosID=centosID_LIKE=centosPRETTY_NAME=Centos")
	fakeOS := [2][]byte{fakeOSCentos, fakeOSDebian}
	for _, value := range fakeOS {
		os.Args = []string{"-fakeOS", string(value)}
		_, err := getUninstallCommands(nil)
		if err != nil {
			t.Errorf(errorDict["getcm-unistall"])
		}
	}
}
func TestGetReconfigurationCommands(t *testing.T) {
	fakeHostName := "TestHost"
	var fakeOSDebian []byte = []byte("NAME=UbuntuID=ubuntuID_LIKE=debianPRETTY_NAME=Ubuntu")
	var fakeOSCentos []byte = []byte("NAME=CentosID=centosID_LIKE=centosPRETTY_NAME=Centos")
	fakeOS := [2][]byte{fakeOSCentos, fakeOSDebian}
	for _, value := range fakeOS {
		os.Args = []string{"-fakeOS", string(value)}
		_, err := getReconfigurationCommands(nil, fakeHostName)
		if err != nil {
			t.Errorf(errorDict["getcm-reconf"])
		}
	}
}
