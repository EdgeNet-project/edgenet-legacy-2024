package infrastructure

import (
	"testing"

	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	namecheap "github.com/billputer/go-namecheap"
)

/*
FAILED TO RUN, DEBUGGING:
--- FAIL: FuzzGetHosts (0.01s)
panic: open ../../configs/namecheap.yaml: The system cannot find the path specified. [recovered]
        panic: open ../../configs/namecheap.yaml: The system cannot find the path specified.
goroutine 20 [running]:
testing.fRunner.func1.2({0x1efba60, 0xc000385480})
        C:/Program Files/Go/src/testing/fuzz.go:660 +0x1e5
testing.fRunner.func1()
        C:/Program Files/Go/src/testing/fuzz.go:663 +0x248
panic({0x1efba60, 0xc000385480})
        C:/Program Files/Go/src/runtime/panic.go:838 +0x207
github.com/EdgeNet-project/edgenet/pkg/bootstrap.CreateNamecheapClient()
        C:/Users/tfa/Documents/GitHub/edgenet/pkg/bootstrap/bootstrap.go:127 +0x1a9
github.com/EdgeNet-project/edgenet/pkg/node/infrastructure.FuzzGetHosts(0x0?)
        C:/Users/tfa/Documents/GitHub/edgenet/pkg/node/infrastructure/fuzz_test.go:10 +0x31
testing.fRunner(0xc0002e23c0, 0x215b4c0)
        C:/Program Files/Go/src/testing/fuzz.go:700 +0xbf
created by testing.runFuzzing
        C:/Program Files/Go/src/testing/fuzz.go:590 +0x78b
exit status 2
FAIL    github.com/EdgeNet-project/edgenet/pkg/node/infrastructure      4.482s
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

// --- FAIL: FuzzGetHosts (0.01s)
// panic: open ../../configs/namecheap.yaml: The system cannot find the path specified. [recovered]
//         panic: open ../../configs/namecheap.yaml: The system cannot find the path specified.

// goroutine 7 [running]:
// testing.fRunner.func1.2({0x207dae0, 0xc0003d3480})
//         C:/Program Files/Go/src/testing/fuzz.go:660 +0x1e5
// testing.fRunner.func1()
//         C:/Program Files/Go/src/testing/fuzz.go:663 +0x248
// panic({0x207dae0, 0xc0003d3480})
//         C:/Program Files/Go/src/runtime/panic.go:838 +0x207
// github.com/EdgeNet-project/edgenet/pkg/bootstrap.CreateNamecheapClient()
//         C:/Users/tfa/Documents/GitHub/edgenet/pkg/bootstrap/bootstrap.go:127 +0x1a9
// github.com/EdgeNet-project/edgenet/pkg/node/infrastructure.FuzzGetHosts(0x0?)
//         C:/Users/tfa/Documents/GitHub/edgenet/pkg/node/infrastructure/fuzz_test.go:34 +0x31
// testing.fRunner(0xc0000005a0, 0x22dd780)
//         C:/Program Files/Go/src/testing/fuzz.go:700 +0xbf
// created by testing.runFuzzTests
//         C:/Program Files/Go/src/testing/fuzz.go:520 +0x73e
// exit status 2
// FAIL    github.com/EdgeNet-project/edgenet/pkg/node/infrastructure      4.512s
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
