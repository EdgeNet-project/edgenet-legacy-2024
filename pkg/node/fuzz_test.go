package node

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// fuzz: elapsed: 0s, gathering baseline coverage: 0/1 completed
// failure while testing seed corpus entry: FuzzGetKubeletVersion/seed#0
// fuzz: elapsed: 2s, gathering baseline coverage: 0/1 completed
// --- FAIL: FuzzGetKubeletVersion (1.98s)
//     --- FAIL: FuzzGetKubeletVersion (0.00s)
func FuzzGetKubeletVersion(f *testing.F) {
	g := testGroup{}
	g.Init()

	headNode := "node-1"
	x := 1
	y := 23
	z := 5
	kubeletVersion := fmt.Sprintf("%d.%d.%d", x, y, z)
	f.Add(headNode, x, y, z)

	f.Fuzz(func(t *testing.T, nodeName string, x int, y int, z int) {
		node := g.nodeObj
		node.SetName(nodeName)
		ver := fmt.Sprintf("%d.%d.%d", x, y, z)
		node.Status.NodeInfo.KubeProxyVersion = ver
		_, err := g.client.CoreV1().Nodes().Create(context.TODO(), node.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		util.Equals(t, kubeletVersion, GetKubeletVersion())
	})
}

func FuzzSetNodeLabels(f *testing.F) {
	g := testGroup{}
	g.Init()
	maxLabelItems := 10

	aName := "node-0"
	aLabel := "label0" // TODO: set correct test value
	aValue := "value0" // TODO: set correct test value
	f.Add(aName, aLabel, aValue)

	f.Fuzz(func(t *testing.T, nodeName string, l string, v string) {
		node := g.nodeObj
		node.SetName(nodeName)
		// Testcase with variable levels of node labels
		rand.Seed(time.Now().UnixNano())
		nb := rand.Intn(maxLabelItems)
		labels := make(map[string]string)
		for i := 0; i < nb; i++ {
			k := l + strconv.Itoa(i)
			v := v + strconv.Itoa(i)
			labels[k] = v
		}
		ret := setNodeLabels(node.GetName(), labels)
		util.Equals(t, true, ret)
		node1, err := g.client.CoreV1().Nodes().Get(context.TODO(), node.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		util.Equals(t, labels, node1.GetLabels())
	})
}

func FuzzSanitizeNodeLabel(f *testing.F) {
	a := "~!@#$%^&*()_+}{\":';?><|][`" //illegal case
	b := "abc"                         // legal case
	c := 10                            //legal case
	d := "-_."                         //legal case
	e := a + b + strconv.Itoa(c) + d   //assembly case,illegal

	f.Add(a, b, c, d, e)
	f.Fuzz(func(t *testing.T, illegal string, alph string, nb int, special string, illegalAssemb string) {
		// user cases
		tc1 := alph + illegal + strconv.Itoa(nb)           // illegal input
		tc2 := illegalAssemb                               //illegal input
		tc3 := special + strconv.Itoa(nb) + alph + special //illegal input
		tc4 := strconv.Itoa(nb) + special + alph           //legal input

		testcases := []string{tc1, tc2, tc3, tc4}
		for _, tc := range testcases {
			s := sanitizeNodeLabel(tc)
			r := regexp.MustCompile("^[A-Za-z0-9]([A-Za-z0-9_-.]*[A-Za-z0-9])?$")
			ret := r.MatchString(s)
			if !ret {
				t.Errorf("Method sanitizeNodeLabel() failed!")
			}
		}
	})
}

func FuzzGetMaxmindLocation(t *testing.T) {
	url := "test.Edgenet.io" // TODO Set correct test url
	accountId := "edgenet"
	licenseKey := "licenseKey"
	address := map[string]int{
		"illegal-addr1": 0,
		"legal-addr1":   1} // TODO Set correct test url

	for addr, flag := range address {
		resp, err := getMaxmindLocation(url, accountId, licenseKey, addr)
		if flag > 0 {
			util.OK(t, err)
		} else {
			util.Equals(t, resp, nil)
		}
	}

}

// func FuzzGetGeolocationByIP(f *testing.F) {

// }

// func FuzzSetHostname(f *testing.F){

// }
