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

package nodecontribution

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"golang.org/x/crypto/ssh"

	corev1alpha1 "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha1"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha1"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha1"
	multiprovider "github.com/EdgeNet-project/edgenet/pkg/multiprovider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "nodecontribution-controller"

// Definitions of the state of the nodecontribution resource
const (
	backoffLimit = 3
	dailyLimit   = 24

	successSynced  = "Synced"
	setupProcedure = "Setup"

	messageResourceSynced       = "Node Contribution synced successfully"
	messageDNSFailed            = "DNS record configuration failed"
	messageDoneSSH              = "SSH connection established"
	messageDoneKubeadm          = "Bootstrap token created and join command has been invoked"
	messageDoneSchedulingPatch  = "Node scheduling updated"
	messageDonePatch            = "Node is patched"
	messageInvalidHost          = "Host field must be an IP Address"
	messageSchedulingFailed     = "Scheduling configuration failed"
	messageUnready              = "Node is unready"
	messageSSHFailed            = "SSH handshake failed"
	messageJoinFailed           = "Node cannot join the cluster"
	messageOwnerReferenceNotSet = "Owner reference is not set"
	messageSuccessful           = "Node is up and running"
	messageReconciled           = "Reconciliation is done"
	messageReconciliation       = "Reconciliation in progress"
	messageFailed               = "Procedure failed"
)

// Controller is the controller implementation for Node Contribution resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgenetclientset is a clientset for the EdgeNet API groups
	edgenetclientset clientset.Interface

	nodesLister corelisters.NodeLister
	nodesSynced cache.InformerSynced

	nodecontributionsLister listers.NodeContributionLister
	nodecontributionsSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder             record.EventRecorder
	route53hostedZone    string
	domainName           string
	multiproviderManager *multiprovider.Manager
}

// NewController returns a new controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	nodeInformer coreinformers.NodeInformer,
	nodecontributionInformer informers.NodeContributionInformer,
	hostedZone, domain string) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	multiproviderManager := multiprovider.NewManager(kubeclientset)

	controller := &Controller{
		kubeclientset:           kubeclientset,
		edgenetclientset:        edgenetclientset,
		nodesLister:             nodeInformer.Lister(),
		nodesSynced:             nodeInformer.Informer().HasSynced,
		nodecontributionsLister: nodecontributionInformer.Lister(),
		nodecontributionsSynced: nodecontributionInformer.Informer().HasSynced,
		workqueue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NodeContributions"),
		recorder:                recorder,
		route53hostedZone:       hostedZone,
		domainName:              domain,
		multiproviderManager:    multiproviderManager,
	}

	klog.Infoln("Setting up event handlers")
	// Set up an event handler for when Node Contribution resources change
	nodecontributionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNodeContribution,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueNodeContribution(new)
		},
	})

	// Below sets incentives for those who contribute nodes to the cluster by indicating tenant.
	// The goal is to attach a resource quota claim based on the capacity of the contributed node.
	// The mechanism removes the quota increment when the node is unavailable or removed.
	// TODO: Contribution incentives should not be limited to CPU and Memory. It should cover any
	// resource the node has.
	// TODO: Be sure that the node is exactly unavailable before removing the quota increment.
	var setIncentives = func(kind, nodeName string, ownerReferences []metav1.OwnerReference, cpuCapacity, memoryCapacity *resource.Quantity) {
		for _, owner := range ownerReferences {
			if owner.Kind == "Tenant" {
				tenantResourceQuota, err := edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Get(context.TODO(), owner.Name, metav1.GetOptions{})
				if err == nil {
					tenantResourceQuotaCopy := tenantResourceQuota.DeepCopy()
					if kind == "incentive" {
						cpuCapacityCopy := cpuCapacity.DeepCopy()
						memoryCapacityCopy := memoryCapacity.DeepCopy()
						cpuAward := int64(float64(cpuCapacity.Value()) * 1.5)
						cpuCapacityCopy.Set(cpuAward)
						memoryAward := int64(float64(memoryCapacity.Value()) * 1.3)
						memoryCapacityCopy.Set(memoryAward)

						if _, elementExists := tenantResourceQuotaCopy.Spec.Claim[nodeName]; elementExists {
							if tenantResourceQuotaCopy.Spec.Claim[nodeName].ResourceList["cpu"].Equal(cpuCapacityCopy) ||
								tenantResourceQuotaCopy.Spec.Claim[nodeName].ResourceList["memory"].Equal(memoryCapacityCopy) {
								tenantResourceQuotaCopy.Spec.Claim[nodeName].ResourceList["cpu"] = cpuCapacityCopy
								tenantResourceQuotaCopy.Spec.Claim[nodeName].ResourceList["memory"] = memoryCapacityCopy
								edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuotaCopy, metav1.UpdateOptions{})
							}
						} else {
							claim := corev1alpha1.ResourceTuning{
								ResourceList: corev1.ResourceList{
									corev1.ResourceCPU:    cpuCapacityCopy,
									corev1.ResourceMemory: memoryCapacityCopy,
								},
							}
							tenantResourceQuotaCopy.Spec.Claim[nodeName] = claim
							edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuotaCopy, metav1.UpdateOptions{})
						}
					} else if kind == "disincentive" {
						if _, elementExists := tenantResourceQuota.Spec.Claim[nodeName]; elementExists {
							delete(tenantResourceQuota.Spec.Claim, nodeName)
							edgenetclientset.CoreV1alpha1().TenantResourceQuotas().Update(context.TODO(), tenantResourceQuota, metav1.UpdateOptions{})
						}
					}
				}
			}
		}
	}

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			nodeObj := obj.(*corev1.Node)
			for key := range nodeObj.Labels {
				if key == "node-role.kubernetes.io/master" {
					return
				}
			}
			if string(corev1.ConditionTrue) == multiprovider.GetConditionReadyStatus(nodeObj) {
				setIncentives("incentive", nodeObj.GetName(), nodeObj.GetOwnerReferences(), nodeObj.Status.Capacity.Cpu(), nodeObj.Status.Capacity.Memory())
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldObj := old.(*corev1.Node)
			newObj := new.(*corev1.Node)
			oldReady := multiprovider.GetConditionReadyStatus(oldObj)
			newReady := multiprovider.GetConditionReadyStatus(newObj)
			if (oldReady == string(corev1.ConditionFalse) && newReady == string(corev1.ConditionTrue)) ||
				(oldReady == string(corev1.ConditionUnknown) && newReady == string(corev1.ConditionTrue)) {
				setIncentives("incentive", newObj.GetName(), newObj.GetOwnerReferences(), newObj.Status.Capacity.Cpu(), newObj.Status.Capacity.Memory())
			} else if (oldReady == string(corev1.ConditionTrue) && newReady == string(corev1.ConditionFalse)) ||
				(oldReady == string(corev1.ConditionTrue) && newReady == string(corev1.ConditionUnknown)) {
				setIncentives("disincentive", newObj.GetName(), newObj.GetOwnerReferences(), nil, nil)
				controller.handleObject(new)
			}
		},
		DeleteFunc: func(obj interface{}) {
			nodeObj := obj.(*corev1.Node)
			ready := multiprovider.GetConditionReadyStatus(nodeObj)
			if ready == string(corev1.ConditionTrue) {
				setIncentives("disincentive", nodeObj.GetName(), nodeObj.GetOwnerReferences(), nil, nil)
				controller.handleObject(obj)
			}
		},
	})

	return controller
}

// Run will set up the event handlers for the types of node contribution and node, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Infoln("Starting Node Contribution controller")

	klog.Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.nodecontributionsSynced,
		c.nodesSynced); !ok {
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
// converge the two. It then updates the Status block of the Node Contribution
// resource with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	nodecontribution, err := c.nodecontributionsLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("nodecontribution '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	c.processNodeContribution(nodecontribution.DeepCopy())

	c.recorder.Event(nodecontribution, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

// enqueueNodeContribution takes a NodeContribution resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than NodeContribution.
func (c *Controller) enqueueNodeContribution(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueNodeContributionAfter takes a NodeContribution resource and converts it into a namespace/name
// string which is then put onto the work queue after the specified date to try establishing an SSH connection with the node.
// This method should *not* be passed resources of any type other than NodeContribution.
func (c *Controller) enqueueNodeContributionAfter(obj interface{}, after time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddAfter(key, after)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the NodeContribution resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that NodeContribution resource to be processed. If the object does not
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
		if ownerRef.Kind != "NodeContribution" {
			return
		}

		nodecontribution, err := c.nodecontributionsLister.Get(ownerRef.Name)
		if err != nil {
			klog.Infof("ignoring orphaned object '%s' of nodecontribution '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueNodeContributionAfter(nodecontribution, 15*time.Minute)
		return
	}
}

func (c *Controller) processNodeContribution(nodecontributionCopy *corev1alpha1.NodeContribution) {
	if nodecontributionCopy.Status.UpdateTimestamp != nil && nodecontributionCopy.Status.UpdateTimestamp.Add(24*time.Hour).After(time.Now()) {
		if exceedsBackoffLimit := nodecontributionCopy.Status.Failed >= backoffLimit; exceedsBackoffLimit {
			c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageFailed)
			return
		}
	}
	recordType := multiprovider.GetRecordType(nodecontributionCopy.Spec.Host)
	if recordType == "" {
		c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageInvalidHost)
		nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
		nodecontributionCopy.Status.Message = messageInvalidHost
		c.updateStatus(context.TODO(), nodecontributionCopy)
		return
	}

	nodeName := fmt.Sprintf("%s.%s", nodecontributionCopy.GetName(), c.domainName)
	switch nodecontributionCopy.Status.State {
	case corev1alpha1.StatusReady:
		if contributedNode, isJoined, isReady, _ := c.getNodeInfo(nodecontributionCopy.GetCreationTimestamp(), nodeName); !isJoined {
			c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusReconciliation, messageReconciliation)
			nodecontributionCopy.Status.State = corev1alpha1.StatusReconciliation
			nodecontributionCopy.Status.Message = messageReconciliation
			c.updateStatus(context.TODO(), nodecontributionCopy)
			return
		} else if !isReady {
			c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusAccessed, messageReconciliation)
			nodecontributionCopy.Status.State = corev1alpha1.StatusAccessed
			nodecontributionCopy.Status.Message = messageReconciliation
			c.updateStatus(context.TODO(), nodecontributionCopy)
			return
		} else {
			if contributedNode.Spec.Unschedulable != !nodecontributionCopy.Spec.Enabled {
				if err := c.multiproviderManager.SetNodeScheduling(nodeName, !nodecontributionCopy.Spec.Enabled); err != nil {
					nodecontributionCopy.Status.State = corev1alpha1.StatusAccessed
					nodecontributionCopy.Status.Message = messageReconciliation
				}
			}
			if ownerRef := metav1.GetControllerOf(contributedNode); (ownerRef == nil) || (ownerRef != nil && ownerRef.Kind != "NodeContribution") {
				nodecontributionCopy.Status.State = corev1alpha1.StatusAccessed
				nodecontributionCopy.Status.Message = messageReconciliation
			}
			if nodecontributionCopy.Spec.Tenant != nil {
				contributorTenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), *nodecontributionCopy.Spec.Tenant, metav1.GetOptions{})
				if err == nil {
					tenantRefExists := false
					nodeOwnerReferences := contributedNode.GetOwnerReferences()
					for _, value := range nodeOwnerReferences {
						if value == contributorTenant.MakeOwnerReference() {
							tenantRefExists = true
						}
					}
					if !tenantRefExists {
						nodecontributionCopy.Status.State = corev1alpha1.StatusAccessed
						nodecontributionCopy.Status.Message = messageReconciliation
					}
				}
			}

			if nodecontributionCopy.Status.State != corev1alpha1.StatusReady {
				nodecontributionCopy.Status.Failed = 0
				c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusAccessed, messageReconciliation)
				c.updateStatus(context.TODO(), nodecontributionCopy)
				return
			}
			c.recorder.Event(nodecontributionCopy, corev1.EventTypeNormal, corev1alpha1.StatusReconciliation, messageReconciled)
		}
	case corev1alpha1.StatusAccessed:
		if _, isJoined, isReady, hasTimedOut := c.getNodeInfo(nodecontributionCopy.GetCreationTimestamp(), nodeName); !isJoined {
			if hasTimedOut {
				c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageJoinFailed)
				nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
				nodecontributionCopy.Status.Message = messageJoinFailed
				c.updateStatus(context.TODO(), nodecontributionCopy)
				return
			}
			c.enqueueNodeContributionAfter(nodecontributionCopy, 1*time.Minute)
		} else {
			if isSynced := c.syncResources(nodecontributionCopy, nodeName); !isSynced {
				return
			}
			if !isReady {
				if hasTimedOut {
					c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageUnready)
					nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
					nodecontributionCopy.Status.Message = messageUnready
					c.updateStatus(context.TODO(), nodecontributionCopy)
					return
				}
				c.enqueueNodeContributionAfter(nodecontributionCopy, 10*time.Minute)
			} else {
				klog.Infof("DNS configuration started: %s", nodeName)
				// Use AWS Route53 for registration
				awsIDPath := "/edgenet/aws/id"
				if flag.Lookup("aws-id-path") != nil {
					awsIDPath = flag.Lookup("aws-id-path").Value.(flag.Getter).Get().(string)
				}
				awsSecretPath := "/edgenet/aws/secret"
				if flag.Lookup("aws-secret-path") != nil {
					awsSecretPath = flag.Lookup("aws-secret-path").Value.(flag.Getter).Get().(string)
				}
				awsID, err := ioutil.ReadFile(awsIDPath)
				if err != nil {
					c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageDNSFailed)
					nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
					nodecontributionCopy.Status.Message = messageDNSFailed
					c.updateStatus(context.TODO(), nodecontributionCopy)
					return
				}
				awsSecret, err := ioutil.ReadFile(awsSecretPath)
				if err != nil {
					c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageDNSFailed)
					nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
					nodecontributionCopy.Status.Message = messageDNSFailed
					c.updateStatus(context.TODO(), nodecontributionCopy)
					return
				}
				if updated, _ := multiprovider.SetHostnameRoute53(awsID, awsSecret, c.route53hostedZone, nodeName, nodecontributionCopy.Spec.Host, recordType); !updated {
					c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageDNSFailed)
					nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
					nodecontributionCopy.Status.Message = messageDNSFailed
					c.updateStatus(context.TODO(), nodecontributionCopy)
					return
				}
				c.recorder.Event(nodecontributionCopy, corev1.EventTypeNormal, corev1alpha1.StatusReady, messageSuccessful)
				nodecontributionCopy.Status.State = corev1alpha1.StatusReady
				nodecontributionCopy.Status.Message = messageSuccessful
				c.updateStatus(context.TODO(), nodecontributionCopy)
			}
		}
	default:
		if _, isJoined, isReady, _ := c.getNodeInfo(nodecontributionCopy.GetCreationTimestamp(), nodeName); isJoined && isReady {
			c.recorder.Event(nodecontributionCopy, corev1.EventTypeNormal, setupProcedure, messageDoneKubeadm)
			nodecontributionCopy.Status.State = corev1alpha1.StatusAccessed
			nodecontributionCopy.Status.Message = messageDoneKubeadm
			c.updateStatus(context.TODO(), nodecontributionCopy)
			return
		}
		// TODO: Include HostKeyCallback
		signer, _, ok := getSSHConfigurations()
		if !ok {
			c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageSSHFailed)
			nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
			nodecontributionCopy.Status.Message = messageSSHFailed
			c.updateStatus(context.TODO(), nodecontributionCopy)
			return
		}
		config := &ssh.ClientConfig{
			User:            nodecontributionCopy.Spec.User,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         15 * time.Second,
		}
		addr := fmt.Sprintf("%s:%d", nodecontributionCopy.Spec.Host, nodecontributionCopy.Spec.Port)
		klog.Infof("Establish SSH connection: %s", nodeName)
		conn := new(ssh.Client)
		isSuccessful := false
		isConnected := make(chan bool, 1)
		go c.sshDialRoutine(conn, config, addr, isConnected)
		select {
		case isSuccessful = <-isConnected:
			break
		case <-time.After(15 * time.Second):
			break
		}
		if !isSuccessful {
			c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageSSHFailed)
			nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
			nodecontributionCopy.Status.Message = messageSSHFailed
			c.updateStatus(context.TODO(), nodecontributionCopy)
			return
		}
		c.recorder.Event(nodecontributionCopy, corev1.EventTypeNormal, setupProcedure, messageDoneSSH)
		defer conn.Close()
		isCompleted := make(chan bool, 1)
		isSuccessful = false
		go c.join(conn, nodeName, isCompleted)
		select {
		case isSuccessful = <-isCompleted:
			break
		case <-time.After(5 * time.Minute):
			break
		}
		if !isSuccessful {
			c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageJoinFailed)
			nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
			nodecontributionCopy.Status.Message = messageJoinFailed
			c.updateStatus(context.TODO(), nodecontributionCopy)
			return
		}
		c.recorder.Event(nodecontributionCopy, corev1.EventTypeNormal, setupProcedure, messageDoneKubeadm)
		nodecontributionCopy.Status.State = corev1alpha1.StatusAccessed
		nodecontributionCopy.Status.Message = messageDoneKubeadm
		c.updateStatus(context.TODO(), nodecontributionCopy)
	}
}

func (c *Controller) syncResources(nodecontributionCopy *corev1alpha1.NodeContribution, nodeName string) bool {
	klog.Infof("Patch node and set owner references: %s", nodeName)
	// Set the node as schedulable or unschedulable according to the node contribution
	if err := c.multiproviderManager.SetNodeScheduling(nodeName, !nodecontributionCopy.Spec.Enabled); err != nil {
		c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageSchedulingFailed)
		nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
		nodecontributionCopy.Status.Message = messageSchedulingFailed
		c.updateStatus(context.TODO(), nodecontributionCopy)
		return false
	}
	c.recorder.Event(nodecontributionCopy, corev1.EventTypeNormal, setupProcedure, messageDoneSchedulingPatch)

	ownerReferences := c.formOwnerReferences(nodecontributionCopy)
	if err := c.multiproviderManager.SetOwnerReferences(nodeName, ownerReferences); err != nil {
		c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageOwnerReferenceNotSet)
		nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
		nodecontributionCopy.Status.Message = messageOwnerReferenceNotSet
		c.updateStatus(context.TODO(), nodecontributionCopy)
		return false
	}

	if vpnPeer, err := c.edgenetclientset.NetworkingV1alpha1().VPNPeers().Get(context.TODO(), nodecontributionCopy.GetName(), metav1.GetOptions{}); err == nil {
		vpnPeerCopy := vpnPeer.DeepCopy()
		vpnPeerCopy.SetOwnerReferences(ownerReferences)
		if _, err := c.edgenetclientset.NetworkingV1alpha1().VPNPeers().Update(context.TODO(), vpnPeerCopy, metav1.UpdateOptions{}); err != nil {
			c.recorder.Event(nodecontributionCopy, corev1.EventTypeWarning, corev1alpha1.StatusFailed, messageOwnerReferenceNotSet)
			nodecontributionCopy.Status.State = corev1alpha1.StatusFailed
			nodecontributionCopy.Status.Message = messageOwnerReferenceNotSet
			c.updateStatus(context.TODO(), nodecontributionCopy)
			return false
		}
	}
	c.recorder.Event(nodecontributionCopy, corev1.EventTypeNormal, setupProcedure, messageDonePatch)
	return true
}

func (c *Controller) formOwnerReferences(nodecontributionCopy *corev1alpha1.NodeContribution) []metav1.OwnerReference {
	ownerReference := nodecontributionCopy.MakeOwnerReference()
	takeControl := true
	ownerReference.Controller = &takeControl
	ownerReferences := []metav1.OwnerReference{ownerReference}
	if nodecontributionCopy.Spec.Tenant != nil {
		contributorTenant, err := c.edgenetclientset.CoreV1alpha1().Tenants().Get(context.TODO(), *nodecontributionCopy.Spec.Tenant, metav1.GetOptions{})
		if err == nil {
			ownerReferences = append(ownerReferences, contributorTenant.MakeOwnerReference())
		}
	}
	return ownerReferences
}
func (c *Controller) getNodeInfo(contributionCreationTimestamp metav1.Time, nodeName string) (contributedNode *corev1.Node, isJoined bool, isReady bool, hasTimedOut bool) {
	var err error
	if contributedNode, err = c.nodesLister.Get(nodeName); err == nil {
		if multiprovider.GetConditionReadyStatus(contributedNode) == string(corev1.ConditionTrue) {
			isJoined, isReady = true, true
		} else {
			isJoined = true
			if contributionCreationTimestamp.Add(10 * time.Minute).Before(time.Now()) {
				hasTimedOut = true
			}
		}
	} else {
		if contributionCreationTimestamp.Add(5 * time.Minute).Before(time.Now()) {
			hasTimedOut = true
		}
	}
	return contributedNode, isJoined, isReady, hasTimedOut
}

func (c *Controller) sshDialRoutine(conn *ssh.Client, config *ssh.ClientConfig, addr string, done chan<- bool) {
	clientConn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		done <- false
		return
	}
	*conn = *clientConn
	done <- true
}

func getSSHConfigurations() (ssh.Signer, ssh.HostKeyCallback, bool) {
	// Set the client config according to the node contribution,
	// with the maximum time of 15 seconds to establist the connection.
	// Get the SSH Private Key of the control plane node
	sshPath := "./.ssh"
	if flag.Lookup("ssh-path") != nil {
		sshPath = flag.Lookup("ssh-path").Value.(flag.Getter).Get().(string)
	}
	key, err := ioutil.ReadFile(fmt.Sprintf("%s/id_rsa", sshPath))
	if err != nil {
		klog.Infoln(err)
		return nil, nil, false
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		klog.Infoln(err)
		return nil, nil, false
	}
	/*hostkeyCallback, err := knownhosts.New(fmt.Sprintf("%s/known_hosts", sshPath))
	if err != nil {
		klog.Infoln(err)
		return nil, nil, false
	}*/
	return signer, nil, true
}

// join creates a token and runs kubeadm join command
func (c *Controller) join(conn *ssh.Client, nodeName string, done chan<- bool) {
	commands := []string{
		"sudo su",
		"kubeadm reset -f",
		c.multiproviderManager.CreateJoinToken("30m", nodeName),
	}
	sess, err := startSession(conn)
	if err != nil {
		klog.Info(err)
		done <- false
		return
	}
	defer sess.Close()
	// StdinPipe for commands
	stdin, err := sess.StdinPipe()
	if err != nil {
		klog.Info(err)
		done <- false
		return
	}
	//sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	sess, err = startShell(sess)
	if err != nil {
		klog.Info(err)
		done <- false
		return
	}
	// Run commands sequentially
	for _, cmd := range commands {
		_, err = fmt.Fprintf(stdin, "%s\n", cmd)
		if err != nil {
			klog.Info(err)
			done <- false
			return
		}
	}
	stdin.Close()
	// Wait for session to finish
	err = sess.Wait()
	if err != nil {
		klog.Info(err)
		done <- false
		return
	}
	done <- true
}

// Start a new session in the connection
func startSession(conn *ssh.Client) (*ssh.Session, error) {
	sess, err := conn.NewSession()
	if err != nil {
		klog.Info(err)
		return nil, err
	}
	return sess, nil
}

// Start a shell in the session
func startShell(sess *ssh.Session) (*ssh.Session, error) {
	// Start remote shell
	if err := sess.Shell(); err != nil {
		klog.Info(err)
		return nil, err
	}
	return sess, nil
}

// updateStatus calls the API to update the slice status.
func (c *Controller) updateStatus(ctx context.Context, nodecontributionCopy *corev1alpha1.NodeContribution) {
	if nodecontributionCopy.Status.State == corev1alpha1.StatusFailed {
		nodecontributionCopy.Status.Failed++
		now := metav1.Now()
		nodecontributionCopy.Status.UpdateTimestamp = &now
	}
	if _, err := c.edgenetclientset.CoreV1alpha1().NodeContributions().UpdateStatus(ctx, nodecontributionCopy, metav1.UpdateOptions{}); err != nil {
		klog.Infoln(err)
	}
}
