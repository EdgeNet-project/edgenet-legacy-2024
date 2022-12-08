/*
Copyright 2021 Contributors to the EdgeNet project.

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
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	appsv1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/apps/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/apps/v1alpha1"
	"github.com/EdgeNet-project/edgenet/pkg/multiprovider"
	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	selection "k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	batchinformers "k8s.io/client-go/informers/batch/v1"
	batchv1beta1informers "k8s.io/client-go/informers/batch/v1beta1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	batchv1beta1listers "k8s.io/client-go/listers/batch/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "selectivedeployment-controller"

// Definitions of the state of the selectivedeployment resource
const (
	successSynced                 = "Synced"
	messageResourceSynced         = "Selective Deployment synced successfully"
	messageWorkloadCreated        = "The desired workload(s) are created successfully"
	failureCreation               = "Creation Failed"
	messageWorkloadFailed         = "A workload defined in the spec could not be created"
	messageExtendedWorkloadFailed = "Workload could not be created: %s %s"
	messageExtendedWorkloadInUse  = "Workload is owned by another selective deployment: %s %s"
	failureGeoJSON                = "GeoJSON Error"
	messageGeoJSONError           = "GeoJSON has a format error"
	failureFewerNodes             = "Fewer nodes issue"
	messageFewerNodes             = "The number of nodes found is lower than desired"
	failure                       = "Failure"
	partial                       = "Running Partially"
	success                       = "Running"
	noSchedule                    = "NoSchedule"
	trueStr                       = "True"
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
	cronjobsLister     batchv1beta1listers.CronJobLister
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
	cronjobInformer batchv1beta1informers.CronJobInformer,
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
			newSelectiveDeployment := new.(*appsv1alpha1.SelectiveDeployment)
			oldSelectiveDeployment := old.(*appsv1alpha1.SelectiveDeployment)
			if newSelectiveDeployment.ResourceVersion == oldSelectiveDeployment.ResourceVersion {
				return
			}
			controller.enqueueSelectiveDeployment(new)
		},
	})

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	})

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

	c.applyCriteria(selectivedeployment)

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

func (c *Controller) recoverSelectiveDeployments(obj interface{}) {
	nodeCopy := obj.(*corev1.Node).DeepCopy()
	if nodeCopy.GetDeletionTimestamp() != nil || multiprovider.GetConditionReadyStatus(nodeCopy) != trueStr || nodeCopy.Spec.Unschedulable {
		ownerRaw, status := c.getByNode(nodeCopy.GetName())
		if status {
			for _, ownerRow := range ownerRaw {
				selectivedeployment, err := c.selectivedeploymentsLister.SelectiveDeployments(ownerRow[0]).Get(ownerRow[1])
				if err != nil {
					klog.Infoln(err.Error())
					continue
				}
				if selectivedeployment.Spec.Recovery {
					c.enqueueSelectiveDeployment(selectivedeployment)
				}
			}
		}
	} else if multiprovider.GetConditionReadyStatus(nodeCopy) == trueStr {
		selectivedeploymentRaw, _ := c.selectivedeploymentsLister.SelectiveDeployments("").List(labels.Everything())
		for _, selectivedeploymentRow := range selectivedeploymentRaw {
			if selectivedeploymentRow.Spec.Recovery {
				if selectivedeploymentRow.Status.State == partial || selectivedeploymentRow.Status.State == failure {
					c.enqueueSelectiveDeployment(selectivedeploymentRow)
				}
			}
		}
	}

}

// getByNode generates selectivedeployment list from the owner references of workloads which contains the node that has an event (add/update/delete)
func (c *Controller) getByNode(nodeName string) ([][]string, bool) {
	ownerList := [][]string{}
	status := false
	setList := func(ctlPodSpec corev1.PodSpec, ownerReferences []metav1.OwnerReference, namespace string) {
		podSpec := ctlPodSpec
		if podSpec.Affinity != nil && podSpec.Affinity.NodeAffinity != nil && podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		nodeSelectorLoop:
			for _, nodeSelectorTerm := range podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
				for _, matchExpression := range nodeSelectorTerm.MatchExpressions {
					if matchExpression.Key == "kubernetes.io/hostname" {
						for _, expressionNodeName := range matchExpression.Values {
							if nodeName == expressionNodeName {
								for _, owner := range ownerReferences {
									if owner.Kind == "SelectiveDeployment" {
										ownerDet := []string{namespace, owner.Name}
										if exists, _ := util.SliceContains(ownerList, ownerDet); !exists {
											ownerList = append(ownerList, ownerDet)
										}
										status = true
									}
								}
								break nodeSelectorLoop
							}
						}
					}
				}
			}
		}
	}
	deploymentRaw, err := c.deploymentsLister.Deployments("").List(labels.Everything())
	if err != nil {
		klog.Infoln(err.Error())
		panic(err.Error())
	}
	for _, deploymentRow := range deploymentRaw {
		setList(deploymentRow.Spec.Template.Spec, deploymentRow.GetOwnerReferences(), deploymentRow.GetNamespace())
	}
	daemonsetRaw, err := c.daemonsetsLister.DaemonSets("").List(labels.Everything())
	if err != nil {
		klog.Infoln(err.Error())
		panic(err.Error())
	}
	for _, daemonsetRow := range daemonsetRaw {
		setList(daemonsetRow.Spec.Template.Spec, daemonsetRow.GetOwnerReferences(), daemonsetRow.GetNamespace())
	}
	statefulsetRaw, err := c.statefulsetsLister.StatefulSets("").List(labels.Everything())
	if err != nil {
		klog.Infoln(err.Error())
		panic(err.Error())
	}
	for _, statefulsetRow := range statefulsetRaw {
		setList(statefulsetRow.Spec.Template.Spec, statefulsetRow.GetOwnerReferences(), statefulsetRow.GetNamespace())
	}
	jobRaw, err := c.jobsLister.Jobs("").List(labels.Everything())
	if err != nil {
		klog.Infoln(err.Error())
		panic(err.Error())
	}
	for _, jobRow := range jobRaw {
		setList(jobRow.Spec.Template.Spec, jobRow.GetOwnerReferences(), jobRow.GetNamespace())
	}
	cronjobRaw, err := c.cronjobsLister.CronJobs("").List(labels.Everything())
	if err != nil {
		klog.Infoln(err.Error())
		panic(err.Error())
	}
	for _, cronjobRow := range cronjobRaw {
		setList(cronjobRow.Spec.JobTemplate.Spec.Template.Spec, cronjobRow.GetOwnerReferences(), cronjobRow.GetNamespace())
	}
	return ownerList, status
}

// applyCriteria picks the nodes according to the selector
func (c *Controller) applyCriteria(selectivedeploymentCopy *appsv1alpha1.SelectiveDeployment) {
	oldStatus := selectivedeploymentCopy.Status
	statusUpdate := func() {
		if !reflect.DeepEqual(oldStatus, selectivedeploymentCopy.Status) {
			c.edgenetclientset.AppsV1alpha1().SelectiveDeployments(selectivedeploymentCopy.GetNamespace()).UpdateStatus(context.TODO(), selectivedeploymentCopy, metav1.UpdateOptions{})
		}
	}
	defer statusUpdate()

	ownerReferences := SetAsOwnerReference(selectivedeploymentCopy)
	workloadCounter := 0
	failureCounter := 0
	if selectivedeploymentCopy.Spec.Workloads.Deployment != nil {
		workloadCounter += len(selectivedeploymentCopy.Spec.Workloads.Deployment)
		for _, deployment := range selectivedeploymentCopy.Spec.Workloads.Deployment {
			if actualDeployment, err := c.deploymentsLister.Deployments(selectivedeploymentCopy.GetNamespace()).Get(deployment.GetName()); errors.IsNotFound(err) {
				desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, deployment.Spec.Template, deployment.Spec.Template, ownerReferences)
				desiredDeployment := deployment.DeepCopy()
				desiredDeployment.Spec.Template = desiredPodTemplate
				desiredDeployment.SetOwnerReferences(ownerReferences)
				if _, err = c.kubeclientset.AppsV1().Deployments(selectivedeploymentCopy.GetNamespace()).Create(context.TODO(), desiredDeployment, metav1.CreateOptions{}); err != nil {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredDeployment.GetObjectKind().GroupVersionKind().Kind, desiredDeployment.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					klog.Infoln(err)
					failureCounter++
				} else {
					if isFailed {
						failureCounter++
					}
				}
			} else {
				if hasOwner := checkOwnerReferences(selectivedeploymentCopy, actualDeployment.GetOwnerReferences()); !hasOwner {
					desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, actualDeployment.Spec.Template, deployment.Spec.Template, ownerReferences)
					if isFailed {
						failureCounter++
					}
					desiredDeployment := actualDeployment.DeepCopy()
					desiredDeployment.Spec.Template = desiredPodTemplate
					desiredDeployment.SetOwnerReferences(ownerReferences)
					if actualDeployment.Status.Replicas != actualDeployment.Status.AvailableReplicas || !reflect.DeepEqual(actualDeployment.Spec.Template, desiredDeployment.Spec.Template) {
						if _, err = c.kubeclientset.AppsV1().Deployments(selectivedeploymentCopy.GetNamespace()).Update(context.TODO(), desiredDeployment, metav1.UpdateOptions{}); err != nil {
							c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredDeployment.GetObjectKind().GroupVersionKind().Kind, desiredDeployment.GetName()))
							selectivedeploymentCopy.Status.State = failure
							selectivedeploymentCopy.Status.Message = messageWorkloadFailed
							klog.Infoln(err)
							if !isFailed {
								failureCounter++
							}
						}
					}
				} else {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadInUse, deployment.GetObjectKind().GroupVersionKind().Kind, deployment.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					failureCounter++
				}
			}
		}
	}
	if selectivedeploymentCopy.Spec.Workloads.DaemonSet != nil {
		workloadCounter += len(selectivedeploymentCopy.Spec.Workloads.DaemonSet)
		for _, daemonset := range selectivedeploymentCopy.Spec.Workloads.DaemonSet {
			if actualDaemonset, err := c.daemonsetsLister.DaemonSets(selectivedeploymentCopy.GetNamespace()).Get(daemonset.GetName()); errors.IsNotFound(err) {
				desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, daemonset.Spec.Template, daemonset.Spec.Template, ownerReferences)
				desiredDaemonset := daemonset.DeepCopy()
				desiredDaemonset.Spec.Template = desiredPodTemplate
				desiredDaemonset.SetOwnerReferences(ownerReferences)
				if _, err = c.kubeclientset.AppsV1().DaemonSets(selectivedeploymentCopy.GetNamespace()).Create(context.TODO(), desiredDaemonset, metav1.CreateOptions{}); err != nil {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredDaemonset.GetObjectKind().GroupVersionKind().Kind, desiredDaemonset.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					klog.Infoln(err)
					failureCounter++
				} else {
					if isFailed {
						failureCounter++
					}
				}
			} else {
				if hasOwner := checkOwnerReferences(selectivedeploymentCopy, actualDaemonset.GetOwnerReferences()); !hasOwner {
					desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, actualDaemonset.Spec.Template, daemonset.Spec.Template, ownerReferences)
					if isFailed {
						failureCounter++
					}
					desiredDaemonset := daemonset.DeepCopy()
					desiredDaemonset.Spec.Template = desiredPodTemplate
					desiredDaemonset.SetOwnerReferences(ownerReferences)
					if actualDaemonset.Status.DesiredNumberScheduled != actualDaemonset.Status.NumberReady || !reflect.DeepEqual(actualDaemonset.Spec.Template, desiredDaemonset.Spec.Template) {
						if _, err = c.kubeclientset.AppsV1().DaemonSets(selectivedeploymentCopy.GetNamespace()).Update(context.TODO(), desiredDaemonset, metav1.UpdateOptions{}); err != nil {
							c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredDaemonset.GetObjectKind().GroupVersionKind().Kind, desiredDaemonset.GetName()))
							selectivedeploymentCopy.Status.State = failure
							selectivedeploymentCopy.Status.Message = messageWorkloadFailed
							klog.Infoln(err)
							if !isFailed {
								failureCounter++
							}
						}
					}
				} else {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadInUse, daemonset.GetObjectKind().GroupVersionKind().Kind, daemonset.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					failureCounter++
				}
			}
		}
	}
	if selectivedeploymentCopy.Spec.Workloads.StatefulSet != nil {
		workloadCounter += len(selectivedeploymentCopy.Spec.Workloads.StatefulSet)
		for _, statefulset := range selectivedeploymentCopy.Spec.Workloads.StatefulSet {
			if actualStatefulset, err := c.statefulsetsLister.StatefulSets(selectivedeploymentCopy.GetNamespace()).Get(statefulset.GetName()); errors.IsNotFound(err) {
				desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, statefulset.Spec.Template, statefulset.Spec.Template, ownerReferences)
				desiredStatefulset := statefulset.DeepCopy()
				desiredStatefulset.Spec.Template = desiredPodTemplate
				desiredStatefulset.SetOwnerReferences(ownerReferences)
				if _, err = c.kubeclientset.AppsV1().StatefulSets(selectivedeploymentCopy.GetNamespace()).Create(context.TODO(), desiredStatefulset, metav1.CreateOptions{}); err != nil {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredStatefulset.GetObjectKind().GroupVersionKind().Kind, desiredStatefulset.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					klog.Infoln(err)
					failureCounter++
				} else {
					if isFailed {
						failureCounter++
					}
				}
			} else {
				if hasOwner := checkOwnerReferences(selectivedeploymentCopy, actualStatefulset.GetOwnerReferences()); !hasOwner {
					desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, actualStatefulset.Spec.Template, statefulset.Spec.Template, ownerReferences)
					if isFailed {
						failureCounter++
					}
					desiredStatefulset := statefulset.DeepCopy()
					desiredStatefulset.Spec.Template = desiredPodTemplate
					desiredStatefulset.SetOwnerReferences(ownerReferences)
					if !reflect.DeepEqual(actualStatefulset.Spec.Template, desiredStatefulset.Spec.Template) {
						if _, err = c.kubeclientset.AppsV1().StatefulSets(selectivedeploymentCopy.GetNamespace()).Update(context.TODO(), desiredStatefulset, metav1.UpdateOptions{}); err != nil {
							c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredStatefulset.GetObjectKind().GroupVersionKind().Kind, desiredStatefulset.GetName()))
							selectivedeploymentCopy.Status.State = failure
							selectivedeploymentCopy.Status.Message = messageWorkloadFailed
							klog.Infoln(err)
							if !isFailed {
								failureCounter++
							}
						}
					}
				} else {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadInUse, statefulset.GetObjectKind().GroupVersionKind().Kind, statefulset.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					failureCounter++
				}
			}
		}
	}
	if selectivedeploymentCopy.Spec.Workloads.Job != nil {
		workloadCounter += len(selectivedeploymentCopy.Spec.Workloads.Job)
		for _, job := range selectivedeploymentCopy.Spec.Workloads.Job {
			if actualJob, err := c.jobsLister.Jobs(selectivedeploymentCopy.GetNamespace()).Get(job.GetName()); errors.IsNotFound(err) {
				desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, job.Spec.Template, job.Spec.Template, ownerReferences)
				desiredJob := job.DeepCopy()
				desiredJob.Spec.Template = desiredPodTemplate
				desiredJob.SetOwnerReferences(ownerReferences)
				if _, err = c.kubeclientset.BatchV1().Jobs(selectivedeploymentCopy.GetNamespace()).Create(context.TODO(), desiredJob, metav1.CreateOptions{}); err != nil {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredJob.GetObjectKind().GroupVersionKind().Kind, desiredJob.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					klog.Infoln(err)
					failureCounter++
				} else {
					if isFailed {
						failureCounter++
					}
				}
			} else {
				if hasOwner := checkOwnerReferences(selectivedeploymentCopy, actualJob.GetOwnerReferences()); !hasOwner {
					desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, actualJob.Spec.Template, job.Spec.Template, ownerReferences)
					if isFailed {
						failureCounter++
					}
					desiredJob := job.DeepCopy()
					desiredJob.Spec.Template = desiredPodTemplate
					desiredJob.SetOwnerReferences(ownerReferences)
					if !reflect.DeepEqual(actualJob.Spec.Template, desiredJob.Spec.Template) {
						if _, err = c.kubeclientset.BatchV1().Jobs(selectivedeploymentCopy.GetNamespace()).Update(context.TODO(), desiredJob, metav1.UpdateOptions{}); err != nil {
							c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredJob.GetObjectKind().GroupVersionKind().Kind, desiredJob.GetName()))
							selectivedeploymentCopy.Status.State = failure
							selectivedeploymentCopy.Status.Message = messageWorkloadFailed
							klog.Infoln(err)
							if !isFailed {
								failureCounter++
							}
						}
					}
				} else {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadInUse, job.GetObjectKind().GroupVersionKind().Kind, job.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					failureCounter++
				}
			}
		}
	}
	if selectivedeploymentCopy.Spec.Workloads.CronJob != nil {
		workloadCounter += len(selectivedeploymentCopy.Spec.Workloads.CronJob)
		for _, cronjob := range selectivedeploymentCopy.Spec.Workloads.CronJob {
			if actualCronjob, err := c.cronjobsLister.CronJobs(selectivedeploymentCopy.GetNamespace()).Get(cronjob.GetName()); errors.IsNotFound(err) {
				desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, cronjob.Spec.JobTemplate.Spec.Template, cronjob.Spec.JobTemplate.Spec.Template, ownerReferences)
				desiredCronjob := cronjob.DeepCopy()
				desiredCronjob.Spec.JobTemplate.Spec.Template = desiredPodTemplate
				desiredCronjob.SetOwnerReferences(ownerReferences)
				if _, err = c.kubeclientset.BatchV1beta1().CronJobs(selectivedeploymentCopy.GetNamespace()).Create(context.TODO(), desiredCronjob, metav1.CreateOptions{}); err != nil {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredCronjob.GetObjectKind().GroupVersionKind().Kind, desiredCronjob.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					klog.Infoln(err)
					failureCounter++
				} else {
					if isFailed {
						failureCounter++
					}
				}
			} else {
				if hasOwner := checkOwnerReferences(selectivedeploymentCopy, actualCronjob.GetOwnerReferences()); !hasOwner {
					desiredPodTemplate, isFailed := c.configureWorkload(selectivedeploymentCopy, actualCronjob.Spec.JobTemplate.Spec.Template, cronjob.Spec.JobTemplate.Spec.Template, ownerReferences)
					if isFailed {
						failureCounter++
					}
					desiredCronjob := cronjob.DeepCopy()
					desiredCronjob.Spec.JobTemplate.Spec.Template = desiredPodTemplate
					desiredCronjob.SetOwnerReferences(ownerReferences)
					if !reflect.DeepEqual(actualCronjob.Spec.JobTemplate.Spec.Template, desiredCronjob.Spec.JobTemplate.Spec.Template) {
						if _, err = c.kubeclientset.BatchV1beta1().CronJobs(selectivedeploymentCopy.GetNamespace()).Update(context.TODO(), desiredCronjob, metav1.UpdateOptions{}); err != nil {
							c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadFailed, desiredCronjob.GetObjectKind().GroupVersionKind().Kind, desiredCronjob.GetName()))
							selectivedeploymentCopy.Status.State = failure
							selectivedeploymentCopy.Status.Message = messageWorkloadFailed
							klog.Infoln(err)
							if !isFailed {
								failureCounter++
							}
						}
					}
				} else {
					c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureCreation, fmt.Sprintf(messageExtendedWorkloadInUse, cronjob.GetObjectKind().GroupVersionKind().Kind, cronjob.GetName()))
					selectivedeploymentCopy.Status.State = failure
					selectivedeploymentCopy.Status.Message = messageWorkloadFailed
					failureCounter++
				}
			}
		}
	}

	if failureCounter == 0 && workloadCounter != 0 {
		c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeNormal, success, messageWorkloadCreated)
		selectivedeploymentCopy.Status.State = success
		selectivedeploymentCopy.Status.Message = messageWorkloadCreated
	} else if workloadCounter == failureCounter {
		selectivedeploymentCopy.Status.State = failure
	} else {
		selectivedeploymentCopy.Status.State = partial
	}
	selectivedeploymentCopy.Status.Ready = fmt.Sprintf("%d/%d", (workloadCounter - failureCounter), workloadCounter)
}

// configureWorkload converges the actual state to the desired state of the workloads defined in selective deployment
func (c *Controller) configureWorkload(selectivedeploymentCopy *appsv1alpha1.SelectiveDeployment, actualPodTemplate, desiredPodTemplate corev1.PodTemplateSpec, ownerReferences []metav1.OwnerReference) (corev1.PodTemplateSpec, bool) {
	klog.Infoln("configureWorkload: start")

	actualNodeSelectorTermList := corev1.NodeSelectorTerm{}

	if actualPodTemplate.Spec.Affinity != nil && len(actualPodTemplate.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
		actualNodeSelectorTermList = actualPodTemplate.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
	}
	desiredNodeSelectorTermList, isFailed := c.setFilter(selectivedeploymentCopy, actualNodeSelectorTermList, "addOrUpdate")
	// Set the new node affinity configuration for the workload and update that
	nodeAffinity := &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: desiredNodeSelectorTermList,
		},
	}
	if len(desiredNodeSelectorTermList) <= 0 {
		affinity := &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: desiredNodeSelectorTermList,
				},
			},
		}
		affinity.Reset()
	}

	if len(desiredNodeSelectorTermList) <= 0 && desiredPodTemplate.Spec.Affinity != nil {
		desiredPodTemplate.Spec.Affinity.Reset()
	} else if desiredPodTemplate.Spec.Affinity != nil {
		desiredPodTemplate.Spec.Affinity.NodeAffinity = nodeAffinity
	} else {
		desiredPodTemplate.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: nodeAffinity,
		}
	}

	return desiredPodTemplate, isFailed
}

// setFilter generates the values in the predefined form and puts those into the node selection fields of the selectivedeployment object
func (c *Controller) setFilter(selectivedeploymentCopy *appsv1alpha1.SelectiveDeployment, actualNodeSelectorTerm corev1.NodeSelectorTerm, event string) ([]corev1.NodeSelectorTerm, bool) {
	var nodeSelectorTermList []corev1.NodeSelectorTerm
	isFailed := false

	for _, selectorRow := range selectivedeploymentCopy.Spec.Selector {
		var matchExpression corev1.NodeSelectorRequirement
		matchExpression.Values = []string{}
		matchExpression.Operator = selectorRow.Operator
		matchExpression.Key = "kubernetes.io/hostname"
		selectorName := strings.ToLower(selectorRow.Name)

		// If the event type is delete then we don't need to run the part below
		if event != "delete" {
			labelKeySuffix := ""
			if selectorName == "state" || selectorName == "country" {
				labelKeySuffix = "-iso"
			}
			labelKey := strings.ToLower(fmt.Sprintf("edge-net.io/%s%s", selectorName, labelKeySuffix))
			// This gets the node list which includes the EdgeNet geolabels
			scheduleReq, _ := labels.NewRequirement("spec.unschedulable", selection.NotEquals, []string{"true"})
			selector := labels.NewSelector()
			selector = selector.Add(*scheduleReq)
			nodesRaw, err := c.nodesLister.List(selector)
			if err != nil {
				klog.Infoln(err.Error())
				panic(err.Error())
			}
			// This loop allows us to process each value defined at the object of selectivedeployment resource

			matchNodeList := []string{}
			desiredNodeList := []string{}
			checkActualList := func(nodeName string) bool {
				if len(actualNodeSelectorTerm.MatchExpressions) > 0 {
					if exists, _ := util.Contains(actualNodeSelectorTerm.MatchExpressions[0].Values, nodeName); exists {
						desiredNodeList = append(desiredNodeList, nodeName)
						return true
					}
				}
				return false
			}
			for _, selectorValue := range selectorRow.Value {
				// The loop to process each node separately
				for _, nodeRow := range nodesRaw {
					taintBlock := false
					for _, taint := range nodeRow.Spec.Taints {
						if (taint.Key == "node-role.kubernetes.io/master" && taint.Effect == noSchedule) ||
							(taint.Key == "node.kubernetes.io/unschedulable" && taint.Effect == noSchedule) {
							taintBlock = true
						}
					}
					conditionBlock := false
					if multiprovider.GetConditionReadyStatus(nodeRow.DeepCopy()) != trueStr {
						conditionBlock = true
					}

					if !conditionBlock && !taintBlock {
						// Turn the key into the predefined form which is determined at the custom resource definition of selectivedeployment
						switch selectorName {
						case "city", "state", "country", "continent":
							if selectorValue == nodeRow.Labels[labelKey] && selectorRow.Operator == "In" {
								if !checkActualList(nodeRow.Labels["kubernetes.io/hostname"]) {
									matchNodeList = append(matchNodeList, nodeRow.Labels["kubernetes.io/hostname"])
								}
							} else if selectorValue != nodeRow.Labels[labelKey] && selectorRow.Operator == "NotIn" {
								if !checkActualList(nodeRow.Labels["kubernetes.io/hostname"]) {
									matchNodeList = append(matchNodeList, nodeRow.Labels["kubernetes.io/hostname"])
								}
							}
						case "polygon":
							var polygon [][]float64
							err = json.Unmarshal([]byte(selectorValue), &polygon)
							if err != nil {
								c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureGeoJSON, messageGeoJSONError)
								selectivedeploymentCopy.Status.State = failure
								selectivedeploymentCopy.Status.Message = messageGeoJSONError
								isFailed = true
								continue
							}
							if nodeRow.Labels["edge-net.io/lon"] != "" && nodeRow.Labels["edge-net.io/lat"] != "" {
								// Because of alphanumeric limitations of Kubernetes on the labels we use "w", "e", "n", and "s" prefixes
								// at the labels of latitude and longitude. Here is the place those prefixes are dropped away.
								lonStr := nodeRow.Labels["edge-net.io/lon"][1:]
								latStr := nodeRow.Labels["edge-net.io/lat"][1:]
								if lon, err := strconv.ParseFloat(lonStr, 64); err == nil {
									if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
										// boundbox is a rectangle which provides to check whether the point is inside polygon
										// without taking all point of the polygon into consideration
										boundbox := multiprovider.Boundbox(polygon)
										status := multiprovider.GeoFence(boundbox, polygon, lon, lat)
										if status && selectorRow.Operator == "In" {
											if !checkActualList(nodeRow.Labels["kubernetes.io/hostname"]) {
												matchNodeList = append(matchNodeList, nodeRow.Labels["kubernetes.io/hostname"])
											}
										} else if !status && selectorRow.Operator == "NotIn" {
											if !checkActualList(nodeRow.Labels["kubernetes.io/hostname"]) {
												matchNodeList = append(matchNodeList, nodeRow.Labels["kubernetes.io/hostname"])
											}
										}
									}
								}
							}
						}
					}
				}
			}

			prePickedNodeCount := len(desiredNodeList)
			if selectorRow.Quantity == 0 {
				desiredNodeList = append(desiredNodeList, matchNodeList...)
			} else {
				if selectorRow.Quantity > prePickedNodeCount {
					if len(matchNodeList) > 0 {
						for i := 0; i < (selectorRow.Quantity - prePickedNodeCount); i++ {
							rand.Seed(time.Now().UnixNano())
							randomSelect := rand.Intn(len(matchNodeList))
							desiredNodeList = append(desiredNodeList, matchNodeList[randomSelect])
							matchNodeList[randomSelect] = matchNodeList[len(matchNodeList)-1]
							matchNodeList = matchNodeList[:len(matchNodeList)-1]
							if len(matchNodeList) == 0 {
								break
							}
						}
					}

					if selectorRow.Quantity != len(desiredNodeList) {
						c.recorder.Event(selectivedeploymentCopy, corev1.EventTypeWarning, failureFewerNodes, messageFewerNodes)
						selectivedeploymentCopy.Status.State = failure
						selectivedeploymentCopy.Status.Message = messageFewerNodes
						isFailed = true
					}

				} else {
					for i := 0; i < (prePickedNodeCount - selectorRow.Quantity); i++ {
						desiredNodeList = desiredNodeList[:len(desiredNodeList)-1]
					}
				}
			}

			matchExpression.Values = desiredNodeList
			var nodeSelectorTerm corev1.NodeSelectorTerm
			nodeSelectorTerm.MatchExpressions = append(nodeSelectorTerm.MatchExpressions, matchExpression)
			nodeSelectorTermList = append(nodeSelectorTermList, nodeSelectorTerm)
		}
	}
	return nodeSelectorTermList, isFailed
}

// SetAsOwnerReference returns the selectivedeployment as owner
func SetAsOwnerReference(selectivedeploymentCopy *appsv1alpha1.SelectiveDeployment) []metav1.OwnerReference {
	// The following section makes selectivedeployment become the owner
	ownerReferences := []metav1.OwnerReference{}
	newRef := *metav1.NewControllerRef(selectivedeploymentCopy, appsv1alpha1.SchemeGroupVersion.WithKind("SelectiveDeployment"))
	takeControl := true
	newRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newRef)
	return ownerReferences
}

func checkOwnerReferences(selectivedeploymentCopy *appsv1alpha1.SelectiveDeployment, ownerReferences []metav1.OwnerReference) bool {
	underControl := false
	for _, reference := range ownerReferences {
		if reference.Kind == "SelectiveDeployment" && reference.UID != selectivedeploymentCopy.GetUID() {
			underControl = true
		}
	}
	return underControl
}
