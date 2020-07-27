package mailer

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestSend(t *testing.T) {
	var smtpServer smtpServer
	// The code below inits the SMTP configuration for sending emails
	// The path of the yaml config file of test smtp server
	file, err := os.Open("../../configs/smtp_test.yaml")
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
	contentData.CommonData.Authority = "TESTAUTHORITY"
	contentData.CommonData.Username = "TESTUSERNAME"
	contentData.CommonData.Name = "TESTNAME"
	contentData.CommonData.Email = []string{"TESTEMAIL"}

	multiProviderData := MultiProviderData{}
	multiProviderData.Name = "multiProviderDataName"
	multiProviderData.Host = "multiProviderDataHost"
	multiProviderData.Status = "multiProvierDataStatus"
	multiProviderData.Message = []string{"multiProviderDataMessage"}
	multiProviderData.CommonData = contentData.CommonData

	resourceAllocationData := ResourceAllocationData{}
	resourceAllocationData.Name = "resourceAllocationDataName"
	resourceAllocationData.OwnerNamespace = "resourceAllocationDataOwnerNS"
	resourceAllocationData.ChildNamespace = "resourceAllocationDataChildNS"
	resourceAllocationData.Authority = "resourceAllocationDataAuthority"
	resourceAllocationData.CommonData = contentData.CommonData

	verifyContentData := VerifyContentData{}
	verifyContentData.Code = "verifyContentDataCode"
	verifyContentData.CommonData = contentData.CommonData

	// Testing across all subjects
	subjects := []string{"user-email-verification", "user-email-verified-alert",
		"user-registration-successful", "authority-email-verification",
		"authority-email-verified-alert", "authority-creation-successful",
		"acceptable-use-policy-accepted", "acceptable-use-policy-renewal",
		"acceptable-use-policy-expired", "acceptable-use-policy-expired",
		"slice-creation", "team-creation", "node-contribution-successful",
		"authority-validation-failure-name", "user-validation-failure-name",
		"user-validation-failure-name"}

	var subject string
	var body bytes.Buffer
	for _, subject = range subjects {
		switch subject {
		case "user-email-verification", "user-email-verification-update":
			err = Send(subject, verifyContentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setUserEmailVerificationContent(verifyContentData, smtpServer.From, subject)
			bodyString := body.String()
			fmt.Printf(bodyString)
			if !strings.Contains(bodyString, verifyContentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, verifyContentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, verifyContentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, verifyContentData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, verifyContentData.Code) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, verifyContentData.Code, "")
			}

		case "user-email-verified-alert", "user-email-verified-notification":
			err = Send(subject, contentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setUserVerifiedAlertContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
			bodyString := body.String()
			if !strings.Contains(bodyString, contentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Name, "")
			}

		case "user-registration-successful":
			// Creating temp config file to be consumed by setUserRegistrationContent()
			var file, err = os.Create(fmt.Sprintf("../../assets/kubeconfigs/edgenet-authority-%s-%s.cfg", contentData.CommonData.Authority,
				contentData.CommonData.Username))
			if err != nil {
				t.Errorf("Failed to create temp ../../assets/kubeconfigs/edgenet-authority-%s-%s.cfg file", contentData.CommonData.Authority,
					contentData.CommonData.Username)
			}
			defer file.Close()
			defer os.Remove(fmt.Sprintf("../../assets/kubeconfigs/edgenet-authority-%s-%s.cfg", contentData.CommonData.Authority,
				contentData.CommonData.Username))
			err = Send(subject, contentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setUserRegistrationContent(contentData, smtpServer.From)
			bodyString := body.String()
			if !strings.Contains(bodyString, contentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Name, "")
			}

		case "authority-email-verification":
			err = Send(subject, verifyContentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setAuthorityEmailVerificationContent(verifyContentData, smtpServer.From)
			bodyString := body.String()
			if !strings.Contains(bodyString, verifyContentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, verifyContentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, verifyContentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, verifyContentData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, verifyContentData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, verifyContentData.CommonData.Name, "")
			}
			if !strings.Contains(bodyString, verifyContentData.Code) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, verifyContentData.Code, "")
			}

		case "authority-email-verified-alert":
			err = Send(subject, contentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setAuthorityVerifiedAlertContent(contentData, smtpServer.From, []string{smtpServer.To})
			bodyString := body.String()
			if !strings.Contains(bodyString, contentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Name, "")
			}

		case "authority-creation-successful":
			err = Send(subject, contentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setAuthorityRequestContent(contentData, smtpServer.From)
			bodyString := body.String()
			if !strings.Contains(bodyString, contentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Username, "")
			}

		case "acceptable-use-policy-accepted":
			err = Send(subject, contentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setAUPConfirmationContent(contentData, smtpServer.From)
			bodyString := body.String()
			if !strings.Contains(bodyString, contentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Username, "")
			}

		case "acceptable-use-policy-renewal":
			err = Send(subject, contentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setAUPRenewalContent(contentData, smtpServer.From)
			bodyString := body.String()
			if !strings.Contains(bodyString, contentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Name, "")
			}

		case "acceptable-use-policy-expired":
			err = Send(subject, contentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setAUPExpiredContent(contentData, smtpServer.From)
			bodyString := body.String()
			if !strings.Contains(bodyString, contentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Name, "")
			}

		case "slice-creation", "slice-removal", "slice-reminder", "slice-deletion", "slice-crash", "slice-total-quota-exceeded", "slice-lack-of-quota",
			"slice-deletion-failed", "slice-collection-deletion-failed":
			err = Send(subject, resourceAllocationData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setSliceContent(resourceAllocationData, smtpServer.From, []string{smtpServer.To}, subject)
			bodyString := body.String()
			if !strings.Contains(bodyString, resourceAllocationData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.CommonData.Name, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.Authority, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.OwnerNamespace) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.OwnerNamespace, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.Name, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.ChildNamespace) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.ChildNamespace, "")
			}

		case "team-creation", "team-removal", "team-deletion", "team-crash":
			err = Send(subject, resourceAllocationData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setTeamContent(resourceAllocationData, smtpServer.From, subject)
			bodyString := body.String()
			if !strings.Contains(bodyString, resourceAllocationData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.CommonData.Name, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.Authority, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.OwnerNamespace) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.OwnerNamespace, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.Name, "")
			}
			if !strings.Contains(bodyString, resourceAllocationData.ChildNamespace) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, resourceAllocationData.ChildNamespace, "")
			}

		case "node-contribution-successful", "node-contribution-failure", "node-contribution-failure-support":
			err = Send(subject, multiProviderData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setNodeContributionContent(multiProviderData, smtpServer.From, []string{smtpServer.To}, subject)
			bodyString := body.String()
			if !strings.Contains(bodyString, multiProviderData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, multiProviderData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, multiProviderData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, multiProviderData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, multiProviderData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, multiProviderData.CommonData.Name, "")
			}
			if !strings.Contains(bodyString, multiProviderData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, multiProviderData.Name, "")
			}
			if !strings.Contains(bodyString, multiProviderData.Host) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, multiProviderData.Host, "")
			}
			if !strings.Contains(bodyString, multiProviderData.Message[0]) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, multiProviderData.Message[0], "")
			}

		case "authority-validation-failure-name", "authority-validation-failure-email", "authority-email-verification-malfunction",
			"authority-creation-failure", "authority-email-verification-dubious":
			err = Send(subject, contentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setAuthorityFailureContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
			bodyString := body.String()
			if !strings.Contains(bodyString, contentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Name, "")
			}
		case "user-validation-failure-name", "user-validation-failure-email", "user-email-verification-malfunction", "user-creation-failure", "user-serviceaccount-failure",
			"user-kubeconfig-failure", "user-email-verification-dubious", "user-email-verification-update-malfunction", "user-deactivation-failure":
			err = Send(subject, contentData)
			if err != nil {
				t.Error("Failed to send email")
			}
			_, body = setUserFailureContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
			bodyString := body.String()
			if !strings.Contains(bodyString, contentData.CommonData.Authority) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Authority, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Username) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Username, "")
			}
			if !strings.Contains(bodyString, contentData.CommonData.Name) {
				t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in template, found \"%v\"\n", subject, contentData.CommonData.Name, "")
			}

		}
	}

}
