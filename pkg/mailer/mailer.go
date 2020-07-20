/*
Copyright 2020 Sorbonne Université

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mailer

import (
	"bytes"
	"crypto/tls"
	"edgenet/pkg/util"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// commonData to have the common data
type commonData struct {
	Authority string
	Username  string
	Name      string
	Email     []string
}

// CommonContentData to set the common variables
type CommonContentData struct {
	CommonData commonData
}

// ResourceAllocationData to set the team and slice variables
type ResourceAllocationData struct {
	CommonData     commonData
	Name           string
	OwnerNamespace string
	ChildNamespace string
	Authority      string
}

// MultiProviderData to set the node contribution variables
type MultiProviderData struct {
	CommonData commonData
	Name       string
	Host       string
	Status     string
	Message    []string
}

// VerifyContentData to set the verification-specific variables
type VerifyContentData struct {
	CommonData commonData
	Code       string
	URL        string
}

// ValidationFailureContentData to set the failure-specific variables
type ValidationFailureContentData struct {
	Kind string
	Name string
}

// smtpServer implementation
type smtpServer struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	From     string `yaml:"from"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	To       string `yaml:"to"`
}

// console implementation
type console struct {
	URL string `yaml:"url"`
}

// address to get URI of smtp server
func (s *smtpServer) address() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

// Send function consumed by the custom resources to send emails
func Send(subject string, contentData interface{}) error {
	// The code below inits the SMTP configuration for sending emails
	// The path of the yaml config file of smtp server
	file, err := os.Open("../../configs/smtp.yaml")
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return err
	}

	if pathSMTP != "" {
		file, err = os.Open(pathSMTP)
		if err != nil {
			log.Printf("Mailer: unexpected error executing command: %v", err)
			return
		}
	}

	decoder := yaml.NewDecoder(file)
	var smtpServer smtpServer
	err = decoder.Decode(&smtpServer)
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return err
	}

	// This section determines which email to send whom
	to := []string{}
	var body bytes.Buffer
	switch subject {
	case "user-email-verification", "user-email-verification-update":
		to, body = setUserEmailVerificationContent(contentData, smtpServer.From, subject)
	case "user-email-verified-alert", "user-email-verified-notification":
		to, body = setUserVerifiedAlertContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
	case "user-registration-successful":
		to, body = setUserRegistrationContent(contentData, smtpServer.From)
	case "authority-email-verification":
		to, body = setAuthorityEmailVerificationContent(contentData, smtpServer.From)
	case "authority-email-verified-alert":
		to, body = setAuthorityVerifiedAlertContent(contentData, smtpServer.From, []string{smtpServer.To})
	case "authority-creation-successful":
		to, body = setAuthorityRequestContent(contentData, smtpServer.From)
	case "acceptable-use-policy-accepted":
		to, body = setAUPConfirmationContent(contentData, smtpServer.From)
	case "acceptable-use-policy-renewal":
		to, body = setAUPRenewalContent(contentData, smtpServer.From)
	case "acceptable-use-policy-expired":
		to, body = setAUPExpiredContent(contentData, smtpServer.From)
	case "slice-creation", "slice-removal", "slice-reminder", "slice-deletion", "slice-crash", "slice-total-quota-exceeded", "slice-lack-of-quota",
		"slice-deletion-failed", "slice-collection-deletion-failed":
		to, body = setSliceContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
	case "team-creation", "team-removal", "team-deletion", "team-crash":
		to, body = setTeamContent(contentData, smtpServer.From, subject)
	case "node-contribution-successful", "node-contribution-failure", "node-contribution-failure-support":
		to, body = setNodeContributionContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
	case "authority-validation-failure-name", "authority-validation-failure-email", "authority-email-verification-malfunction",
		"authority-creation-failure", "authority-email-verification-dubious":
		to, body = setAuthorityFailureContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
	case "user-validation-failure-name", "user-validation-failure-email", "user-email-verification-malfunction", "user-creation-failure", "user-cert-failure",
		"user-kubeconfig-failure", "user-email-verification-dubious", "user-email-verification-update-malfunction", "user-deactivation-failure":
		to, body = setUserFailureContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
	}

	// Create a new Client connected to the SMTP server
	client, err := smtp.Dial(smtpServer.address())
	if err != nil {
		log.Println(err)
		return err
	}
	// Check if the server supports TLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		// Start TLS to encrypt all further communication
		cfg := &tls.Config{ServerName: smtpServer.Host, InsecureSkipVerify: true}
		if err = client.StartTLS(cfg); err != nil {
			log.Println(err)
			return err
		}
	}
	// Check if the server supports SMTP authentication
	if ok, _ := client.Extension("AUTH"); ok {
		// To authenticate if needed
		auth := smtp.PlainAuth("", smtpServer.Username, smtpServer.Password, smtpServer.Host)
		if err = client.Auth(auth); err != nil {
			log.Println(err)
			return err
		}
	}
	// The part below starts a mail transaction by using the provided email address
	if err = client.Mail(smtpServer.From); err != nil {
		log.Println(err)
		return err
	}
	// Add recipients to the email
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			log.Println(err)
			return err
		}
	}
	// To write the mail headers and body
	w, err := client.Data()
	if err != nil {
		log.Println(err)
		return err
	}
	_, err = w.Write(body.Bytes())
	if err != nil {
		log.Println(err)
		return err
	}
	err = w.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	// Close the connection to the server
	client.Quit()
	log.Printf("Mailer: email sent to  %s!", to)
	return err
}

// setCommonEmailHeaders to create an email body by subject and common headers
func setCommonEmailHeaders(subject string, from string, to []string, delimiter string) bytes.Buffer {
	var headerTo string
	for i, addr := range to {
		if i == 0 {
			headerTo = addr
		} else {
			headerTo = fmt.Sprintf("%s, %s", headerTo, addr)
		}
	}

	var body bytes.Buffer
	headers := fmt.Sprintf("From: %s\r\n", from)
	headers += fmt.Sprintf("To: %s\r\n", headerTo)
	headers += fmt.Sprintf("Subject: %s\r\n", subject)
	headers += "MIME-Version: 1.0\r\n"
	if delimiter != "" {
		headers += fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", delimiter)
		headers += fmt.Sprintf("\r\n--%s\r\n", delimiter)
	}
	headers += "Content-Type: text/html; charset=\"utf-8\"\r\n"
	headers += "Content-Transfer-Encoding: 8bit\r\n\r\n"
	if delimiter != "" {
		headers += "\r\n"
	}
	log.Println(headers)
	//body.Write([]byte(fmt.Sprintf("Subject: %s\n%s\n\n", subject, headers)))
	body.Write([]byte(headers))
	return body
}

// setUserFailureContent to create an email body related to failures during user creation
func setUserFailureContent(contentData interface{}, from string, to []string, subject string) ([]string, bytes.Buffer) {
	NCData := contentData.(CommonContentData)
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("../../assets/templates/email/%s.html", subject))
	delimiter := ""
	title := "[EdgeNet Admin] User Creation Failure"
	if subject == "user-validation-failure-name" || subject == "user-validation-failure-email" ||
		subject == "user-creation-failure" {
		title = "[EdgeNet] User Creation Failure"
		// This represents receivers' email addresses
		to = NCData.CommonData.Email
	}
	body := setCommonEmailHeaders(title, from, to, delimiter)
	t.Execute(&body, NCData)

	return to, body
}

// setAuthorityFailureContent to create an email body related to failures during authority creation
func setAuthorityFailureContent(contentData interface{}, from string, to []string, subject string) ([]string, bytes.Buffer) {
	NCData := contentData.(CommonContentData)
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("../../assets/templates/email/%s.html", subject))
	delimiter := ""
	title := "[EdgeNet Admin] Authority Establishment Failure"
	if subject == "authority-validation-failure-name" || subject == "authority-validation-failure-email" {
		title = "[EdgeNet] Authority Establishment Failure"
		// This represents receivers' email addresses
		to = NCData.CommonData.Email
	}
	body := setCommonEmailHeaders(title, from, to, delimiter)
	t.Execute(&body, NCData)

	return to, body
}

// setNodeContributionContent to create an email body related to the node contribution notification
func setNodeContributionContent(contentData interface{}, from string, to []string, subject string) ([]string, bytes.Buffer) {
	NCData := contentData.(MultiProviderData)
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("../../assets/templates/email/%s.html", subject))
	delimiter := ""
	title := "[EdgeNet] Node contribution event"
	switch subject {
	case "node-contribution-successful":
		// This represents receivers' email addresses
		to = NCData.CommonData.Email
		title = "[EdgeNet] Node Contribution - Successful"
	case "node-contribution-failure":
		to = NCData.CommonData.Email
		title = "[EdgeNet] Node Contribution - Failed"
	case "node-contribution-failure-support":
		title = "[EdgeNet Admin] Node Contribution - Failure"
	}
	body := setCommonEmailHeaders(title, from, to, delimiter)
	t.Execute(&body, NCData)

	return to, body
}

// setTeamContent to create an email body related to the team invitation
func setTeamContent(contentData interface{}, from, subject string) ([]string, bytes.Buffer) {
	teamData := contentData.(ResourceAllocationData)
	// This represents receivers' email addresses
	to := teamData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("../../assets/templates/email/%s.html", subject))
	delimiter := ""
	title := "[EdgeNet] Team event"
	switch subject {
	case "team-creation":
		title = "[EdgeNet] Team invitation"
	case "team-removal":
		title = "[EdgeNet] Team farewell message"
	case "team-deletion":
		title = "[EdgeNet] Team deleted"
	case "team-crash":
		title = "[EdgeNet] Team creation failed"
	}
	body := setCommonEmailHeaders(title, from, to, delimiter)
	t.Execute(&body, teamData)

	return to, body
}

// setSliceContent to create an email body related to the slice emails
func setSliceContent(contentData interface{}, from string, to []string, subject string) ([]string, bytes.Buffer) {
	sliceData := contentData.(ResourceAllocationData)
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("../../assets/templates/email/%s.html", subject))
	delimiter := ""
	title := "[EdgeNet] Slice event"
	switch subject {
	case "slice-creation":
		// This represents receivers' email addresses
		to = sliceData.CommonData.Email
		title = "[EdgeNet] Slice invitation"
	case "slice-removal":
		to = sliceData.CommonData.Email
		title = "[EdgeNet] Slice farewell message"
	case "slice-reminder":
		to = sliceData.CommonData.Email
		title = "[EdgeNet] Slice renewal reminder"
	case "slice-deletion":
		to = sliceData.CommonData.Email
		title = "[EdgeNet] Slice deleted"
	case "slice-crash":
		to = sliceData.CommonData.Email
		title = "[EdgeNet] Slice creation failed"
	case "slice-total-quota-exceeded":
		to = sliceData.CommonData.Email
		title = "[EdgeNet] Slice could not be created"
	case "slice-lack-of-quota":
		to = sliceData.CommonData.Email
		title = "[EdgeNet] Slice profile could not be changed"
	case "slice-deletion-failed", "slice-collection-deletion-failed":
		title = "[EdgeNet] Slice deletion failed"
	}
	body := setCommonEmailHeaders(title, from, to, delimiter)
	t.Execute(&body, sliceData)

	return to, body
}

// setAUPConfirmationContent to create an email body related to the acceptable use policy confirmation
func setAUPConfirmationContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	AUPData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := AUPData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/acceptable-use-policy-confirmation.html")
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] Acceptable Use Policy Confirmed", from, to, delimiter)
	t.Execute(&body, AUPData)

	return to, body
}

// setAUPExpiredContent to create an email body related to the acceptable use policy expired
func setAUPExpiredContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	AUPData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := AUPData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/acceptable-use-policy-expired.html")
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] Acceptable Use Policy Expired", from, to, delimiter)
	t.Execute(&body, AUPData)

	return to, body
}

// setAUPRenewalContent to create an email body related to the acceptable use policy renewal
func setAUPRenewalContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	AUPData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := AUPData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/acceptable-use-policy-renewal.html")
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] Acceptable Use Policy Expiring", from, to, delimiter)
	t.Execute(&body, AUPData)

	return to, body
}

// setAuthorityRequestContent to create an email body related to the authority creation activity
func setAuthorityRequestContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	registrationData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := registrationData.CommonData.Email
	// The HTML template
	// Getting the command line argument(if existed) and use another path for authority-creation.html file
	var authorityCreationTemplate string
	commandLine := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	commandLine.StringVar(&authorityCreationTemplate, "authorityCreationTemplate-path", "", "authorityCreationTemplate-path")
	commandLine.Parse(os.Args[6:8])
	t, err := template.ParseFiles("../../assets/templates/email/authority-creation.html")
	if err != nil {
		t, err = template.ParseFiles(authorityCreationTemplate)
	}
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] Authority Successfully Created", from, to, delimiter)
	t.Execute(&body, registrationData)

	return to, body
}

// setAuthorityEmailVerificationContent to create an email body related to the email verification
func setAuthorityEmailVerificationContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	verificationData := contentData.(VerifyContentData)
	file, err := os.Open("../../configs/console.yaml")
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
	}
	decoder := yaml.NewDecoder(file)
	var console console
	err = decoder.Decode(&console)
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
	}
	verificationData.URL = console.URL
	// This represents receivers' email addresses
	to := verificationData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/authority-email-verification.html")
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] Authority Registration Request - Do You Confirm?", from, to, delimiter)
	t.Execute(&body, verificationData)

	return to, body
}

// setAuthorityVerifiedAlertContent to create an email body related to the email verified alert
func setAuthorityVerifiedAlertContent(contentData interface{}, from string, to []string) ([]string, bytes.Buffer) {
	alertData := contentData.(CommonContentData)
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/authority-email-verified-alert.html")
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet Admin] Authority Request - Email Verified", from, to, delimiter)
	t.Execute(&body, alertData)

	return to, body
}

// setUserRegistrationContent to create an email body related to the user registration activity
func setUserRegistrationContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	registrationData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := registrationData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/user-registration.html")
	delimiter := util.GenerateRandomString(10)
	body := setCommonEmailHeaders("[EdgeNet] User Registration Successful", from, to, delimiter)
	t.Execute(&body, registrationData)

	headers := fmt.Sprintf("--%s\r\n", delimiter)
	headers += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
	headers += "Content-Transfer-Encoding: base64\r\n"
	headers += "Content-Disposition: attachment;filename=\"edgenet-kubeconfig.cfg\"\r\n"
	// Read the kubeconfig file created for web authentication
	// It will be in the attachment of email
	rawFile, fileErr := ioutil.ReadFile(fmt.Sprintf("../../assets/kubeconfigs/%s-%s.cfg", registrationData.CommonData.Authority,
		registrationData.CommonData.Username))
	if fileErr != nil {
		log.Panic(fileErr)
	}
	attachment := "\r\n" + base64.StdEncoding.EncodeToString(rawFile)
	body.Write([]byte(fmt.Sprintf("%s%s\r\n\r\n--%s--", headers, attachment, delimiter)))

	return to, body
}

// setUserEmailVerificationContent to create an email body related to the email verification
func setUserEmailVerificationContent(contentData interface{}, from, subject string) ([]string, bytes.Buffer) {
	verificationData := contentData.(VerifyContentData)
	file, err := os.Open("../../configs/console.yaml")
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
	}
	decoder := yaml.NewDecoder(file)
	var console console
	err = decoder.Decode(&console)
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
	}
	verificationData.URL = console.URL
	// This represents receivers' email addresses
	to := verificationData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("../../assets/templates/email/%s.html", subject))
	delimiter := ""
	title := "[EdgeNet] Email Verification"
	switch subject {
	case "user-email-verification":
		title = "[EdgeNet] Signed Up - Email Verification"
	case "user-email-verification-update":
		title = "[EdgeNet] User Updated - Email Verification"
	}
	body := setCommonEmailHeaders(title, from, to, delimiter)
	t.Execute(&body, verificationData)

	return to, body
}

// setUserVerifiedAlertContent to create an email body related to the email verified alert
func setUserVerifiedAlertContent(contentData interface{}, from string, to []string, subject string) ([]string, bytes.Buffer) {
	alertData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	if len(alertData.CommonData.Email) > 0 {
		to = alertData.CommonData.Email
	}
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("../../assets/templates/email/%s.html", subject))
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] User Email Verified", from, to, delimiter)
	t.Execute(&body, alertData)

	return to, body
}
