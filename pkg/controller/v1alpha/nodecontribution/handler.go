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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	apps_v1alpha "headnode/pkg/apis/apps/v1alpha"
	"headnode/pkg/authorization"
	"headnode/pkg/client/clientset/versioned"
	"headnode/pkg/mailer"
	"headnode/pkg/node"

	log "github.com/Sirupsen/logrus"
	namecheap "github.com/billputer/go-namecheap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// HandlerInterface interface contains the methods that are required
type HandlerInterface interface {
	Init() error
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

// Handler implementation
type Handler struct {
	clientset        *kubernetes.Clientset
	edgenetClientset *versioned.Clientset
	publicKey        ssh.Signer
}

// Init handles any handler initialization
func (t *Handler) Init() error {
	log.Info("NCHandler.Init")
	var err error
	t.clientset, err = authorization.CreateClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
	t.edgenetClientset, err = authorization.CreateEdgeNetClientSet()
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}
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
	return err
}

// ObjectCreated is called when an object is created
func (t *Handler) ObjectCreated(obj interface{}) {
	log.Info("NCHandler.ObjectCreated")
	// Create a copy of the node contribution object to make changes on it
	NCCopy := obj.(*apps_v1alpha.NodeContribution).DeepCopy()
	// Find the authority from the namespace in which the object is
	NCOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(NCCopy.GetNamespace(), metav1.GetOptions{})
	// If the object's kind is AuthorityRequest, `registration` namespace hosts the node contribution object.
	// Otherwise, the object belongs to the namespace that the authority created.
	nodeName := fmt.Sprintf("%s.%s.edge-net.io", NCOwnerNamespace.Labels["authority-name"], NCCopy.GetName())
	var authorityEnabled bool
	if NCOwnerNamespace.GetName() == "authority-edgenet" {
		nodeName = fmt.Sprintf("%s.edge-net.io", NCCopy.GetName())
		authorityEnabled = true
	} else {
		NCOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(NCOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
		authorityEnabled = NCOwnerAuthority.Status.Enabled
	}
	log.Println("AUTHORITY CHECK")
	// Check if the authority is active
	if authorityEnabled {
		log.Println("AUTHORITY ENABLED")
		// If the service restarts, it creates all objects again
		// Because of that, this section covers a variety of possibilities
		config := &ssh.ClientConfig{
			User:            NCCopy.Spec.Username,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(t.publicKey), ssh.Password(NCCopy.Spec.Password)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		addr := fmt.Sprintf("%s:%d", NCCopy.Spec.Host, NCCopy.Spec.Port)
		// SSH into the node
		conn, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			log.Println(err)
			NCCopy.Status.State = failure
			NCCopy.Status.Message = "SSH handshake failed"
			t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
			return
		}
		defer conn.Close()
		contributedNode, err := t.clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err == nil {
			log.Println("NODE FOUND")
			if node.GetConditionReadyStatus(contributedNode.DeepCopy()) != trueStr {
				recordType := getRecordType(NCCopy.Spec.Host)
				if recordType == "" {
					NCCopy.Status.State = failure
					NCCopy.Status.Message = "Host field must be an IP Address"
					t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
					return
				}
				go t.startRecoveringProcedure(addr, config, nodeName, NCCopy, contributedNode)
				NCCopy.Status.State = recover
				NCCopy.Status.Message = "Node recovering"
				t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
			}
		} else {
			log.Println("NODE NOT FOUND")
			recordType := getRecordType(NCCopy.Spec.Host)
			if recordType == "" {
				NCCopy.Status.State = failure
				NCCopy.Status.Message = "Host field must be an IP Address"
				t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
				return
			}
			hostRecord := namecheap.DomainDNSHost{
				Name:    nodeName,
				Type:    recordType,
				Address: NCCopy.Spec.Host,
			}
			result, state := node.SetHostname(hostRecord)
			if !result {
				if state == "exist" {
					log.Printf("Error: Hostname %s or address %s already exists", hostRecord.Name, hostRecord.Address)
				} else {
					log.Printf("Error: Hostname %s or address %s couldn't added", hostRecord.Name, hostRecord.Address)
				}
			}

			err := t.cleanInstallation(conn, nodeName, NCCopy)
			if err != nil {
				log.Println(err)
				NCCopy.Status.State = failure
				NCCopy.Status.Message = "Node installation failed"
				t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
				return
			}
			// Create a patch slice and initialize it to the label size
			nodePatchArr := make([]interface{}, 1)
			//nodePatchByOwnerReferences := patchByOwnerReferenceValue{}
			nodePatchByBool := patchByBoolValue{}
			/*nodePatchOwnerReference := patchOwnerReference{}
			nodePatchOwnerReference.APIVersion = "apps.edgenet.io/v1alpha"
			nodePatchOwnerReference.BlockOwnerDeletion = true
			nodePatchOwnerReference.Controller = false
			nodePatchOwnerReference.Kind = "Namespace"
			nodePatchOwnerReference.Name = NCOwnerNamespace.GetName()
			nodePatchOwnerReference.UID = string(NCOwnerNamespace.GetUID())
			nodePatchOwnerReferences := append([]patchOwnerReference{}, nodePatchOwnerReference)

			// Append the data existing in the label map to the slice
			nodePatchByOwnerReferences.Op = "add"
			nodePatchByOwnerReferences.Path = "/metadata/ownerReferences"
			nodePatchByOwnerReferences.Value = nodePatchOwnerReferences
			nodePatchArr[0] = nodePatchByOwnerReferences*/
			nodePatchByBool.Op = "replace"
			nodePatchByBool.Path = "/spec/unschedulable"
			nodePatchByBool.Value = !NCCopy.Spec.Enabled
			nodePatchArr[0] = nodePatchByBool

			nodePatchJSON, _ := json.Marshal(nodePatchArr)
			// Patch the nodes with the arguments:
			// hostname, patch type, and patch data
			_, err = t.clientset.CoreV1().Nodes().Patch(nodeName, types.JSONPatchType, nodePatchJSON)
			if err != nil {
				log.Println(err.Error())
			} else {
				NCCopy.Status.State = success
				NCCopy.Status.Message = "Node installation successful"
				t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
			}
		}
	} else {
		log.Println("AUTHORITY NOT ENABLED")
		NCCopy.Spec.Enabled = false
		NCCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).Update(NCCopy)
		if err == nil {
			NCCopy = NCCopyUpdated
			NCCopy.Status.State = failure
			NCCopy.Status.Message = "Authority disabled"
			t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
		}
	}
}

// ObjectUpdated is called when an object is updated
func (t *Handler) ObjectUpdated(obj interface{}) {
	log.Info("NCHandler.ObjectUpdated")
	// Create a copy of the node contribution object to make changes on it
	NCCopy := obj.(*apps_v1alpha.NodeContribution).DeepCopy()
	NCOwnerNamespace, _ := t.clientset.CoreV1().Namespaces().Get(NCCopy.GetNamespace(), metav1.GetOptions{})
	nodeName := fmt.Sprintf("%s.%s.edge-net.io", NCOwnerNamespace.Labels["authority-name"], NCCopy.GetName())
	var authorityEnabled bool
	if NCOwnerNamespace.GetName() == "registration" {
		nodeName = fmt.Sprintf("%s.edge-net.io", NCCopy.GetName())
		authorityEnabled = true
	} else {
		NCOwnerAuthority, _ := t.edgenetClientset.AppsV1alpha().Authorities().Get(NCOwnerNamespace.Labels["authority-name"], metav1.GetOptions{})
		authorityEnabled = NCOwnerAuthority.Status.Enabled
	}
	log.Println("AUTHORITY CHECK")
	// Check whether the authority enabled
	if authorityEnabled {
		log.Println("AUTHORITY ENABLED")
		// Check whether the node contribution is done
		recordType := getRecordType(NCCopy.Spec.Host)
		if recordType == "" {
			NCCopy.Status.State = failure
			NCCopy.Status.Message = "Host field must be an IP Address"
			t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
			return
		}
		config := &ssh.ClientConfig{
			User:            NCCopy.Spec.Username,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(t.publicKey), ssh.Password(NCCopy.Spec.Password)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		addr := fmt.Sprintf("%s:%d", NCCopy.Spec.Host, NCCopy.Spec.Port)
		contributedNode, err := t.clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err == nil {
			log.Println("NODE FOUND")
			if contributedNode.Spec.Unschedulable != !NCCopy.Spec.Enabled {
				// Create a patch slice and initialize it to the label size
				nodePatchArr := make([]patchByBoolValue, 1)
				nodePatch := patchByBoolValue{}
				// Append the data existing in the label map to the slice
				nodePatch.Op = "replace"
				nodePatch.Path = "/spec/unschedulable"
				nodePatch.Value = !NCCopy.Spec.Enabled
				nodePatchArr[0] = nodePatch
				nodePatchJSON, _ := json.Marshal(nodePatchArr)
				// Patch the nodes with the arguments:
				// hostname, patch type, and patch data
				t.clientset.CoreV1().Nodes().Patch(contributedNode.GetName(), types.JSONPatchType, nodePatchJSON)
			}

			if NCCopy.Status.State == failure {
				go t.startRecoveringProcedure(addr, config, nodeName, NCCopy, contributedNode)
				NCCopy.Status.State = recover
				NCCopy.Status.Message = "Node recovering"
				t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
			}
		} else {
			log.Println("NODE NOT FOUND")
			conn, err := ssh.Dial("tcp", addr, config)
			if err != nil {
				log.Println(err)
				NCCopy.Status.State = failure
				NCCopy.Status.Message = "SSH handshake failed"
				t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
				return
			}
			defer conn.Close()
			hostRecord := namecheap.DomainDNSHost{
				Name:    nodeName,
				Type:    recordType,
				Address: NCCopy.Spec.Host,
			}
			result, state := node.SetHostname(hostRecord)
			if !result {
				if state == "exist" {
					log.Printf("Error: Hostname %s or address %s already exists", hostRecord.Name, hostRecord.Address)
				} else {
					log.Printf("Error: Hostname %s or address %s couldn't added", hostRecord.Name, hostRecord.Address)
				}
			}

			err = t.cleanInstallation(conn, nodeName, NCCopy)
			if err != nil {
				log.Println(err)
				NCCopy.Status.State = failure
				NCCopy.Status.Message = "Node installation failed"
				t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
				return
			}
			// Create a patch slice and initialize it to the label size
			nodePatchArr := make([]interface{}, 1)
			//nodePatchByOwnerReferences := patchByOwnerReferenceValue{}
			nodePatchByBool := patchByBoolValue{}
			/*nodePatchOwnerReference := patchOwnerReference{}
			nodePatchOwnerReference.APIVersion = "apps.edgenet.io/v1alpha"
			nodePatchOwnerReference.BlockOwnerDeletion = true
			nodePatchOwnerReference.Controller = false
			nodePatchOwnerReference.Kind = "Namespace"
			nodePatchOwnerReference.Name = NCOwnerNamespace.GetName()
			nodePatchOwnerReference.UID = string(NCOwnerNamespace.GetUID())
			nodePatchOwnerReferences := append([]patchOwnerReference{}, nodePatchOwnerReference)

			// Append the data existing in the label map to the slice
			nodePatchByOwnerReferences.Op = "add"
			nodePatchByOwnerReferences.Path = "/metadata/ownerReferences"
			nodePatchByOwnerReferences.Value = nodePatchOwnerReferences
			nodePatchArr[0] = nodePatchByOwnerReferences*/
			nodePatchByBool.Op = "replace"
			nodePatchByBool.Path = "/spec/unschedulable"
			nodePatchByBool.Value = !NCCopy.Spec.Enabled
			nodePatchArr[0] = nodePatchByBool

			nodePatchJSON, _ := json.Marshal(nodePatchArr)
			// Patch the nodes with the arguments:
			// hostname, patch type, and patch data
			_, err = t.clientset.CoreV1().Nodes().Patch(nodeName, types.JSONPatchType, nodePatchJSON)
			if err != nil {
				log.Println(err.Error())
			} else {
				NCCopy.Status.State = success
				NCCopy.Status.Message = "Node installation successful"
				t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
			}
		}
	} else {
		log.Println("AUTHORITY NOT ENABLED")

		NCCopy.Spec.Enabled = false
		NCCopyUpdated, err := t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).Update(NCCopy)
		if err == nil {
			NCCopy = NCCopyUpdated
			NCCopy.Status.State = failure
			NCCopy.Status.Message = "Authority disabled"
			t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
		}
	}
}

// ObjectDeleted is called when an object is deleted
func (t *Handler) ObjectDeleted(obj interface{}) {
	log.Info("NCHandler.ObjectDeleted")
	// Mail notification, TBD
}

// sendEmail to send notification to cluster admins and authority managers about node contribution
func (t *Handler) sendEmail(kind, authority, namespace, username, fullname string) {
	// Set the HTML template variables
	contentData := mailer.CommonContentData{}
	contentData.CommonData.Authority = authority
	contentData.CommonData.Username = username
	contentData.CommonData.Name = fullname
	contentData.CommonData.Email = []string{}
	if kind == "user-email-verified-alert" {
		// Put the email addresses of the authority-admins and managers in the email to be sent list
		userRaw, _ := t.edgenetClientset.AppsV1alpha().Users(namespace).List(metav1.ListOptions{})
		for _, userRow := range userRaw.Items {
			for _, userRole := range userRow.Spec.Roles {
				if strings.ToLower(userRole) == "admin" || strings.ToLower(userRole) == "manager" {
					contentData.CommonData.Email = append(contentData.CommonData.Email, userRow.Spec.Email)
				}
			}
		}
	}
	mailer.Send(kind, contentData)
}

func (t *Handler) startRecoveringProcedure(addr string, config *ssh.ClientConfig,
	nodeName string, NCCopy *apps_v1alpha.NodeContribution, contributedNode *corev1.Node) {
	endProcedure := make(chan bool, 1)
	startReboot := make(chan bool, 1)
	establishConnection := make(chan bool, 1)

	// Watch the events of node object
	watchNode, err := t.clientset.CoreV1().Nodes().Watch(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name==%s", contributedNode.GetName())})
	if err == nil {
		go func() {
			// Get events from watch interface
			for nodeEvent := range watchNode.ResultChan() {
				// Get updated email verification object
				updatedNode, status := nodeEvent.Object.(*corev1.Node)
				if status {
					if nodeEvent.Type == "DELETED" {
						continue
					}

					if node.GetConditionReadyStatus(updatedNode) == trueStr {
						endProcedure <- true
					}
				}
			}
		}()
	} else {
		// In case of any malfunction of watching emailverification resources,
		// terminate the function
		endProcedure <- true
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Println(err)
		return
	}
	rebootCounter := 0
	startReboot <- true

checkNodeRecovery:
	for {
		select {
		case <-startReboot:
			err = rebootNode(conn)
			if err != nil {
				log.Println(err)
				endProcedure <- true
			}
			conn.Close()
			time.Sleep(3 * time.Minute)
			establishConnection <- true
		case <-establishConnection:
			conn, err = ssh.Dial("tcp", addr, config)
			if err != nil && rebootCounter < 3 {
				log.Println(err)
				time.Sleep(3 * time.Minute)
				establishConnection <- true
				rebootCounter++
			} else if err == nil {
				defer conn.Close()
				err = reconfigureNode(conn, contributedNode.GetName())
				if err != nil {
					log.Println(err)
					endProcedure <- true
				}
				time.Sleep(5 * time.Minute)
				err := t.cleanInstallation(conn, nodeName, NCCopy)
				if err != nil {
					log.Println(err)
					endProcedure <- true
				}
				watchNode.Stop()
				break checkNodeRecovery
			} else {
				log.Println(err)
				endProcedure <- true
			}
		case <-endProcedure:
			NCCopy.Status.State = failure
			NCCopy.Status.Message = "Node recovery failed"
			t.edgenetClientset.AppsV1alpha().NodeContributions(NCCopy.GetNamespace()).UpdateStatus(NCCopy)
			watchNode.Stop()
			break checkNodeRecovery
		case <-time.After(25 * time.Minute):
			endProcedure <- true
		}
	}
}

func (t *Handler) cleanInstallation(conn *ssh.Client, nodeName string, NCCopy *apps_v1alpha.NodeContribution) error {
	uninstallationCommands, err := getUninstallationCommands(conn)
	if err != nil {
		log.Println(err)
		return err
	}
	installationCommands, err := getInstallationCommands(conn, nodeName, t.getKubernetesVersion()[1:])
	if err != nil {
		log.Println(err)
		return err
	}
	commands := append(uninstallationCommands, installationCommands...)
	sess, err := startSession(conn)
	if err != nil {
		log.Println(err)
		return err
	}
	completed := make(chan bool, 1)
	closeSession := func() {
	timeoutOptions:
		select {
		case <-completed:
			break timeoutOptions
		case <-time.After(600 * time.Second):
			sess.Close()
		}
	}
	go closeSession()
	// StdinPipe for commands
	stdin, err := sess.StdinPipe()
	if err != nil {
		log.Println(err)
		return err
	}
	//sess.Stdout = os.Stdout
	//sess.Stderr = os.Stderr

	sess, err = startShell(sess)
	if err != nil {
		log.Println(err)
		return err
	}
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
	sess.Close()
	completed <- true
	return nil
}

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

func reconfigureNode(conn *ssh.Client, hostname string) error {
	commands, err := getReconfigurationCommands(conn, hostname)
	if err != nil {
		log.Println(err)
		return err
	}
	sess, err := startSession(conn)
	if err != nil {
		log.Println(err)
		return err
	}
	defer sess.Close()
	completed := make(chan bool, 1)
	closeSession := func() {
	timeoutOptions:
		select {
		case <-completed:
			break timeoutOptions
		case <-time.After(600 * time.Second):
			sess.Close()
		}
	}
	go closeSession()
	// StdinPipe for commands
	stdin, err := sess.StdinPipe()
	if err != nil {
		log.Println(err)
		return err
	}
	//sess.Stdout = os.Stdout
	//sess.Stderr = os.Stderr

	sess, err = startShell(sess)
	if err != nil {
		log.Println(err)
		return err
	}

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
	sess.Close()
	return nil
}

func startSession(conn *ssh.Client) (*ssh.Session, error) {
	// Start session
	sess, err := conn.NewSession()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return sess, nil
}

func startShell(sess *ssh.Session) (*ssh.Session, error) {
	// Start remote shell
	if err := sess.Shell(); err != nil {
		log.Println(err)
		return nil, err
	}
	return sess, nil
}

func getInstallationCommands(conn *ssh.Client, hostname string, kubernetesVersion string) ([]string, error) {
	sess, err := startSession(conn)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer sess.Close()
	output, err := sess.Output("cat /etc/os-release")
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if ubuntuOrDebian, _ := regexp.MatchString("ID=\"ubuntu\".*|ID=ubuntu.*|ID=\"debian\".*|ID=debian.*", string(output[:])); ubuntuOrDebian {
		// The commands to be sent
		commands := []string{
			"sudo apt-get update -y && apt-get install -y apt-transport-https -y",
			"sudo apt install curl -y",
			"sudo curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -",
			"sudo cat << EOF >/etc/apt/sources.list.d/kubernetes.list",
			"deb http://apt.kubernetes.io/ kubernetes-xenial main",
			"EOF",
			"sudo apt-get update",
			fmt.Sprintf("sudo apt-get install docker.io kubeadm=%s-00 kubernetes-cni -y", kubernetesVersion),
			"sudo swapoff -a",
			fmt.Sprintf("sudo hostname %s", hostname),
			"sudo systemctl enable docker",
			"sudo systemctl start docker",
			"sudo systemctl enable kubelet",
			"sudo systemctl start kubelet",
			fmt.Sprintf("sudo %s", node.CreateJoinToken("3600s", hostname)),
		}
		return commands, nil
	} else if centos, _ := regexp.MatchString("ID=\"centos\".*|ID=centos.*", string(output[:])); centos {
		// The commands to be sent
		commands := []string{
			"sudo yum install yum-utils -y",
			"sudo yum install epel-release -y",
			"sudo yum update -y",
			"sudo setenforce 0",
			"sudo sed -i --follow-symlinks 's/SELINUX=enforcing/SELINUX=disabled/g' /etc/sysconfig/selinux",
			"sudo firewall-cmd --permanent --add-port=6783/tcp",
			"sudo firewall-cmd --permanent --add-port=10250/tcp",
			"sudo firewall-cmd --permanent --add-port=10255/tcp",
			"sudo firewall-cmd --permanent --add-port=30000-32767/tcp",
			"sudo firewall-cmd  --reload",
			"sudo echo '1' > /proc/sys/net/bridge/bridge-nf-call-iptables",
			"sudo cat <<EOF > /etc/yum.repos.d/kubernetes.repo",
			"[kubernetes]",
			"name=Kubernetes",
			"baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64",
			"enabled=1",
			"gpgcheck=1",
			"repo_gpgcheck=1",
			"gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg",
			"EOF",
			fmt.Sprintf("sudo yum install docker kubeadm-%[1]s-0 kubectl-%[1]s-0 kubelet-%[1]s-0 kubernetes-cni -y", kubernetesVersion),
			"sudo swapoff -a",
			"sudo sed -e '/swap/ s/^#*/#/' -i /etc/fstab",
			fmt.Sprintf("sudo hostname %s", hostname),
			"sudo systemctl enable docker",
			"sudo systemctl start docker",
			"sudo systemctl enable kubelet",
			"sudo systemctl start kubelet",
			fmt.Sprintf("sudo %s", node.CreateJoinToken("3600s", hostname)),
		}
		return commands, nil
	}
	return nil, fmt.Errorf("unknown")
}

func getUninstallationCommands(conn *ssh.Client) ([]string, error) {
	sess, err := startSession(conn)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer sess.Close()
	output, err := sess.Output("cat /etc/os-release")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if ubuntuOrDebian, _ := regexp.MatchString("ID=\"ubuntu\".*|ID=ubuntu.*|ID=\"debian\".*|ID=debian.*", string(output[:])); ubuntuOrDebian {
		// The commands to be sent
		commands := []string{
			"sudo kubeadm reset -f",
			"sudo apt-get purge kubeadm kubectl kubelet kubernetes-cni kube* docker-engine docker docker.io docker-ce -y",
			"sudo apt-get autoremove -y",
		}
		return commands, nil
	} else if centos, _ := regexp.MatchString("ID=\"centos\".*|ID=centos.*", string(output[:])); centos {
		// The commands to be sent
		commands := []string{
			"sudo kubeadm reset -f",
			"sudo yum remove kubeadm kubectl kubelet kubernetes-cni kube* docker docker-ce docker-ce-cli docker-client docker-client-latest docker-common docker-latest docker-latest-logrotate docker-logrotate docker-engine -y",
			"sudo yum clean all -y",
			"sudo yum autoremove -y",
		}
		return commands, nil
	}
	return nil, fmt.Errorf("unknown")
}

func getReconfigurationCommands(conn *ssh.Client, hostname string) ([]string, error) {
	sess, err := startSession(conn)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer sess.Close()
	output, err := sess.Output("cat /etc/os-release")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if ubuntuOrDebian, _ := regexp.MatchString("ID=\"ubuntu\".*|ID=ubuntu.*|ID=\"debian\".*|ID=debian.*", string(output[:])); ubuntuOrDebian {
		commands := []string{
			fmt.Sprintf("sudo hostname %s", hostname),
			"sudo systemctl stop docker",
			"sudo systemctl stop kubelet",
			"sudo iptables --flush",
			"sudo iptables -tnat --flush",
			"sudo systemctl start docker",
			"sudo systemctl start kubelet",
		}
		return commands, nil
	} else if centos, _ := regexp.MatchString("ID=\"centos\".*|ID=centos.*", string(output[:])); centos {
		commands := []string{
			fmt.Sprintf("sudo hostname %s", hostname),
			"sudo systemctl stop docker",
			"sudo systemctl stop kubelet",
			"sudo iptables -F",
			"sudo iptables -tnat -F",
			"sudo systemctl start docker",
			"sudo systemctl start kubelet",
		}
		return commands, nil
	}
	return nil, fmt.Errorf("unknown")
}

func (t *Handler) getKubernetesVersion() string {
	nodeRaw, err := t.clientset.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/master"})
	if err != nil {
		log.Println(err.Error())
	}
	kubeletVersion := ""
	for _, nodeRow := range nodeRaw.Items {
		kubeletVersion = nodeRow.Status.NodeInfo.KubeletVersion
	}
	return kubeletVersion
}

func getRecordType(ip string) string {
	if net.ParseIP(ip) == nil {
		return ""
	}
	for i := 0; i < len(ip); i++ {
		switch ip[i] {
		case '.':
			return "A"
		case ':':
			return "AAAA"
		}
	}
	return ""
}
