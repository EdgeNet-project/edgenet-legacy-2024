package infrastructure

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	namecheap "github.com/billputer/go-namecheap"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestMain(m *testing.M) {
	flag.String("ca-path", "../../../configs/ca_sample.crt", "Set CA path.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestCreateJoinToken(t *testing.T) {
	ttl, err := time.ParseDuration("600s")
	util.OK(t, err)
	_, err = CreateToken(testclient.NewSimpleClientset(), ttl, "test.edgenet.io")
	util.OK(t, err)
}

/*
TODO: DEBUGGING:
panic: open ../../configs/namecheap.yaml: no such file or directory
*/
func FuzzGetHosts(f *testing.F) {

	testcases := 0 // testcase claimed as a placeholder, to take advantage of the Fuzz feature
	f.Add(testcases)
	f.Fuzz(func(t *testing.T, tc int) {
		client, err := bootstrap.CreateNamecheapClient()
		util.OK(t, err)
		hostList := getHosts(client)
		if len(hostList.Domain) < 2 {
			t.Errorf("Method getHosts() returns Domain with value nil")
		}
		if len(hostList.Hosts) < 1 {
			t.Errorf("Method getHosts() returns DomainDNSHost with value nil")
		}
	})
}

// TODO: DEBUGGING:
//panic: open ../../configs/namecheap.yaml: no such file or directory
func FuzzSetHostName(f *testing.F) {

	client, err := bootstrap.CreateNamecheapClient()
	util.OK(f, err)
	hostList := getHosts(client)
	size := len(hostList.Hosts)

	hostRecord := make([]namecheap.DomainDNSHost, size+1)

	// In case hostList is null
	hostRecord[size].Name = "fuzzTest.node1.lip6.fr"
	hostRecord[size].Address = "127.0.0.1"
	hostRecord[size].Type = "AnyType" // TODO: set correct type value here
	f.Add(hostRecord[size].Name, hostRecord[size].Address)

	// Take existed record to test update, then auto-generated ones to test add
	if size > 0 {
		for key, host := range hostList.Hosts {
			hostRecord[key].Name = host.Name
			hostRecord[key].Type = host.Type
			hostRecord[key].Address = host.Address
			f.Add(host.Name, host.Address, host.Type)
		}
	}

	f.Fuzz(func(t *testing.T, hostName string, address string, recordType string) {
		if err != nil {
			t.Errorf("Method CreateNamecheapClient() failed!")
			return
		}
		hostRecord := namecheap.DomainDNSHost{
			Name:    hostName,
			Type:    address,
			Address: recordType,
		}
		ret, _ := SetHostname(client, hostRecord)
		if ret == false {
			t.Errorf("Method SetHostName() failed!")
		}
	})
}
