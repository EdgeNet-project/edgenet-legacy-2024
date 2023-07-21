package notification

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"k8s.io/klog"
)

const (
	clusterRoleRequestApproveCmd = "kubectl patch clusterrolerequest %s --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/approved\", \"value\":true}]' --kubeconfig ./edgenet-kubeconfig.cfg"
	roleRequestApproveCmd        = "kubectl patch rolerequest %s -n %s --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/approved\", \"value\":true}]' --kubeconfig ./edgenet-kubeconfig.cfg"
	tenantRequestApproveCmd      = "kubectl patch tenantrequest %s --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/approved\", \"value\":true}]' --kubeconfig ./admin.cfg"
)

func (c *Content) slack(purpose string) error {
	authTokenPath := "./token"
	if flag.Lookup("slack-token-path") != nil {
		authTokenPath = flag.Lookup("slack-token-path").Value.(flag.Getter).Get().(string)
	}
	channelIDPath := "./channelid"
	if flag.Lookup("slack-channel-id-path") != nil {
		channelIDPath = flag.Lookup("slack-channel-id-path").Value.(flag.Getter).Get().(string)
	}
	authToken, err := ioutil.ReadFile(authTokenPath)
	if err != nil {
		return err
	}
	channelID, err := ioutil.ReadFile(channelIDPath)
	if err != nil {
		return err
	}

	client := slack.New(strings.TrimSpace(string(authToken)))

	googleScholarLink := fmt.Sprintf("<https://scholar.google.com/scholar?hl=en&as_sdt=0%%2C5&q=%s+%s&oq=|Google Scholar>", c.FirstName, c.LastName)

	fields := []slack.AttachmentField{
		{
			Title: "User Information",
			Value: fmt.Sprintf("%s %s, %s, %s", c.FirstName, c.LastName, c.User, googleScholarLink),
		},
		{
			Title: "Request Information",
			Value: c.getRequestInformation(),
		},
		{
			Title: "Cluster Information",
			Value: c.Cluster,
		},
		{
			Title: "Notification Date",
			Value: time.Now().Format(time.RFC1123),
		},
	}

	// If the command for approval exists also add it to the slack notification
	if command, isRequestMade := c.getCommand(purpose); isRequestMade {
		fields = append(fields, slack.AttachmentField{
			Title: "Approve via Console",
			Value: "Please click on this <https://console.edge-net.org/|link> to access the web console.",
		}, slack.AttachmentField{
			Title: "Approve via Kubectl Command",
			Value: command,
		})
	}

	// Set edgenet colors
	attachment := slack.Attachment{
		Pretext: c.Subject,
		Text:    "Please review the following details to make sure that they are corresponding information to the request owner and correct.",
		Color:   "#3e7fb8",
		Fields:  fields,
	}

	_, timestamp, err := client.PostMessage(
		strings.TrimSpace(string(channelID)),
		slack.MsgOptionAttachments(attachment),
	)
	if err != nil {
		return err
	}
	klog.V(4).Infof("Slack notification sent on %q", timestamp)
	return nil
}

func (c *Content) getRequestInformation() string {
	if c.RoleRequest != nil {
		return fmt.Sprintf("Name: %s, Namespace: %s", c.RoleRequest.Name, c.RoleRequest.Namespace)
	} else if c.TenantRequest != nil {
		return fmt.Sprintf("Name: %s", c.TenantRequest.Tenant)
	} else {
		return fmt.Sprintf("Name: %s", c.ClusterRoleRequest.Name)
	}
}

func (c *Content) getCommand(purpose string) (string, bool) {
	if purpose == "clusterrole-request-made" {
		return fmt.Sprintf(clusterRoleRequestApproveCmd, c.ClusterRoleRequest.Name), true
	} else if purpose == "rolerequest-made" {
		return fmt.Sprintf(roleRequestApproveCmd, c.RoleRequest.Name, c.RoleRequest.Namespace), true
	} else if purpose == "tenant-request-made" {
		return fmt.Sprintf(tenantRequestApproveCmd, c.TenantRequest.Tenant), true
	} else {
		return "", false
	}
}
