/*
Copyright 2021 Contributors to the EdgeNet project.

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
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"

	"github.com/EdgeNet-project/edgenet/pkg/util"

	yaml "gopkg.in/yaml.v2"
)

// commonData to have the common data
type commonData struct {
	Tenant      string
	Namespace   string
	Cluster     string
	Name        string
	Username    string
	RoleRequest string
	Email       []string
	AuthMethod  []string
}

// CommonContentData to set the common variables
type CommonContentData struct {
	CommonData commonData
}

// ResourceAllocationData to set the team and subnamespace variables
type ResourceAllocationData struct {
	CommonData     commonData
	Name           string
	OwnerNamespace string
	ChildNamespace string
	Tenant         string
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

var dir = "../.."

// address to get URI of smtp server
func (s *smtpServer) address() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

// Send function consumed by the custom resources to send emails
func Send(subject string, contentData interface{}) error {
	// The code below inits the SMTP configuration for sending emails
	if flag.Lookup("dir") != nil {
		dir = flag.Lookup("dir").Value.(flag.Getter).Get().(string)
	}
	// The path of the yaml config file of smtp server
	var pathSMTP string
	if flag.Lookup("smtp-path") != nil {
		pathSMTP = flag.Lookup("smtp-path").Value.(flag.Getter).Get().(string)
	}
	if pathSMTP == "" {
		pathSMTP = fmt.Sprintf("%s/configs/smtp.yaml", dir)
	}
	file, err := os.Open(pathSMTP)
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return err
	}
	decoder := yaml.NewDecoder(file)
	var smtpServer smtpServer
	err = decoder.Decode(&smtpServer)
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return err
	}

	// This section determines which email to send whom
	to, body := prepareNotification(subject, contentData, smtpServer)

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

func prepareNotification(subject string, contentData interface{}, smtpServer smtpServer) ([]string, bytes.Buffer) {
	to := []string{}
	var body bytes.Buffer
	switch subject {
	case "user-email-verification", "user-email-verification-update":
		to, body = setUserEmailVerificationContent(contentData, smtpServer.From, subject)
	case "user-email-verified-alert", "user-email-verified-notification":
		to, body = setUserVerifiedAlertContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
	case "user-registration-successful":
		to, body = setUserRegistrationContent(contentData, smtpServer.From)
	case "tenant-email-verification":
		to, body = setTenantEmailVerificationContent(contentData, smtpServer.From)
	case "tenant-email-verified-alert":
		to, body = setTenantVerifiedAlertContent(contentData, smtpServer.From, []string{smtpServer.To})
	case "tenant-creation-successful":
		to, body = setTenantRequestContent(contentData, smtpServer.From)
	case "acceptable-use-policy-accepted":
		to, body = setAUPConfirmationContent(contentData, smtpServer.From)
	case "acceptable-use-policy-renewal":
		to, body = setAUPRenewalContent(contentData, smtpServer.From)
	case "acceptable-use-policy-expired":
		to, body = setAUPExpiredContent(contentData, smtpServer.From)
	case "node-contribution-successful", "node-contribution-failure", "node-contribution-failure-support":
		to, body = setNodeContributionContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
	case "tenant-validation-failure-name", "tenant-validation-failure-email", "tenant-email-verification-malfunction",
		"tenant-creation-failure", "tenant-email-verification-dubious":
		to, body = setTenantFailureContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
	case "user-validation-failure-name", "user-validation-failure-email", "user-email-verification-malfunction", "user-creation-failure", "user-cert-failure",
		"user-kubeconfig-failure", "user-email-verification-dubious", "user-email-verification-update-malfunction", "user-deactivation-failure":
		to, body = setUserFailureContent(contentData, smtpServer.From, []string{smtpServer.To}, subject)
	}
	return to, body
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
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/%s.html", dir, subject))
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

// setTenantFailureContent to create an email body related to failures during tenant creation
func setTenantFailureContent(contentData interface{}, from string, to []string, subject string) ([]string, bytes.Buffer) {
	NCData := contentData.(CommonContentData)
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/%s.html", dir, subject))
	delimiter := ""
	title := "[EdgeNet Admin] Tenant Establishment Failure"
	if subject == "tenant-validation-failure-name" || subject == "tenant-validation-failure-email" {
		title = "[EdgeNet] Tenant Establishment Failure"
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
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/%s.html", dir, subject))
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

// setAUPConfirmationContent to create an email body related to the acceptable use policy confirmation
func setAUPConfirmationContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	AUPData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := AUPData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/acceptable-use-policy-confirmation.html", dir))
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
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/acceptable-use-policy-expired.html", dir))
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
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/acceptable-use-policy-renewal.html", dir))
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] Acceptable Use Policy Expiring", from, to, delimiter)
	t.Execute(&body, AUPData)

	return to, body
}

// setTenantRequestContent to create an email body related to the tenant creation activity
func setTenantRequestContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	registrationData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := registrationData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/tenant-creation.html", dir))
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] Tenant Successfully Created", from, to, delimiter)
	t.Execute(&body, registrationData)

	return to, body
}

// setTenantEmailVerificationContent to create an email body related to the email verification
func setTenantEmailVerificationContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	verificationData := contentData.(VerifyContentData)
	file, err := os.Open(fmt.Sprintf("%s/configs/console.yaml", dir))
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
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/tenant-email-verification.html", dir))
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] Tenant Registration Request - Do You Confirm?", from, to, delimiter)
	t.Execute(&body, verificationData)

	return to, body
}

// setTenantVerifiedAlertContent to create an email body related to the email verified alert
func setTenantVerifiedAlertContent(contentData interface{}, from string, to []string) ([]string, bytes.Buffer) {
	alertData := contentData.(CommonContentData)
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/tenant-email-verified-alert.html", dir))
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet Admin] Tenant Request - Email Verified", from, to, delimiter)
	t.Execute(&body, alertData)

	return to, body
}

// setUserRegistrationContent to create an email body related to the user registration activity
func setUserRegistrationContent(contentData interface{}, from string) ([]string, bytes.Buffer) {
	registrationData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := registrationData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/user-registration.html", dir))
	delimiter := util.GenerateRandomString(10)
	body := setCommonEmailHeaders("[EdgeNet] User Registration Successful", from, to, delimiter)
	t.Execute(&body, registrationData)

	headers := fmt.Sprintf("--%s\r\n", delimiter)
	headers += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
	headers += "Content-Transfer-Encoding: base64\r\n"
	headers += "Content-Disposition: attachment;filename=\"edgenet-kubeconfig.cfg\"\r\n"
	// Read the kubeconfig file created for web authentication
	// It will be in the attachment of email
	rawFile, fileErr := ioutil.ReadFile(fmt.Sprintf("%s/assets/kubeconfigs/%s-%s.cfg", dir, registrationData.CommonData.Tenant,
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
	file, err := os.Open(fmt.Sprintf("%s/configs/console.yaml", dir))
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
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/%s.html", dir, subject))
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
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/%s.html", dir, subject))
	delimiter := ""
	body := setCommonEmailHeaders("[EdgeNet] User Email Verified", from, to, delimiter)
	t.Execute(&body, alertData)

	return to, body
}
