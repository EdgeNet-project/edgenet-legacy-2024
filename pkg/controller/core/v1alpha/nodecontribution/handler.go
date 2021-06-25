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

	// Get the SSH Private Key of the headnode
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
		if contributedNode.Spec.Unschedulable != !nodeContribution.Spec.Enabled {
			node.SetNodeScheduling(nodeName, !nodeContribution.Spec.Enabled)
		}
		if node.GetConditionReadyStatus(contributedNode.DeepCopy()) != trueStr {
			t.balanceMultiThreading(5)
			go t.setup(nodeContribution.Spec.Tenant, addr, nodeName, recordType, "recovery", config, nodeContribution)
		} else {
			nodeContribution.Status.State = success
			nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["succesful"])
			t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
		}
	} else {
		t.balanceMultiThreading(5)
		go t.setup(nodeContribution.Spec.Tenant, addr, nodeName, recordType, "initial", config, nodeContribution)
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("NCHandler.ObjectDeleted")
	// Mail notification, TBD
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

// setup registers DNS record and makes the node join into the cluster
func (t *Handler) setup(tenantName, addr, nodeName, recordType, procedure string, config *ssh.ClientConfig, nodeContribution *corev1alpha.NodeContribution) error {
	// Steps in the procedure
	endProcedure := make(chan bool, 1)
	dnsConfiguration := make(chan bool, 1)
	establishConnection := make(chan bool, 1)
	setup := make(chan bool, 1)
	nodePatch := make(chan bool, 1)
	reboot := make(chan bool, 1)
	// Set the status as recovering
	nodeContribution.Status.State = inprogress
	nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["in-progress"])
	nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
	if err == nil {
		nodeContribution = nodeContributionUpdated
	}

	var conn *ssh.Client
	// connCounter to try establishing a connection for several times when the node is rebooted
	connCounter := 0
	if procedure == "recovery" {
		// Watch the events of node object
		watchNode, err := t.clientset.CoreV1().Nodes().Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", nodeName)})
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
							nodeContribution.Status.State = success
							nodeContribution.Status.Message = append([]string{}, statusDict["successful"])
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
		}

		go func() {
			conn, err = ssh.Dial("tcp", addr, config)
			if err != nil {
				log.Println(err)
				nodeContribution.Status.State = failure
				nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["ssh-failure"])
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
	} else {
		// Start DNS configuration of `edge-net.io`
		dnsConfiguration <- true
	}
	// This statement to organize tasks and put a general timeout on
nodeSetupLoop:
	for {
		select {
		case <-dnsConfiguration:
			log.Printf("DNS configuration started: %s", nodeName)
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
			establishConnection <- true
		case <-establishConnection:
			log.Printf("Establish SSH connection: %s", nodeName)
			go func() {
				conn, err = ssh.Dial("tcp", addr, config)
				if err != nil && connCounter < 3 {
					log.Println(err)
					// Wait three minutes to try establishing a connection again
					time.Sleep(3 * time.Minute)
					establishConnection <- true
					connCounter++
				} else if err != nil && connCounter >= 3 {
					nodeContribution.Status.State = failure
					nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["ssh-failure"])
					nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
					log.Println(err)
					if err == nil {
						nodeContribution = nodeContributionUpdated
					}
					endProcedure <- true
					return
				}
				setup <- true
			}()
		case <-setup:
			log.Printf("Create a token and run kubadm join: %s", nodeName)
			// To prevent hanging forever during establishing a connection
			go func() {
				defer func() {
					if conn != nil {
						conn.Close()
					}
				}()
				err = t.join(conn, nodeName, nodeContribution)
				if err != nil {
					nodeContribution.Status.State = failure
					nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["join-failure"])
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
			log.Printf("Patch scheduling option: %s", nodeName)
			// Set the node as schedulable or unschedulable according to the node contribution
			err := node.SetNodeScheduling(nodeName, !nodeContribution.Spec.Enabled)
			if err != nil {
				nodeContribution.Status.State = incomplete
				nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["configuration-failure"])
				t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
				endProcedure <- true
			}
			ncTenant, err := t.edgenetClientset.CoreV1alpha().Tenants().Get(context.TODO(), tenantName, metav1.GetOptions{})
			if err == nil {
				ownerReferences := tenant.SetAsOwnerReference(ncTenant)
				err = node.SetOwnerReferences(nodeName, ownerReferences)
				if err != nil {
					nodeContribution.Status.State = incomplete
					nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["owner-reference-failure"])
					t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
					endProcedure <- true
				}
			}
			nodeContribution.Status.State = success
			nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["successful"])
			t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
			if procedure == "initial" {
				endProcedure <- true
			}
		case <-reboot:
			log.Printf("Reboot the node: %s", nodeName)
			// Reboot the node in a minute
			err = rebootNode(conn)
			if err != nil {
				nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["reboot-failure"])
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
			log.Printf("Procedure completed: %s", nodeName)
			break nodeSetupLoop
		case <-time.After(5 * time.Minute):
			log.Printf("Timeout: %s", nodeName)
			// Terminate the procedure after 5 minutes
			nodeContribution.Status.State = failure
			nodeContribution.Status.Message = append(nodeContribution.Status.Message, statusDict["timeout"])
			nodeContributionUpdated, err := t.edgenetClientset.CoreV1alpha().NodeContributions().UpdateStatus(context.TODO(), nodeContribution, metav1.UpdateOptions{})
			log.Println(err)
			if err == nil {
				nodeContribution = nodeContributionUpdated
			}
			break nodeSetupLoop
		}
	}
	return err
}

// join creates a token and runs kubeadm join command
func (t *Handler) join(conn *ssh.Client, nodeName string, nodeContribution *corev1alpha.NodeContribution) error {
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
