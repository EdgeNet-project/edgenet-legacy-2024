package infrastructure

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestMain(m *testing.M) {
	flag.String("ca-path", "../../../configs/ca_sample.crt", "Set CA path.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestCreateJoinToken(t *testing.T) {
	ttl, err := time.ParseDuration("600s")
	util.OK(t, err)
	_, err = CreateToken(testclient.NewSimpleClientset(), ttl, "test.edgenet.io")
	util.OK(t, err)
}
