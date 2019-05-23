package infrastructure

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	nodebootstraptokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
)

// createToken creates the token used to add node
// and return the token
func CreateToken(client clientset.Interface, duration int, hostname string) (string, error) {
	tokenStr, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return "", err
		//"error generating token to upload certs"
	}
	token, err := kubeadmapi.NewBootstrapTokenString(tokenStr)
	if err != nil {
		return "", err
		//"error creating upload certs token"
	}
	tokens := []kubeadmapi.BootstrapToken{{
		Token:       token,
		Description: fmt.Sprintf("EdgeNet token for adding node called %s", hostname),
		TTL: &metav1.Duration{
			Duration: time.Duration(duration),
		},
		Usages: []string{"authentication", "signing"},
		Groups: []string{"system:bootstrappers:kubeadm:default-node-token"},
	}}

	if err := nodebootstraptokenphase.CreateNewTokens(client, tokens); err != nil {
		return "", err
		//"error creating token"
	}
	return tokens[0].Token.String(), nil
}
