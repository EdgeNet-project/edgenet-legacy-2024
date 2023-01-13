package multiprovider

import (
	"flag"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"
)

func TestCreateToken(t *testing.T) {
	flag.String("kubeconfig-path", "../../configs/public.cfg", "Set kubeconfig path.")
	flag.Parse()
	g := testGroup{}
	g.Init()
	ttl, err := time.ParseDuration("600s")
	util.OK(t, err)
	_, err = g.multiproviderManager.createToken(ttl, "test.edgenet.io")
	util.OK(t, err)
}

func TestGetOperations(t *testing.T) {
	flag.String("configs-path", "../../configs", "Set Namecheap path.")
	flag.Parse()

	t.Run("config view", func(t *testing.T) {
		_, err := getConfigView()
		util.OK(t, err)
	})
	t.Run("server from current context", func(t *testing.T) {
		_, err := getServerOfCurrentContext()
		util.OK(t, err)
	})
}
