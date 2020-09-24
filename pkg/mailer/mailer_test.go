package mailer

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	yaml "gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	flag.String("smtp-path", "../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestNotification(t *testing.T) {
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

	contentData := CommonContentData{}
	contentData.CommonData.Authority = "test"
	contentData.CommonData.Username = "johndoe"
	contentData.CommonData.Name = "John"
	contentData.CommonData.Email = []string{"john.doe@edge-net.org"}

	multiProviderData := MultiProviderData{}
	multiProviderData.Name = "test"
	multiProviderData.Host = "12.12.123.123"
	multiProviderData.Status = "Status"
	multiProviderData.Message = []string{"Status Message"}
	multiProviderData.CommonData = contentData.CommonData

	resourceAllocationData := ResourceAllocationData{}
	resourceAllocationData.Name = "test"
	resourceAllocationData.OwnerNamespace = "authority-test"
	resourceAllocationData.ChildNamespace = "authority-test-namespace-test"
	resourceAllocationData.Authority = "test"
	resourceAllocationData.CommonData = contentData.CommonData

	verifyContentData := VerifyContentData{}
	verifyContentData.Code = "verificationcode"
	verifyContentData.CommonData = contentData.CommonData

	createKubeconfig := func(contentData interface{}, done chan bool) {
		registrationData := contentData.(CommonContentData)
		// Creating temp config file to be consumed by setUserRegistrationContent()
		var file, err = os.Create(fmt.Sprintf("%s/assets/kubeconfigs/%s-%s.cfg", dir, registrationData.CommonData.Authority,
			registrationData.CommonData.Username))
		if err != nil {
			t.Errorf("Failed to create temp %s/assets/kubeconfigs/%s-%s.cfg file", dir, registrationData.CommonData.Authority,
				registrationData.CommonData.Username)
		}
		<-done
		file.Close()
		os.Remove(fmt.Sprintf("%s/assets/kubeconfigs/%s-%s.cfg", dir, registrationData.CommonData.Authority,
			registrationData.CommonData.Username))
	}

	cases := map[string]struct {
		Content  interface{}
		Expected []string
	}{
		"user-email-verification":                    {verifyContentData, []string{verifyContentData.CommonData.Authority, verifyContentData.CommonData.Username, verifyContentData.Code}},
		"user-email-verification-update":             {verifyContentData, []string{verifyContentData.CommonData.Authority, verifyContentData.CommonData.Username, verifyContentData.Code}},
		"user-email-verified-alert":                  {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-email-verified-notification":           {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-registration-successful":               {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"authority-email-verification":               {verifyContentData, []string{verifyContentData.CommonData.Authority, verifyContentData.CommonData.Username, verifyContentData.CommonData.Name, verifyContentData.Code}},
		"authority-email-verified-alert":             {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"authority-creation-successful":              {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username}},
		"acceptable-use-policy-accepted":             {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username}},
		"acceptable-use-policy-renewal":              {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"acceptable-use-policy-expired":              {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"slice-creation":                             {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name, resourceAllocationData.ChildNamespace}},
		"slice-removal":                              {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name, resourceAllocationData.ChildNamespace}},
		"slice-reminder":                             {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name, resourceAllocationData.ChildNamespace}},
		"slice-deletion":                             {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name, resourceAllocationData.ChildNamespace}},
		"slice-crash":                                {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name}},
		"slice-total-quota-exceeded":                 {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name}},
		"slice-lack-of-quota":                        {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name}},
		"slice-deletion-failed":                      {resourceAllocationData, []string{resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name}},
		"slice-collection-deletion-failed":           {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name}},
		"team-creation":                              {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name, resourceAllocationData.ChildNamespace}},
		"team-removal":                               {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name, resourceAllocationData.ChildNamespace}},
		"team-deletion":                              {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name, resourceAllocationData.ChildNamespace}},
		"team-crash":                                 {resourceAllocationData, []string{resourceAllocationData.CommonData.Authority, resourceAllocationData.CommonData.Username, resourceAllocationData.CommonData.Name, resourceAllocationData.Authority, resourceAllocationData.OwnerNamespace, resourceAllocationData.Name}},
		"node-contribution-successful":               {multiProviderData, []string{multiProviderData.CommonData.Authority, multiProviderData.CommonData.Username, multiProviderData.CommonData.Name, multiProviderData.Name, multiProviderData.Host, multiProviderData.Message[0]}},
		"node-contribution-failure":                  {multiProviderData, []string{multiProviderData.CommonData.Authority, multiProviderData.CommonData.Username, multiProviderData.CommonData.Name, multiProviderData.Name, multiProviderData.Host, multiProviderData.Message[0]}},
		"node-contribution-failure-support":          {multiProviderData, []string{multiProviderData.CommonData.Authority, multiProviderData.Name, multiProviderData.Host, multiProviderData.Message[0]}},
		"authority-validation-failure-name":          {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"authority-validation-failure-email":         {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"authority-email-verification-malfunction":   {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username}},
		"authority-creation-failure":                 {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"authority-email-verification-dubious":       {contentData, []string{contentData.CommonData.Authority}},
		"user-validation-failure-name":               {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-validation-failure-email":              {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-email-verification-malfunction":        {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username}},
		"user-creation-failure":                      {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-cert-failure":                          {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-kubeconfig-failure":                    {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-email-verification-dubious":            {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username}},
		"user-email-verification-update-malfunction": {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username}},
		"user-deactivation-failure":                  {contentData, []string{contentData.CommonData.Authority, contentData.CommonData.Username}},
	}

	for k, tc := range cases {
		t.Run(fmt.Sprintf("%s", k), func(t *testing.T) {
			if k == "user-registration-successful" {
				done := make(chan bool)
				go createKubeconfig(tc.Content, done)
				defer func() { done <- true }()
			}

			t.Run("template", func(t *testing.T) {
				_, body := prepareNotification(k, tc.Content, smtpServer)
				bodyString := body.String()
				for _, expected := range tc.Expected {
					if !strings.Contains(bodyString, expected) {
						t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in the template not found\n", k, expected)
					}
				}
			})

			/*t.Run("send", func(t *testing.T) {
				err = Send(k, tc.Content)
				util.OK(t, err)
				time.Sleep(200 * time.Millisecond)
			})*/
		})
	}
}
