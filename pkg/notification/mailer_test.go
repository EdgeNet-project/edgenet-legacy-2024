package notification

import (
	"flag"
	"io"
	"log"
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"k8s.io/klog"
)

func TestMain(m *testing.M) {
	klog.SetOutput(io.Discard)
	log.SetOutput(io.Discard)
	logrus.SetOutput(io.Discard)

	flag.String("smtp-path", "../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	os.Exit(m.Run())
}
