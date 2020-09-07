package nodelabeler

import (
	"edgenet/pkg/util"
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartingController(t *testing.T) {
	g := testGroup{}
	g.Init()
	// Run the controller in a goroutine
	go Start(g.client)

	// Prepare cases
	nodeFR := g.nodeObj
	nodeFR.ObjectMeta = metav1.ObjectMeta{
		Name: "fr.edge-net.io",
		Labels: map[string]string{
			"kubernetes.io/hostname": "fr.edge-net.io",
		},
	}
	nodeFR.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "InternalIP",
			Address: "132.227.123.47",
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
	nodeUS := g.nodeObj
	nodeUS.ObjectMeta = metav1.ObjectMeta{
		Name: "us.edge-net.io",
		Labels: map[string]string{
			"kubernetes.io/hostname": "us.edge-net.io",
		},
	}
	nodeUS.Status.Addresses = []corev1.NodeAddress{
		corev1.NodeAddress{
			Type:    "ExternalIP",
			Address: "206.196.180.220",
		},
	}
	nodeUSToUpdate := nodeUS
	nodeUSToUpdate.Status.Addresses[0].Address = "204.102.228.171"
	geolabelsUSUpdated := map[string]string{
		"edge-net.io/continent":   "North_America",
		"edge-net.io/state-iso":   "CA",
		"edge-net.io/country-iso": "US",
		"edge-net.io/city":        "Seaside",
		"edge-net.io/lat":         "n36.621700",
		"edge-net.io/lon":         "w-121.793500",
	}

	cases := map[string]struct {
		Operation string
		Node      corev1.Node
		Expected  map[string]string
	}{
		"fr":        {"create", nodeFR, geolabelsFR},
		"us update": {"update", nodeUSToUpdate, geolabelsUSUpdated},
	}

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			if tc.Operation == "create" {
				g.client.CoreV1().Nodes().Create(tc.Node.DeepCopy())
			} else if tc.Operation == "update" {
				g.client.CoreV1().Nodes().Create(tc.Node.DeepCopy())
				// Wait for the object to be up to date
				time.Sleep(time.Millisecond * 500)
				updatedNode, _ := g.client.CoreV1().Nodes().Get(tc.Node.GetName(), metav1.GetOptions{})
				g.client.CoreV1().Nodes().Update(updatedNode)
			}
			// Wait for the object to be up to date
			time.Sleep(time.Millisecond * 500)
			node, _ := g.client.CoreV1().Nodes().Get(tc.Node.GetName(), metav1.GetOptions{})
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
