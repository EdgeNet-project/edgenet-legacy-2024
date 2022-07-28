package selectivedeployment

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	appsv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions"
	"github.com/EdgeNet-project/edgenet/pkg/signals"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

type TestGroup struct {
	sdObj          appsv1alpha1.SelectiveDeployment
	selector       appsv1alpha1.Selector
	deploymentObj  appsv1.Deployment
	daemonsetObj   appsv1.DaemonSet
	statefulsetObj appsv1.StatefulSet
	jobObj         batchv1.Job
	cronjobObj     batchv1beta.CronJob
	nodeObj        corev1.Node
}

var controller *Controller
var kubeclientset kubernetes.Interface = testclient.NewSimpleClientset()
var edgenetclientset versioned.Interface = edgenettestclient.NewSimpleClientset()

func TestMain(m *testing.M) {
	klog.SetOutput(ioutil.Discard)
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)

	stopCh := signals.SetupSignalHandler()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)
	edgenetInformerFactory := informers.NewSharedInformerFactory(edgenetclientset, time.Second*30)

	newController := NewController(kubeclientset,
		edgenetclientset,
		kubeInformerFactory.Core().V1().Nodes(),
		kubeInformerFactory.Apps().V1().Deployments(),
		kubeInformerFactory.Apps().V1().DaemonSets(),
		kubeInformerFactory.Apps().V1().StatefulSets(),
		kubeInformerFactory.Batch().V1().Jobs(),
		kubeInformerFactory.Batch().V1beta1().CronJobs(),
		edgenetInformerFactory.Apps().V1alpha1().SelectiveDeployments())

	kubeInformerFactory.Start(stopCh)
	edgenetInformerFactory.Start(stopCh)
	controller = newController
	go func() {
		if err := controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}()

	time.Sleep(500 * time.Millisecond)

	os.Exit(m.Run())
	<-stopCh
}

// Init syncs the test group
func (g *TestGroup) Init() {
	nodeRaw, _ := kubeclientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	for _, nodeRow := range nodeRaw.Items {
		kubeclientset.CoreV1().Nodes().Delete(context.TODO(), nodeRow.GetName(), metav1.DeleteOptions{})
	}
	selectiveDeploymentRaw, _ := edgenetclientset.AppsV1alpha1().SelectiveDeployments("").List(context.TODO(), metav1.ListOptions{})
	for _, selectiveDeploymentRow := range selectiveDeploymentRaw.Items {
		edgenetclientset.AppsV1alpha1().SelectiveDeployments(selectiveDeploymentRow.GetNamespace()).Delete(context.TODO(), selectiveDeploymentRow.GetName(), metav1.DeleteOptions{})
	}
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
						{
							Name:  "nginx",
							Image: "nginx:1.7.9",
							Ports: []corev1.ContainerPort{
								{
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
						{
							Name:  "nginx",
							Image: "nginx:1.7.9",
							Ports: []corev1.ContainerPort{
								{
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
						{
							Name:  "nginx",
							Image: "nginx:1.7.9",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	jobObj := batchv1.Job{
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
		Spec: batchv1.JobSpec{
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
						{
							Name:  "nginx",
							Image: "nginx:1.7.9",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	cronjobObj := batchv1beta.CronJob{
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
		Spec: batchv1beta.CronJobSpec{
			Schedule: "*/1 * * * *",
			JobTemplate: batchv1beta.JobTemplateSpec{
				Spec: batchv1.JobSpec{
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
								{
									Name:  "nginx",
									Image: "nginx:1.7.9",
									Ports: []corev1.ContainerPort{
										{
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
	}
	selectorObj := appsv1alpha1.Selector{
		Value:    []string{"Paris"},
		Operator: "In",
		Name:     "city",
	}
	sdObj := appsv1alpha1.SelectiveDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelectiveDeployment",
			APIVersion: "apps.edgenet.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			UID:  "sd",
		},
		Spec: appsv1alpha1.SelectiveDeploymentSpec{
			Recovery: false,
			Workloads: appsv1alpha1.Workloads{
				Deployment: []appsv1.Deployment{
					deploymentObj,
				},
				DaemonSet: []appsv1.DaemonSet{
					daemonsetObj,
				},
				StatefulSet: []appsv1.StatefulSet{
					statefulsetObj,
				},
				Job: []batchv1.Job{
					jobObj,
				},
				CronJob: []batchv1beta.CronJob{
					cronjobObj,
				},
			},
			Selector: []appsv1alpha1.Selector{
				selectorObj,
			},
		},
	}
	nodeObj := corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "apps.edgenet.io/v1alpha1",
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
				{
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
	g.jobObj = jobObj
	g.cronjobObj = cronjobObj
	g.selector = selectorObj
	g.sdObj = sdObj

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}
	kubeclientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
}

func TestStartController(t *testing.T) {
	g := TestGroup{}
	g.Init()
	// Creating nodes
	nodeParis := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeParis.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 250)
	kubeclientset.AppsV1().Deployments("default").Create(context.TODO(), g.deploymentObj.DeepCopy(), metav1.CreateOptions{})
	kubeclientset.AppsV1().DaemonSets("default").Create(context.TODO(), g.daemonsetObj.DeepCopy(), metav1.CreateOptions{})
	kubeclientset.AppsV1().StatefulSets("default").Create(context.TODO(), g.statefulsetObj.DeepCopy(), metav1.CreateOptions{})
	kubeclientset.BatchV1().Jobs("default").Create(context.TODO(), g.jobObj.DeepCopy(), metav1.CreateOptions{})
	kubeclientset.BatchV1beta1().CronJobs("default").Create(context.TODO(), g.cronjobObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 250)
	// Invoking the create function
	sdObj := g.sdObj.DeepCopy()
	uid := types.UID(uuid.New().String())
	sdObj.SetUID(uid)
	edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Create(context.TODO(), sdObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err := edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, "5/5", sdCopy.Status.Ready)

	useu := g.selector
	useu.Value = []string{"US", "FR"}
	useu.Quantity = 2
	useu.Name = "Country"
	countryUSEU := []appsv1alpha1.Selector{useu}
	sdCopy.Spec.Selector = countryUSEU
	sdCopy.SetResourceVersion(util.GenerateRandomString(6))
	edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Update(context.TODO(), sdCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, failure, sdCopy.Status.State)
	util.Equals(t, "0/5", sdCopy.Status.Ready)

	nodeRichardson := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeRichardson.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, failure, sdCopy.Status.State)
	util.Equals(t, "0/5", sdCopy.Status.Ready)

	sdCopy.Spec.Recovery = true
	sdCopy.SetResourceVersion(util.GenerateRandomString(6))
	_, err = edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Update(context.TODO(), sdCopy, metav1.UpdateOptions{})
	util.OK(t, err)
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, "5/5", sdCopy.Status.Ready)

	nodeSeaside := g.nodeObj.DeepCopy()
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
	nodeSeaside.Status.Conditions[0].Type = "NotReady"
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeSeaside.DeepCopy(), metav1.CreateOptions{})

	useu = g.selector
	useu.Value = []string{"US", "FR"}
	useu.Quantity = 3
	useu.Name = "Country"
	countryUSEU = []appsv1alpha1.Selector{useu}
	sdCopy.Spec.Selector = countryUSEU
	sdCopy.SetResourceVersion(util.GenerateRandomString(6))
	edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Update(context.TODO(), sdCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, failure, sdCopy.Status.State)
	util.Equals(t, "0/5", sdCopy.Status.Ready)
	nodeSeaside.Status.Conditions[0].Type = "Ready"
	nodeSeaside.ResourceVersion = "1"
	kubeclientset.CoreV1().Nodes().Update(context.TODO(), nodeSeaside.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, "5/5", sdCopy.Status.Ready)

	nodeCopy, _ := kubeclientset.CoreV1().Nodes().Get(context.TODO(), nodeSeaside.GetName(), metav1.GetOptions{})
	nodeCopy.Status.Conditions[0].Type = "NotReady"
	nodeCopy.ResourceVersion = "2"
	kubeclientset.CoreV1().Nodes().Update(context.TODO(), nodeCopy.DeepCopy(), metav1.UpdateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, failure, sdCopy.Status.State)
	util.Equals(t, "0/5", sdCopy.Status.Ready)

	nodeCollegePark := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeCollegePark.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err = edgenetclientset.AppsV1alpha1().SelectiveDeployments("default").Get(context.TODO(), sdCopy.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, "5/5", sdCopy.Status.Ready)
}

func TestCreate(t *testing.T) {
	g := TestGroup{}
	g.Init()

	// Creating nodes
	nodeParis := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeParis.DeepCopy(), metav1.CreateOptions{})
	nodeRichardson := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeRichardson.DeepCopy(), metav1.CreateOptions{})

	sdObj := g.sdObj.DeepCopy()
	sdObj.SetName("create")
	sdRepeatedObj := g.sdObj.DeepCopy()
	sdRepeatedObj.SetName("repeated")
	sdRepeatedObj.SetUID("repeated")
	sdPartiallyRepeatedObj := g.sdObj.DeepCopy()
	sdPartiallyRepeatedObj.SetName("partial")
	sdPartiallyRepeatedObj.SetUID("partial")
	deploymentPartial := g.deploymentObj
	deploymentPartial.SetName("partial")
	kubeclientset.AppsV1().Deployments("create").Create(context.TODO(), deploymentPartial.DeepCopy(), metav1.CreateOptions{})
	sdPartiallyRepeatedObj.Spec.Workloads.Deployment = append(sdObj.Spec.Workloads.Deployment, deploymentPartial)
	// Deployment, DaemonSet, and StatefulSet created already before the creation of Selective Deployment
	deploymentIrrelevant := g.deploymentObj
	deploymentIrrelevant.SetName("irrelevant")
	kubeclientset.AppsV1().Deployments("create").Create(context.TODO(), deploymentIrrelevant.DeepCopy(), metav1.CreateOptions{})
	daemonsetIrrelevant := g.daemonsetObj
	daemonsetIrrelevant.SetName("irrelevant")
	kubeclientset.AppsV1().DaemonSets("create").Create(context.TODO(), daemonsetIrrelevant.DeepCopy(), metav1.CreateOptions{})
	statefulsetIrrelevant := g.statefulsetObj
	statefulsetIrrelevant.SetName("irrelevant")
	kubeclientset.AppsV1().StatefulSets("create").Create(context.TODO(), statefulsetIrrelevant.DeepCopy(), metav1.CreateOptions{})
	jobIrrelevant := g.jobObj
	jobIrrelevant.SetName("irrelevant")
	kubeclientset.BatchV1().Jobs("create").Create(context.TODO(), jobIrrelevant.DeepCopy(), metav1.CreateOptions{})
	cronjobIrrelevant := g.cronjobObj
	cronjobIrrelevant.SetName("irrelevant")
	kubeclientset.BatchV1beta1().CronJobs("create").Create(context.TODO(), cronjobIrrelevant.DeepCopy(), metav1.CreateOptions{})

	deploymentCreated := g.deploymentObj
	deploymentCreated.SetName("created")
	kubeclientset.AppsV1().Deployments("create").Create(context.TODO(), deploymentCreated.DeepCopy(), metav1.CreateOptions{})
	sdObj.Spec.Workloads.Deployment = append(sdObj.Spec.Workloads.Deployment, deploymentCreated)
	daemonsetCreated := g.daemonsetObj
	daemonsetCreated.SetName("created")
	kubeclientset.AppsV1().DaemonSets("create").Create(context.TODO(), daemonsetCreated.DeepCopy(), metav1.CreateOptions{})
	sdObj.Spec.Workloads.DaemonSet = append(sdObj.Spec.Workloads.DaemonSet, daemonsetCreated)
	statefulsetCreated := g.statefulsetObj
	statefulsetCreated.SetName("created")
	kubeclientset.AppsV1().StatefulSets("create").Create(context.TODO(), statefulsetCreated.DeepCopy(), metav1.CreateOptions{})
	sdObj.Spec.Workloads.StatefulSet = append(sdObj.Spec.Workloads.StatefulSet, statefulsetCreated)
	jobCreated := g.jobObj
	jobCreated.SetName("created")
	kubeclientset.BatchV1().Jobs("create").Create(context.TODO(), jobCreated.DeepCopy(), metav1.CreateOptions{})
	sdObj.Spec.Workloads.Job = append(sdObj.Spec.Workloads.Job, jobCreated)
	cronjobCreated := g.cronjobObj
	cronjobCreated.SetName("created")
	kubeclientset.BatchV1beta1().CronJobs("create").Create(context.TODO(), cronjobCreated.DeepCopy(), metav1.CreateOptions{})
	sdObj.Spec.Workloads.CronJob = append(sdObj.Spec.Workloads.CronJob, cronjobCreated)

	// Invoke the create function
	_, err := edgenetclientset.AppsV1alpha1().SelectiveDeployments("create").Create(context.TODO(), sdObj.DeepCopy(), metav1.CreateOptions{})
	util.OK(t, err)
	time.Sleep(time.Millisecond * 500)
	for _, workload := range sdObj.Spec.Workloads.Deployment {
		log.Println(workload)
	}
	sdCopy, err := edgenetclientset.AppsV1alpha1().SelectiveDeployments("create").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
	t.Run("status", func(t *testing.T) {
		util.OK(t, err)
		util.Equals(t, success, sdCopy.Status.State)
		util.Equals(t, messageWorkloadCreated, sdCopy.Status.Message)
		util.Equals(t, "10/10", sdCopy.Status.Ready)
	})
	edgenetclientset.AppsV1alpha1().SelectiveDeployments("create").Create(context.TODO(), sdRepeatedObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdRepeatedCopy, err := edgenetclientset.AppsV1alpha1().SelectiveDeployments("create").Get(context.TODO(), sdRepeatedObj.GetName(), metav1.GetOptions{})
	t.Run("status of failure", func(t *testing.T) {
		util.OK(t, err)
		util.Equals(t, failure, sdRepeatedCopy.Status.State)
		util.Equals(t, "0/5", sdRepeatedCopy.Status.Ready)
	})
	edgenetclientset.AppsV1alpha1().SelectiveDeployments("create").Create(context.TODO(), sdPartiallyRepeatedObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdPartialCopy, err := edgenetclientset.AppsV1alpha1().SelectiveDeployments("create").Get(context.TODO(), sdPartiallyRepeatedObj.GetName(), metav1.GetOptions{})
	t.Run("status of failure", func(t *testing.T) {
		util.OK(t, err)
		util.Equals(t, partial, sdPartialCopy.Status.State)
		util.Equals(t, "1/6", sdPartialCopy.Status.Ready)
	})
	cases := map[string]struct {
		kind     string
		name     string
		expected string
	}{
		"configure/deployment":   {"Deployment", deploymentCreated.GetName(), nodeParis.GetName()},
		"create/deployment":      {"Deployment", g.sdObj.Spec.Workloads.Deployment[0].GetName(), nodeParis.GetName()},
		"configure/daemonset":    {"DaemonSet", daemonsetCreated.GetName(), nodeParis.GetName()},
		"create/daemonset":       {"DaemonSet", g.sdObj.Spec.Workloads.DaemonSet[0].GetName(), nodeParis.GetName()},
		"configure/statefulset":  {"StatefulSet", statefulsetCreated.GetName(), nodeParis.GetName()},
		"create/statefulset":     {"StatefulSet", g.sdObj.Spec.Workloads.StatefulSet[0].GetName(), nodeParis.GetName()},
		"configure/job":          {"Job", jobCreated.GetName(), nodeParis.GetName()},
		"create/job":             {"Job", g.sdObj.Spec.Workloads.Job[0].GetName(), nodeParis.GetName()},
		"configure/cronjob":      {"CronJob", cronjobCreated.GetName(), nodeParis.GetName()},
		"create/cronjob":         {"CronJob", g.sdObj.Spec.Workloads.CronJob[0].GetName(), nodeParis.GetName()},
		"irrelevant/deployment":  {"Deployment", deploymentIrrelevant.GetName(), ""},
		"irrelevant/daemonset":   {"DaemonSet", daemonsetIrrelevant.GetName(), ""},
		"irrelevant/statefulset": {"StatefulSet", statefulsetIrrelevant.GetName(), ""},
		"irrelevant/job":         {"Job", jobIrrelevant.GetName(), ""},
		"irrelevant/cronjob":     {"CronJob", cronjobIrrelevant.GetName(), ""},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			var affinityValue string
			if tc.kind == "Deployment" {
				deploymentCopy, err := kubeclientset.AppsV1().Deployments("create").Get(context.TODO(), tc.name, metav1.GetOptions{})
				util.OK(t, err)
				if deploymentCopy.Spec.Template.Spec.Affinity != nil && len(deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
					util.Equals(t, 1, len(deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values))
					affinityValue = deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
				}
			} else if tc.kind == "DaemonSet" {
				daemonsetCopy, err := kubeclientset.AppsV1().DaemonSets("create").Get(context.TODO(), tc.name, metav1.GetOptions{})
				util.OK(t, err)
				if daemonsetCopy.Spec.Template.Spec.Affinity != nil && len(daemonsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
					util.Equals(t, 1, len(daemonsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values))
					affinityValue = daemonsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
				}
			} else if tc.kind == "StatefulSet" {
				statefulsetCopy, err := kubeclientset.AppsV1().StatefulSets("create").Get(context.TODO(), tc.name, metav1.GetOptions{})
				util.OK(t, err)
				if statefulsetCopy.Spec.Template.Spec.Affinity != nil && len(statefulsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
					util.Equals(t, 1, len(statefulsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values))
					affinityValue = statefulsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
				}
			} else if tc.kind == "Job" {
				jobCopy, err := kubeclientset.BatchV1().Jobs("create").Get(context.TODO(), tc.name, metav1.GetOptions{})
				util.OK(t, err)
				if jobCopy.Spec.Template.Spec.Affinity != nil && len(jobCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
					util.Equals(t, 1, len(jobCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values))
					affinityValue = jobCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
				}
			} else if tc.kind == "CronJob" {
				jobCopy, err := kubeclientset.BatchV1beta1().CronJobs("create").Get(context.TODO(), tc.name, metav1.GetOptions{})
				util.OK(t, err)
				if jobCopy.Spec.JobTemplate.Spec.Template.Spec.Affinity != nil && len(jobCopy.Spec.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
					util.Equals(t, 1, len(jobCopy.Spec.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values))
					affinityValue = jobCopy.Spec.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
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

	// Creating nodes
	nodeParis := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeParis.DeepCopy(), metav1.CreateOptions{})
	nodeCollegePark := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeCollegePark.DeepCopy(), metav1.CreateOptions{})
	nodeSeaside := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeSeaside.DeepCopy(), metav1.CreateOptions{})

	sdObj := g.sdObj.DeepCopy()
	sdObj.SetName("update")
	edgenetclientset.AppsV1alpha1().SelectiveDeployments("update").Create(context.TODO(), sdObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err := edgenetclientset.AppsV1alpha1().SelectiveDeployments("update").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, messageWorkloadCreated, sdCopy.Status.Message)
	util.Equals(t, "5/5", sdCopy.Status.Ready)

	deploymentCopy, err := kubeclientset.AppsV1().Deployments("update").Get(context.TODO(), sdObj.Spec.Workloads.Deployment[0].GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t,
		nodeParis.GetName(),
		deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
	daemonsetCopy, err := kubeclientset.AppsV1().DaemonSets("update").Get(context.TODO(), sdObj.Spec.Workloads.DaemonSet[0].GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t,
		nodeParis.GetName(),
		daemonsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
	statefulsetCopy, err := kubeclientset.AppsV1().StatefulSets("update").Get(context.TODO(), sdObj.Spec.Workloads.StatefulSet[0].GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t,
		nodeParis.GetName(),
		statefulsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
	jobCopy, err := kubeclientset.BatchV1().Jobs("update").Get(context.TODO(), sdObj.Spec.Workloads.Job[0].GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t,
		nodeParis.GetName(),
		jobCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
	cronjobCopy, err := kubeclientset.BatchV1beta1().CronJobs("update").Get(context.TODO(), sdObj.Spec.Workloads.CronJob[0].GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t,
		nodeParis.GetName(),
		cronjobCopy.Spec.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0])

	seaside := g.selector
	seaside.Value = []string{"Seaside"}
	seaside.Quantity = 1
	seaside.Name = "City"
	citySeaside := []appsv1alpha1.Selector{seaside}

	ca := g.selector
	ca.Value = []string{"CA"}
	ca.Quantity = 1
	ca.Name = "State"
	stateCA := []appsv1alpha1.Selector{ca}

	us := g.selector
	us.Value = []string{"US"}
	us.Name = "Country"
	countryUSAll := []appsv1alpha1.Selector{us}
	us.Operator = "NotIn"
	countryUSOut := []appsv1alpha1.Selector{us}
	us.Operator = "In"
	us.Quantity = 1
	fr := g.selector
	fr.Value = []string{"FR"}
	fr.Quantity = 1
	fr.Name = "Country"
	countryUSEU := []appsv1alpha1.Selector{us, fr}

	eu := g.selector
	eu.Value = []string{"Europe"}
	eu.Quantity = 1
	eu.Name = "Continent"
	continentEU := []appsv1alpha1.Selector{eu}

	paris := g.selector
	paris.Value = []string{"[ [2.2150567, 48.8947616], [2.2040704, 48.8084639], [2.3393396, 48.7835862], [2.4519494, 48.8416903], [2.3932412, 48.9171024] ]"}
	paris.Quantity = 1
	paris.Name = "Polygon"
	polygonParis := []appsv1alpha1.Selector{paris}

	countryUScityParis := []appsv1alpha1.Selector{us, paris}

	paris.Quantity = 4
	polygonParisFewer := []appsv1alpha1.Selector{paris}
	us.Quantity = 3
	countryUSFewer := []appsv1alpha1.Selector{us}

	cases := map[string]struct {
		input          []appsv1alpha1.Selector
		expectedStatus string
		expected       [][]string
	}{
		"city/seaside":          {citySeaside, success, [][]string{{nodeSeaside.GetName()}}},
		"polygon/paris":         {polygonParis, success, [][]string{{nodeParis.GetName()}}},
		"state/ca":              {stateCA, success, [][]string{{nodeSeaside.GetName()}}},
		"country/us/all":        {countryUSAll, success, [][]string{{nodeSeaside.GetName()}}},
		"country/us/out":        {countryUSOut, success, [][]string{{nodeParis.GetName()}}},
		"continent/europe":      {continentEU, success, [][]string{{nodeParis.GetName()}}},
		"country/us-eu":         {countryUSEU, success, [][]string{{nodeSeaside.GetName()}, {nodeParis.GetName()}}},
		"country/us|city/paris": {countryUScityParis, success, [][]string{{nodeSeaside.GetName()}, {nodeParis.GetName()}}},
		"polygon/paris/fewer":   {polygonParisFewer, failure, [][]string{{nodeParis.GetName()}}},
		"country/us/fewer":      {countryUSFewer, failure, [][]string{{nodeSeaside.GetName()}}},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			sdCopy, _ := edgenetclientset.AppsV1alpha1().SelectiveDeployments("update").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
			sdCopy.Spec.Selector = tc.input
			sdCopy.SetResourceVersion(util.GenerateRandomString(6))
			edgenetclientset.AppsV1alpha1().SelectiveDeployments("update").Update(context.TODO(), sdCopy, metav1.UpdateOptions{})
			time.Sleep(time.Millisecond * 500)
			sdCopy, _ = edgenetclientset.AppsV1alpha1().SelectiveDeployments("update").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
			util.Equals(t, tc.expectedStatus, sdCopy.Status.State)
			deploymentCopy, err := kubeclientset.AppsV1().Deployments("update").Get(context.TODO(), deploymentCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			for i, expected := range tc.expected {
				util.Equals(t,
					expected,
					deploymentCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[i].MatchExpressions[0].Values)
			}
			daemonsetCopy, err := kubeclientset.AppsV1().DaemonSets("update").Get(context.TODO(), daemonsetCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			for j, expected := range tc.expected {
				util.Equals(t,
					expected,
					daemonsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[j].MatchExpressions[0].Values)
			}
			statefulsetCopy, err := kubeclientset.AppsV1().StatefulSets("update").Get(context.TODO(), statefulsetCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			for k, expected := range tc.expected {
				util.Equals(t,
					expected,
					statefulsetCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[k].MatchExpressions[0].Values)
			}
			jobCopy, err := kubeclientset.BatchV1().Jobs("update").Get(context.TODO(), jobCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			for l, expected := range tc.expected {
				util.Equals(t,
					expected,
					jobCopy.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[l].MatchExpressions[0].Values)
			}
			cronjobCopy, err := kubeclientset.BatchV1beta1().CronJobs("update").Get(context.TODO(), cronjobCopy.GetName(), metav1.GetOptions{})
			util.OK(t, err)
			for m, expected := range tc.expected {
				util.Equals(t,
					expected,
					cronjobCopy.Spec.JobTemplate.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[m].MatchExpressions[0].Values)
			}
		})
	}

	t.Run("workload spec", func(t *testing.T) {
		util.Equals(t, sdCopy.Spec.Workloads.Deployment[0].Spec.Template.Spec.Containers[0].Image, deploymentCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, sdCopy.Spec.Workloads.DaemonSet[0].Spec.Template.Spec.Containers[0].Image, daemonsetCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, sdCopy.Spec.Workloads.StatefulSet[0].Spec.Template.Spec.Containers[0].Image, statefulsetCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, sdCopy.Spec.Workloads.Job[0].Spec.Template.Spec.Containers[0].Image, jobCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, sdCopy.Spec.Workloads.CronJob[0].Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image, cronjobCopy.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

		sdCopy.Spec.Workloads.Deployment[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.0"
		sdCopy.Spec.Workloads.DaemonSet[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.1"
		sdCopy.Spec.Workloads.StatefulSet[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.2"
		sdCopy.Spec.Workloads.Job[0].Spec.Template.Spec.Containers[0].Image = "nginx:1.8.3"
		sdCopy.Spec.Workloads.CronJob[0].Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image = "nginx:1.8.4"

		sdCopy.SetResourceVersion(util.GenerateRandomString(6))
		edgenetclientset.AppsV1alpha1().SelectiveDeployments("update").Update(context.TODO(), sdCopy, metav1.UpdateOptions{})
		time.Sleep(time.Millisecond * 500)
		deploymentCopy, err := kubeclientset.AppsV1().Deployments("update").Get(context.TODO(), deploymentCopy.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		daemonsetCopy, err := kubeclientset.AppsV1().DaemonSets("update").Get(context.TODO(), daemonsetCopy.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		statefulsetCopy, err := kubeclientset.AppsV1().StatefulSets("update").Get(context.TODO(), statefulsetCopy.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		jobCopy, err := kubeclientset.BatchV1().Jobs("update").Get(context.TODO(), jobCopy.GetName(), metav1.GetOptions{})
		util.OK(t, err)
		cronjobCopy, err := kubeclientset.BatchV1beta1().CronJobs("update").Get(context.TODO(), cronjobCopy.GetName(), metav1.GetOptions{})
		util.OK(t, err)

		util.Equals(t, "nginx:1.8.0", deploymentCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, "nginx:1.8.1", daemonsetCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, "nginx:1.8.2", statefulsetCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, "nginx:1.8.3", jobCopy.Spec.Template.Spec.Containers[0].Image)
		util.Equals(t, "nginx:1.8.4", cronjobCopy.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	})
}

func TestGetByNode(t *testing.T) {
	g := TestGroup{}
	g.Init()

	deploymentRaw, _ := kubeclientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	for _, deploymentRow := range deploymentRaw.Items {
		kubeclientset.AppsV1().Deployments(deploymentRow.GetNamespace()).Delete(context.TODO(), deploymentRow.GetName(), metav1.DeleteOptions{})
	}
	daemonsetRaw, _ := kubeclientset.AppsV1().DaemonSets("").List(context.TODO(), metav1.ListOptions{})
	for _, daemonsetRow := range daemonsetRaw.Items {
		kubeclientset.AppsV1().DaemonSets(daemonsetRow.GetNamespace()).Delete(context.TODO(), daemonsetRow.GetName(), metav1.DeleteOptions{})
	}
	statefulsetRaw, _ := kubeclientset.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	for _, statefulsetRow := range statefulsetRaw.Items {
		kubeclientset.AppsV1().StatefulSets(statefulsetRow.GetNamespace()).Delete(context.TODO(), statefulsetRow.GetName(), metav1.DeleteOptions{})
	}
	jobRaw, _ := kubeclientset.BatchV1().Jobs("").List(context.TODO(), metav1.ListOptions{})
	for _, jobRow := range jobRaw.Items {
		kubeclientset.BatchV1().Jobs(jobRow.GetNamespace()).Delete(context.TODO(), jobRow.GetName(), metav1.DeleteOptions{})
	}
	cronjobRaw, _ := kubeclientset.BatchV1beta1().CronJobs("").List(context.TODO(), metav1.ListOptions{})
	for _, cronjobRow := range cronjobRaw.Items {
		kubeclientset.BatchV1beta1().CronJobs(cronjobRow.GetNamespace()).Delete(context.TODO(), cronjobRow.GetName(), metav1.DeleteOptions{})
	}

	// Creating nodes
	nodeParis := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeParis.DeepCopy(), metav1.CreateOptions{})
	nodeRichardson := g.nodeObj.DeepCopy()
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
	kubeclientset.CoreV1().Nodes().Create(context.TODO(), nodeRichardson.DeepCopy(), metav1.CreateOptions{})

	// Invoke the create function
	sdObj := g.sdObj.DeepCopy()
	sdObj.SetName("getbynode")
	useu := g.selector
	useu.Value = []string{"US", "FR"}
	useu.Quantity = 2
	useu.Name = "Country"
	sdObj.Spec.Selector = []appsv1alpha1.Selector{useu}

	edgenetclientset.AppsV1alpha1().SelectiveDeployments("getbynode").Create(context.TODO(), sdObj.DeepCopy(), metav1.CreateOptions{})
	time.Sleep(time.Millisecond * 500)
	sdCopy, err := edgenetclientset.AppsV1alpha1().SelectiveDeployments("getbynode").Get(context.TODO(), sdObj.GetName(), metav1.GetOptions{})
	util.OK(t, err)
	util.Equals(t, success, sdCopy.Status.State)
	util.Equals(t, messageWorkloadCreated, sdCopy.Status.Message)
	util.Equals(t, "5/5", sdCopy.Status.Ready)

	ownerList, status := controller.getByNode(nodeParis.GetName())
	util.Equals(t, true, status)
	util.Equals(t, "getbynode", ownerList[0][0])
	util.Equals(t, sdObj.GetName(), ownerList[0][1])

	ownerList, status = controller.getByNode(nodeRichardson.GetName())
	util.Equals(t, true, status)
	util.Equals(t, "getbynode", ownerList[0][0])
	util.Equals(t, sdObj.GetName(), ownerList[0][1])
}
