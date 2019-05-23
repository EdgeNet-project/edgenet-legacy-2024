package node

import (
	"encoding/json"
	"fmt"

	"headnode/pkg/authorization"
	"headnode/pkg/node/infrastructure"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateJoinToken(kubeconfig *string, duration int, hostname string) string {
	clientset, err := authorization.CreateClientSet(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	token, err := infrastructure.CreateToken(clientset, duration, hostname)
	if err != nil {
		return "error"
	}
	return token
}

func GetList(kubeconfig *string) []string {
	clientset, err := authorization.CreateClientSet(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	nodesRaw, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	nodes := make([]string, len(nodesRaw.Items))
	for i, nodeRow := range nodesRaw.Items {
		nodes[i] = nodeRow.Name
	}

	return nodes
}

func GetStatusList(kubeconfig *string) []byte {
	type nodeStatus struct {
		Node  string `json:"node"`
		Ready string `json:"ready"`
	}
	clientset, err := authorization.CreateClientSet(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	nodesRaw, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	nodesArr := make([]nodeStatus, len(nodesRaw.Items))
	for i, nodeRow := range nodesRaw.Items {
		nodesArr[i].Node = nodeRow.Name
		for _, conditionRow := range nodeRow.Status.Conditions {
			if conditionType := conditionRow.Type; conditionType == "Ready" {
				nodesArr[i].Ready = string(conditionRow.Status)
			}
		}
	}
	nodesJSON, _ := json.Marshal(nodesArr)

	return nodesJSON
}

func getNodeByHostname(kubeconfig *string, hostname string) (string, error) {
	clientset, err := authorization.CreateClientSet(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err = clientset.CoreV1().Nodes().Get(hostname, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Node %s not found\n", hostname)
		return "Node not found", err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting node %s: %v\n",
			hostname, statusError.ErrStatus)
		return "Error getting node", err
	} else if err != nil {
		panic(err.Error())
	} else {
		return "", nil
	}
}
