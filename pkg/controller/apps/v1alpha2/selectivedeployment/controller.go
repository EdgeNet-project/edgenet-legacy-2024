/*
Copyright 2022 Contributors to the EdgeNet project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package selectivedeployment

import (
	"context"
	"fmt"
	"time"

	appsv1alpha2 "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha2"
	federationv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/federation/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/bootstrap"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/apps/v1alpha2"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/apps/v1alpha2"
	"github.com/EdgeNet-project/edgenet/pkg/multiprovider"
	"github.com/EdgeNet-project/edgenet/pkg/multitenancy"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	batchinformers "k8s.io/client-go/informers/batch/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "selectivedeployment-controller"

// Definitions of the state of the selectivedeployment resource
const (
	successSynced = "Synced"

	messageResourceSynced = "Selective deployment synced successfully"

	messageServiceAccountFailed = "Service account creation failed"
	messageAuthSecretFailed     = "Secret storing selective deployment's token cannot be created"
	messageBindingFailed        = "Role binding failed"
)

// Controller is the controller implementation for Selective Deployment resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	nodesLister corelisters.NodeLister
	nodesSynced cache.InformerSynced

	deploymentsLister  appslisters.DeploymentLister
	deploymentsSynced  cache.InformerSynced
	daemonsetsLister   appslisters.DaemonSetLister
	daemonsetsSynced   cache.InformerSynced
	statefulsetsLister appslisters.StatefulSetLister
	statefulsetsSynced cache.InformerSynced
	jobsLister         batchlisters.JobLister
	jobsSynced         cache.InformerSynced
	cronjobsLister     batchlisters.CronJobLister
	cronjobsSynced     cache.InformerSynced

	selectivedeploymentsLister listers.SelectiveDeploymentLister
	selectivedeploymentsSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	nodeInformer coreinformers.NodeInformer,
	deploymentInformer appsinformers.DeploymentInformer,
	daemonsetInformer appsinformers.DaemonSetInformer,
	statefulsetInformer appsinformers.StatefulSetInformer,
	jobInformer batchinformers.JobInformer,
	cronjobInformer batchinformers.CronJobInformer,
	selectivedeploymentInformer informers.SelectiveDeploymentInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:              kubeclientset,
		edgenetclientset:           edgenetclientset,
		nodesLister:                nodeInformer.Lister(),
		nodesSynced:                nodeInformer.Informer().HasSynced,
		deploymentsLister:          deploymentInformer.Lister(),
		deploymentsSynced:          deploymentInformer.Informer().HasSynced,
		daemonsetsLister:           daemonsetInformer.Lister(),
		daemonsetsSynced:           daemonsetInformer.Informer().HasSynced,
		statefulsetsLister:         statefulsetInformer.Lister(),
		statefulsetsSynced:         statefulsetInformer.Informer().HasSynced,
		jobsLister:                 jobInformer.Lister(),
		jobsSynced:                 jobInformer.Informer().HasSynced,
		cronjobsLister:             cronjobInformer.Lister(),
		cronjobsSynced:             cronjobInformer.Informer().HasSynced,
		selectivedeploymentsLister: selectivedeploymentInformer.Lister(),
		selectivedeploymentsSynced: selectivedeploymentInformer.Informer().HasSynced,
		workqueue:                  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "SelectiveDeployments"),
		recorder:                   recorder,
	}

	klog.Infoln("Setting up event handlers")
	// Set up an event handler for when Selective Deployment resources change
	selectivedeploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueSelectiveDeployment,
		UpdateFunc: func(old, new interface{}) {
			newSelectiveDeployment := new.(*appsv1alpha2.SelectiveDeployment)
			oldSelectiveDeployment := old.(*appsv1alpha2.SelectiveDeployment)
			if newSelectiveDeployment.ResourceVersion == oldSelectiveDeployment.ResourceVersion {
				return
			}
			controller.enqueueSelectiveDeployment(new)
		},
	})

	/*nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.recoverSelectiveDeployments,
		UpdateFunc: func(old, new interface{}) {
			newNode := new.(*corev1.Node)
			oldNode := old.(*corev1.Node)
			if newNode.ResourceVersion == oldNode.ResourceVersion {
				return
			}
			controller.recoverSelectiveDeployments(new)
		},
		DeleteFunc: controller.recoverSelectiveDeployments,
	})*/

	/*deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDeployment := new.(*appsv1.Deployment)
			oldDeployment := old.(*appsv1.Deployment)
			if newDeployment.ResourceVersion == oldDeployment.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	daemonsetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDaemonSet := new.(*appsv1.DaemonSet)
			oldDaemonSet := old.(*appsv1.DaemonSet)
			if newDaemonSet.ResourceVersion == oldDaemonSet.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	statefulsetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newStatefulSet := new.(*appsv1.StatefulSet)
			oldStatefulSet := old.(*appsv1.StatefulSet)
			if newStatefulSet.ResourceVersion == oldStatefulSet.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newJob := new.(*batchv1.Job)
			oldJob := old.(*batchv1.Job)
			if newJob.ResourceVersion == oldJob.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	cronjobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newCronJob := new.(*batchv1beta1.CronJob)
			oldCronJob := old.(*batchv1beta1.CronJob)
			if newCronJob.ResourceVersion == oldCronJob.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})*/

	return controller
}

// Run will set up the event handlers for the types of selective deployment and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting Selective Deployment controller")

	klog.Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.selectivedeploymentsSynced,
		c.nodesSynced,
		c.deploymentsSynced,
		c.daemonsetsSynced,
		c.statefulsetsSynced,
		c.jobsSynced,
		c.cronjobsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Infoln("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Infoln("Started workers")
	<-stopCh
	klog.Infoln("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Selective Deployment
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	selectivedeployment, err := c.selectivedeploymentsLister.SelectiveDeployments(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("selectivedeployment '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	c.processSelectiveDeployment(selectivedeployment)

	c.recorder.Event(selectivedeployment, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueSelectiveDeployment takes a SelectiveDeployment resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than SelectiveDeployment.
func (c *Controller) enqueueSelectiveDeployment(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueSelectiveDeploymentAfter takes a SelectiveDeployment resource and converts it into a namespace/name
// string which is then put onto the work queue after the expiry date to be deleted. This method should *not* be
// passed resources of any type other than SelectiveDeployment.
func (c *Controller) enqueueSelectiveDeploymentAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the SelectiveDeployment resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that SelectiveDeployment resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		if ownerRef.Kind != "SelectiveDeployment" {
			return
		}

		selectivedeployment, err := c.selectivedeploymentsLister.SelectiveDeployments(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.Infof("ignoring orphaned object '%s' of selectivedeployment '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		//c.enqueueSelectiveDeploymentAfter(selectivedeployment, 5*time.Minute)
		c.enqueueSelectiveDeployment(selectivedeployment)
		return
	}
}

func (c *Controller) processSelectiveDeployment(selectivedeploymentCopy *appsv1alpha2.SelectiveDeployment) {
	multitenancyManager := multitenancy.NewManager(c.kubeclientset, c.edgenetclientset)
	permitted, _, _ := multitenancyManager.EligibilityCheck(selectivedeploymentCopy.GetNamespace())
	if permitted {

		switch selectivedeploymentCopy.Status.State {
		case appsv1alpha2.StatusReady:
			// edge-net.io/origin-selective-deployment-uid

		case appsv1alpha2.StatusCreated:
			annotations := selectivedeploymentCopy.GetAnnotations()
			if value, ok := annotations["edge-net.io/selective-deployment"]; ok && value == "follower" {
				if len(selectivedeploymentCopy.Spec.Workloads.Deployment) > 0 {
					for _, deployment := range selectivedeploymentCopy.Spec.Workloads.Deployment {
						name := selectivedeploymentCopy.GetName() + "-" + deployment.GetName()
						if _, err := c.kubeclientset.AppsV1().Deployments(selectivedeploymentCopy.GetNamespace()).Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
							return
						}
					}
				}
				if len(selectivedeploymentCopy.Spec.Workloads.StatefulSet) > 0 {
					for _, statefulset := range selectivedeploymentCopy.Spec.Workloads.StatefulSet {
						name := selectivedeploymentCopy.GetName() + "-" + statefulset.GetName()
						if _, err := c.kubeclientset.AppsV1().StatefulSets(selectivedeploymentCopy.GetNamespace()).Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
							return
						}
					}
				}
				if len(selectivedeploymentCopy.Spec.Workloads.DaemonSet) > 0 {
					for _, daemonset := range selectivedeploymentCopy.Spec.Workloads.DaemonSet {
						name := selectivedeploymentCopy.GetName() + "-" + daemonset.GetName()
						if _, err := c.kubeclientset.AppsV1().DaemonSets(selectivedeploymentCopy.GetNamespace()).Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
							return
						}
					}
				}
				if len(selectivedeploymentCopy.Spec.Workloads.Job) > 0 {
					for _, job := range selectivedeploymentCopy.Spec.Workloads.Job {
						name := selectivedeploymentCopy.GetName() + "-" + job.GetName()
						if _, err := c.kubeclientset.BatchV1().Jobs(selectivedeploymentCopy.GetNamespace()).Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
							return
						}
					}
				}
				if len(selectivedeploymentCopy.Spec.Workloads.CronJob) > 0 {
					for _, cronjob := range selectivedeploymentCopy.Spec.Workloads.CronJob {
						name := selectivedeploymentCopy.GetName() + "-" + cronjob.GetName()
						if _, err := c.kubeclientset.BatchV1().CronJobs(selectivedeploymentCopy.GetNamespace()).Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
							return
						}
					}
				}
				// Update the status of the original selectivedeployment
			} else {
				secretFMAuth, _ := c.kubeclientset.CoreV1().Secrets("edgenet").Get(context.TODO(), "federation", metav1.GetOptions{})
				propagationNamespace := fmt.Sprintf(federationv1alpha1.FederationManagerNamespace, secretFMAuth.Data["cluster-uid"])

				config := bootstrap.PrepareRestConfig(string(secretFMAuth.Data["server"]), string(secretFMAuth.Data["token"]), secretFMAuth.Data["ca.crt"])
				remotekubeclientset, _ := bootstrap.CreateKubeClientset(config)
				remoteedgeclientset, _ := bootstrap.CreateEdgeNetClientset(config)
				if _, err := remotekubeclientset.CoreV1().Secrets(propagationNamespace).Get(context.TODO(), string(selectivedeploymentCopy.GetUID()), metav1.GetOptions{}); err != nil {
					return
				}
				if _, err := remoteedgeclientset.FederationV1alpha1().SelectiveDeploymentAnchors(propagationNamespace).Get(context.TODO(), string(selectivedeploymentCopy.GetUID()), metav1.GetOptions{}); err != nil {
					return
				}
			}
		default:
			annotations := selectivedeploymentCopy.GetAnnotations()
			if value, ok := annotations["edge-net.io/selective-deployment"]; ok && value == "follower" {
				if len(selectivedeploymentCopy.Spec.Workloads.Deployment) > 0 {
					for _, deployment := range selectivedeploymentCopy.Spec.Workloads.Deployment {
						deploymentCopy := deployment.DeepCopy()
						deploymentCopy.Namespace = selectivedeploymentCopy.GetNamespace()
						deploymentCopy.Name = selectivedeploymentCopy.GetName() + "-" + deploymentCopy.GetName()
						deploymentCopy.SetOwnerReferences([]metav1.OwnerReference{selectivedeploymentCopy.MakeOwnerReference()})
						deploymentCopy.Labels["edge-net.io/selective-deployment"] = "follower"
						deploymentCopy.Labels["edge-net.io/selective-deployment-name"] = selectivedeploymentCopy.GetName()
						_, err := c.kubeclientset.AppsV1().Deployments(deploymentCopy.GetNamespace()).Create(context.TODO(), deploymentCopy, metav1.CreateOptions{})
						if err != nil {
							if errors.IsAlreadyExists(err) {
								_, err := c.kubeclientset.AppsV1().Deployments(deploymentCopy.GetNamespace()).Update(context.TODO(), deploymentCopy, metav1.UpdateOptions{})
								if err != nil {
									klog.Errorf("Couldn't update deployment %s in namespace %s", deploymentCopy.GetName(), deploymentCopy.GetNamespace())
								}
							} else {
								klog.Errorf("Couldn't create deployment %s in namespace %s", deploymentCopy.GetName(), deploymentCopy.GetNamespace())
							}
						}
					}
				}
				if len(selectivedeploymentCopy.Spec.Workloads.StatefulSet) > 0 {
					for _, statefulset := range selectivedeploymentCopy.Spec.Workloads.StatefulSet {
						statefulsetCopy := statefulset.DeepCopy()
						statefulsetCopy.Namespace = selectivedeploymentCopy.GetNamespace()
						statefulsetCopy.Name = selectivedeploymentCopy.GetName() + "-" + statefulsetCopy.GetName()
						statefulsetCopy.SetOwnerReferences([]metav1.OwnerReference{selectivedeploymentCopy.MakeOwnerReference()})
						statefulsetCopy.Labels["edge-net.io/selective-deployment"] = "follower"
						statefulsetCopy.Labels["edge-net.io/selective-deployment-name"] = selectivedeploymentCopy.GetName()
						_, err := c.kubeclientset.AppsV1().StatefulSets(statefulsetCopy.GetNamespace()).Create(context.TODO(), statefulsetCopy, metav1.CreateOptions{})
						if err != nil {
							if errors.IsAlreadyExists(err) {
								_, err := c.kubeclientset.AppsV1().StatefulSets(statefulsetCopy.GetNamespace()).Update(context.TODO(), statefulsetCopy, metav1.UpdateOptions{})
								if err != nil {
									klog.Errorf("Couldn't update statefulset %s in namespace %s", statefulsetCopy.GetName(), statefulsetCopy.GetNamespace())
								}
							} else {
								klog.Errorf("Couldn't create statefulset %s in namespace %s", statefulsetCopy.GetName(), statefulsetCopy.GetNamespace())
							}
						}
					}
				}
				if len(selectivedeploymentCopy.Spec.Workloads.DaemonSet) > 0 {
					for _, daemonset := range selectivedeploymentCopy.Spec.Workloads.DaemonSet {
						daemonsetCopy := daemonset.DeepCopy()
						daemonsetCopy.Namespace = selectivedeploymentCopy.GetNamespace()
						daemonsetCopy.Name = selectivedeploymentCopy.GetName() + "-" + daemonsetCopy.GetName()
						daemonsetCopy.SetOwnerReferences([]metav1.OwnerReference{selectivedeploymentCopy.MakeOwnerReference()})
						daemonsetCopy.Labels["edge-net.io/selective-deployment"] = "follower"
						daemonsetCopy.Labels["edge-net.io/selective-deployment-name"] = selectivedeploymentCopy.GetName()
						_, err := c.kubeclientset.AppsV1().DaemonSets(daemonsetCopy.GetNamespace()).Create(context.TODO(), daemonsetCopy, metav1.CreateOptions{})
						if err != nil {
							if errors.IsAlreadyExists(err) {
								_, err := c.kubeclientset.AppsV1().DaemonSets(daemonsetCopy.GetNamespace()).Update(context.TODO(), daemonsetCopy, metav1.UpdateOptions{})
								if err != nil {
									klog.Errorf("Couldn't update daemonset %s in namespace %s", daemonsetCopy.GetName(), daemonsetCopy.GetNamespace())
								}
							} else {
								klog.Errorf("Couldn't create daemonset %s in namespace %s", daemonsetCopy.GetName(), daemonsetCopy.GetNamespace())
							}
						}
					}
				}
				if len(selectivedeploymentCopy.Spec.Workloads.Job) > 0 {
					for _, job := range selectivedeploymentCopy.Spec.Workloads.Job {
						jobCopy := job.DeepCopy()
						jobCopy.Namespace = selectivedeploymentCopy.GetNamespace()
						jobCopy.Name = selectivedeploymentCopy.GetName() + "-" + jobCopy.GetName()
						jobCopy.SetOwnerReferences([]metav1.OwnerReference{selectivedeploymentCopy.MakeOwnerReference()})
						jobCopy.Labels["edge-net.io/selective-deployment"] = "follower"
						jobCopy.Labels["edge-net.io/selective-deployment-name"] = selectivedeploymentCopy.GetName()
						_, err := c.kubeclientset.BatchV1().Jobs(jobCopy.GetNamespace()).Create(context.TODO(), jobCopy, metav1.CreateOptions{})
						if err != nil {
							if errors.IsAlreadyExists(err) {
								_, err := c.kubeclientset.BatchV1().Jobs(jobCopy.GetNamespace()).Update(context.TODO(), jobCopy, metav1.UpdateOptions{})
								if err != nil {
									klog.Errorf("Couldn't update job %s in namespace %s", jobCopy.GetName(), jobCopy.GetNamespace())
								}
							} else {
								klog.Errorf("Couldn't create job %s in namespace %s", jobCopy.GetName(), jobCopy.GetNamespace())
							}
						}
					}
				}
				if len(selectivedeploymentCopy.Spec.Workloads.CronJob) > 0 {
					for _, cronjob := range selectivedeploymentCopy.Spec.Workloads.CronJob {
						cronjobCopy := cronjob.DeepCopy()
						cronjobCopy.Namespace = selectivedeploymentCopy.GetNamespace()
						cronjobCopy.Name = selectivedeploymentCopy.GetName() + "-" + cronjobCopy.GetName()
						cronjobCopy.SetOwnerReferences([]metav1.OwnerReference{selectivedeploymentCopy.MakeOwnerReference()})
						cronjobCopy.Labels["edge-net.io/selective-deployment"] = "follower"
						cronjobCopy.Labels["edge-net.io/selective-deployment-name"] = selectivedeploymentCopy.GetName()
						_, err := c.kubeclientset.BatchV1().CronJobs(cronjobCopy.GetNamespace()).Create(context.TODO(), cronjobCopy, metav1.CreateOptions{})
						if err != nil {
							if errors.IsAlreadyExists(err) {
								_, err := c.kubeclientset.BatchV1().CronJobs(cronjobCopy.GetNamespace()).Update(context.TODO(), cronjobCopy, metav1.UpdateOptions{})
								if err != nil {
									klog.Errorf("Couldn't update cronjob %s in namespace %s", cronjobCopy.GetName(), cronjobCopy.GetNamespace())
								}
							} else {
								klog.Errorf("Couldn't create cronjob %s in namespace %s", cronjobCopy.GetName(), cronjobCopy.GetNamespace())
							}
						}
					}
				}
			} else {
				c.createTokenForFollowers(selectivedeploymentCopy, selectivedeploymentCopy.GetNamespace())

				secretFMAuth, _ := c.kubeclientset.CoreV1().Secrets("edgenet").Get(context.TODO(), "federation", metav1.GetOptions{})
				propagationNamespace := fmt.Sprintf(federationv1alpha1.FederationManagerNamespace, secretFMAuth.Data["cluster-uid"])
				klog.Infof("%s", secretFMAuth.Data)
				config := bootstrap.PrepareRestConfig(string(secretFMAuth.Data["server"]), string(secretFMAuth.Data["token"]), secretFMAuth.Data["ca.crt"])
				remotekubeclientset, _ := bootstrap.CreateKubeClientset(config)
				remoteedgeclientset, _ := bootstrap.CreateEdgeNetClientset(config)

				authSecret, err := c.kubeclientset.CoreV1().Secrets(selectivedeploymentCopy.GetNamespace()).Get(context.TODO(), string(selectivedeploymentCopy.GetUID()), metav1.GetOptions{})
				klog.Infoln(err)
				remoteAuthSecret := new(corev1.Secret)
				remoteAuthSecret.SetName(authSecret.GetName())
				remoteAuthSecret.SetNamespace(propagationNamespace)
				remoteAuthSecret.Data = authSecret.Data
				_, err = remotekubeclientset.CoreV1().Secrets(remoteAuthSecret.GetNamespace()).Create(context.TODO(), remoteAuthSecret, metav1.CreateOptions{})
				klog.Infoln(err)
				selectivedeploymentanchor := new(federationv1alpha1.SelectiveDeploymentAnchor)
				selectivedeploymentanchor.SetName(string(selectivedeploymentCopy.GetUID()))
				selectivedeploymentanchor.SetNamespace(propagationNamespace)
				selectivedeploymentanchor.Spec.ClusterAffinity = selectivedeploymentCopy.Spec.ClusterAffinity
				selectivedeploymentanchor.Spec.ClusterReplicas = selectivedeploymentCopy.Spec.ClusterReplicas
				selectivedeploymentanchor.Spec.OriginRef.Name = selectivedeploymentCopy.GetName()
				selectivedeploymentanchor.Spec.OriginRef.Namespace = selectivedeploymentCopy.GetNamespace()
				selectivedeploymentanchor.Spec.OriginRef.UID = string(selectivedeploymentCopy.GetUID())
				selectivedeploymentanchor.Spec.SecretName = remoteAuthSecret.GetName()
				_, err = remoteedgeclientset.FederationV1alpha1().SelectiveDeploymentAnchors(selectivedeploymentanchor.GetNamespace()).Create(context.TODO(), selectivedeploymentanchor, metav1.CreateOptions{})
				klog.Infoln(err)
			}
		}
	} else {
		c.edgenetclientset.AppsV1alpha2().SelectiveDeployments(selectivedeploymentCopy.GetNamespace()).Delete(context.TODO(), selectivedeploymentCopy.GetName(), metav1.DeleteOptions{})
	}
}

// updateStatus calls the API to update the selectivedeployment status.
func (c *Controller) updateStatus(ctx context.Context, selectivedeploymentCopy *appsv1alpha2.SelectiveDeployment) {
	if selectivedeploymentCopy.Status.State == appsv1alpha2.StatusFailed {
		selectivedeploymentCopy.Status.Failed++
	}
	if _, err := c.edgenetclientset.AppsV1alpha2().SelectiveDeployments(selectivedeploymentCopy.GetNamespace()).UpdateStatus(ctx, selectivedeploymentCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}

// createTokenForFollowers creates a service account, a secret, and required permissions for the follower selective deployments to access the original selective deployment
func (c *Controller) createTokenForFollowers(selectivedeploymentCopy *appsv1alpha2.SelectiveDeployment, propagationNamespace string) error {
	serviceAccount := new(corev1.ServiceAccount)
	serviceAccount.SetName(string(selectivedeploymentCopy.GetUID()))
	serviceAccount.SetNamespace(propagationNamespace)
	if _, err := c.kubeclientset.CoreV1().ServiceAccounts(propagationNamespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, appsv1alpha2.StatusFailed, messageServiceAccountFailed)
		selectivedeploymentCopy.Status.State = appsv1alpha2.StatusFailed
		selectivedeploymentCopy.Status.Message = messageServiceAccountFailed
		c.updateStatus(context.TODO(), selectivedeploymentCopy)
		return err
	}
	authSecret := new(corev1.Secret)
	authSecret.Name = string(selectivedeploymentCopy.GetUID())
	authSecret.Namespace = propagationNamespace
	authSecret.Type = corev1.SecretTypeServiceAccountToken
	authSecret.Data = make(map[string][]byte)
	authSecret.Data["serviceaccount"] = []byte(fmt.Sprintf("system:serviceaccount:%s:%s", propagationNamespace, serviceAccount.GetName()))
	authSecret.Data["namespace"] = []byte(propagationNamespace)
	var address string
	nodeRaw, _ := c.kubeclientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/control-plane"})
	for _, node := range nodeRaw.Items {
		if internal, external := multiprovider.GetNodeIPAddresses(node.DeepCopy()); external == "" && internal == "" {
			continue
		} else if external != "" {
			address = external + ":8443"
		} else {
			address = internal + ":8443"
		}
		break
	}
	authSecret.Data["server"] = []byte(address)
	authSecret.Annotations = map[string]string{"kubernetes.io/service-account.name": serviceAccount.GetName()}
	if _, err := c.kubeclientset.CoreV1().Secrets(propagationNamespace).Create(context.TODO(), authSecret, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, appsv1alpha2.StatusFailed, messageAuthSecretFailed)
		selectivedeploymentCopy.Status.State = appsv1alpha2.StatusFailed
		selectivedeploymentCopy.Status.Message = messageAuthSecretFailed
		c.updateStatus(context.TODO(), selectivedeploymentCopy)
		return err
	}

	roleRef := rbacv1.RoleRef{Kind: "ClusterRole", Name: appsv1alpha2.RemoteSelectiveDeploymentRole}
	rbSubjects := []rbacv1.Subject{{Kind: "ServiceAccount", Name: serviceAccount.GetName(), Namespace: serviceAccount.GetNamespace()}}
	roleBind := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", appsv1alpha2.RemoteSelectiveDeploymentRole, selectivedeploymentCopy.GetUID()), Namespace: propagationNamespace},
		Subjects: rbSubjects, RoleRef: roleRef}
	roleBindLabels := map[string]string{"edge-net.io/generated": "true"}
	roleBind.SetLabels(roleBindLabels)
	if _, err := c.kubeclientset.RbacV1().RoleBindings(propagationNamespace).Create(context.TODO(), roleBind, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if roleBinding, err := c.kubeclientset.RbacV1().RoleBindings(propagationNamespace).Get(context.TODO(), roleBind.GetName(), metav1.GetOptions{}); err == nil {
				roleBindingCopy := roleBinding.DeepCopy()
				roleBindingCopy.RoleRef = roleBind.RoleRef
				roleBindingCopy.Subjects = roleBind.Subjects
				roleBindingCopy.SetLabels(roleBind.GetLabels())
				if _, err := c.kubeclientset.RbacV1().RoleBindings(propagationNamespace).Update(context.TODO(), roleBindingCopy, metav1.UpdateOptions{}); err == nil {
					return nil
				}
			}
		}
		c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, appsv1alpha2.StatusFailed, messageBindingFailed)
		selectivedeploymentCopy.Status.State = appsv1alpha2.StatusFailed
		selectivedeploymentCopy.Status.Message = messageBindingFailed
		c.updateStatus(context.TODO(), selectivedeploymentCopy)
		klog.Infoln(err)
		return err
	}
	return nil
}
