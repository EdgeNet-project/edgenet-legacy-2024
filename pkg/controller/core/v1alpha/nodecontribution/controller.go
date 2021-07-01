/*
Copyright 2020 Sorbonne Université

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
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	namecheap "github.com/billputer/go-namecheap"
	"golang.org/x/crypto/ssh"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenant"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/node"
	"github.com/EdgeNet-project/edgenet/pkg/remoteip"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	SuccessSynced         = "Synced"
	MessageResourceSynced = "Node Contribution synced successfully"
	inqueue               = "In Queue"
	inprogress            = "In Progress"
	failure               = "Failure"
	incomplete            = "Halting"
	success               = "Successful"
	create                = "create"
	update                = "update"
	delete                = "delete"
	trueStr               = "True"
	falseStr              = "False"
	unknownStr            = "Unknown"
)

// Dictionary of status messages
var statusDict = map[string]string{
	"successful":              "Node is up and running",
	"failure":                 "Node is unready",
	"in-progress":             "Node setup in progress",
	"invalid-host":            "Error: Host field must be an IP Address",
	"ssh-failure":             "Error: SSH handshake failed",
	"join-failure":            "Error: Node cannot join the cluster",
	"configuration-failure":   "Error: Scheduling configuration failed",
	"owner-reference-failure": "Error: Setting owner reference failed",
	"reboot-failure":          "Error: Node cannot get rebooted",
	"timeout":                 "Error: Node contribution failed due to timeout",
}

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
	recorder  record.EventRecorder
	publicKey ssh.Signer
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	nodeInformer coreinformers.NodeInformer,
	nodecontributionInformer informers.NodeContributionInformer) *Controller {

	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	// Get the SSH Private Key of the control plane node
	key, err := ioutil.ReadFile("../../.ssh/id_edgenet_2021")
	if err != nil {
		klog.V(4).Info(err.Error())
		panic(err.Error())
	}

	publicKey, err := ssh.ParsePrivateKey(key)
	if err != nil {
		klog.V(4).Info(err.Error())
		panic(err.Error())
	}

	controller := &Controller{
		kubeclientset:           kubeclientset,
		edgenetclientset:        edgenetclientset,
		nodesLister:             nodeInformer.Lister(),
		nodesSynced:             nodeInformer.Informer().HasSynced,
		nodecontributionsLister: nodecontributionInformer.Lister(),
		nodecontributionsSynced: nodecontributionInformer.Informer().HasSynced,
		workqueue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NodeContributions"),
		recorder:                recorder,
		publicKey:               publicKey,
	}

	klog.V(4).Infoln("Setting up event handlers")
	// Set up an event handler for when Node Contribution resources change
	nodecontributionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNodeContribution,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueNodeContribution(new)
		},
	})

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newNode := new.(*corev1.Node)
			oldNode := old.(*corev1.Node)
			if newNode.ResourceVersion == oldNode.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
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

	klog.V(4).Infoln("Starting Node Contribution controller")

	klog.V(4).Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.nodecontributionsSynced,
		c.nodesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.V(4).Infoln("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.V(4).Infoln("Started workers")
	<-stopCh
	klog.V(4).Infoln("Shutting down workers")

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
		klog.V(4).Infof("Successfully synced '%s'", key)
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

	// Make a copy of the node contribution object to make changes on it
	nodecontribution.Status.Message = []string{}

	nodeName := fmt.Sprintf("%s.edge-net.io", nodecontribution.GetName())

	recordType := remoteip.GetRecordType(nodecontribution.Spec.Host)
	if recordType == "" {
		nodecontribution.Status.State = failure
		nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["invalid-host"])
		c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
		return nil
	}
	// Set the client config according to the node contribution,
	// with the maximum time of 15 seconds to establist the connection.
	config := &ssh.ClientConfig{
		User:            nodecontribution.Spec.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(c.publicKey)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", nodecontribution.Spec.Host, nodecontribution.Spec.Port)
	contributedNode, err := c.nodesLister.Get(nodeName)
	if err == nil {
		if contributedNode.Spec.Unschedulable != !nodecontribution.Spec.Enabled {
			node.SetNodeScheduling(nodeName, !nodecontribution.Spec.Enabled)
		}
		if node.GetConditionReadyStatus(contributedNode.DeepCopy()) != trueStr {
			c.balanceMultiThreading(5)
			go c.setup(nodecontribution.Spec.Tenant, addr, nodeName, recordType, "recovery", config, nodecontribution)
		} else {
			nodecontribution.Status.State = success
			nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["succesful"])
			c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
		}
	} else {
		c.balanceMultiThreading(5)
		go c.setup(nodecontribution.Spec.Tenant, addr, nodeName, recordType, "initial", config, nodecontribution)
	}

	c.recorder.Event(nodecontribution, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
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
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		if ownerRef.Kind != "NodeContribution" {
			return
		}

		nodecontribution, err := c.nodecontributionsLister.Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of nodecontribution '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueNodeContribution(nodecontribution)
		return
	}
}

// balanceMultiThreading is a simple algorithm to limit concurrent threads
func (c *Controller) balanceMultiThreading(limit int) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
check:
	for ; true; <-ticker.C {
		var threads int
		ncRaw, err := c.nodecontributionsLister.List(labels.Everything())
		if err == nil {
			for _, ncRow := range ncRaw {
				if ncRow.Status.State == inprogress {
					threads++
				}
			}
			if threads < limit {
				break check
			}
		}
	}
}

// setup registers DNS record and makes the node join into the cluster
func (c *Controller) setup(tenantName, addr, nodeName, recordType, procedure string, config *ssh.ClientConfig, nodecontribution *corev1alpha.NodeContribution) error {
	// Steps in the procedure
	endProcedure := make(chan bool, 1)
	dnsConfiguration := make(chan bool, 1)
	establishConnection := make(chan bool, 1)
	setup := make(chan bool, 1)
	nodePatch := make(chan bool, 1)
	reboot := make(chan bool, 1)
	// Set the status as recovering
	nodecontribution.Status.State = inprogress
	nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["in-progress"])
	nodecontributionUpdated, err := c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
	if err == nil {
		nodecontribution = nodecontributionUpdated
	}

	var conn *ssh.Client
	// connCounter to try establishing a connection for several times when the node is rebooted
	connCounter := 0
	if procedure == "recovery" {
		// Watch the events of node object
		watchNode, err := c.kubeclientset.CoreV1().Nodes().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", nodeName)})
		defer watchNode.Stop()
		if err == nil {
			go func() {
				// Get events from watch interface
				for nodeEvent := range watchNode.ResultChan() {
					// Get updated node object
					updatedNode, status := nodeEvent.Object.(*corev1.Node)
					if status {
						if nodeEvent.Type == "DELETED" {
							endProcedure <- true
						}
						if node.GetConditionReadyStatus(updatedNode) == trueStr {
							nodecontribution.Status.State = success
							nodecontribution.Status.Message = append([]string{}, statusDict["successful"])
							nodecontributionUpdated, err := c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
							klog.V(4).Info(err)
							if err == nil {
								nodecontribution = nodecontributionUpdated
							}
							endProcedure <- true
						}
					}
				}
			}()
		}

		go func() {
			conn, err = ssh.Dial("tcp", addr, config)
			if err != nil {
				klog.V(4).Info(err)
				nodecontribution.Status.State = failure
				nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["ssh-failure"])
				nodecontributionUpdated, err := c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
				klog.V(4).Info(err)
				if err == nil {
					nodecontribution = nodecontributionUpdated
				}
				endProcedure <- true
			} else {
				reboot <- true
			}
		}()
	} else {
		// Start DNS configuration of `edge-net.io`
		dnsConfiguration <- true
	}
	// This statement to organize tasks and put a general timeout on
nodeSetupLoop:
	for {
		select {
		case <-dnsConfiguration:
			klog.V(4).Infof("DNS configuration started: %s", nodeName)
			// Use Namecheap API for registration
			hostRecord := namecheap.DomainDNSHost{
				Name:    strings.TrimSuffix(nodeName, ".edge-net.io"),
				Type:    recordType,
				Address: nodecontribution.Spec.Host,
			}
			result, state := node.SetHostname(hostRecord)
			// If the host record already exists, update the status of the node contribution.
			// However, the setup procedure keeps going on, so, it is not terminated.
			if !result {
				var hostnameError string
				if state == "exist" {
					hostnameError = fmt.Sprintf("Error: Hostname %s or address %s already exists", hostRecord.Name, hostRecord.Address)
				} else {
					hostnameError = fmt.Sprintf("Error: Hostname %s or address %s couldn't added", hostRecord.Name, hostRecord.Address)
				}
				nodecontribution.Status.State = incomplete
				nodecontribution.Status.Message = append(nodecontribution.Status.Message, hostnameError)
				nodecontributionUpdated, err := c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
				if err == nil {
					nodecontribution = nodecontributionUpdated
				}
				klog.V(4).Info(hostnameError)
			}
			establishConnection <- true
		case <-establishConnection:
			klog.V(4).Infof("Establish SSH connection: %s", nodeName)
			go func() {
				conn, err = ssh.Dial("tcp", addr, config)
				if err != nil && connCounter < 3 {
					klog.V(4).Info(err)
					// Wait three minutes to try establishing a connection again
					time.Sleep(3 * time.Minute)
					establishConnection <- true
					connCounter++
				} else if err != nil && connCounter >= 3 {
					nodecontribution.Status.State = failure
					nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["ssh-failure"])
					nodecontributionUpdated, err := c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
					klog.V(4).Info(err)
					if err == nil {
						nodecontribution = nodecontributionUpdated
					}
					endProcedure <- true
					return
				}
				setup <- true
			}()
		case <-setup:
			klog.V(4).Infof("Create a token and run kubadm join: %s", nodeName)
			// To prevent hanging forever during establishing a connection
			go func() {
				defer func() {
					if conn != nil {
						conn.Close()
					}
				}()
				err = c.join(conn, nodeName, nodecontribution)
				if err != nil {
					nodecontribution.Status.State = failure
					nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["join-failure"])
					nodecontributionUpdated, err := c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
					klog.V(4).Info(err)
					if err == nil {
						nodecontribution = nodecontributionUpdated
					}
					endProcedure <- true
					return
				}
				_, err = c.nodesLister.Get(nodeName)
				if err == nil {
					nodePatch <- true
				}
			}()
		case <-nodePatch:
			klog.V(4).Infof("Patch scheduling option: %s", nodeName)
			// Set the node as schedulable or unschedulable according to the node contribution
			err := node.SetNodeScheduling(nodeName, !nodecontribution.Spec.Enabled)
			if err != nil {
				nodecontribution.Status.State = incomplete
				nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["configuration-failure"])
				c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
				endProcedure <- true
			}
			ownerReferences := SetAsOwnerReference(nodecontribution)
			contributorTenant, err := c.edgenetclientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			if err == nil {
				ownerReferences = append(ownerReferences, tenant.SetAsOwnerReference(contributorTenant)...)
			}
			err = node.SetOwnerReferences(nodeName, ownerReferences)
			if err != nil {
				nodecontribution.Status.State = incomplete
				nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["owner-reference-failure"])
				c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
				endProcedure <- true
			}
			nodecontribution.Status.State = success
			nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["successful"])
			c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
			if procedure == "initial" {
				endProcedure <- true
			}
		case <-reboot:
			klog.V(4).Infof("Reboot the node: %s", nodeName)
			// Reboot the node in a minute
			err = rebootNode(conn)
			if err != nil {
				nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["reboot-failure"])
				nodecontributionUpdated, err := c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
				klog.V(4).Info(err)
				if err == nil {
					nodecontribution = nodecontributionUpdated
				}
			}
			conn.Close()
			time.Sleep(3 * time.Minute)
			establishConnection <- true
		case <-endProcedure:
			klog.V(4).Infof("Procedure completed: %s", nodeName)
			break nodeSetupLoop
		case <-time.After(5 * time.Minute):
			klog.V(4).Infof("Timeout: %s", nodeName)
			// Terminate the procedure after 5 minutes
			nodecontribution.Status.State = failure
			nodecontribution.Status.Message = append(nodecontribution.Status.Message, statusDict["timeout"])
			nodecontributionUpdated, err := c.edgenetclientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodecontribution, metav1.UpdateOptions{})
			klog.V(4).Info(err)
			if err == nil {
				nodecontribution = nodecontributionUpdated
			}
			break nodeSetupLoop
		}
	}
	return err
}

// join creates a token and runs kubeadm join command
func (c *Controller) join(conn *ssh.Client, nodeName string, nodecontribution *corev1alpha.NodeContribution) error {
	commands := []string{
		"sudo su",
		"kubeadm reset -f",
		node.CreateJoinToken("30m", nodeName),
	}
	sess, err := startSession(conn)
	if err != nil {
		klog.V(4).Info(err)
		return err
	}
	defer sess.Close()
	// StdinPipe for commands
	stdin, err := sess.StdinPipe()
	if err != nil {
		klog.V(4).Info(err)
		return err
	}
	//sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	sess, err = startShell(sess)
	if err != nil {
		klog.V(4).Info(err)
		return err
	}
	// Run commands sequentially
	for _, cmd := range commands {
		_, err = fmt.Fprintf(stdin, "%s\n", cmd)
		if err != nil {
			klog.V(4).Info(err)
			return err
		}
	}
	stdin.Close()
	// Wait for session to finish
	err = sess.Wait()
	if err != nil {
		klog.V(4).Info(err)
		return err
	}
	return nil
}

// rebootNode restarts node after a minute
func rebootNode(conn *ssh.Client) error {
	sess, err := startSession(conn)
	if err != nil {
		klog.V(4).Info(err)
		return err
	}
	defer sess.Close()
	err = sess.Run("sudo shutdown -r +1")
	if err != nil {
		klog.V(4).Info(err)
		return err
	}
	return nil
}

// Start a new session in the connection
func startSession(conn *ssh.Client) (*ssh.Session, error) {
	sess, err := conn.NewSession()
	if err != nil {
		klog.V(4).Info(err)
		return nil, err
	}
	return sess, nil
}

// Start a shell in the session
func startShell(sess *ssh.Session) (*ssh.Session, error) {
	// Start remote shell
	if err := sess.Shell(); err != nil {
		klog.V(4).Info(err)
		return nil, err
	}
	return sess, nil
}

// SetAsOwnerReference returns the nodecontribution as owner
func SetAsOwnerReference(nodecontributionCopy *corev1alpha.NodeContribution) []metav1.OwnerReference {
	// The following section makes nodecontribution become the owner
	ownerReferences := []metav1.OwnerReference{}
	newRef := *metav1.NewControllerRef(nodecontributionCopy, corev1alpha.SchemeGroupVersion.WithKind("NodeContribution"))
	takeControl := true
	newRef.Controller = &takeControl
	ownerReferences = append(ownerReferences, newRef)
	return ownerReferences
}
