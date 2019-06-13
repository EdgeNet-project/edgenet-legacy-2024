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
		log.Printf("%s: %s", r.URL.RequestURI(), err)
		fmt.Fprintf(w, "%s", err)
		return
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
		log.Printf("%s: %s", r.URL.RequestURI(), err)
		fmt.Fprintf(w, "%s", err)
		return
	}
	// Add the user
	result, status := registration.MakeUser(user)
	w.WriteHeader(status)
	fmt.Fprintf(w, "%s", result)
	return
}

// Get the status of nodes currently known by the headnode. Call is /get_status
func getNodeStatus(w http.ResponseWriter, r *http.Request) {
	// Get the node list
	result := node.GetStatusList()
	fmt.Fprintf(w, "%s", result)
	return
}

// Get a shared secret that the add_node script can run on the client and pass
// to this node to ensure that the add_node is legitimate. Call is /get_secret
func getSecret(w http.ResponseWriter, r *http.Request) {
	// Returns same result with the result of the command
	// $ sudo kubeadm token create --print-join-command --ttl 3600s
	result := node.CreateJoinToken("3600s", "edge-net.io")
	fmt.Fprintf(w, "%s", result)
	return
}

// Add a node to the DNS records for edge-net.io.
// Call: /add_node?ip_address=<ip_address>&sitename=<sitename>&record_type=<A or AAA>
func addNode(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip_address")
	site := r.URL.Query().Get("sitename")
	recordType := r.URL.Query().Get("record_type")
	if ip == "" {
		err := errors.New("No 'ip_address' arg in request")
		log.Printf("%s: %s", r.URL.RequestURI(), err)
		fmt.Fprintf(w, "%s", err)
		return
	} else if site == "" {
		err := errors.New("No 'sitename' arg in request")
		log.Printf("%s: %s", r.URL.RequestURI(), err)
		fmt.Fprintf(w, "%s", err)
		return
	} else if recordType == "" {
		err := errors.New("No 'record_type' arg in request")
		log.Printf("%s: %s", r.URL.RequestURI(), err)
		fmt.Fprintf(w, "%s", err)
		return
	}

	hostRecord := namecheap.DomainDNSHost{
		Name:    site,
		Type:    recordType,
		Address: ip,
	}
	result, state := node.SetHostname(hostRecord)
	if result {
		fmt.Fprintf(w, "Site %s.edge-net.io added at ip %s", hostRecord.Name, hostRecord.Address)
		return
	}
	if state == "exist" {
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "application/json")
		err := fmt.Errorf("Error: Site name %s or address %s already exists", hostRecord.Name, hostRecord.Address)
		log.Printf("%s: %s", r.URL.RequestURI(), err)
		fmt.Fprintf(w, "%s", err)
		return
	}
	err := fmt.Errorf("Error: Site name %s or address %s couldn't added", hostRecord.Name, hostRecord.Address)
	log.Printf("%s: %s", r.URL.RequestURI(), err)
	fmt.Fprintf(w, "%s", err)
	return
}

// Get the set of namespaces. Call: /get_namespaces
func getNamespaces(w http.ResponseWriter, r *http.Request) {
	// Get the namespaces currently exist on the headnode except "default",
	// "kube-system", and "kube-public"
	result := namespace.GetList()
	fmt.Fprintf(w, "%s", result)
	return
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

	log.Fatal(http.ListenAndServe("127.0.0.1:8181", router))
}
