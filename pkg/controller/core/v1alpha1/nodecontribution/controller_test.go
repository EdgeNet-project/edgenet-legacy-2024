package nodecontribution

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}
