package nodelabeler

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"

	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubetestclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

// The main structure of test group
type TestGroup struct {
	nodeObj corev1.Node
}

var controller *Controller
var kubeclientset kubernetes.Interface = kubetestclient.NewSimpleClientset()
var edgenetclientset clientset.Interface = edgenettestclient.NewSimpleClientset()

func TestMain(m *testing.M) {
	//klog.SetOutput(ioutil.Discard)
	//log.SetOutput(ioutil.Discard)
	//logrus.SetOutput(ioutil.Discard)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer ts.Close()

	stopCh := signals.SetupSignalHandler()

	go func() {
		kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)

		newController := NewController(
			kubeclientset,
			edgenetclientset,
			kubeInformerFactory.Core().V1().Nodes(),
			ts.URL+"/",
			"null-account-id",
			"null-license-key",
		)

		kubeInformerFactory.Start(stopCh)
		controller = newController
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	os.Exit(m.Run())
	<-stopCh
}

func (g *TestGroup) Init() {
	// Delete the existing Nodes
	nodeRaw, _ := kubeclientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	for _, nodeRow := range nodeRaw.Items {
		kubeclientset.CoreV1().Nodes().Delete(context.TODO(), nodeRow.GetName(), metav1.DeleteOptions{})
	}

	nodeObj := corev1.Node{
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
				{
					Type:   "Ready",
					Status: "True",
				},
			},
		},
	}

	g.nodeObj = nodeObj
}

func TestAssigningGeoLabels(t *testing.T) {
	g := TestGroup{}
	g.Init()

	// Create the Paris Node
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

	// Create the US Node
	nodeUS := g.nodeObj.DeepCopy()
	nodeUS.ObjectMeta = metav1.ObjectMeta{
		Name: "us.edge-net.io",
		Labels: map[string]string{
			"kubernetes.io/hostname": "us.edge-net.io",
		},
	}
	nodeUS.Status.Addresses = []corev1.NodeAddress{
		{
			Type:    "ExternalIP",
			Address: "206.196.180.220",
		},
	}
	expectedLabelsUS := map[string]string{
		"kubernetes.io/hostname":  "us.edge-net.io",
		"edge-net.io/continent":   "North_America",
		"edge-net.io/state-iso":   "MD",
		"edge-net.io/country-iso": "US",
		"edge-net.io/city":        "College_Park",
		"edge-net.io/lat":         "n38.996500",
		"edge-net.io/lon":         "w-76.934000",
		"edge-net.io/isp":         "University_of_Maryland",
		"edge-net.io/as":          "MAX-GIGAPOP",
		"edge-net.io/asn":         "10886",
	}

	cases := map[string]struct {
		Node     *corev1.Node
		Expected map[string]string
	}{
		"fr": {nodeFR, expectedLabelsFR},
		"us": {nodeUS, expectedLabelsUS},
	}

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			kubeclientset.CoreV1().Nodes().Create(context.TODO(), tc.Node.DeepCopy(), metav1.CreateOptions{})
			time.Sleep(time.Millisecond * 500)
			node, _ := kubeclientset.CoreV1().Nodes().Get(context.TODO(), tc.Node.GetName(), metav1.GetOptions{})
			util.Equals(t, tc.Expected, node.Labels)
		})
	}
}
