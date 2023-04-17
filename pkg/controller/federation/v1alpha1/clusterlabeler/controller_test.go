package clusterlabeler

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

	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubetestclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

// The main structure of test group
type TestGroup struct {
	clusterObj federationv1alpha1.Cluster
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
		edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

		newController := NewController(
			kubeclientset,
			edgenetclientset,
			edgenetInformerFactory.Federation().V1alpha1().Clusters(),
			ts.URL+"/",
			"null-account-id",
			"null-license-key",
		)

		edgenetInformerFactory.Start(stopCh)
		controller = newController
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	os.Exit(m.Run())
	<-stopCh
}

func (g *TestGroup) Init() {
	// Delete the existing clusters
	clusterRaw, _ := edgenetclientset.FederationV1alpha1().Clusters("").List(context.TODO(), metav1.ListOptions{})
	for _, clusterRow := range clusterRaw.Items {
		edgenetclientset.FederationV1alpha1().Clusters(clusterRow.GetNamespace()).Delete(context.TODO(), clusterRow.GetName(), metav1.DeleteOptions{})
	}

	// Create the cluster object
	clusterObj := federationv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "federation.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-namespace",
		},
		Spec: federationv1alpha1.ClusterSpec{
			UID:        "test-uid",
			Role:       "workload",
			Server:     "8.8.8.8",
			Visibility: "Public",
			SecretName: "test-secret",
		},
		Status: federationv1alpha1.ClusterStatus{
			State: federationv1alpha1.StatusReady,
		},
	}
	g.clusterObj = clusterObj
}

func TestAssigningGeoLabels(t *testing.T) {
	g := TestGroup{}
	g.Init()

	// Create a Paris Cluster
	clusterFR := g.clusterObj.DeepCopy()
	clusterFR.Spec.Server = "132.227.123.51"

	expectedLabelsFR := map[string]string{
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

	// Create a US Node
	clusterUS := g.clusterObj.DeepCopy()
	clusterUS.SetName("test-cluster-us")
	clusterUS.Spec.Server = "206.196.180.220"

	expectedLabelsUS := map[string]string{
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
		Cluster  *federationv1alpha1.Cluster
		Expected map[string]string
	}{
		"fr": {clusterFR, expectedLabelsFR},
		"us": {clusterUS, expectedLabelsUS},
	}

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			edgenetclientset.FederationV1alpha1().Clusters(tc.Cluster.GetNamespace()).Create(context.TODO(), tc.Cluster.DeepCopy(), metav1.CreateOptions{})
			time.Sleep(time.Millisecond * 500)
			cluster, _ := edgenetclientset.FederationV1alpha1().Clusters(tc.Cluster.GetNamespace()).Get(context.TODO(), tc.Cluster.GetName(), metav1.GetOptions{})
			util.Equals(t, tc.Expected, cluster.Labels)
		})
	}
}
