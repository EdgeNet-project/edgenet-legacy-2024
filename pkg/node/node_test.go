package node

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestUnique(t *testing.T) {
	var tests = []struct {
		input  []string
		expect []string
	}{
		{[]string{"test1", "test1", "test2", "test2", "test3", "t"}, []string{"test1", "test2", "test3", "t"}},
		{[]string{"test1", "test1", "test3", "test2", "test3", "test4", "test444", "r"}, []string{"test1", "test3", "test2", "test4", "test444", "r"}},
		{[]string{"test2", "test4", "test4", "test4", "test5", "test7", "test"}, []string{"test2", "test4", "test5", "test7", "test"}},
		{[]string{"test3", "test33", "ttest6", "test6", "test6"}, []string{"test3", "test33", "ttest6", "test6"}},
	}
	for _, v := range tests {
		ret := unique(v.input)
		ok := Equal(ret, v.expect)
		if ok {
			t.Logf("pass")
		} else {
			t.Errorf("fail, want %+v, get %+v\n", v.expect, ret)
		}
	}
}
func Equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestSetNodeLabels(t *testing.T) {
	data := []struct {
		clientset      kubernetes.Interface
		hostname       string
		labelsExpected map[string]string
		expected       bool
	}{
		{clientset: testclient.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node1",
				Namespace: "",
				Labels: map[string]string{
					"edge-net.io~1continent":   "",
					"edge-net.io~1country-iso": "",
					"edge-net.io~1state-iso":   "",
					"edge-net.io~1city":        "",
					"edge-net.io~1lon":         "",
					"edge-net.io~1lat":         "",
				},
			},
		}),
			hostname: "node1",
			expected: true,
			labelsExpected: map[string]string{
				"edge-net.io~1continent":   "Europe",
				"edge-net.io~1country-iso": "FR",
				"edge-net.io~1state-iso":   "IDF",
				"edge-net.io~1city":        "Paris",
				"edge-net.io~1lon":         "e2.352700",
				"edge-net.io~1lat":         "n48.854300",
			},
		},
	}

	for _, test := range data {
		//t.Logf("TEST, %v #\n %v #\n %v", test.hostname, test.labelsExpected, test.clientset)
		if output := setNodeLabels(test.hostname, test.labelsExpected, test.clientset); output != test.expected {
			t.Error("Error")
		}
	}
}

func TestGetList(t *testing.T) {
	data := []struct {
		clientset    kubernetes.Interface
		expectedNode []string
	}{
		{clientset: testclient.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node1",
				Namespace: "default",
			},
		}, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node2",
				Namespace: "default"},
		}),
			expectedNode: []string{"node1", "node2"},
		},
		{clientset: testclient.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node3",
				Namespace: "default"},
		}, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node4",
				Namespace: "default"},
		}),
			expectedNode: []string{"node3", "node4"},
		}}
	for _, single := range data {
		if !Equal(GetList(single.clientset), single.expectedNode) {
			t.Fatal("error")
		}
	}
}

func TestGetGeolocationByIP(t *testing.T) {
	/* //hostname := "Farhad"
	ipStr := "46.193.66.93"
	geoLabelsFirstIP := map[string]string{
		"edge-net.io~1continent":   "Europe",
		"edge-net.io~1country-iso": "FR",
		"edge-net.io~1state-iso":   "IDF",
		"edge-net.io~1city":        "Paris",
		"edge-net.io~1lon":         "e2.352700",
		"edge-net.io~1lat":         "n48.854300",
	}
	ip := net.ParseIP(ipStr)
	db, err := geoip2.Open("../../assets/database/GeoLite2-City/GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	record, err := db.City(ip)
	if err != nil {
		log.Fatal(err)
	}
	continent := strings.Replace(record.Continent.Names["en"], " ", "_", -1)
	country := record.Country.IsoCode
	state := record.Country.IsoCode
	city := strings.Replace(record.City.Names["en"], " ", "_", -1)
	var lon string
	var lat string
	if record.Location.Longitude >= 0 {
		lon = fmt.Sprintf("e%.6f", record.Location.Longitude)
	} else {
		lon = fmt.Sprintf("w%.6f", record.Location.Longitude)
	}
	if record.Location.Latitude >= 0 {
		lat = fmt.Sprintf("n%.6f", record.Location.Latitude)
	} else {
		lat = fmt.Sprintf("s%.6f", record.Location.Latitude)
	}
	if len(record.Subdivisions) > 0 {
		state = record.Subdivisions[0].IsoCode
	}

	// Create label map to attach to the node
	geoLabels := map[string]string{
		"edge-net.io~1continent":   continent,
		"edge-net.io~1country-iso": country,
		"edge-net.io~1state-iso":   state,
		"edge-net.io~1city":        city,
		"edge-net.io~1lon":         lon,
		"edge-net.io~1lat":         lat,
	}

	if setNodeLabels("Farhad", geoLabelsFirstIP) != GetGeolocationByIP("Farhad", ipStr) {
		t.Logf("Failed it's not equall")
	}

	// Attach geolabels to the node
	//result := setNodeLabels(hostname, geoLabels)

	t.Logf("It worked %v\n", record)
	t.Logf("result %v", geoLabels)*/
}

func TestGetNodeByHostname(t *testing.T) {
	data := []struct {
		clientset kubernetes.Interface
		node      string
		expected  string
	}{
		{clientset: testclient.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node1",
				Namespace: "default"},
		}, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node2",
				Namespace: "default"},
		}),
			node:     "node1",
			expected: "true"},
		{clientset: testclient.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node3",
				Namespace: "default"},
		}, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node4",
				Namespace: "default"},
		}),
			node:     "node4",
			expected: "true"},
		{clientset: testclient.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node5",
				Namespace: "default"},
		}, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node6",
				Namespace: "default",
			},
		}),
			node:     "node4",
			expected: "false",
		},
	}

	for _, test := range data {
		if output, err := getNodeByHostname(test.node, test.clientset); output != test.expected {
			t.Error(err)
		}
	}
}

func TestGetNodeIPAddresses(t *testing.T) {

	node1 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1", UID: "01"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.1", Type: "InternalIP"}, {Address: "10.0.0.1", Type: "ExternalIP"}}}}
	node2 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-2", UID: "01"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.2", Type: "InternalIP"}, {Address: "10.0.0.2", Type: "ExternalIP"}}}}

	data := []struct {
		node       *corev1.Node
		expectedip []string
	}{
		{&node1, []string{"192.168.0.1", "10.0.0.1"}},
		{&node2, []string{"192.168.0.2", "10.0.0.2"}},
	}

	for _, test := range data {
		if outputInternal, outputExternal := GetNodeIPAddresses(test.node); !Equal([]string{outputInternal, outputExternal}, test.expectedip) {
			t.Error("error")
		}
	}

}

func TestCompareIPAddresses(t *testing.T) {
	node1 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1", UID: "01"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.1", Type: "InternalIP"}, {Address: "10.0.0.1", Type: "ExternalIP"}}}}
	node2 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-2", UID: "02"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.2", Type: "InternalIP"}, {Address: "10.0.0.2", Type: "ExternalIP"}}}}

	node3 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-3", UID: "03"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.1", Type: "InternalIP"}, {Address: "10.0.0.6", Type: "ExternalIP"}}}}
	node4 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-4", UID: "04"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.5", Type: "InternalIP"}, {Address: "10.0.0.2", Type: "ExternalIP"}}}}
	data := []struct {
		oldnode  *corev1.Node
		newnode  *corev1.Node
		expected bool
	}{
		{&node1, &node1, false},
		{&node2, &node1, true},
		{&node2, &node4, true},
		{&node1, &node3, true},
	}

	for _, test := range data {
		if output := CompareIPAddresses(test.oldnode, test.newnode); output != test.expected {
			t.Error("error")
		}
	}

}

func TestGetConditionReadyStatus(t *testing.T) {
	node1 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1", UID: "01"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.1", Type: "InternalIP"}, {Address: "10.0.0.1", Type: "ExternalIP"}},
			Conditions: []corev1.NodeCondition{{Type: "Ready"}}},
	}
	node2 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-2", UID: "02"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.2", Type: "InternalIP"}, {Address: "10.0.0.2", Type: "ExternalIP"}},
			Conditions: []corev1.NodeCondition{{Status: "true", Type: "Ready"}}},
	}
	node3 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-3", UID: "03"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.3", Type: "InternalIP"}, {Address: "10.0.0.3", Type: "ExternalIP"}},
			Conditions: []corev1.NodeCondition{{Status: "unknown", Type: "on"}}},
	}
	node4 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-4", UID: "04"},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "192.168.0.4", Type: "InternalIP"}, {Address: "10.0.0.4", Type: "ExternalIP"}},
			Conditions: []corev1.NodeCondition{{Status: "", Type: "Ready"}}},
	}
	data := []struct {
		node     *corev1.Node
		expected string
	}{
		{&node1, ""},
		{&node2, "true"},
		{&node3, ""},
		{&node4, ""},
	}

	for _, test := range data {
		if output := GetConditionReadyStatus(test.node); output != test.expected {
			t.Error("error")
		}
	}

}
