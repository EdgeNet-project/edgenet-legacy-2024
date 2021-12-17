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
	"flag"
	"fmt"
	"log"
	"os"
	"text/template"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/klog"
)

// smtpServer implementation
type smtpServer struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	From     string `yaml:"from"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	To       string `yaml:"to"`
}

type Content struct {
	Cluster             string
	User                string
	FirstName           string
	LastName            string
	Subject             string
	Recipient           []string
	RoleRequest         *RoleRequest
	TenantRequest       *TenantRequest
	EmailVerification   *EmailVerification
	AcceptableUsePolicy *AcceptableUsePolicy
}
type RoleRequest struct {
	Name       string
	Namespace  string
	AuthMethod []string
}
type TenantRequest struct {
	Tenant     string
	AuthMethod []string
}
type EmailVerification struct {
	Code string
	URL  string
}
type AcceptableUsePolicy struct {
	Name string
}

var dir = "../.."

func (c *Content) Send(purpose string) error {
	server := mail.NewSMTPClient()

	// Prepare SMTP server configuration
	smtpInfo, err := getSMTPInformation()
	if err != nil {
		klog.V(4).Infoln(err)
		return err
	}
	server.Host = smtpInfo.Host
	server.Port = smtpInfo.Port
	server.Username = smtpInfo.Username
	server.Password = smtpInfo.Password
	server.Encryption = mail.EncryptionSTARTTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second
	server.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	// Prepare SMTP client
	smtpClient, err := server.Connect()
	if err != nil {
		klog.V(4).Infoln(err)
		return err
	}
	var htmlBody bytes.Buffer
	t, _ := template.ParseFiles(fmt.Sprintf("%s/assets/templates/email/%s.html", dir, purpose))
	t.Execute(&htmlBody, c)
	if len(c.Recipient) == 0 {
		c.Recipient = append(c.Recipient, smtpInfo.To)
	}
	for _, to := range c.Recipient {
		email := mail.NewMSG()
		email.SetFrom(smtpInfo.From).
			AddTo(to).
			SetSubject(c.Subject)
		email.SetBodyData(mail.TextHTML, htmlBody.Bytes())
		if c.RoleRequest != nil && purpose == "role-request-approved" || (c.TenantRequest != nil && purpose == "tenant-request-approved") {
			for _, method := range c.RoleRequest.AuthMethod {
				if method == "client-certificate" {
					email.Attach(&mail.File{FilePath: fmt.Sprintf("%s/assets/kubeconfigs/%s.cfg", dir, c.User), Name: fmt.Sprintf("edgenet.cfg"), Inline: true})
				}
				if method == "oidc" {
					email.Attach(&mail.File{FilePath: fmt.Sprintf("%s/assets/kubeconfigs/oidc.cfg", dir), Name: fmt.Sprintf("edgenet-oidc.cfg"), Inline: true})
				}
			}
			for _, method := range c.TenantRequest.AuthMethod {
				if method == "client-certificate" {
					email.Attach(&mail.File{FilePath: fmt.Sprintf("%s/assets/kubeconfigs/%s.cfg", dir, c.User), Name: fmt.Sprintf("edgenet.cfg"), Inline: true})
				}
				if method == "oidc" {
					email.Attach(&mail.File{FilePath: fmt.Sprintf("%s/assets/kubeconfigs/oidc.cfg", dir), Name: fmt.Sprintf("edgenet-oidc.cfg"), Inline: true})
				}
			}
		}
		if email.Error != nil {
			klog.V(4).Infoln(email.Error)
		}
		err = email.Send(smtpClient)
		if err != nil {
			klog.V(4).Infoln(err)
		} else {
			klog.V(4).Infoln(fmt.Sprintf("Email sent to %s: %s", to, c.Subject))
		}
	}
	return err
}

func getSMTPInformation() (*smtpServer, error) {
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
		return nil, err
	}
	decoder := yaml.NewDecoder(file)
	var smtpServer smtpServer
	err = decoder.Decode(&smtpServer)
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return nil, err
	}
	return &smtpServer, nil
}

// console implementation
/*type console struct {
	URL string `yaml:"url"`
}
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
verificationData.URL = console.URL*/
