// Package main stands as the interface between the EdgeNet portal and headnode.
// This uses gorilla/mux to handle requests from the EdgeNet and make use of the
// functions of packages according to the requests. All authoritative state for
// the EdgeNet system is kept by the portal, which issues http requests through
// this server to ensure that the headnode is maintaining this state.

// This server also acts as an intermediary for DNS entries, currently kept by
// namecheap.com.  Namecheap uses IP whitelisting as its major security tool,
// and since the portal currently runs on the Google App Engine and this doesn't
// offer fixed IP addresses, we use this as the DNS intermediary. Once we either
// switch away from namecheap and/or switch to the Google App Engine Flex, we'll
// move that to the portal.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"headnode/pkg/authorization"
	"headnode/pkg/namespace"
	"headnode/pkg/node"
	"headnode/pkg/registration"

	namecheap "github.com/billputer/go-namecheap"
	"github.com/gorilla/mux"
)

// This creates a kubeconfig file based on the namespace in which the user is
// represented at the Kubernetes level. Call is ?user=<username>
func hello(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		err := errors.New("No 'user' arg in request")
		fmt.Fprintf(w, "%s", err)
	}
	// Create a kubeconfig file for the user
	result := registration.MakeConfig(user)
	fmt.Fprintf(w, "%s", result)
	return
}

// Add a user to the headnode (actually, a namespace, which generates a
// configuration file). Call is /make-user?user=<username>
func makeUser(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		err := errors.New("No 'user' arg in request")
		fmt.Fprintf(w, "%s", err)
	}
	// Add the user
	result := registration.MakeUser(user)
	fmt.Fprintf(w, "%s", result)
	return
}

// Get the status of nodes currently known by the headnode. Call is /get_status
func getNodeStatus(w http.ResponseWriter, r *http.Request) {
	// Get the node list
	result := node.GetStatusList()
	fmt.Fprintf(w, "%s", result)
}

// Get a shared secret that the add_node script can run on the client and pass
// to this node to ensure that the add_node is legitimate. Call is /get_secret
func getSecret(w http.ResponseWriter, r *http.Request) {
	// Returns same result with the result of the command
	// $ sudo kubeadm token create --print-join-command --ttl 3600s
	result := node.CreateJoinToken("3600s", "edge-net.io")
	fmt.Fprintf(w, "%s", result)
}

// Old reference below
// /add_node?sitename=<node_name>&ip_address=address
// called from portal, no error-checking
//@app.route("/add_node")
func addNode(w http.ResponseWriter, r *http.Request) {
	// Config needed
	apiUser := "."
	apiToken := "."
	userName := "."

	type hostDetails struct {
		ID      int    `json:"ID"`
		Name    string `json:"Name"`
		Type    string `json:"Type"`
		Address string `json:"Address"`
		MXPref  int    `json:"MXPref"`
		TTL     int    `json:"TTL"`
	}

	type hostResponse struct {
		Domain        string        `json:"Domain"`
		IsUsingOurDNS bool          `json:"IsUsingOurDNS"`
		Hosts         []hostDetails `json:"Hosts"`
	}

	client := namecheap.NewClient(apiUser, apiToken, userName)

	ip := r.URL.Query().Get("ip_address")
	site := r.URL.Query().Get("sitename")
	//recordType := r.URL.Query().Get("record_type")
	hostsResponse, err := client.DomainsDNSGetHosts("edge-net", "io")
	if err != nil {
		panic(err.Error())
	}
	responseJSON, err := json.Marshal(hostsResponse)
	if err != nil {
		panic(err.Error())
	}
	hostList := hostResponse{}
	json.Unmarshal([]byte(responseJSON), &hostList)
	exist := false
	for _, host := range hostList.Hosts {
		if host.Name == site || host.Address == ip {
			exist = true
			break
		}
	}
	// Should return as JSON
	if exist {
		fmt.Fprintf(w, "Error: Site name %s or address %s already exists", site, ip)
	}
	return
	//hosts.append((site, ip, record_type))
	//namecheap_lib.set_hosts('edge-net.io', hosts)
	//return Response("Site %s.edge-net.io added at ip %s" % (site, ip))
	// !!!!! WILL BE FURTHER DEVELOPED
}

// Get the set of namespaces. Call: /get_namespaces
func getNamespaces(w http.ResponseWriter, r *http.Request) {
	// Get the namespaces currently exist on the headnode except "default",
	// "kube-system", and "kube-public"
	result := namespace.GetList()
	fmt.Fprintf(w, "%s", result)
}

func main() {
	// Set kubeconfig to be used to create clientsets
	authorization.SetKubeConfig()

	// URL paths and handlers. Deprecated URL paths and handlers below
	// ("/nodes", getNodes), ("/show_ip", showIP), ("/get_setup", getSetup), ("/show_headers", getHeaders)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", hello)                   // In use
	router.HandleFunc("/make-user", makeUser)       // In use
	router.HandleFunc("/get_status", getNodeStatus) // In use
	router.HandleFunc("/get_secret", getSecret)     // In use
	router.HandleFunc("/add_node", addNode)         // In use
	router.HandleFunc("/namespaces", getNamespaces) // In use

	log.Fatal(http.ListenAndServe(":8181", router))
}
