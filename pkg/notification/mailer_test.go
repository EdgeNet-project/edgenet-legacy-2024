package notification

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"k8s.io/klog"
)

func TestMain(m *testing.M) {
	klog.SetOutput(ioutil.Discard)
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)

	flag.String("smtp-path", "../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	os.Exit(m.Run())
}

/*func TestNotification(t *testing.T) {
	var smtpServer smtpServer
	// The code below inits the SMTP configuration for sending emails
	// The path of the yaml config file of test smtp server
	file, err := os.Open(flag.Lookup("smtp-path").Value.(flag.Getter).Get().(string))
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return
	}
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&smtpServer)
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return
	}

	email := new(Content)
	email.Cluster = "cluster-uid"
	email.User = "john.doe@edge-net.org"
	email.FirstName = "John"
	email.LastName = "Doe"
	email.Subject = "Role Request Approval"
	email.Recipient = []string{"john.doe@edge-net.org"}

	email.RoleRequest = new(RoleRequest)
	email.RoleRequest.Name = "johndoe"
	email.RoleRequest.Namespace = "edgenet"
	err = email.Send("role-request-approved")
	util.OK(t, err)
}*/
