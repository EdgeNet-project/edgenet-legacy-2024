package node

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// The main structure of test group
type testGroup struct {
	client  kubernetes.Interface
	nodeObj corev1.Node
}

func TestMain(m *testing.M) {
	flag.String("ca-path", "../../configs/ca_sample.crt", "Set CA path.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *testGroup) Init() {
	g.client = testclient.NewSimpleClientset()
	Clientset = g.client
	g.nodeObj = corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("3781924"),
				corev1.ResourceCPU:              resource.MustParse("2"),
				corev1.ResourceEphemeralStorage: resource.MustParse("51493088"),
				corev1.ResourcePods:             resource.MustParse("100"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("3781924"),
				corev1.ResourceCPU:              resource.MustParse("2"),
				corev1.ResourceEphemeralStorage: resource.MustParse("51493088"),
				corev1.ResourcePods:             resource.MustParse("100"),
			},
			Conditions: []corev1.NodeCondition{
				corev1.NodeCondition{
					Type:   "Ready",
					Status: "True",
				},
			},
		},
	}
}

func TestUnique(t *testing.T) {
	cases := []struct {
		input    []string
		expected []string
	}{
		{
			[]string{"test1", "test1", "test2", "test2", "test3", "t", "test-1"},
			[]string{"test1", "test2", "test3", "t", "test-1"},
		},
		{
			[]string{"test1", "test1", "test3", "test2", "test3", "test4", "test444", "r"},
			[]string{"test1", "test3", "test2", "test4", "test444", "r"},
		},
		{
			[]string{"test2", "test4", "test4", "test4", "test5", "test7", "test"},
			[]string{"test2", "test4", "test5", "test7", "test"},
		},
		{
			[]string{"test3", "test33", "ttest6", "test-2", "test6", "test6", "test-2"},
			[]string{"test3", "test33", "ttest6", "test-2", "test6"},
		},
	}
	for _, tc := range cases {
		util.Equals(t, tc.expected, unique(tc.input))
	}
}

func TestBoundbox(t *testing.T) {
	cases := []struct {
		points   [][]float64
		expected []float64
	}{
		{
			[][]float64{{2.352700, 48.854300}, {-0.039305, 51.421792}, {10.035233, 51.780464}},
			[]float64{-0.039305, 10.035233, 48.8543, 51.780464},
		},
		{
			[][]float64{{12.422600, 38.854300}, {-4.032105, 21.621372}, {11.126233, 0.780464}, {13.012115, -8.120456}},
			[]float64{-4.032105, 13.012115, -8.120456, 38.854300},
		},
		{
			[][]float64{{2.325100, 8.152300}, {-0.032105, 0.621372}},
			[]float64{-0.032105, 2.325100, 0.621372, 8.152300},
		},
	}
	for _, tc := range cases {
		util.Equals(t, tc.expected, Boundbox(tc.points))
	}
}

func TestGeofence(t *testing.T) {
	cases := []struct {
		point    []float64
		polygon  [][]float64
		expected bool
	}{
		{
			[]float64{41.0121814, 28.977277},
			[][]float64{
				{40.9700482, 28.9009094},
				{41.0387075, 28.9160156},
				{41.0503595, 28.9874268},
				{41.0293844, 29.0196991},
				{41.0014072, 29.0550613},
				{40.9796389, 29.0670776},
				{40.9612339, 29.0543747},
				{40.9625302, 29.0334320},
				{40.9716035, 29.0227890},
				{40.9868958, 29.0138626},
				{41.0059413, 29.0059662},
				{41.0074958, 28.9942932},
				{41.0029618, 28.9872551},
				{40.9992047, 28.9754105},
				{40.9697890, 28.9014244},
			},
			true,
		},
		{
			[]float64{41.0121814, 28.977277},
			[][]float64{
				{40.9990104, 28.9692307},
				{41.0191534, 28.9744663},
				{41.0186353, 28.9861393},
				{41.0161096, 28.9889717},
				{41.0087912, 28.9891434},
				{41.0073015, 29.0071678},
				{40.9931153, 29.0106869},
				{40.9786669, 29.0171242},
				{40.9791205, 29.0361786},
				{41.0282189, 29.0329170},
				{41.0262764, 29.0110302},
				{41.0327512, 28.9923191},
				{41.0258231, 28.9671707},
				{41.0021197, 28.9524078},
				{40.9977796, 28.9548969},
				{40.9941519, 28.9700890},
				{41.0063299, 28.9918900},
				{41.0080140, 28.9898300},
				{41.0080787, 28.9878559},
				{41.0000145, 28.9770842},
				{40.9988485, 28.9693165},
			},
			false,
		},
		{
			[]float64{38.3845201, 26.7419811},
			[][]float64{
				{38.4965935, 26.9178772},
				{38.3459645, 26.9480896},
				{38.3384248, 27.2158813},
				{38.4396066, 27.3449707},
				{38.5223841, 27.0524597},
				{38.4976683, 26.9165039},
			},
			false,
		},
	}
	for _, tc := range cases {
		util.Equals(t, tc.expected, GeoFence(Boundbox(tc.polygon), tc.polygon, tc.point[0], tc.point[1]))
	}
}

func TestGetList(t *testing.T) {
	g := testGroup{}
	g.Init()
	node1 := g.nodeObj
	node1.SetName("node-1")
	node2 := g.nodeObj
	node2.SetName("node-2")
	node3 := g.nodeObj
	node3.SetName("node-3")
	node4 := g.nodeObj
	node4.SetName("node-4")

	cases := []struct {
		node     corev1.Node
		expected []string
	}{
		{
			node1,
			[]string{"node-1"},
		},
		{
			node2,
			[]string{"node-1", "node-2"},
		},
		{
			node3,
			[]string{"node-1", "node-2", "node-3"},
		},
		{
			node4,
			[]string{"node-1", "node-2", "node-3", "node-4"},
		},
	}
	for _, tc := range cases {
		_, err := g.client.CoreV1().Nodes().Create(context.TODO(), tc.node.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)
		util.Equals(t, tc.expected, GetList())
	}
}

func TestGetNodeByHostname(t *testing.T) {
	g := testGroup{}
	g.Init()
	node1 := g.nodeObj
	node1.SetName("node-1")
	node2 := g.nodeObj
	node2.SetName("node-2")
	node3 := g.nodeObj
	node3.SetName("node-3")
	node4 := g.nodeObj
	node4.SetName("node-4")

	cases := []struct {
		node     corev1.Node
		name     string
		expected bool
	}{
		{
			node1,
			"node-1",
			true,
		},
		{
			node2,
			"node-2",
			true,
		},
		{
			node3,
			"node-3",
			true,
		},
		{
			node4,
			"node5",
			false,
		},
	}
	for _, tc := range cases {
		_, err := g.client.CoreV1().Nodes().Create(context.TODO(), tc.node.DeepCopy(), metav1.CreateOptions{})
		util.OK(t, err)

		_, err = getNodeByHostname(tc.name)
		util.Equals(t, tc.expected, !errors.IsNotFound(err))
	}
}

func TestGetNodeIPAddresses(t *testing.T) {
	g := testGroup{}
	g.Init()
	node1 := g.nodeObj
	node1.SetName("node-1")
	node1.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "192.168.0.1",
		},
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "10.0.0.1",
		},
	}
	node2 := g.nodeObj
	node2.SetName("node-2")
	node2.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "192.168.0.2",
		},
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "10.0.0.2",
		},
	}
	node3 := g.nodeObj
	node3.SetName("node-3")
	node3.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "192.168.0.3",
		},
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "10.0.0.3",
		},
	}
	node4 := g.nodeObj
	node4.SetName("node-4")
	node4.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "192.168.0.4",
		},
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "10.0.0.4",
		},
	}

	cases := []struct {
		node     corev1.Node
		expected []string
	}{
		{
			node1,
			[]string{"192.168.0.1", "10.0.0.1"},
		},
		{
			node2,
			[]string{"192.168.0.2", "10.0.0.2"},
		},
		{
			node3,
			[]string{"192.168.0.3", "10.0.0.3"},
		},
		{
			node4,
			[]string{"192.168.0.4", "10.0.0.4"},
		},
	}
	for _, tc := range cases {
		internal, external := GetNodeIPAddresses(tc.node.DeepCopy())
		util.Equals(t, tc.expected, []string{internal, external})
	}
}

func TestCompareIPAddresses(t *testing.T) {
	g := testGroup{}
	g.Init()
	node1 := g.nodeObj
	node1.SetName("node-1")
	node1.SetUID("01")
	node1.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "192.168.0.1",
		},
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "10.0.0.1",
		},
	}
	node1Updated := node1
	node1Updated.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "192.168.0.10",
		},
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "10.0.0.10",
		},
	}
	node2 := g.nodeObj
	node2.SetName("node-2")
	node2.SetUID("02")
	node2.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "192.168.0.2",
		},
	}
	node2Updated := node2
	node2Updated.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "192.168.0.3",
		},
	}
	node3 := g.nodeObj
	node3.SetName("node-3")
	node3.SetUID("03")
	node3.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "10.0.0.3",
		},
	}
	node3Updated := node3
	node3Updated.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "10.0.0.30",
		},
	}

	cases := []struct {
		oldObj   corev1.Node
		newObj   corev1.Node
		expected bool
	}{
		{
			node1,
			node1Updated,
			true,
		},
		{
			node1,
			node1,
			false,
		},
		{
			node2,
			node2Updated,
			true,
		},
		{
			node2,
			node2,
			false,
		},
		{
			node3,
			node3,
			false,
		},
		{
			node3,
			node3Updated,
			true,
		},
	}
	for _, tc := range cases {
		util.Equals(t, tc.expected, CompareIPAddresses(tc.oldObj.DeepCopy(), tc.newObj.DeepCopy()))
	}
}

func TestGetConditionReadyStatus(t *testing.T) {
	g := testGroup{}
	g.Init()
	node1 := g.nodeObj
	node1.SetName("node-1")
	node1.SetUID("01")
	node1.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "192.168.0.1",
		},
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "10.0.0.1",
		},
	}
	node1.Status.Conditions = []corev1.NodeCondition{{Status: "True", Type: "Ready"}}
	node2 := g.nodeObj
	node2.SetName("node-2")
	node2.SetUID("02")
	node2.Status.Conditions = []corev1.NodeCondition{{Status: "False", Type: "Ready"}}
	node3 := g.nodeObj
	node3.SetName("node-3")
	node3.SetUID("03")
	node3.Status.Conditions = []corev1.NodeCondition{{Status: "Unknown", Type: "Ready"}}
	node4 := g.nodeObj
	node4.SetName("node-4")
	node4.SetUID("04")
	node4.Status.Conditions = []corev1.NodeCondition{}

	cases := []struct {
		node     corev1.Node
		expected string
	}{
		{
			node1,
			"True",
		},
		{
			node2,
			"False",
		},
		{
			node3,
			"Unknown",
		},
		{
			node4,
			"",
		},
	}
	for _, tc := range cases {
		util.Equals(t, tc.expected, GetConditionReadyStatus(tc.node.DeepCopy()))
	}
}

func TestSetOwnerReferences(t *testing.T) {
	g := testGroup{}
	g.Init()
	// Prepare cases
	node1 := g.nodeObj
	node1.SetName("node-1")
	node2 := g.nodeObj
	node2.SetName("node-2")

	newRef := *metav1.NewControllerRef(node2.DeepCopy(), corev1.SchemeGroupVersion.WithKind("Node"))
	takeControl := false
	newRef.Controller = &takeControl
	ownerReferences := []metav1.OwnerReference{newRef}

	g.client.CoreV1().Nodes().Create(context.TODO(), node1.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), node2.DeepCopy(), metav1.CreateOptions{})

	err := SetOwnerReferences(node1.GetName(), ownerReferences)
	util.OK(t, err)

	node, err := g.client.CoreV1().Nodes().Get(context.TODO(), node1.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, ownerReferences, node.GetOwnerReferences())
}

func TestSetNodeScheduling(t *testing.T) {
	g := testGroup{}
	g.Init()
	// Prepare cases
	node1 := g.nodeObj
	node1.SetName("node-1")

	g.client.CoreV1().Nodes().Create(context.TODO(), node1.DeepCopy(), metav1.CreateOptions{})

	cases := map[string]struct {
		input    bool
		expected bool
	}{
		"true":  {true, true},
		"false": {false, false},
	}

	for k, tc := range cases {
		t.Run(fmt.Sprintf("%s", k), func(t *testing.T) {
			err := SetNodeScheduling(node1.GetName(), tc.input)
			util.OK(t, err)
			node, err := g.client.CoreV1().Nodes().Get(context.TODO(), node1.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			util.Equals(t, tc.expected, node.Spec.Unschedulable)
		})
	}
}

func TestCreateJoinToken(t *testing.T) {
	token := CreateJoinToken("600s", "test.edgenet.io")
	if token == "error" {
		t.Errorf("Token cannot be created")
	}
}

// TODO fuzz fail
/*
	exp: "1.23.5"
	got: ""
*/
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

// TODO fuzz fail
/*
panic: nodes "node-0" not found
*/
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

//TODO  Fuzz fail
/*
panic: regexp: Compile(`^[A-Za-z0-9]([A-Za-z0-9_-.]*[A-Za-z0-9])?$`): error parsing regexp: invalid character class range: `_-.`
*/
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

//TODO test fail
func TestGetMaxmindLocation(t *testing.T) {
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

// func FuzzGetGeolocationByIP(f *testing.F) {}

// func FuzzSetHostname(f *testing.F){}
