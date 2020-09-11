package selectivedeployment

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"

	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Dictionary for error messages
var errorDict = map[string]string{
	"k8-sync":                    "Kubernetes clientset sync problem",
	"edgnet-sync":                "EdgeNet clientset sync problem",
	"SD-deployment-fail":         "Selective deployment failed with Deployment as a controller",
	"SD-daemonSet-fail":          "Selective deployment failed with DaemonSet as a controller",
	"SD-statefulSet-fail":        "Selective deployment failed with StatefulSet as a controller",
	"SD-deploymentPolygon-fail":  "Selective deployment failed with Deployment as a controller and using polygon",
	"select-deployment-fail":     "Deployment is not in the currect node",
	"select-daemonset-fail":      "Daemonset is not in the currect node",
	"select-statefulset-fail":    "Statefulset is not in the currect node",
	"SD-deploymentExisted-fail":  "SD failed with Deployment as a controller and existed Deployment",
	"SD-daemonSetExisted-fail":   "SD failed with DaemonSet as a controller and existed DaemonSet",
	"SD-statefulSetExisted-fail": "SD failed with StatefulSet as a controller and existed StatefulSet",
	"checkCon-fail":              "Check controller func failed",
	"GetbyNode-fail":             "GetbyNode status failed",
	"GetbyNode-fail-owner":       "GetbyNode ownerList failed",
}

type SDTestGroup struct {
	client                 kubernetes.Interface
	edgenetclient          versioned.Interface
	statefulsetService     corev1.Service
	nodeFRObj              corev1.Node
	nodeUSObj              corev1.Node
	nodeUSSecondObj        corev1.Node
	nodeUSThirdObj         corev1.Node
	sdObjDeployment        apps_v1alpha.SelectiveDeployment
	sdObjDeploymentPolygon apps_v1alpha.SelectiveDeployment
	sdObjStatefulset       apps_v1alpha.SelectiveDeployment
	sdObjDaemonset         apps_v1alpha.SelectiveDeployment
	deploymentObj          v1.Deployment
	statefulSetObj         v1.StatefulSet
	daemonsetObj           v1.DaemonSet
	handler                SDHandler
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

// Init syncs the test group
func (g *SDTestGroup) Init() {
	sdObjDeployment := apps_v1alpha.SelectiveDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelectiveDeployment",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "country",
		},
		Spec: apps_v1alpha.SelectiveDeploymentSpec{
			Controllers: apps_v1alpha.Controllers{
				Deployment: []v1.Deployment{
					v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "deployment1",
							Labels: map[string]string{
								"app": "nginx",
							},
						},
						Spec: v1.DeploymentSpec{
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
					},
				},
			},
			Selector: []apps_v1alpha.Selector{
				{
					Value:    []string{"FR"},
					Operator: "In",
					Quantity: 1,
					Name:     "country",
				},
			},
		},
	}
	sdObjDeploymentPolygon := apps_v1alpha.SelectiveDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelectiveDeployment",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "polygon",
		},
		Spec: apps_v1alpha.SelectiveDeploymentSpec{
			Controllers: apps_v1alpha.Controllers{
				Deployment: []v1.Deployment{
					v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "deployment1",
							Labels: map[string]string{
								"app": "nginx",
							},
						},
						Spec: v1.DeploymentSpec{
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
					},
				},
			},
			Selector: []apps_v1alpha.Selector{
				{
					Value: []string{
						"[ [2.2150567, 48.8947616], [2.2040704, 48.8084639], [2.3393396, 48.7835862], [2.4519494, 48.8416903], [2.3932412, 48.9171024] ]",
					},
					Operator: "In",
					Quantity: 1,
					Name:     "polygon",
				},
			},
		},
	}
	sdObjStatefulset := apps_v1alpha.SelectiveDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelectiveDeployment",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "state",
		},
		Spec: apps_v1alpha.SelectiveDeploymentSpec{
			Controllers: apps_v1alpha.Controllers{
				StatefulSet: []v1.StatefulSet{
					v1.StatefulSet{
						TypeMeta: metav1.TypeMeta{
							Kind:       "StatefulSet",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "statefulset",
							Labels: map[string]string{
								"app": "nginx",
							},
						},
						Spec: v1.StatefulSetSpec{
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
					},
				},
			},
			Selector: []apps_v1alpha.Selector{
				{
					Value:    []string{"MD"},
					Operator: "In",
					Quantity: 1,
					Name:     "State",
				},
			},
		},
	}
	sdObjDaemonset := apps_v1alpha.SelectiveDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelectiveDeployment",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "country",
		},
		Spec: apps_v1alpha.SelectiveDeploymentSpec{
			Controllers: apps_v1alpha.Controllers{
				DaemonSet: []v1.DaemonSet{
					v1.DaemonSet{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "apps/v1",
							Kind:       "DaemonSet",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "daemonset",
							Labels: map[string]string{
								"app": "nginx",
							},
						},
						Spec: v1.DaemonSetSpec{
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
					},
				},
			},
			Selector: []apps_v1alpha.Selector{
				{
					Value:    []string{"FR"},
					Operator: "In",
					Quantity: 1,
					Name:     "Country",
				},
			},
		},
	}
	statefulsetService := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port: 80,
					Name: "web",
				},
			},
			ClusterIP: "None",
			Selector: map[string]string{
				"app": "nginx",
			},
		},
	}
	nodeFRObj := corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet.planet-lab.eu",
			Labels: map[string]string{
				"kubernetes.io/hostname":  "edgenet.planet-lab.eu",
				"edge-net.io/city":        "Paris",
				"edge-net.io/country-iso": "FR",
				"edge-net.io/state-iso":   "FR",
				"edge-net.io/continent":   "Europe",
				"edge-net.io/lon":         "e2.34",
				"edge-net.io/lat":         "n48.86",
			},
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
	nodeUSObj := corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "maxgigapop-1.edge-net.io",
			Labels: map[string]string{
				"kubernetes.io/hostname":  "maxgigapop-1.edge-net.io",
				"edge-net.io/city":        "College Park",
				"edge-net.io/country-iso": "US",
				"edge-net.io/state-iso":   "MD",
				"edge-net.io/continent":   "North America",
				"edge-net.io/lon":         "w-76.94",
				"edge-net.io/lat":         "n38.99",
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("3000000"),
				corev1.ResourceCPU:              resource.MustParse("2"),
				corev1.ResourceEphemeralStorage: resource.MustParse("50100880"),
				corev1.ResourcePods:             resource.MustParse("100"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("3000000"),
				corev1.ResourceCPU:              resource.MustParse("2"),
				corev1.ResourceEphemeralStorage: resource.MustParse("50100880"),
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
	nodeUSSecondObj := corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "nps-1.edge-net.io",
			Labels: map[string]string{
				"kubernetes.io/hostname":  "nps-1.edge-net.io",
				"edge-net.io/city":        "Seaside",
				"edge-net.io/country-iso": "US",
				"edge-net.io/state-iso":   "CA",
				"edge-net.io/continent":   "North America",
				"edge-net.io/lon":         "w-121.79",
				"edge-net.io/lat":         "n36.62",
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("30000"),
				corev1.ResourceCPU:              resource.MustParse("1"),
				corev1.ResourceEphemeralStorage: resource.MustParse("50100"),
				corev1.ResourcePods:             resource.MustParse("10"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("30000"),
				corev1.ResourceCPU:              resource.MustParse("1"),
				corev1.ResourceEphemeralStorage: resource.MustParse("50100"),
				corev1.ResourcePods:             resource.MustParse("10"),
			},
			Conditions: []corev1.NodeCondition{
				corev1.NodeCondition{
					Type:   "Ready",
					Status: "True",
				},
			},
		},
	}
	nodeUSThirdObj := corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "apps.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "utdallas-1.edge-net.io",
			Labels: map[string]string{
				"kubernetes.io/hostname":  "utdallas-1.edge-net.io",
				"edge-net.io/city":        "Richardson",
				"edge-net.io/country-iso": "US",
				"edge-net.io/state-iso":   "TX",
				"edge-net.io/continent":   "North America",
				"edge-net.io/lon":         "w-96.78",
				"edge-net.io/lat":         "n32.77",
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("30000"),
				corev1.ResourceCPU:              resource.MustParse("1"),
				corev1.ResourceEphemeralStorage: resource.MustParse("50100"),
				corev1.ResourcePods:             resource.MustParse("10"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("30000"),
				corev1.ResourceCPU:              resource.MustParse("1"),
				corev1.ResourceEphemeralStorage: resource.MustParse("50100"),
				corev1.ResourcePods:             resource.MustParse("10"),
			},
			Conditions: []corev1.NodeCondition{
				corev1.NodeCondition{
					Type:   "Ready",
					Status: "True",
				},
			},
		},
	}
	deploymentObj := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "deployment1",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: v1.DeploymentSpec{
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
	daemonsetObj := v1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "daemonset",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: v1.DaemonSetSpec{
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
	statefulSetObj := v1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "statefulset",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: v1.StatefulSetSpec{
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
	g.nodeFRObj = nodeFRObj
	g.nodeUSObj = nodeUSObj
	g.nodeUSSecondObj = nodeUSSecondObj
	g.nodeUSThirdObj = nodeUSThirdObj
	g.sdObjDeployment = sdObjDeployment
	g.sdObjDeploymentPolygon = sdObjDeploymentPolygon
	g.statefulsetService = statefulsetService
	g.sdObjStatefulset = sdObjStatefulset
	g.sdObjDaemonset = sdObjDaemonset
	g.deploymentObj = deploymentObj
	g.daemonsetObj = daemonsetObj
	g.statefulSetObj = statefulSetObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
}

// TestHandlerInit for handler initialization
func TestHandlerInit(t *testing.T) {
	// Sync the test group
	g := SDTestGroup{}
	g.Init()
	// Initialize the handler
	g.handler.Init(g.client, g.edgenetclient)
	if g.handler.clientset != g.client {
		t.Error(errorDict["k8-sync"])
	}
	if g.handler.edgenetClientset != g.edgenetclient {
		t.Error(errorDict["edgenet-sync"])
	}
}

func TestObjectCreated(t *testing.T) {
	g := SDTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creating four nodes
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeFRObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSSecondObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSThirdObj.DeepCopy(), metav1.CreateOptions{})

	t.Run("creation of SD, Deployment as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDeployment.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDeployment.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-deployment-fail"])
		}
		// Checking the node name
		deployment, _ := g.client.AppsV1().Deployments("").Get(context.TODO(), g.sdObjDeployment.Spec.Controllers.Deployment[0].GetName(), metav1.GetOptions{})
		deploymentNodeName := deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
		if deploymentNodeName != g.nodeFRObj.GetName() {
			t.Errorf(errorDict["select-deployment-fail"])
		}
		// pods, _ := g.client.CoreV1().Pods("").List(metav1.ListOptions{})
		// TBD: The pod list is at this point empty and looks like it's because of the Fake client
	})

	t.Run("creation of SD, DaemonSet as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDaemonset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDaemonset.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-daemonSet-fail"])
		}
		// Checking the node name
		daemonset, _ := g.client.AppsV1().DaemonSets("").Get(context.TODO(), g.sdObjDaemonset.Spec.Controllers.DaemonSet[0].GetName(), metav1.GetOptions{})
		daemonsetNodeName := daemonset.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
		if daemonsetNodeName != g.nodeFRObj.GetName() {
			t.Errorf(errorDict["select-daemonset-fail"])
		}
	})

	t.Run("creation of SD, StatefulSet as a controller", func(t *testing.T) {
		// Creating a service for StatefulSet
		g.client.CoreV1().Services("").Create(context.TODO(), g.statefulsetService.DeepCopy(), metav1.CreateOptions{})
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjStatefulset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjStatefulset.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-statefulSet-fail"])
		}
		// Checking the node name
		statefulset, _ := g.client.AppsV1().StatefulSets("").Get(context.TODO(), g.sdObjStatefulset.Spec.Controllers.StatefulSet[0].GetName(), metav1.GetOptions{})
		statefulsetNodeName := statefulset.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
		if statefulsetNodeName != g.nodeUSObj.GetName() {
			t.Errorf(errorDict["select-statefulset-fail"])
		}
	})

	t.Run("creation of SD, Deployment as a controller with Polygon", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDeploymentPolygon.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDeploymentPolygon.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeploymentPolygon.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-deploymentPolygon-fail"])
		}
		// Checking the node name
		deployment, _ := g.client.AppsV1().Deployments("").Get(context.TODO(), g.sdObjDeploymentPolygon.Spec.Controllers.Deployment[0].GetName(), metav1.GetOptions{})
		deploymentNodeName := deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
		if deploymentNodeName != g.nodeFRObj.GetName() {
			t.Errorf(errorDict["select-deployment-fail"])
		}
	})

	t.Run("creation of SD, Deployment already existed", func(t *testing.T) {
		// Creating deployment before creating the SD
		g.client.AppsV1().Deployments("").Create(context.TODO(), g.deploymentObj.DeepCopy(), metav1.CreateOptions{})
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDeployment.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDeployment.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-deploymentExisted-fail"])
		}
		// Checking the node name
		deployment, _ := g.client.AppsV1().Deployments("").Get(context.TODO(), g.sdObjDeployment.Spec.Controllers.Deployment[0].GetName(), metav1.GetOptions{})
		deploymentNodeName := deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
		if deploymentNodeName != g.nodeFRObj.GetName() {
			t.Errorf(errorDict["select-deployment-fail"])
		}
	})

	t.Run("creation of SD, DaemonSet already existed", func(t *testing.T) {
		// Creating a Daemonset before creating the SD
		g.client.AppsV1().DaemonSets("").Create(context.TODO(), g.daemonsetObj.DeepCopy(), metav1.CreateOptions{})
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDaemonset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDaemonset.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-daemonSetExisted-fail"])
		}
		// Checking the node name
		daemonset, _ := g.client.AppsV1().DaemonSets("").Get(context.TODO(), g.sdObjDaemonset.Spec.Controllers.DaemonSet[0].GetName(), metav1.GetOptions{})
		daemonsetNodeName := daemonset.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
		if daemonsetNodeName != g.nodeFRObj.GetName() {
			t.Errorf(errorDict["select-daemonset-fail"])
		}
	})

	t.Run("creation of SD, StatefulSet already existed", func(t *testing.T) {
		// Creating a service for StatefulSet
		g.client.CoreV1().Services("").Create(context.TODO(), g.statefulsetService.DeepCopy(), metav1.CreateOptions{})
		// Creating StatefulSet before creating SD
		g.client.AppsV1().StatefulSets("").Create(context.TODO(), g.statefulSetObj.DeepCopy(), metav1.CreateOptions{})
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjStatefulset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjStatefulset.DeepCopy())
		// Get the selectiveDeployment
		sdStatefulSet, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sdStatefulSet.Status.State != success {
			t.Errorf(errorDict["SD-statefulSetExisted-fail"])
		}
		// Checking the node name
		statefulset, _ := g.client.AppsV1().StatefulSets("").Get(context.TODO(), g.sdObjStatefulset.Spec.Controllers.StatefulSet[0].GetName(), metav1.GetOptions{})
		statefulsetNodeName := statefulset.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
		if statefulsetNodeName != g.nodeUSObj.GetName() {
			t.Errorf(errorDict["select-statefulset-fail"])
		}
	})
}

func TestObjectUpdated(t *testing.T) {
	g := SDTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creating four nodes
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeFRObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSSecondObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSThirdObj.DeepCopy(), metav1.CreateOptions{})

	t.Run("Update Image of SD, Deployment as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDeployment.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDeployment.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-deployment-fail"])
		}
		sd.Spec.Controllers.Deployment[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.0"
		// Invoke ObjectUpdated function and check the status
		g.handler.ObjectUpdated(sd.DeepCopy(), "")
		sdUpdated, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sdUpdated.Status.State != success || sdUpdated.Spec.Controllers.Deployment[0].Spec.Template.Spec.Containers[0].Image != "nginx:1.8.0" {
			t.Errorf("Update-Failed")
		}
	})

	t.Run("Update Selector of SD, Deployment as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDeployment.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDeployment.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-deployment-fail"])
		}
		// Updating Selector
		sd.Spec.Selector = []apps_v1alpha.Selector{
			{
				Value:    []string{"US"},
				Operator: "In",
				Quantity: 2,
				Name:     "Country",
			},
		}
		// Apending the second deployment object
		sd.Spec.Controllers.Deployment = append(sd.Spec.Controllers.Deployment, g.deploymentObj)
		// Invoke ObjectUpdated function and check the status
		g.handler.ObjectUpdated(sd.DeepCopy(), "")
		sdUpdated, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sdUpdated.Status.State != success {
			t.Errorf("Update-Failed")
		}
		// Checking the node name
		deployment, _ := g.client.AppsV1().Deployments("").Get(context.TODO(), g.sdObjDeployment.Spec.Controllers.Deployment[0].GetName(), metav1.GetOptions{})
		deploymentNodeNames := deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values
		usNodesNames := map[string]bool{
			g.nodeUSObj.GetName():       true,
			g.nodeUSSecondObj.GetName(): true,
			g.nodeUSThirdObj.GetName():  true,
		}
		if !usNodesNames[deploymentNodeNames[0]] || !usNodesNames[deploymentNodeNames[1]] {
			t.Errorf(errorDict["select-deployment-fail"])
		}
	})

	t.Run("Update Image of SD, DaemonSet as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDaemonset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDaemonset.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-daemonSet-fail"])
		}
		sd.Spec.Controllers.DaemonSet[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.0"
		// Invoke ObjectUpdated function and check the status
		g.handler.ObjectUpdated(sd.DeepCopy(), "")
		sdUpdated, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sdUpdated.Status.State != success || sdUpdated.Spec.Controllers.DaemonSet[0].Spec.Template.Spec.Containers[0].Image != "nginx:1.8.0" {
			t.Errorf("Update-Failed")
		}
	})

	t.Run("Update Selector of SD, DaemonSet as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDaemonset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDaemonset.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-daemonSet-fail"])
		}
		// Updating Selector
		sd.Spec.Selector = []apps_v1alpha.Selector{
			{
				Value:    []string{"US"},
				Operator: "In",
				Quantity: 2,
				Name:     "Country",
			},
		}
		// Apending the second deployment object
		sd.Spec.Controllers.DaemonSet = append(sd.Spec.Controllers.DaemonSet, g.daemonsetObj)
		// Invoke ObjectUpdated function and check the status
		g.handler.ObjectUpdated(sd.DeepCopy(), "")
		sdUpdated, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sdUpdated.Status.State != success {
			t.Errorf("Update-Failed")
		}
		// Checking the node name
		daemonset, _ := g.client.AppsV1().DaemonSets("").Get(context.TODO(), g.sdObjDaemonset.Spec.Controllers.DaemonSet[0].GetName(), metav1.GetOptions{})
		daemonsetNodeNames := daemonset.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values
		usNodesNames := map[string]bool{
			g.nodeUSObj.GetName():       true,
			g.nodeUSSecondObj.GetName(): true,
			g.nodeUSThirdObj.GetName():  true,
		}
		if !usNodesNames[daemonsetNodeNames[0]] || !usNodesNames[daemonsetNodeNames[1]] {
			t.Errorf(errorDict["select-daemonset-fail"])
		}
	})

	t.Run("Update Image of SD, StatefulSet as a controller", func(t *testing.T) {
		// Creating a service for StatefulSet
		g.client.CoreV1().Services("").Create(context.TODO(), g.statefulsetService.DeepCopy(), metav1.CreateOptions{})
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjStatefulset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjStatefulset.DeepCopy())
		// Get the selectiveDeployment
		sdStatefulSet, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sdStatefulSet.Status.State != success {
			t.Errorf(errorDict["SD-statefulSet-fail"])
		}
		sdStatefulSet.Spec.Controllers.StatefulSet[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.0"
		g.handler.ObjectUpdated(sdStatefulSet.DeepCopy(), "")
		sdUpdated, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sdUpdated.Status.State != success || sdUpdated.Spec.Controllers.StatefulSet[0].Spec.Template.Spec.Containers[0].Image != "nginx:1.8.0" {
			t.Errorf("Update-Failed")
		}
	})

	t.Run("Update Selector of SD, StatefulSet as a controller", func(t *testing.T) {
		// Creating a service for StatefulSet
		g.client.CoreV1().Services("").Create(context.TODO(), g.statefulsetService.DeepCopy(), metav1.CreateOptions{})
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjStatefulset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjStatefulset.DeepCopy())
		// Get the selectiveDeployment
		sdStatefulSet, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sdStatefulSet.Status.State != success {
			t.Errorf(errorDict["SD-statefulSet-fail"])
		}
		// Updating Selector
		sdStatefulSet.Spec.Selector = []apps_v1alpha.Selector{
			{
				Value:    []string{"US"},
				Operator: "In",
				Quantity: 2,
				Name:     "Country",
			},
		}
		// Apending the second deployment object
		sdStatefulSet.Spec.Controllers.StatefulSet = append(sdStatefulSet.Spec.Controllers.StatefulSet, g.statefulSetObj)
		// Invoke ObjectUpdated function and check the status
		g.handler.ObjectUpdated(sdStatefulSet.DeepCopy(), "")
		sdUpdated, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sdUpdated.Status.State != success {
			t.Errorf("Update-Failed")
		}
		// Checking the node name
		statefulset, _ := g.client.AppsV1().StatefulSets("").Get(context.TODO(), g.sdObjStatefulset.Spec.Controllers.StatefulSet[0].GetName(), metav1.GetOptions{})
		statefulsetNodeNames := statefulset.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values
		usNodesNames := map[string]bool{
			g.nodeUSObj.GetName():       true,
			g.nodeUSSecondObj.GetName(): true,
			g.nodeUSThirdObj.GetName():  true,
		}
		if !usNodesNames[statefulsetNodeNames[0]] || !usNodesNames[statefulsetNodeNames[1]] {
			t.Errorf(errorDict["select-statefulset-fail"])
		}
	})
}

func TestGetByNode(t *testing.T) {
	g := SDTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creating four nodes
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeFRObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSSecondObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSThirdObj.DeepCopy(), metav1.CreateOptions{})

	t.Run("Testing getByNode with Deployment", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDeployment.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDeployment.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-deployment-fail"])
		}
		ownerList, status := g.handler.getByNode(g.nodeFRObj.GetName())
		if reflect.DeepEqual(ownerList[0][0], (g.deploymentObj.GetNamespace() + g.deploymentObj.GetName())) {
			t.Errorf(errorDict["GetbyNode-fail-owner"])
		}
		if status != true {
			t.Errorf(errorDict["GetbyNode-fail"])
		}
	})

	t.Run("Testing getByNode with DaemonSet", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDaemonset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDaemonset.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf(errorDict["SD-daemonSet-fail"])
		}
		ownerList, status := g.handler.getByNode(g.nodeFRObj.GetName())
		if reflect.DeepEqual(ownerList[0][0], (g.daemonsetObj.GetNamespace() + g.daemonsetObj.GetName())) {
			t.Errorf(errorDict["GetbyNode-fail-owner"])
		}
		if status != true {
			t.Errorf(errorDict["GetbyNode-fail"])
		}
	})

	t.Run("Testing checkController func with StatefulSet as a controller", func(t *testing.T) {
		// Creating a service for StatefulSet
		g.client.CoreV1().Services("").Create(context.TODO(), g.statefulsetService.DeepCopy(), metav1.CreateOptions{})
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjStatefulset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjStatefulset.DeepCopy())
		// Get the selectiveDeployment
		sdStatefulSet, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sdStatefulSet.Status.State != success {
			t.Errorf(errorDict["SD-statefulSet-fail"])
		}
		ownerList, status := g.handler.getByNode(g.nodeUSObj.GetName())
		if reflect.DeepEqual(ownerList[0][0], (g.statefulSetObj.GetNamespace() + g.statefulSetObj.GetName())) {
			t.Errorf(errorDict["GetbyNode-fail-owner"])
		}
		if status != true {
			t.Errorf(errorDict["GetbyNode-fail"])
		}
	})

}

func TestCheckController(t *testing.T) {
	g := SDTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creating two nodes
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeFRObj.DeepCopy(), metav1.CreateOptions{})
	g.client.CoreV1().Nodes().Create(context.TODO(), g.nodeUSObj.DeepCopy(), metav1.CreateOptions{})

	t.Run("Testing checkController func with Deployment as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDeployment.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDeployment.DeepCopy())
		// Get the selectiveDeployment
		sdDeployment, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sdDeployment.Status.State != success {
			t.Errorf(errorDict["SD-deployment-fail"])
		}
		// Invoking checkController function to get the related SD obj to the controller data that we have
		sdCheck, _ := g.handler.checkController(g.sdObjDeployment.Spec.Controllers.Deployment[0].GetName(), g.sdObjDeployment.Spec.Controllers.Deployment[0].Kind, "")
		if !reflect.DeepEqual(sdCheck.GetName(), g.sdObjDeployment.GetName()) {
			t.Errorf(errorDict["checkCon-fail"])
		}
	})

	t.Run("Testing checkController func with DaemonSet as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjDaemonset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjDaemonset.DeepCopy())
		// Get the selectiveDeployment
		sdDaemonset, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sdDaemonset.Status.State != success {
			t.Errorf(errorDict["SD-daemonSet-fail"])
		}
		// Invoking checkController function to get the related SD obj to the controller data that we have
		sdCheck, _ := g.handler.checkController(g.sdObjDaemonset.Spec.Controllers.DaemonSet[0].GetName(), g.sdObjDaemonset.Spec.Controllers.DaemonSet[0].Kind, "")
		if !reflect.DeepEqual(sdCheck.GetName(), g.sdObjDaemonset.GetName()) {
			t.Errorf(errorDict["checkCon-fail"])
		}
	})

	t.Run("Testing checkController func with StatefulSet as a controller", func(t *testing.T) {
		// Creating a service for StatefulSet
		g.client.CoreV1().Services("").Create(context.TODO(), g.statefulsetService.DeepCopy(), metav1.CreateOptions{})
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(context.TODO(), g.sdObjStatefulset.DeepCopy(), metav1.CreateOptions{})
		g.handler.ObjectCreated(g.sdObjStatefulset.DeepCopy())
		// Get the selectiveDeployment
		sdStatefulSet, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(context.TODO(), g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sdStatefulSet.Status.State != success {
			t.Errorf(errorDict["SD-statefulSet-fail"])
		}
		// Invoking checkController function to get the related SD obj to the controller data that we have
		sdCheck, _ := g.handler.checkController(g.sdObjStatefulset.Spec.Controllers.StatefulSet[0].GetName(), g.sdObjStatefulset.Spec.Controllers.StatefulSet[0].Kind, "")
		if !reflect.DeepEqual(sdCheck.GetName(), g.sdObjStatefulset.GetName()) {
			t.Errorf(errorDict["checkCon-fail"])
		}
	})
}
