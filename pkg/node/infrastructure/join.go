package infrastructure

import (
	"crypto/sha256"
	"fmt"
	s "strings"
	"time"

	custconfig "headnode/pkg/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	nodebootstraptokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
)

// CreateToken creates the token to be used to add node
// and return the token
func CreateToken(clientset clientset.Interface, duration time.Duration, hostname string) (string, error) {
	tokenStr, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return "", err
		// "error generating token to upload certs"
	}
	token, err := kubeadmapi.NewBootstrapTokenString(tokenStr)
	if err != nil {
		return "", err
		// "error creating upload certs token"
	}
	tokens := []kubeadmapi.BootstrapToken{{
		Token:       token,
		Description: fmt.Sprintf("EdgeNet token for adding node called %s", hostname),
		TTL: &metav1.Duration{
			Duration: duration,
		},
		Usages: []string{"authentication", "signing"},
		Groups: []string{"system:bootstrappers:kubeadm:default-node-token"},
	}}

	if err := nodebootstraptokenphase.CreateNewTokens(clientset, tokens); err != nil {
		return "", err
		// "error creating token"
	}
	// This reads server info of the current context from the config file
	server, err := custconfig.GetServerOfCurrentContext()
	if err != nil {
		fmt.Printf("Err: %s", err)
	}
	server = s.Trim(server, "https://")
	server = s.Trim(server, "http://")
	// This reads CA cert to be hashed
	certs, err := cert.CertsFromFile("/etc/kubernetes/pki/ca.crt")
	if err != nil {
		fmt.Printf("Err: %s", err)
	}
	var CA string
	for i, cert := range certs {
		if i == 0 {
			hashedCA := sha256.Sum256([]byte(cert.RawSubjectPublicKeyInfo))
			CA = fmt.Sprintf("sha256:%x", hashedCA)
		}
	}

	joinCommand := fmt.Sprintf("kubeadm join %s --token %s --discovery-token-ca-cert-hash %s", server, tokens[0].Token.String(), CA)
	return joinCommand, nil
}
