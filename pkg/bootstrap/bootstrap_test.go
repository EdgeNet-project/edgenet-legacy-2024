package bootstrap

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/EdgeNet-project/edgenet/pkg/util"
)

func TestHomeDir(t *testing.T) {
	home := homeDir()
	if home == "" {
		t.Fatal("Returned home directory is empty")
	}
	if !filepath.IsAbs(home) {
		t.Fatalf("Returned path is not absolute: %s", home)
	}
}

func TestClientSetCreation(t *testing.T) {
	kubeconfigPath := getKubeconfigPath()
	config, err := GetRestConfig("kubeconfig")
	util.OK(t, err)
	t.Run("preparing kubeconfig file", func(t *testing.T) {
		util.Equals(t, filepath.Join(homeDir(), ".kube", "config"), kubeconfigPath)
	})
	t.Run("create edgenet clientset", func(t *testing.T) {
		_, err := CreateEdgeNetClientset(config)
		util.OK(t, err)
	})
	t.Run("create kubernetes clientset", func(t *testing.T) {
		_, err := CreateKubeClientset(config)
		util.OK(t, err)
	})
}

func TestNamecheapClient(t *testing.T) {
	flag.String("configs-path", "../../configs", "Set Namecheap path.")
	flag.Parse()
	_, err := CreateNamecheapClient()
	util.OK(t, err)
}
