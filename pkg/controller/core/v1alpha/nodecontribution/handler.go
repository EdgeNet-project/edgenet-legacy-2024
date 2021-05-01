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

	apps_v1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/apps/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/controller/v1alpha/authority"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	"github.com/EdgeNet-project/edgenet/pkg/mailer"
	ns "github.com/EdgeNet-project/edgenet/pkg/namespace"
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
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
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
	key, err := ioutil.ReadFile("../../.ssh/id_rsa")
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

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("NCHandler.ObjectCreated")
	// Create a copy of the node contribution object to make changes on it
	ncCopy := obj.(*corev1alpha.NodeContribution).DeepCopy()
	ncCopy.Status.Message = []string{}
	// Find the authority from the namespace in which the object is
	NCOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), ncCopy.GetNamespace(), metav1.GetOptions{})
	nodeName := fmt.Sprintf("%s.%s.edge-net.io", NCOwnerNamespace.Labels["authority-name"], ncCopy.GetName())
	// Don't use the authority name if the node belongs to EdgeNet
	if NCOwnerNamespace.GetName() == "authority-edgenet" {
		nodeName = fmt.Sprintf("%s.edge-net.io", ncCopy.GetName())
	}
	NCOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), NCOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	authorityEnabled := NCOwnerAuthority.Spec.Enabled
	log.Println("AUTHORITY CHECK")
	// Check if the authority is active
	if authorityEnabled {
		log.Println("AUTHORITY ENABLED")
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		// Check whether the host has been given as an IP address or else
		recordType := remoteip.GetRecordType(ncCopy.Spec.Host)
		if recordType == "" {
			ncCopy.Status.State = failure
			ncCopy.Status.Message = append(ncCopy.Status.Message, statusDict["invalid-host"])
			t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
			t.sendEmail(ncCopy)
			return
		}
		// Set the client config according to the node contribution,
		// with the maximum time of 15 seconds to establist the connection.
		config := &ssh.ClientConfig{
			User:            ncCopy.Spec.User,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(t.publicKey), ssh.Password(ncCopy.Spec.Password)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         15 * time.Second,
		}
		addr := fmt.Sprintf("%s:%d", ncCopy.Spec.Host, ncCopy.Spec.Port)
		contributedNode, err := t.clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err == nil {
			// The node corresponding to the contributed node exists in the cluster
			log.Println("NODE FOUND")
			if node.GetConditionReadyStatus(contributedNode.DeepCopy()) != trueStr {
				t.balanceMultiThreading(5)
				go t.runRecoveryProcedure(addr, config, nodeName, ncCopy, contributedNode)
			} else {
				ncCopy.Status.State = success
				ncCopy.Status.Message = append(ncCopy.Status.Message, statusDict["node-ok"])
				t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
			}
		} else {
			// There isn't any node corresponding to the node contribution
			log.Println("NODE NOT FOUND")
			t.balanceMultiThreading(5)
			go t.runSetupProcedure(NCOwnerNamespace.Labels["authority-name"], addr, nodeName, recordType, config, ncCopy)
		}
	} else {
		log.Println("AUTHORITY NOT ENABLED")
		// Disable scheduling on the node if the authority is disabled
		ncCopy.Spec.Enabled = false
		ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).Update(context.TODO(), ncCopy, metav1.UpdateOptions{})
		if err == nil {
			ncCopy = ncCopyUpdated
			ncCopy.Status.State = failure
			ncCopy.Status.Message = append(ncCopy.Status.Message, statusDict["authority-disabled"])
			t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
		}
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("NCHandler.ObjectUpdated")
	// Create a copy of the node contribution object to make changes on it
	ncCopy := obj.(*corev1alpha.NodeContribution).DeepCopy()
	ncCopy.Status.Message = []string{}
	NCOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(context.TODO(), ncCopy.GetNamespace(), metav1.GetOptions{})
	nodeName := fmt.Sprintf("%s.%s.edge-net.io", NCOwnerNamespace.Labels["authority-name"], ncCopy.GetName())
	if NCOwnerNamespace.GetName() == "authority-edgenet" {
		nodeName = fmt.Sprintf("%s.edge-net.io", ncCopy.GetName())
	}
	NCOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), NCOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
	authorityEnabled := NCOwnerAuthority.Spec.Enabled
	log.Println("AUTHORITY CHECK")
	// Check if the authority is active
	if authorityEnabled {
		log.Println("AUTHORITY ENABLED")
		recordType := remoteip.GetRecordType(ncCopy.Spec.Host)
		if recordType == "" {
			ncCopy.Status.State = failure
			ncCopy.Status.Message = append(ncCopy.Status.Message, statusDict["invalid-host"])
			t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
			t.sendEmail(ncCopy)
			return
		}
		config := &ssh.ClientConfig{
			User:            ncCopy.Spec.User,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(t.publicKey), ssh.Password(ncCopy.Spec.Password)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         15 * time.Second,
		}
		addr := fmt.Sprintf("%s:%d", ncCopy.Spec.Host, ncCopy.Spec.Port)
		contributedNode, err := t.clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err == nil {
			log.Println("NODE FOUND")
			if contributedNode.Spec.Unschedulable != !ncCopy.Spec.Enabled {
				node.SetNodeScheduling(nodeName, !ncCopy.Spec.Enabled)
			}
			if node.GetConditionReadyStatus(contributedNode.DeepCopy()) != trueStr {
				t.balanceMultiThreading(5)
				go t.runRecoveryProcedure(addr, config, nodeName, ncCopy, contributedNode)
			} else {
				ncCopy.Status.State = success
				ncCopy.Status.Message = append(ncCopy.Status.Message, statusDict["node-ok"])
				t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
			}
		} else {
			log.Println("NODE NOT FOUND")
			t.balanceMultiThreading(5)
			go t.runSetupProcedure(NCOwnerNamespace.Labels["authority-name"], addr, nodeName, recordType, config, ncCopy)
		}
	} else {
		log.Println("AUTHORITY NOT ENABLED")
		ncCopy.Spec.Enabled = false
		ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).Update(context.TODO(), ncCopy, metav1.UpdateOptions{})
		if err == nil {
			ncCopy = ncCopyUpdated
			ncCopy.Status.State = failure
			ncCopy.Status.Message = append(ncCopy.Status.Message, "Authority disabled")
			t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("NCHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to participants
func (t *Handler) sendEmail(ncCopy *corev1alpha.NodeContribution) {
	// For those who are authority-admin and authorized users of the authority
	userRaw, err := t.edgenetClientset.AppsV1alpha().Users(ncCopy.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		contentData := mailer.MultiProviderData{}
		contentData.Name = ncCopy.GetName()
		contentData.Host = ncCopy.Spec.Host
		contentData.Status = ncCopy.Status.State
		contentData.Message = ncCopy.Status.Message
		for _, userRow := range userRaw.Items {
			if userRow.Spec.Active && userRow.Status.AUP && userRow.Status.Type == "admin" {
				if err == nil && userRow.Spec.Active && userRow.Status.AUP {
					// Set the HTML template variables
					contentData.CommonData.Authority = userRow.GetNamespace()
					contentData.CommonData.Username = userRow.GetName()
					contentData.CommonData.Name = fmt.Sprintf("%s %s", userRow.Spec.FirstName, userRow.Spec.LastName)
					contentData.CommonData.Email = []string{userRow.Spec.Email}
					if contentData.Status == failure {
						mailer.Send("node-contribution-failure", contentData)
					} else if contentData.Status == success {
						mailer.Send("node-contribution-successful", contentData)
					}
				}
			}
		}
		if contentData.Status == failure {
			mailer.Send("node-contribution-failure-support", contentData)
		}
	}
}

// balanceMultiThreading is a simple algorithm to limit concurrent threads
func (t *Handler) balanceMultiThreading(limit int) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
check:
	for ; true; <-ticker.C {
		var threads int
		ncRaw, err := t.edgenetClientset.AppsV1alpha().NodeContributions("").List(context.TODO(), metav1.ListOptions{})
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
func (t *Handler) runSetupProcedure(authorityName, addr, nodeName, recordType string, config *ssh.ClientConfig,
	ncCopy *corev1alpha.NodeContribution) error {
	// Steps in the procedure
	endProcedure := make(chan bool, 1)
	dnsConfiguration := make(chan bool, 1)
	installation := make(chan bool, 1)
	nodePatch := make(chan bool, 1)
	// Set the status as recovering
	ncCopy.Status.State = inprogress
	ncCopy.Status.Message = append(ncCopy.Status.Message, "Installation procedure has started")
	ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
	if err == nil {
		ncCopy = ncCopyUpdated
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
				Address: ncCopy.Spec.Host,
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
				ncCopy.Status.State = incomplete
				ncCopy.Status.Message = append(ncCopy.Status.Message, hostnameError)
				ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
				if err == nil {
					ncCopy = ncCopyUpdated
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
					ncCopy.Status.State = failure
					ncCopy.Status.Message = append(ncCopy.Status.Message, "SSH handshake failed")
					ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
					log.Println(err)
					if err == nil {
						ncCopy = ncCopyUpdated
					}
					endProcedure <- true
					return
				}
				defer conn.Close()
				// Uninstall all existing packages related, do a clean installation, and make the node join to the cluster
				err = t.cleanInstallation(conn, nodeName, ncCopy)
				if err != nil {
					ncCopy.Status.State = failure
					ncCopy.Status.Message = append(ncCopy.Status.Message, "Node installation failed")
					ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
					log.Println(err)
					if err == nil {
						ncCopy = ncCopyUpdated
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
			err := node.SetNodeScheduling(nodeName, !ncCopy.Spec.Enabled)
			if err != nil {
				ncCopy.Status.State = incomplete
				ncCopy.Status.Message = append(ncCopy.Status.Message, "Scheduling configuration failed")
				t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
				t.sendEmail(ncCopy)
				patchStatus = false
			}
			var ownerReferences []metav1.OwnerReference
			authorityCopy, err := t.edgenetClientset.AppsV1alpha().Authorities().Get(context.TODO(), authorityName, metav1.GetOptions{})
			if err == nil {
				ownerReferences = authority.SetAsOwnerReference(authorityCopy)
			}
			NCOwnerNamespace, err := t.clientset.CoreV1().Namespaces().Get(context.TODO(), fmt.Sprintf("authority-%s", authorityName), metav1.GetOptions{})
			if err == nil {
				ownerReferences = append(ownerReferences, ns.SetAsOwnerReference(NCOwnerNamespace)...)
			}
			err = node.SetOwnerReferences(nodeName, ownerReferences)
			if err != nil {
				ncCopy.Status.State = incomplete
				ncCopy.Status.Message = append(ncCopy.Status.Message, "Setting owner reference failed")
				t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
				t.sendEmail(ncCopy)
				patchStatus = false
			}
			if patchStatus {
				break nodeInstallLoop
			}
			ncCopy.Status.State = success
			ncCopy.Status.Message = append(ncCopy.Status.Message, "Node installation successful")
			t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
			endProcedure <- true
		case <-endProcedure:
			log.Println("***************Procedure Terminated***************")
			t.sendEmail(ncCopy)
			break nodeInstallLoop
		case <-time.After(25 * time.Minute):
			log.Println("***************Timeout***************")
			// Terminate the procedure after 25 minutes
			ncCopy.Status.State = failure
			ncCopy.Status.Message = append(ncCopy.Status.Message, "Node installation failed: timeout")
			ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
			log.Println(err)
			if err == nil {
				ncCopy = ncCopyUpdated
			}
			t.sendEmail(ncCopy)
			break nodeInstallLoop
		}
	}
	return err
}

// runRecoveryProcedure applies predefined methods to recover the node
func (t *Handler) runRecoveryProcedure(addr string, config *ssh.ClientConfig,
	nodeName string, ncCopy *corev1alpha.NodeContribution, contributedNode *corev1.Node) {
	// Steps in the procedure
	endProcedure := make(chan bool, 1)
	establishConnection := make(chan bool, 1)
	installation := make(chan bool, 1)
	reboot := make(chan bool, 1)
	// Set the status as recovering
	ncCopy.Status.State = recover
	ncCopy.Status.Message = append(ncCopy.Status.Message, "Node recovering")
	ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
	if err == nil {
		ncCopy = ncCopyUpdated
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
						ncCopy.Status.State = success
						ncCopy.Status.Message = append([]string{}, "Node recovery successful")
						ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
						log.Println(err)
						if err == nil {
							ncCopy = ncCopyUpdated
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
			ncCopy.Status.State = failure
			ncCopy.Status.Message = append(ncCopy.Status.Message, "Node recovery failed: SSH handshake failed")
			ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
			log.Println(err)
			if err == nil {
				ncCopy = ncCopyUpdated
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
					ncCopy.Status.State = failure
					ncCopy.Status.Message = append(ncCopy.Status.Message, "Node recovery failed: SSH handshake failed")
					ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
					log.Println(err)
					if err == nil {
						ncCopy = ncCopyUpdated
					}
					<-endProcedure
					return
				}
				installation <- true
			}()
		case <-installation:
			log.Println("***************Installation***************")
			// Uninstall all existing packages related, do a clean installation, and make the node join to the cluster
			err := t.cleanInstallation(conn, nodeName, ncCopy)
			if err != nil {
				ncCopy.Status.State = failure
				ncCopy.Status.Message = append(ncCopy.Status.Message, "Node recovery failed: installation step")
				ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
				log.Println(err)
				if err == nil {
					ncCopy = ncCopyUpdated
				}
				t.sendEmail(ncCopy)
				watchNode.Stop()
				break nodeRecoveryLoop
			}
		case <-reboot:
			log.Println("***************Reboot***************")
			// Reboot the node in a minute
			err = rebootNode(conn)
			if err != nil {
				ncCopy.Status.Message = append(ncCopy.Status.Message, "Node recovery failed: reboot step")
				ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
				log.Println(err)
				if err == nil {
					ncCopy = ncCopyUpdated
				}
			}
			conn.Close()
			time.Sleep(3 * time.Minute)
			establishConnection <- true
		case <-endProcedure:
			log.Println("***************Procedure Terminated***************")
			t.sendEmail(ncCopy)
			watchNode.Stop()
			break nodeRecoveryLoop
		case <-time.After(25 * time.Minute):
			log.Println("***************Timeout***************")
			// Terminate the procedure after 25 minutes
			ncCopy.Status.State = failure
			ncCopy.Status.Message = append(ncCopy.Status.Message, "Node recovery failed: timeout")
			ncCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(ncCopy.GetNamespace()).UpdateStatus(context.TODO(), ncCopy, metav1.UpdateOptions{})
			log.Println(err)
			if err == nil {
				ncCopy = ncCopyUpdated
			}
			t.sendEmail(ncCopy)
			watchNode.Stop()
			break nodeRecoveryLoop
		}
	}
	if conn != nil {
		conn.Close()
	}
}

// cleanInstallation gets and runs the uninstallation and installation commands prepared
func (t *Handler) cleanInstallation(conn *ssh.Client, nodeName string, ncCopy *corev1alpha.NodeContribution) error {
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
