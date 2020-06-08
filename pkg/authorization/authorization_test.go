package authorization
import (
	"testing"
	"path/filepath"
	"flag"
	
)
func TestHomeDir(t *testing.T) {
	home := homeDir()
	if home == "" {
		t.Fatal("returned home directory is empty")
	}
	if !filepath.IsAbs(home) {
		t.Fatalf("returned path is not absolute: %s", home)
	}
}

func TestSetKubeConfig(t *testing.T) {
SetKubeConfig()
var r string
flag.StringVar(&r, "r", filepath.Join(homeDir(), ".kube", "config"), "")
if(kubeconfig != "" && kubeconfig != r){
	t.Fatal("Error, another path has been detected")
}
flag.Parse()
}











