package selectivedeployment

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type TestGroup struct {
	client         kubernetes.Interface
	edgenetClient  versioned.Interface
	sdObj          apps_v1alpha.SelectiveDeployment
	selector       apps_v1alpha.Selector
	deploymentObj  appsv1.Deployment
	daemonsetObj   appsv1.DaemonSet
	statefulsetObj appsv1.StatefulSet
	nodeObj        corev1.Node
	handler        SDHandler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *TestGroup) Init() {
	deploymentObj := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "nginx",
							Image: "nginx:1.7.9",
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	daemonsetObj := appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "nginx",
							Image: "nginx:1.7.9",
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	statefulsetObj := appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			ServiceName: "nginx",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "nginx",
							Image: "nginx:1.7.9",
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	selectorObj := apps_v1alpha.Selector{
		Value:    []string{"Paris"},
		Operator: "In",
		Name:     "city",
	}
	sdObj := apps_v1alpha.SelectiveDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelectiveDeployment",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: apps_v1alpha.SelectiveDeploymentSpec{
			Controllers: apps_v1alpha.Controllers{
				Deployment: []appsv1.Deployment{
					deploymentObj,
				},
				DaemonSet: []appsv1.DaemonSet{
					daemonsetObj,
				},
				StatefulSet: []appsv1.StatefulSet{
					statefulsetObj,
				},
			},
			Selector: []apps_v1alpha.Selector{
				selectorObj,
			},
		},
	}
	nodeObj := corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		Spec: corev1.NodeSpec{
			Unschedulable: false,
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
	g.nodeObj = nodeObj
	g.statefulsetObj = statefulsetObj
	g.daemonsetObj = daemonsetObj
	g.deploymentObj = deploymentObj
	g.selector = selectorObj
	g.sdObj = sdObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetClient = edgenettestclient.NewSimpleClientset()
}

// TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := TestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetClient)
	util.Equals(t, g.client, g.handler.clientset)
	util.Equals(t, g.edgenetClient, g.handler.edgenetClientset)
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Creating nodes
	nodeParis := g.nodeObj
	nodeParis.SetName("edgenet.planet-lab.eu")
	nodeParis.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "edgenet.planet-lab.eu",
		"edge-net.io/city":        "Paris",
		"edge-net.io/country-iso": "FR",
		"edge-net.io/state-iso":   "IDF",
		"edge-net.io/continent":   "Europe",
		"edge-net.io/lon":         "e2.34",
		"edge-net.io/lat":         "n48.86",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeParis.DeepCopy(), metav1.CreateOptions{})
	nodeRichardson := g.nodeObj
	nodeRichardson.SetName("utdallas-1.edge-net.io")
	nodeRichardson.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "utdallas-1.edge-net.io",
		"edge-net.io/city":        "Richardson",
		"edge-net.io/country-iso": "US",
		"edge-net.io/state-iso":   "TX",
		"edge-net.io/continent":   "North America",
		"edge-net.io/lon":         "w-96.78",
		"edge-net.io/lat":         "n32.77",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeRichardson.DeepCopy(), metav1.CreateOptions{})

	sdObj := g.sdObj.DeepCopy()
	sdRepeatedObj := g.sdObj.DeepCopy()
	sdRepeatedObj.SetName("repeated")
	sdRepeatedObj.SetUID("repeated")
	sdPartiallyRepeatedObj := g.sdObj.DeepCopy()
	sdPartiallyRepeatedObj.SetName("partial")
	sdPartiallyRepeatedObj.SetUID("partial")
	deploymentPartial := g.deploymentObj
	deploymentPartial.SetName("partial")
	g.client.AppsV1().Deployments("").Create(context.TODO(), deploymentPartial.DeepCopy(), metav1.CreateOptions{})
	sdPartiallyRepeatedObj.Spec.Controllers.Deployment = append(sdObj.Spec.Controllers.Deployment, deploymentPartial)
	// Deployment, DaemonSet, and StatefulSet created already before the creation of Selective Deployment
	deploymentIrrelevant := g.deploymentObj
	deploymentIrrelevant.SetName("irrelevant")
	g.client.AppsV1().Deployments(deploymentIrrelevant.GetNamespace()).Create(context.TODO(), deploymentIrrelevant.DeepCopy(), metav1.CreateOptions{})
	daemonsetIrrelevant := g.daemonsetObj
	daemonsetIrrelevant.SetName("irrelevant")
	g.client.AppsV1().DaemonSets("").Create(context.TODO(), daemonsetIrrelevant.DeepCopy(), metav1.CreateOptions{})
	statefulsetIrrelevant := g.statefulsetObj
	statefulsetIrrelevant.SetName("irrelevant")
	g.client.AppsV1().StatefulSets("").Create(context.TODO(), statefulsetIrrelevant.DeepCopy(), metav1.CreateOptions{})

	deploymentCreated := g.deploymentObj
	deploymentCreated.SetName("created")
	g.client.AppsV1().Deployments("").Create(context.TODO(), deploymentCreated.DeepCopy(), metav1.CreateOptions{})
	sdObj.Spec.Controllers.Deployment = append(sdObj.Spec.Controllers.Deployment, deploymentCreated)
	daemonsetCreated := g.daemonsetObj
	daemonsetCreated.SetName("created")
	g.client.AppsV1().DaemonSets("").Create(context.TODO(), daemonsetCreated.DeepCopy(), metav1.CreateOptions{})
	sdObj.Spec.Controllers.DaemonSet = append(sdObj.Spec.Controllers.DaemonSet, daemonsetCreated)
	statefulsetCreated := g.statefulsetObj
	statefulsetCreated.SetName("created")
	g.client.AppsV1().StatefulSets("").Create(context.TODO(), statefulsetCreated.DeepCopy(), metav1.CreateOptions{})
	sdObj.Spec.Controllers.StatefulSet = append(sdObj.Spec.Controllers.StatefulSet, statefulsetCreated)
	// Invoke the create function
	g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), sdObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(sdObj.DeepCopy())
	sdCopy, err := g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
	t.Run("status", func(t *testing.T) {
		util.OK(t, err)
		util.Equals(t, success, sdCopy.Status.State)
		util.Equals(t, statusDict["sd-success"], sdCopy.Status.Message[0])
		util.Equals(t, "6/6", sdCopy.Status.Ready)
	})
	g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), sdRepeatedObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(sdRepeatedObj.DeepCopy())
	sdRepeatedCopy, err := g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdRepeatedObj.GetName(), metav1.GetOptions{})
	t.Run("status of failure", func(t *testing.T) {
		util.OK(t, err)
		util.Equals(t, failure, sdRepeatedCopy.Status.State)
		util.Equals(t, "0/3", sdRepeatedCopy.Status.Ready)
	})
	g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), sdPartiallyRepeatedObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(sdPartiallyRepeatedObj.DeepCopy())
	sdPartialCopy, err := g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdPartiallyRepeatedObj.GetName(), metav1.GetOptions{})
	t.Run("status of failure", func(t *testing.T) {
		util.OK(t, err)
		util.Equals(t, partial, sdPartialCopy.Status.State)
		util.Equals(t, "1/4", sdPartialCopy.Status.Ready)
	})
	cases := map[string]struct {
		kind     string
		name     string
		expected string
	}{
		"configure/deployment":   {"Deployment", deploymentCreated.GetName(), nodeParis.GetName()},
		"create/deployment":      {"Deployment", g.sdObj.Spec.Controllers.Deployment[0].GetName(), nodeParis.GetName()},
		"configure/daemonset":    {"DaemonSet", daemonsetCreated.GetName(), nodeParis.GetName()},
		"create/daemonset":       {"DaemonSet", g.sdObj.Spec.Controllers.DaemonSet[0].GetName(), nodeParis.GetName()},
		"configure/statefulset":  {"StatefulSet", statefulsetCreated.GetName(), nodeParis.GetName()},
		"create/statefulset":     {"StatefulSet", g.sdObj.Spec.Controllers.StatefulSet[0].GetName(), nodeParis.GetName()},
		"irrelevant/deployment":  {"Deployment", deploymentIrrelevant.GetName(), ""},
		"irrelevant/daemonset":   {"DaemonSet", daemonsetIrrelevant.GetName(), ""},
		"irrelevant/statefulset": {"StatefulSet", statefulsetIrrelevant.GetName(), ""},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			var affinityValue string
			if tc.kind == "Deployment" {
				deploymentCopy, err := g.client.AppsV1().Deployments("").Get(context.TODO(), tc.name, metav1.GetOptions{})
				util.OK(t, err)
				if deploymentCopy.Spec.Template.Spec.Affinity != nil {
					util.Equals(t, 1, len(deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values))
					affinityValue = deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
				}
			} else if tc.kind == "DaemonSet" {
				daemonsetCopy, err := g.client.AppsV1().DaemonSets("").Get(context.TODO(), tc.name, metav1.GetOptions{})
				util.OK(t, err)
				if daemonsetCopy.Spec.Template.Spec.Affinity != nil {
					util.Equals(t, 1, len(daemonsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values))
					affinityValue = daemonsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]

				}
			} else if tc.kind == "StatefulSet" {
				statfulsetCopy, err := g.client.AppsV1().StatefulSets("").Get(context.TODO(), tc.name, metav1.GetOptions{})
				util.OK(t, err)
				if statfulsetCopy.Spec.Template.Spec.Affinity != nil {
					util.Equals(t, 1, len(statfulsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values))
					affinityValue = statfulsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
				}
			}
			t.Run("node affinity", func(t *testing.T) {
				util.Equals(
					t,
					tc.expected,
					affinityValue)
			})
		})
	}
}

func TestUpdate(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Creating nodes
	nodeParis := g.nodeObj
	nodeParis.SetName("edgenet.planet-lab.eu")
	nodeParis.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "edgenet.planet-lab.eu",
		"edge-net.io/city":        "Paris",
		"edge-net.io/country-iso": "FR",
		"edge-net.io/state-iso":   "IDF",
		"edge-net.io/continent":   "Europe",
		"edge-net.io/lon":         "e2.34",
		"edge-net.io/lat":         "n48.86",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeParis.DeepCopy(), metav1.CreateOptions{})
	nodeRichardson := g.nodeObj
	nodeRichardson.SetName("utdallas-1.edge-net.io")
	nodeRichardson.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "utdallas-1.edge-net.io",
		"edge-net.io/city":        "Richardson",
		"edge-net.io/country-iso": "US",
		"edge-net.io/state-iso":   "TX",
		"edge-net.io/continent":   "North America",
		"edge-net.io/lon":         "w-96.78",
		"edge-net.io/lat":         "n32.77",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeRichardson.DeepCopy(), metav1.CreateOptions{})
	nodeSeaside := g.nodeObj
	nodeSeaside.SetName("nps-1.edge-net.io")
	nodeSeaside.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "nps-1.edge-net.io",
		"edge-net.io/city":        "Seaside",
		"edge-net.io/country-iso": "US",
		"edge-net.io/state-iso":   "CA",
		"edge-net.io/continent":   "North America",
		"edge-net.io/lon":         "w-121.79",
		"edge-net.io/lat":         "n36.62",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeSeaside.DeepCopy(), metav1.CreateOptions{})
	nodeCollegePark := g.nodeObj
	nodeCollegePark.SetName("maxgigapop-1.edge-net.io")
	nodeCollegePark.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "maxgigapop-1.edge-net.io",
		"edge-net.io/city":        "College Park",
		"edge-net.io/country-iso": "US",
		"edge-net.io/state-iso":   "MD",
		"edge-net.io/continent":   "North America",
		"edge-net.io/lon":         "w-76.94",
		"edge-net.io/lat":         "n38.99",
	}
	nodeCollegePark.Status.Conditions[0].Type = "NotReady"
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeCollegePark.DeepCopy(), metav1.CreateOptions{})

	// Invoke the create function
	sdObj := g.sdObj.DeepCopy()
	g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), sdObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(sdObj.DeepCopy())
	sdCopy, err := g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, statusDict["sd-success"], sdCopy.Status.Message[0])
	util.Equals(t, "3/3", sdCopy.Status.Ready)

	deploymentCopy, err := g.client.AppsV1().Deployments("").Get(context.TODO(), sdObj.Spec.Controllers.Deployment[0].GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t,
		nodeParis.GetName(),
		deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
	daemonsetCopy, err := g.client.AppsV1().DaemonSets("").Get(context.TODO(), sdObj.Spec.Controllers.DaemonSet[0].GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t,
		nodeParis.GetName(),
		daemonsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
	statfulsetCopy, err := g.client.AppsV1().StatefulSets("").Get(context.TODO(), sdObj.Spec.Controllers.StatefulSet[0].GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t,
		nodeParis.GetName(),
		statfulsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])

	seaside := g.selector
	seaside.Value = []string{"Seaside"}
	seaside.Quantity = 1
	seaside.Name = "City"
	citySeaside := []apps_v1alpha.Selector{seaside}

	ca := g.selector
	ca.Value = []string{"CA"}
	ca.Quantity = 1
	ca.Name = "State"
	stateCA := []apps_v1alpha.Selector{ca}

	us := g.selector
	us.Value = []string{"US"}
	us.Name = "Country"
	countryUSAll := []apps_v1alpha.Selector{us}
	us.Operator = "NotIn"
	countryUSOut := []apps_v1alpha.Selector{us}
	us.Operator = "In"
	us.Quantity = 1
	countryUS := []apps_v1alpha.Selector{us}
	useu := g.selector
	useu.Value = []string{"US", "FR"}
	useu.Quantity = 2
	useu.Name = "Country"
	countryUSEU1 := []apps_v1alpha.Selector{useu}
	fr := g.selector
	fr.Value = []string{"FR"}
	fr.Quantity = 1
	fr.Name = "Country"
	countryUSEU2 := []apps_v1alpha.Selector{us, fr}

	eu := g.selector
	eu.Value = []string{"Europe"}
	eu.Quantity = 1
	eu.Name = "Continent"
	continentEU := []apps_v1alpha.Selector{eu}

	paris := g.selector
	paris.Value = []string{"[ [2.2150567, 48.8947616], [2.2040704, 48.8084639], [2.3393396, 48.7835862], [2.4519494, 48.8416903], [2.3932412, 48.9171024] ]"}
	paris.Quantity = 1
	paris.Name = "Polygon"
	polygonParis := []apps_v1alpha.Selector{paris}

	countryUScityParis := []apps_v1alpha.Selector{us, paris}

	paris.Quantity = 4
	polygonParisFewer := []apps_v1alpha.Selector{paris}
	us.Quantity = 3
	countryUSFewer := []apps_v1alpha.Selector{us}

	cases := map[string]struct {
		input          []apps_v1alpha.Selector
		expectedStatus string
		expected       [][]string
	}{
		"city/seaside":          {citySeaside, success, [][]string{[]string{nodeSeaside.GetName()}}},
		"polygon/paris":         {polygonParis, success, [][]string{[]string{nodeParis.GetName()}}},
		"state/ca":              {stateCA, success, [][]string{[]string{nodeSeaside.GetName()}}},
		"country/us":            {countryUS, success, [][]string{[]string{nodeSeaside.GetName()}}},
		"country/us/all":        {countryUSAll, success, [][]string{[]string{nodeSeaside.GetName(), nodeRichardson.GetName()}}},
		"country/us/out":        {countryUSOut, success, [][]string{[]string{nodeParis.GetName()}}},
		"continent/europe":      {continentEU, success, [][]string{[]string{nodeParis.GetName()}}},
		"country/us-eu/1":       {countryUSEU1, success, [][]string{[]string{nodeSeaside.GetName(), nodeRichardson.GetName()}}},
		"country/us-eu/2":       {countryUSEU2, success, [][]string{[]string{nodeSeaside.GetName()}, []string{nodeParis.GetName()}}},
		"country/us|city/paris": {countryUScityParis, success, [][]string{[]string{nodeSeaside.GetName()}, []string{nodeParis.GetName()}}},
		"polygon/paris/fewer":   {polygonParisFewer, failure, [][]string{[]string{nodeParis.GetName()}}},
		"country/us/fewer":      {countryUSFewer, failure, [][]string{[]string{nodeSeaside.GetName(), nodeRichardson.GetName()}}},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			sdCopy, _ := g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
			sdCopy.Spec.Selector = tc.input
			g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Update(context.TODO(), sdCopy, metav1.UpdateOptions{})
			g.handler.ObjectUpdated(sdCopy)
			sdCopy, _ = g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
			util.Equals(t, tc.expectedStatus, sdCopy.Status.State)
			deploymentCopy, err := g.client.AppsV1().Deployments("").Get(context.TODO(), deploymentCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			for i, expected := range tc.expected {
				util.Equals(t,
					expected,
					deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[i].MatchExpressions[0].Values)
			}
			daemonsetCopy, err := g.client.AppsV1().DaemonSets("").Get(context.TODO(), daemonsetCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			for j, expected := range tc.expected {
				util.Equals(t,
					expected,
					daemonsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[j].MatchExpressions[0].Values)
			}
			statfulsetCopy, err := g.client.AppsV1().StatefulSets("").Get(context.TODO(), statfulsetCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			for z, expected := range tc.expected {
				util.Equals(t,
					expected,
					statfulsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[z].MatchExpressions[0].Values)
			}
		})
	}

	t.Run("controller spec", func(t *testing.T) {
		util.Equals(t, sdCopy.Spec.Controllers.Deployment[0].Spec.Template.Spec.Containers[0].Image, deploymentCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, sdCopy.Spec.Controllers.DaemonSet[0].Spec.Template.Spec.Containers[0].Image, daemonsetCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, sdCopy.Spec.Controllers.StatefulSet[0].Spec.Template.Spec.Containers[0].Image, statfulsetCopy.Spec.Template.Spec.Containers[0].Image)

		sdCopy.Spec.Controllers.Deployment[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.0"
		sdCopy.Spec.Controllers.DaemonSet[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.1"
		sdCopy.Spec.Controllers.StatefulSet[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.2"

		g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Update(context.TODO(), sdCopy, metav1.UpdateOptions{})
		g.handler.ObjectUpdated(sdCopy)
		deploymentCopy, err := g.client.AppsV1().Deployments("").Get(context.TODO(), deploymentCopy.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		daemonsetCopy, err := g.client.AppsV1().DaemonSets("").Get(context.TODO(), daemonsetCopy.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		statfulsetCopy, err := g.client.AppsV1().StatefulSets("").Get(context.TODO(), statfulsetCopy.GetName(), metav1.GetOptions{})
		util.OK(t, err)

		util.Equals(t, "nginx:1.8.0", deploymentCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, "nginx:1.8.1", daemonsetCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, "nginx:1.8.2", statfulsetCopy.Spec.Template.Spec.Containers[0].Image)
	})
}

func TestGetByNode(t *testing.T) {
	g := TestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetClient)
	// Creating nodes
	nodeParis := g.nodeObj
	nodeParis.SetName("edgenet.planet-lab.eu")
	nodeParis.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "edgenet.planet-lab.eu",
		"edge-net.io/city":        "Paris",
		"edge-net.io/country-iso": "FR",
		"edge-net.io/state-iso":   "IDF",
		"edge-net.io/continent":   "Europe",
		"edge-net.io/lon":         "e2.34",
		"edge-net.io/lat":         "n48.86",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeParis.DeepCopy(), metav1.CreateOptions{})
	nodeRichardson := g.nodeObj
	nodeRichardson.SetName("utdallas-1.edge-net.io")
	nodeRichardson.ObjectMeta.Labels = map[string]string{
		"kubernetes.io/hostname":  "utdallas-1.edge-net.io",
		"edge-net.io/city":        "Richardson",
		"edge-net.io/country-iso": "US",
		"edge-net.io/state-iso":   "TX",
		"edge-net.io/continent":   "North America",
		"edge-net.io/lon":         "w-96.78",
		"edge-net.io/lat":         "n32.77",
	}
	g.client.CoreV1().Nodes().Create(context.TODO(), nodeRichardson.DeepCopy(), metav1.CreateOptions{})

	// Invoke the create function
	sdObj := g.sdObj.DeepCopy()
	useu := g.selector
	useu.Value = []string{"US", "FR"}
	useu.Quantity = 2
	useu.Name = "Country"
	sdObj.Spec.Selector = []apps_v1alpha.Selector{useu}

	g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), sdObj.DeepCopy(), metav1.CreateOptions{})
	g.handler.ObjectCreated(sdObj.DeepCopy())
	sdCopy, err := g.edgenetClient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, statusDict["sd-success"], sdCopy.Status.Message[0])
	util.Equals(t, "3/3", sdCopy.Status.Ready)

	ownerList, status := g.handler.getByNode(nodeParis.GetName())
	util.Equals(t, true, status)
	util.Equals(t, "", ownerList[0][0])
	util.Equals(t, sdObj.GetName(), ownerList[0][1])

	ownerList, status = g.handler.getByNode(nodeRichardson.GetName())
	util.Equals(t, true, status)
	util.Equals(t, "", ownerList[0][0])
	util.Equals(t, sdObj.GetName(), ownerList[0][1])
}
