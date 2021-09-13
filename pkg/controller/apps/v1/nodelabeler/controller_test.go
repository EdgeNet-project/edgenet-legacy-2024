package nodelabeler

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

// The main structure of test group
type TestGroup struct {
	nodeObj corev1.Node
}

var controller *Controller
var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()

func TestMain(m *testing.M) {
	flag.String("geolite-path", "../../../../../assets/database/GeoLite2-City/GeoLite2-City.mmdb", "Set GeoIP DB path.")
	flag.Parse()

	klog.SetOutput(ioutil.Discard)
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)

	stopCh := signals.SetupSignalHandler()

	go func() {
		kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)

		newController := NewController(kubeclientset,
			kubeInformerFactory.Core().V1().Nodes())

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
	geolabelsFR := map[string]string{
		"edge-net.io/continent":   "Europe",
		"edge-net.io/state-iso":   "IDF",
		"edge-net.io/country-iso": "FR",
		"edge-net.io/city":        "Paris",
		"edge-net.io/lat":         "n48.860700",
		"edge-net.io/lon":         "e2.328100",
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
	geolabelsUS := map[string]string{
		"edge-net.io/continent":   "North_America",
		"edge-net.io/state-iso":   "MD",
		"edge-net.io/country-iso": "US",
		"edge-net.io/city":        "College_Park",
		"edge-net.io/lat":         "n38.989600",
		"edge-net.io/lon":         "w-76.945700",
	}

	cases := map[string]struct {
		Node     *corev1.Node
		Expected map[string]string
	}{
		"fr": {nodeFR, geolabelsFR},
		"us": {nodeUS, geolabelsUS},
	}

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			kubeclientset.CoreV1().Nodes().Create(context.TODO(), tc.Node.DeepCopy(), metav1.CreateOptions{})

			time.Sleep(time.Millisecond * 500)

			node, _ := kubeclientset.CoreV1().Nodes().Get(context.TODO(), tc.Node.GetName(), metav1.GetOptions{})

			if !reflect.DeepEqual(node.Labels, tc.Expected) {
				for actualKey, actualValue := range node.Labels {
					for expectedKey, expectedValue := range tc.Expected {
						if actualKey == expectedKey {
							util.Equals(t, expectedValue, actualValue)
						}
					}
				}
			}
		})
	}
}
