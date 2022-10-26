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
	CLUSTER_ROLE_REQUEST_MADE = "kubectl patch clusterrolerequest %s --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/approved\", \"value\":true}]' --kubeconfig ./edgenet-kubeconfig.cfg"
	ROLE_REQUEST_MADE         = "kubectl patch rolerequest %s -n %s --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/approved\", \"value\":true}]' --kubeconfig ./edgenet-kubeconfig.cfg"
	TENANT_REQUEST_MADE       = "kubectl patch tenantrequest %s --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/approved\", \"value\":true}]' --kubeconfig ./admin.cfg"
)

func (c *Content) slack(purpose string) error {
	authTokenPath := "./token"
	if flag.Lookup("slack-token-path") != nil {
		authTokenPath = flag.Lookup("slack-token-path").Value.(flag.Getter).Get().(string)
	}
	channelIdPath := "./channelid"
	if flag.Lookup("slack-channel-id-path") != nil {
		channelIdPath = flag.Lookup("slack-channel-id-path").Value.(flag.Getter).Get().(string)
	}
	authToken, err := ioutil.ReadFile(authTokenPath)
	if err != nil {
		return err
	}
	channelId, err := ioutil.ReadFile(channelIdPath)
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
			Title: "Cluster Information",
			Value: c.Cluster,
		},
		{
			Title: "Notification Date",
			Value: time.Now().Format(time.RFC1123),
		},
		{
			Title: "Approve via Console",
			Value: "Please click on this <https://console.edge-net.org/|link> to access the web console.",
		},
	}

	// If the command for approval exists also add it to the slack notification
	if purpose == "clusterrole-request-made" {
		fields = append(fields, slack.AttachmentField{
			Title: "Approve via Kubectl Command",
			Value: fmt.Sprintf(CLUSTER_ROLE_REQUEST_MADE, c.ClusterRoleRequest.Name),
		})
	} else if purpose == "rolerequest-made" {
		fields = append(fields, slack.AttachmentField{
			Title: "Approve via Kubectl Command",
			Value: fmt.Sprintf(ROLE_REQUEST_MADE, c.RoleRequest.Name, c.RoleRequest.Namespace),
		})
	} else if purpose == "tenant-request-made" {
		fields = append(fields, slack.AttachmentField{
			Title: "Approve via Kubectl Command",
			Value: fmt.Sprintf(TENANT_REQUEST_MADE, c.TenantRequest.Tenant),
		})
	}

	// Set edgenet colors
	attachment := slack.Attachment{
		Pretext: c.Subject,
		Text:    "Please review the following details to make sure that they are corresponding information to the tenant and correct.",
		Color:   "#3e7fb8",
		Fields:  fields,
	}

	_, timestamp, err := client.PostMessage(
		strings.TrimSpace(string(channelId)),
		slack.MsgOptionAttachments(attachment),
	)

	if err != nil {
		return err
	}

	klog.V(4).Infof("Slack notification sent on %q", timestamp)

	return nil
}
