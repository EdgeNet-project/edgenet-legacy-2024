package selectivedeployment

import (
	apps_v1alpha "edgenet/pkg/apis/apps/v1alpha"
	"edgenet/pkg/client/clientset/versioned"
	"io/ioutil"
	"os"
	"testing"

	edgenettestclient "edgenet/pkg/client/clientset/versioned/fake"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// Dictionary for status messages
var errorDict = map[string]string{
	"k8-sync":     "Kubernetes clientset sync problem",
	"edgnet-sync": "EdgeNet clientset sync problem",
}

type SDTestGroup struct {
	client                 kubernetes.Interface
	statefulsetService     corev1.Service
	nodeFRObj              corev1.Node
	nodeUSObj              corev1.Node
	sdObjDeployment        apps_v1alpha.SelectiveDeployment
	sdObjDeploymentPolygon apps_v1alpha.SelectiveDeployment
	sdObjStatefulset       apps_v1alpha.SelectiveDeployment
	sdObjDaemonset         apps_v1alpha.SelectiveDeployment
	deploymentObj          v1.Deployment
	statefulSetObj         v1.StatefulSet
	daemonsetObj           v1.DaemonSet
	edgenetclient          versioned.Interface
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
						"[ [48.8947616, 2.2150567], [48.8084639, 2.2040704], [48.7835862, 2.3393396], [48.8416903, 2.4519494], [48.9171024,2.3932412] ]",
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
				"edge-net.io/lon":         "n2.34",
				"edge-net.io/lat":         "s48.86",
			},
		},
		Status: corev1.NodeStatus{
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
			},
		},
		Status: corev1.NodeStatus{
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
	// Creating two nodes
	g.client.CoreV1().Nodes().Create(g.nodeFRObj.DeepCopy())
	g.client.CoreV1().Nodes().Create(g.nodeUSObj.DeepCopy())

	t.Run("creation of SD, Deployment as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjDeployment.DeepCopy())
		g.handler.ObjectCreated(g.sdObjDeployment.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf("Selective deployment failed with Deployment as a controller")
		}
	})

	t.Run("creation of SD, DaemonSet as a controller", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjDaemonset.DeepCopy())
		g.handler.ObjectCreated(g.sdObjDaemonset.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf("Selective deployment failed with DaemonSet as a controller")
		}
	})

	t.Run("creation of SD, StatefulSet as a controller", func(t *testing.T) {
		// Creating a service for StatefulSet
		g.client.CoreV1().Services("").Create(g.statefulsetService.DeepCopy())
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjStatefulset.DeepCopy())
		g.handler.ObjectCreated(g.sdObjStatefulset.DeepCopy())
		// Get the selectiveDeployment
		sdStatefulSet, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sdStatefulSet.Status.State != success {
			t.Errorf("Selective deployment failed with StatefulSet as a controller")
		}
	})

	t.Run("creation of SD, Deployment as a controller with Polygon", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjDeploymentPolygon.DeepCopy())
		g.handler.ObjectCreated(g.sdObjDeploymentPolygon.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDeploymentPolygon.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf("Selective deployment failed with Deployment as a controller and using polygon")
		}
	})

	t.Run("creation of SD, Deployment already existed", func(t *testing.T) {
		// Creating deployment before creating the SD
		g.client.AppsV1().Deployments("").Create(g.deploymentObj.DeepCopy())
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjDeployment.DeepCopy())
		g.handler.ObjectCreated(g.sdObjDeployment.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf("SD failed with Deployment as controller and existed deployment")
		}
	})

	t.Run("creation of SD, DaemonSet already existed", func(t *testing.T) {
		// Creating a Daemonset before creating the SD
		g.client.AppsV1().DaemonSets("").Create(g.daemonsetObj.DeepCopy())
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjDaemonset.DeepCopy())
		g.handler.ObjectCreated(g.sdObjDaemonset.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDaemonset.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf("SD failed with DaemonSet as a controller and existed before")
		}
	})

	t.Run("creation of SD, StatefulSet already existed", func(t *testing.T) {
		// Creating a service for StatefulSet
		g.client.CoreV1().Services("").Create(g.statefulsetService.DeepCopy())
		// Creating StatefulSet before creating SD
		g.client.AppsV1().StatefulSets("").Create(g.statefulSetObj.DeepCopy())
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjStatefulset.DeepCopy())
		g.handler.ObjectCreated(g.sdObjStatefulset.DeepCopy())
		// Get the selectiveDeployment
		sdStatefulSet, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjStatefulset.GetName(), metav1.GetOptions{})
		if sdStatefulSet.Status.State != success {
			t.Errorf("SD failed with StatefulSet as a controller and existed StatefulSet")
		}
	})
}

func TestGetByNode(t *testing.T) {
	g := SDTestGroup{}
	g.Init()
	g.handler.Init(g.client, g.edgenetclient)
	// Creating two nodes
	g.client.CoreV1().Nodes().Create(g.nodeFRObj.DeepCopy())
	g.client.CoreV1().Nodes().Create(g.nodeUSObj.DeepCopy())

	t.Run("Testing getByNode with Deployment", func(t *testing.T) {
		// Invoking the Create function of SD
		g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Create(g.sdObjDeployment.DeepCopy())
		g.handler.ObjectCreated(g.sdObjDeployment.DeepCopy())
		// Get the selectiveDeployment
		sd, _ := g.edgenetclient.AppsV1alpha().SelectiveDeployments("").Get(g.sdObjDeployment.GetName(), metav1.GetOptions{})
		if sd.Status.State != success {
			t.Errorf("Selective deployment failed with Deployment as a controller")
		}
		_, status := g.handler.getByNode(g.nodeFRObj.GetName())
		if status != true {
			t.Errorf("GetbyNode Failed")
		}
	})

}