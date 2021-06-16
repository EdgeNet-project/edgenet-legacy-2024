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

	"golang.org/x/crypto/ssh"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/core/v1alpha/tenant"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	"github.com/EdgeNet-project/edgenet/pkg/node"
	"github.com/EdgeNet-project/edgenet/pkg/remoteip"

	namecheap "github.com/billputer/go-namecheap"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) error
	ObjectCreatedOrUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        kubernetes.Interface
	edgenetClientset versioned.Interface
	publicKey        ssh.Signer
}

// Init handles any handler initialization
func (t *Handler) Init(kubernetes kubernetes.Interface, edgenet versioned.Interface) error {
	log.Info("NCHandler.Init")
	t.clientset = kubernetes
	t.edgenetClientset = edgenet

	// Get the SSH Public Key of the headnode
	key, err := ioutil.ReadFile("../../.ssh/id_edgenet_2021")
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	t.publicKey, err = ssh.ParsePrivateKey(key)
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	node.Clientset = t.clientset
	return err
}

// ObjectCreatedOrUpdated is called when an object is created
func (t *Handler) ObjectCreatedOrUpdated(obj interface{}) {
	log.Info("NCHandler.ObjectCreated")
	// Make a copy of the node contribution object to make changes on it
	nodeContribution := obj.(*corev1alpha.NodeContribution).DeepCopy()
	nodeContribution.Status.Message = []string{}

	nodeName := fmt.Sprintf("%s.edge-net.io", nodeContribution.GetName())

	recordType := remoteip.GetRecordType(nodeContribution.Spec.Host)
	if recordType == "" {
		nodeContribution.Status.State = failure
		nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["invalid-host"])
		t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
		t.sendEmail(nodeContribution)
		return
	}
	// Set the client config according to the node contribution,
	// with the maximum time of 15 seconds to establist the connection.
	config := &ssh.ClientConfig{
		User:            nodeContribution.Spec.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(t.publicKey)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", nodeContribution.Spec.Host, nodeContribution.Spec.Port)
	contributedNode, err := t.clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err == nil {
		// The node corresponding to the contributed node exists in the cluster
		if contributedNode.Spec.Unschedulable != !nodeContribution.Spec.Enabled {
			node.SetNodeScheduling(nodeName, !nodeContribution.Spec.Enabled)
		}
		if node.GetConditionReadyStatus(contributedNode.DeepCopy()) != trueStr {
			t.balanceMultiThreading(5)
			go t.runRecoveryProcedure(addr, config, nodeName, nodeContribution, contributedNode)
		} else {
			nodeContribution.Status.State = success
			nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["node-ok"])
			t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
		}
	} else {
		// There is no node corresponding to the node contribution
		t.balanceMultiThreading(5)
		nodeContributionLabels := nodeContribution.GetLabels()
		tenantName := nodeContributionLabels["edge-net.io/tenant"]
		go t.runSetupProcedure(tenantName, addr, nodeName, recordType, config, nodeContribution)
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("NCHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(nodeContribution *corev1alpha.NodeContribution) {
	// TODO: Proper implementation is missing here
	// For those who are tenant owner and authorized users of the tenant
	contentData := mailer.MultiProviderData{}
	contentData.Name = nodeContribution.GetName()
	contentData.Host = nodeContribution.Spec.Host
	contentData.Status = nodeContribution.Status.State
	contentData.Message = nodeContribution.Status.Message

	// Set the HTML template variables
	/*contentData.CommonData.Tenant = userRow.GetNamespace()
	contentData.CommonData.Username = userRow.GetName()
	contentData.CommonData.Name = fmt.Sprintf("%s %s", userRow.Spec.FirstName, userRow.Spec.LastName)
	contentData.CommonData.Email = []string{userRow.Spec.Email}
	if contentData.Status == failure {
		mailer.Send("node-contribution-failure", contentData)
	} else if contentData.Status == success {
		mailer.Send("node-contribution-successful", contentData)
	}

	if contentData.Status == failure {
		mailer.Send("node-contribution-failure-support", contentData)
	}*/

}

// balanceMultiThreading is a simple algorithm to limit concurrent threads
func (t *Handler) balanceMultiThreading(limit int) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
check:
	for ; true; <-ticker.C {
		var threads int
		ncRaw, err := t.edgenetClientset.CoreV1alpha().NodeContributions().List(context.TODO(), metav1.ListOptions{})
		if err == nil {
			for _, ncRow := range ncRaw.Items {
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

// runSetupProcedure installs necessary packages from scratch and makes the node join into the cluster
func (t *Handler) runSetupProcedure(tenantName, addr, nodeName, recordType string, config *ssh.ClientConfig,
	nodeContribution *corev1alpha.NodeContribution) error {
	// Steps in the procedure
	endProcedure := make(chan bool, 1)
	dnsConfiguration := make(chan bool, 1)
	installation := make(chan bool, 1)
	nodePatch := make(chan bool, 1)
	// Set the status as recovering
	nodeContribution.Status.State = inprogress
	nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Installation procedure has started")
	nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
	if err == nil {
		nodeContribution = nodeContributionUpdated
	}
	// Start DNS configuration of `edge-net.io`
	dnsConfiguration <- true
	// This statement to organize tasks and put a general timeout on
nodeInstallLoop:
	for {
		select {
		case <-dnsConfiguration:
			log.Println("***************DNS Configuration***************")
			// Use Namecheap API for registration
			hostRecord := namecheap.DomainDNSHost{
				Name:    strings.TrimSuffix(nodeName, ".edge-net.io"),
				Type:    recordType,
				Address: nodeContribution.Spec.Host,
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
				nodeContribution.Status.State = incomplete
				nodeContribution.Status.Message = append(nodeContribution.Status.Message, hostnameError)
				nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
				if err == nil {
					nodeContribution = nodeContributionUpdated
				}
				log.Println(hostnameError)
			}
			installation <- true
		case <-installation:
			log.Println("***************Installation***************")
			// To prevent hanging forever during establishing a connection
			go func() {
				// SSH into the node
				conn, err := ssh.Dial("tcp", addr, config)
				if err != nil {
					log.Println(err)
					nodeContribution.Status.State = failure
					nodeContribution.Status.Message = append(nodeContribution.Status.Message, "SSH handshake failed")
					nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
					log.Println(err)
					if err == nil {
						nodeContribution = nodeContributionUpdated
					}
					endProcedure <- true
					return
				}
				defer conn.Close()
				// Uninstall all existing packages related, do a clean installation, and make the node join to the cluster
				err = t.cleanInstallation(conn, nodeName, nodeContribution)
				if err != nil {
					nodeContribution.Status.State = failure
					nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Node installation failed")
					nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
					log.Println(err)
					if err == nil {
						nodeContribution = nodeContributionUpdated
					}
					endProcedure <- true
					return
				}
				_, err = t.clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
				if err == nil {
					nodePatch <- true
				}
			}()
		case <-nodePatch:
			log.Println("***************Node Patch***************")
			// Set the node as schedulable or unschedulable according to the node contribution
			patchStatus := true
			err := node.SetNodeScheduling(nodeName, !nodeContribution.Spec.Enabled)
			if err != nil {
				nodeContribution.Status.State = incomplete
				nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Scheduling configuration failed")
				t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
				t.sendEmail(nodeContribution)
				patchStatus = false
			}
			var ownerReferences []metav1.OwnerReference
			ncTenant, err := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			if err == nil {
				ownerReferences = tenant.SetAsOwnerReference(ncTenant)
			}
			err = node.SetOwnerReferences(nodeName, ownerReferences)
			if err != nil {
				nodeContribution.Status.State = incomplete
				nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Setting owner reference failed")
				t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
				t.sendEmail(nodeContribution)
				patchStatus = false
			}
			if patchStatus {
				break nodeInstallLoop
			}
			nodeContribution.Status.State = success
			nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Node installation successful")
			t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
			endProcedure <- true
		case <-endProcedure:
			log.Println("***************Procedure Terminated***************")
			t.sendEmail(nodeContribution)
			break nodeInstallLoop
		case <-time.After(25 * time.Minute):
			log.Println("***************Timeout***************")
			// Terminate the procedure after 25 minutes
			nodeContribution.Status.State = failure
			nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Node installation failed: timeout")
			nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
			log.Println(err)
			if err == nil {
				nodeContribution = nodeContributionUpdated
			}
			t.sendEmail(nodeContribution)
			break nodeInstallLoop
		}
	}
	return err
}

// runRecoveryProcedure applies predefined methods to recover the node
func (t *Handler) runRecoveryProcedure(addr string, config *ssh.ClientConfig,
	nodeName string, nodeContribution *corev1alpha.NodeContribution, contributedNode *corev1.Node) {
	// Steps in the procedure
	endProcedure := make(chan bool, 1)
	establishConnection := make(chan bool, 1)
	installation := make(chan bool, 1)
	reboot := make(chan bool, 1)
	// Set the status as recovering
	nodeContribution.Status.State = recover
	nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Node recovering")
	nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
	if err == nil {
		nodeContribution = nodeContributionUpdated
	}
	// Watch the events of node object
	watchNode, err := t.clientset.CoreV1().Nodes().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", contributedNode.GetName())})
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
						nodeContribution.Status.State = success
						nodeContribution.Status.Message = append([]string{}, "Node recovery successful")
						nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
						log.Println(err)
						if err == nil {
							nodeContribution = nodeContributionUpdated
						}
						endProcedure <- true
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching node resources,
		// terminate the function
		endProcedure <- true
	}

	var conn *ssh.Client
	go func() {
		conn, err = ssh.Dial("tcp", addr, config)
		if err != nil {
			log.Println(err)
			nodeContribution.Status.State = failure
			nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Node recovery failed: SSH handshake failed")
			nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
			log.Println(err)
			if err == nil {
				nodeContribution = nodeContributionUpdated
			}
			endProcedure <- true
		} else {
			reboot <- true
		}
	}()

	// connCounter to try establishing a connection for several times when the node is rebooted
	connCounter := 0

	// This statement to organize tasks and put a general timeout on
nodeRecoveryLoop:
	for {
		select {
		case <-establishConnection:
			log.Printf("***************Establish Connection***************%s", nodeName)
			go func() {
				// SSH into the node
				conn, err = ssh.Dial("tcp", addr, config)
				if err != nil && connCounter < 3 {
					log.Println(err)
					// Wait three minutes to try establishing a connection again
					time.Sleep(3 * time.Minute)
					establishConnection <- true
					connCounter++
				} else if err != nil && connCounter >= 3 {
					nodeContribution.Status.State = failure
					nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Node recovery failed: SSH handshake failed")
					nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
					log.Println(err)
					if err == nil {
						nodeContribution = nodeContributionUpdated
					}
					<-endProcedure
					return
				}
				installation <- true
			}()
		case <-installation:
			log.Println("***************Installation***************")
			// Uninstall all existing packages related, do a clean installation, and make the node join to the cluster
			err := t.cleanInstallation(conn, nodeName, nodeContribution)
			if err != nil {
				nodeContribution.Status.State = failure
				nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Node recovery failed: installation step")
				nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
				log.Println(err)
				if err == nil {
					nodeContribution = nodeContributionUpdated
				}
				t.sendEmail(nodeContribution)
				watchNode.Stop()
				break nodeRecoveryLoop
			}
		case <-reboot:
			log.Println("***************Reboot***************")
			// Reboot the node in a minute
			err = rebootNode(conn)
			if err != nil {
				nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Node recovery failed: reboot step")
				nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
				log.Println(err)
				if err == nil {
					nodeContribution = nodeContributionUpdated
				}
			}
			conn.Close()
			time.Sleep(3 * time.Minute)
			establishConnection <- true
		case <-endProcedure:
			log.Println("***************Procedure Terminated***************")
			t.sendEmail(nodeContribution)
			watchNode.Stop()
			break nodeRecoveryLoop
		case <-time.After(25 * time.Minute):
			log.Println("***************Timeout***************")
			// Terminate the procedure after 25 minutes
			nodeContribution.Status.State = failure
			nodeContribution.Status.Message = append(nodeContribution.Status.Message, "Node recovery failed: timeout")
			nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
			log.Println(err)
			if err == nil {
				nodeContribution = nodeContributionUpdated
			}
			t.sendEmail(nodeContribution)
			watchNode.Stop()
			break nodeRecoveryLoop
		}
	}
	if conn != nil {
		conn.Close()
	}
}

// cleanInstallation gets and runs the uninstallation and installation commands prepared
func (t *Handler) cleanInstallation(conn *ssh.Client, nodeName string, nodeContribution *corev1alpha.NodeContribution) error {
	commands := []string{
		"sudo su",
		"kubeadm reset -f",
		node.CreateJoinToken("30m", nodeName),
	}
	sess, err := startSession(conn)
	if err != nil {
		log.Println(err)
		return err
	}
	defer sess.Close()
	// StdinPipe for commands
	stdin, err := sess.StdinPipe()
	if err != nil {
		log.Println(err)
		return err
	}
	//sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	sess, err = startShell(sess)
	if err != nil {
		log.Println(err)
		return err
	}
	// Run commands sequentially
	for _, cmd := range commands {
		_, err = fmt.Fprintf(stdin, "%s\n", cmd)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	stdin.Close()
	// Wait for session to finish
	err = sess.Wait()
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// rebootNode restarts node after a minute
func rebootNode(conn *ssh.Client) error {
	sess, err := startSession(conn)
	if err != nil {
		log.Println(err)
		return err
	}
	defer sess.Close()
	err = sess.Run("sudo shutdown -r +1")
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// Start a new session in the connection
func startSession(conn *ssh.Client) (*ssh.Session, error) {
	sess, err := conn.NewSession()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return sess, nil
}

// Start a shell in the session
func startShell(sess *ssh.Session) (*ssh.Session, error) {
	// Start remote shell
	if err := sess.Shell(); err != nil {
		log.Println(err)
		return nil, err
	}
	return sess, nil
}
