package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"headnode/pkg/namespace"
	"headnode/pkg/node"
	"headnode/pkg/remoteip"

	"github.com/gorilla/mux"

	namecheap "github.com/billputer/go-namecheap"
)

var kubeconfig string

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// Old reference below
//@app.route("/")
//def hello():
func hello(w http.ResponseWriter, r *http.Request) {
	if user := r.URL.Query().Get("user"); user == "" {
		err := errors.New("No 'user' arg in request")
		fmt.Fprintf(w, "%s", err)
	}
	// !!!!! WILL BE FURTHER DEVELOPED
	return
}

// Old reference below
//@app.route('/make-user')
//def make_user():
func makeUser(w http.ResponseWriter, r *http.Request) {
	if user := r.URL.Query().Get("user"); user == "" {
		err := errors.New("No 'user' arg in request")
		fmt.Fprintf(w, "%s", err)
	}
	// !!!!! WILL BE FURTHER DEVELOPED
	return
}

// Old reference below
//@app.route("/nodes")
//def get_nodes():
func getNodes(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", node.GetList(&kubeconfig))
}

// Old reference below
//@app.route("/get_status")
//def get_status():
func getNodeStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", node.GetStatusList(&kubeconfig))
}

func getSecret(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", node.CreateJoinToken(&kubeconfig, 100000000000, "edge-net.io"))
}

// Old reference below
//@app.route("/show_ip")
//def show_ip():
func showIP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Your request is from %s", remoteip.GetIPAdress(r))
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

// Old reference below
//@app.route("/get_setup")
//def get_setup():
func getSetup(writer http.ResponseWriter, request *http.Request) {
	//First of check if Get is set in the URL
	Filename := request.URL.Query().Get("file")
	if Filename == "" {
		//Get not set, send a 400 bad request
		http.Error(writer, "Get 'file' not specified in url.", 400)
		return
	}
	fmt.Println("Client requests: " + Filename)

	//Check if file exists and open
	Openfile, err := os.Open("setup_node.sh")
	defer Openfile.Close() //Close after function return
	if err != nil {
		//File not found, send 404
		http.Error(writer, "File not found.", 404)
		return
	}

	//File is found, create and send the correct headers

	//Get the Content-Type of the file
	//Create a buffer to store the header of the file in
	FileHeader := make([]byte, 512)
	//Copy the headers into the FileHeader buffer
	Openfile.Read(FileHeader)
	//Get content type of file
	FileContentType := http.DetectContentType(FileHeader)

	//Get the file size
	FileStat, _ := Openfile.Stat()                     //Get info from file
	FileSize := strconv.FormatInt(FileStat.Size(), 10) //Get file size as a string

	//Send the headers
	writer.Header().Set("Content-Disposition", "attachment; filename=setup_node.sh")
	writer.Header().Set("Content-Type", FileContentType)
	writer.Header().Set("Content-Length", FileSize)

	//Send the file
	//We read 512 bytes from the file already, so we reset the offset back to 0
	Openfile.Seek(0, 0)
	io.Copy(writer, Openfile) //'Copy' the file to the client
	return
	// !!!!! WILL BE FURTHER DEVELOPED
}

// Old reference below
//@app.route("/show_headers")
//def get_headers():
func getHeaders(w http.ResponseWriter, r *http.Request) {
	headers, err := json.Marshal(r.Header)

	if err != nil {
		return
	}

	fmt.Fprintf(w, "%s", headers)
}

// Old reference below
//@app.route("/namespaces")
//def get_namespaces():
func getNamespaces(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", namespace.GetList(&kubeconfig))
}

func main() {
	if home := homeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", hello)
	router.HandleFunc("/make-user", makeUser)
	router.HandleFunc("/nodes", getNodes)
	router.HandleFunc("/get_status", getNodeStatus)
	router.HandleFunc("/get_secret", getSecret)
	router.HandleFunc("/show_ip", showIP)
	router.HandleFunc("/add_node", addNode)
	router.HandleFunc("/get_setup", getSetup)
	router.HandleFunc("/show_headers", getHeaders)
	router.HandleFunc("/namespaces", getNamespaces)

	log.Fatal(http.ListenAndServe(":8181", router))
}
