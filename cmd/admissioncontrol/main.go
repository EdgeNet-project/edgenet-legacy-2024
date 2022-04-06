package main

import (
	"errors"
	"os"

	admissioncontrol "github.com/EdgeNet-project/edgenet/pkg/admissioncontrol"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog"
)

var (
	tlsCert          string
	tlsKey           string
	containerRuntime string
)

func init() {
	tlsCert = os.Getenv("TLS_CERTIFICATE")
	tlsKey = os.Getenv("TLS_PRIVATE_KEY")
	if tlsCert == "" || tlsKey == "" {
		err := errors.New("TLS_CERTIFICATE and TLS_PRIVATE_KEY required")
		klog.Fatalf("Error running admission control webhook: %s", err.Error())
		os.Exit(1)
	}
	containerRuntime = os.Getenv("CONTAINER_RUNTIME")
}

func main() {
	webhook := admissioncontrol.Webhook{}
	webhook.CertFile = tlsCert
	webhook.KeyFile = tlsKey
	webhook.Codecs = serializer.NewCodecFactory(runtime.NewScheme())
	webhook.Runtime = containerRuntime
	webhook.RunServer()
}
