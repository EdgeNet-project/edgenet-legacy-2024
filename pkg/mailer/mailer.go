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
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/smtp"
	"os"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// commonData to have the common data
type commonData struct {
	Site     string
	Username string
	Name     string
	Email    []string
}

// CommonContentData to set the common variables
type CommonContentData struct {
	CommonData commonData
}

// ResourceAllocationData to set the common variables
type ResourceAllocationData struct {
	CommonData commonData
	Name       string
	Namespace  string
	Site       string
}

// LoginContentData to set the login-specific variables
type LoginContentData struct {
	CommonData commonData
	Token      string
}

// VerifyContentData to set the verification-specific variables
type VerifyContentData struct {
	CommonData commonData
	Code       string
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

// address to get URI of smtp server
func (s *smtpServer) address() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

// Send function consumed by the custom resources to send emails
func Send(subject string, contentData interface{}) {
	// The code below inits the SMTP configuration for sending emails
	// The path of the yaml config file of smtp server
	file, err := os.Open("../../config/smtp.yaml")
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return
	}
	decoder := yaml.NewDecoder(file)
	var smtpServer smtpServer
	err = decoder.Decode(&smtpServer)
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return
	}

	// This section determines which email to send whom
	to := []string{}
	var body bytes.Buffer
	switch subject {
	case "login":
		to, body = setLoginContent(contentData)
	case "user-email-verification":
		to, body = setUserEmailVerificationContent(contentData)
	case "user-email-verified-alert":
		to, body = setUserVerifiedAlertContent(contentData)
		if len(to) == 0 {
			to = []string{smtpServer.To}
		}
	case "user-registration-successful":
		to, body = setUserRegistrationContent(contentData)
	case "site-email-verification":
		to, body = setSiteEmailVerificationContent(contentData)
	case "site-email-verified-alert":
		_, body = setSiteVerifiedAlertContent(contentData)
		to = []string{smtpServer.To}
	case "site-registration-successful":
		to, body = setSiteRegistrationContent(contentData)
	case "acceptable-use-policy-renewal":
		to, body = setAUPRenewalContent(contentData)
	case "slice-invitation":
		to, body = setSliceInvitationContent(contentData)
	case "project-invitation":
		to, body = setProjectInvitationContent(contentData)
	}

	// Create a new Client connected to the SMTP server
	client, err := smtp.Dial(smtpServer.address())
	if err != nil {
		log.Println(err)
		return
	}
	// Check if the server supports TLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		// Start TLS to encrypt all further communication
		cfg := &tls.Config{ServerName: smtpServer.Host, InsecureSkipVerify: true}
		if err = client.StartTLS(cfg); err != nil {
			log.Println(err)
			return
		}
	}
	// Check if the server supports SMTP authentication
	if ok, _ := client.Extension("AUTH"); ok {
		// To authenticate if needed
		auth := smtp.PlainAuth("", smtpServer.Username, smtpServer.Password, smtpServer.Host)
		if err = client.Auth(auth); err != nil {
			log.Println(err)
			return
		}
	}
	// The part below starts a mail transaction by using the provided email address
	if err = client.Mail(smtpServer.From); err != nil {
		log.Println(err)
		return
	}
	// Add recipients to the email
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			log.Println(err)
			return
		}
	}
	// To write the mail headers and body
	w, err := client.Data()
	if err != nil {
		log.Println(err)
		return
	}
	_, err = w.Write(body.Bytes())
	if err != nil {
		log.Println(err)
		return
	}
	err = w.Close()
	if err != nil {
		log.Println(err)
		return
	}
	// Close the connection to the server
	client.Quit()
	log.Printf("Mailer: email sent to  %s!", to)
}

// setCommonEmailHeaders to create an email body by subject and common headers
func setCommonEmailHeaders(subject string, delimiter string) bytes.Buffer {
	var body bytes.Buffer
	headers := "MIME-Version: 1.0\r\n"
	headers += fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", delimiter)
	headers += fmt.Sprintf("\r\n--%s\r\n", delimiter)
	headers += "Content-Type: text/html; charset=\"utf-8\"\r\n"
	headers += "Content-Transfer-Encoding: 7bit\r\n"
	body.Write([]byte(fmt.Sprintf("Subject: %s\n%s\n\n", subject, headers)))

	return body
}

// setLoginContent to create an email body related to the user login activity
func setLoginContent(contentData interface{}) ([]string, bytes.Buffer) {
	loginData := contentData.(LoginContentData)
	// This represents receivers' email addresses
	to := loginData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/web-token.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("Logged In - Web Token Delivery", delimiter)
	t.Execute(&body, loginData)

	headers := fmt.Sprintf("\r\n--%s\r\n", delimiter)
	headers += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
	headers += "Content-Transfer-Encoding: base64\r\n"
	headers += "Content-Disposition: attachment;filename=\"edgenet-web-kubeconfig.cfg\"\r\n"
	// Read the kubeconfig file created for web authentication
	// It will be in the attachment of email
	rawFile, fileErr := ioutil.ReadFile(fmt.Sprintf("../../assets/kubeconfigs/edgenet-site-%s-%s-webauth.cfg", loginData.CommonData.Site,
		loginData.CommonData.Username))
	if fileErr != nil {
		log.Panic(fileErr)
	}
	attachment := "\r\n" + base64.StdEncoding.EncodeToString(rawFile)
	body.Write([]byte(fmt.Sprintf("%s%s", headers, attachment)))

	return to, body
}

// setProjectInvitationContent to create an email body related to the project invitation
func setProjectInvitationContent(contentData interface{}) ([]string, bytes.Buffer) {
	projectData := contentData.(ResourceAllocationData)
	// This represents receivers' email addresses
	to := projectData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/project-invitation.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("New Project Created", delimiter)
	t.Execute(&body, projectData)

	return to, body
}

// setSliceInvitationContent to create an email body related to the slice invitation
func setSliceInvitationContent(contentData interface{}) ([]string, bytes.Buffer) {
	sliceData := contentData.(ResourceAllocationData)
	// This represents receivers' email addresses
	to := sliceData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/slice-invitation.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("New Slice Created", delimiter)
	t.Execute(&body, sliceData)

	return to, body
}

// setAUPRenewalContent to create an email body related to the acceptable use policy renewal
func setAUPRenewalContent(contentData interface{}) ([]string, bytes.Buffer) {
	AUPData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := AUPData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/acceptable-use-policy-renewal.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("Acceptable Use Policy Expiring", delimiter)
	t.Execute(&body, AUPData)

	return to, body
}

// setSiteRegistrationContent to create an email body related to the site registration activity
func setSiteRegistrationContent(contentData interface{}) ([]string, bytes.Buffer) {
	registrationData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := registrationData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/site-registration.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("Site Registration Successful", delimiter)
	t.Execute(&body, registrationData)

	return to, body
}

// setSiteEmailVerificationContent to create an email body related to the email verification
func setSiteEmailVerificationContent(contentData interface{}) ([]string, bytes.Buffer) {
	verificationData := contentData.(VerifyContentData)
	// This represents receivers' email addresses
	to := verificationData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/site-email-verify.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("Site Registration - Email Verification", delimiter)
	t.Execute(&body, verificationData)

	return to, body
}

// setSiteVerifiedAlertContent to create an email body related to the email verified alert
func setSiteVerifiedAlertContent(contentData interface{}) ([]string, bytes.Buffer) {
	alertData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := alertData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/site-email-verified-alert.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("Site Registration - Email Verified", delimiter)
	t.Execute(&body, alertData)

	return to, body
}

// setUserRegistrationContent to create an email body related to the user registration activity
func setUserRegistrationContent(contentData interface{}) ([]string, bytes.Buffer) {
	registrationData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := registrationData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/user-registration.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("User Registration Successful", delimiter)
	t.Execute(&body, registrationData)

	headers := fmt.Sprintf("\r\n--%s\r\n", delimiter)
	headers += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
	headers += "Content-Transfer-Encoding: base64\r\n"
	headers += "Content-Disposition: attachment;filename=\"edgenet-kubeconfig.cfg\"\r\n"
	// Read the kubeconfig file created for web authentication
	// It will be in the attachment of email
	rawFile, fileErr := ioutil.ReadFile(fmt.Sprintf("../../assets/kubeconfigs/edgenet-site-%s-%s.cfg", registrationData.CommonData.Site,
		registrationData.CommonData.Username))
	if fileErr != nil {
		log.Panic(fileErr)
	}
	attachment := "\r\n" + base64.StdEncoding.EncodeToString(rawFile)
	body.Write([]byte(fmt.Sprintf("%s%s", headers, attachment)))

	return to, body
}

// setUserEmailVerificationContent to create an email body related to the email verification
func setUserEmailVerificationContent(contentData interface{}) ([]string, bytes.Buffer) {
	verificationData := contentData.(VerifyContentData)
	// This represents receivers' email addresses
	to := verificationData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/user-email-verify.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("Signed Up - Email Verification", delimiter)
	t.Execute(&body, verificationData)

	return to, body
}

// setUserVerifiedAlertContent to create an email body related to the email verified alert
func setUserVerifiedAlertContent(contentData interface{}) ([]string, bytes.Buffer) {
	alertData := contentData.(CommonContentData)
	// This represents receivers' email addresses
	to := alertData.CommonData.Email
	// The HTML template
	t, _ := template.ParseFiles("../../assets/templates/email/user-email-verified-alert.html")
	delimiter := generateRandomString(10)
	body := setCommonEmailHeaders("User Email Verified", delimiter)
	t.Execute(&body, alertData)

	return to, body
}

// generateRandomString to have a unique string
func generateRandomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
