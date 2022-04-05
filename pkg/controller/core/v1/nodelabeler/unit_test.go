package nodelabeler

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
)

var kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)
var ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	s := strings.Split(r.URL.Path, "../../../../../configs/nodelabeler/")
	// 1.2.3.4 -> 1-2-3-4.json
	filename := strings.Replace(s[len(s)-1], ".", "-", -1) + ".json"
	response, err := ioutil.ReadFile(fmt.Sprintf("../../../../../configs/nodelabeler/%s", filename))
	if err != nil {
		w.WriteHeader(400)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			panic(err)
		}
	}
	_, err = w.Write(response)
	if err != nil {
		panic(err)
	}
}))

var c = NewController(
	kubeclientset,
	edgenetclientset,
	kubeInformerFactory.Core().V1().Nodes(),
	ts.URL+"/",
	"null-account-id",
	"null-license-key",
)

func getTestResource() *corev1.Node {
	g := TestGroup{}
	g.Init()
	nodeFR := g.nodeObj.DeepCopy()
	nodeFR.ObjectMeta = metav1.ObjectMeta{
		Name: "fr.edge-net.io",
		Labels: map[string]string{
			"kubernetes.io/hostname": "fr.edge-net.io",
		},
	}
	nodeFR.Status.Addresses = []corev1.NodeAddress{
		{
			Type:    "InternalIP",
			Address: "132.227.123.51",
		},
	}
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeFR.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	node, _ := kubeclientset.CoreV1().Nodes().Get(context.TODO(), nodeFR.GetName(), metav1.GetOptions{})
	return node
}

func TestSyncHandler(t *testing.T) {
	key := "default/fr.edge-net.io"
	node := getTestResource()
	expectedLabelsFR := map[string]string{
		"kubernetes.io/hostname":  "fr.edge-net.io",
		"edge-net.io/continent":   "Europe",
		"edge-net.io/state-iso":   "IDF",
		"edge-net.io/country-iso": "FR",
		"edge-net.io/city":        "Pantin",
		"edge-net.io/lat":         "n48.895800",
		"edge-net.io/lon":         "e2.406400",
		"edge-net.io/isp":         "Renater",
		"edge-net.io/as":          "Renater",
		"edge-net.io/asn":         "1307",
	}
	err := c.syncHandler(key)
	util.OK(t, err)
	util.Equals(t, expectedLabelsFR, node.Labels)
}

// TODO: More test cases
func processNextWorkItem(t *testing.T) {
	node_1 := getTestResource()
	node_2 := getTestResource()
	c.enqueueNodelabeler(node_1)
	c.enqueueNodelabeler(node_2)
	c.processNextWorkItem()
	util.Equals(t, 1, c.workqueue.Len())
	c.processNextWorkItem()
	util.Equals(t, 0, c.workqueue.Len())
}

// TODO: test failed
/*
=== RUN   TestEnqueueNodelabeler
I0405 01:08:34.822932    3504 controller.go:82] Setting up event handlers
unit_test.go:109:

        exp: 2

        got: 1

--- FAIL: TestEnqueueNodelabeler (1.01s)
*/
func TestEnqueueNodelabeler(t *testing.T) {
	node_1 := getTestResource()
	node_2 := getTestResource()

	c.enqueueNodelabeler(node_1)
	util.Equals(t, 1, c.workqueue.Len())
	c.enqueueNodelabeler(node_2)
	util.Equals(t, 2, c.workqueue.Len())
}
