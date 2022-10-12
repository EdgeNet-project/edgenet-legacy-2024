package slack

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
	"k8s.io/klog"
)

const (
	CLUSTER_ROLE_REQUEST_MADE = "kubectl patch clusterrolerequest %s --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/approved\", \"value\":true}]' --kubeconfig ./edgenet-kubeconfig.cfg"
	ROLE_REQUEST_MADE         = "kubectl patch rolerequest %s -n %s --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/approved\", \"value\":true}]' --kubeconfig ./edgenet-kubeconfig.cfg"
	TENANT_REQUEST_MADE       = "kubectl patch tenantrequest %s --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/approved\", \"value\":true}]' --kubeconfig ./admin.cfg"
)

type Content struct {
	Cluster   string
	User      string
	FirstName string
	LastName  string
	Subject   string
	AuthToken string
	ChannelId string
	// There are not recipient since this the notification will only be sent to a channel.
	// If requested this may be changed to support multiple Slack channels.
	// Recipient     []string
	ClusterRolerequest *ClusterRoleRequest
	RoleRequest        *RoleRequest
	TenantRequest      *TenantRequest
}

type ClusterRoleRequest struct {
	Name string
}

type RoleRequest struct {
	Name      string
	Namespace string
}

type TenantRequest struct {
	Tenant string
}

func (c *Content) Send(purpose string) error {
	client := slack.New(c.AuthToken)

	googleScholarLink := fmt.Sprintf("<https://scholar.google.com/scholar?hl=en&as_sdt=0%%2C5&q=%s+%s&oq=|Google Scholar>", c.FirstName, c.LastName)

	fields := []slack.AttachmentField{
		{
			Title: "User Information",
			Value: fmt.Sprintf("%s (%s %s) Google Scholar: %s", c.User, c.FirstName, c.LastName, googleScholarLink),
		},
		{
			Title: "Cluster Information",
			Value: c.Cluster,
		},
		{
			Title: "Date",
			Value: time.Now().String(),
		},
		{
			Title: "Console Link",
			Value: "Use this <https://www.edge-net.org/|link>",
		},
	}

	// If the command for approval exists also add it to the slack notification
	if purpose == "clusterrole-request-made" {
		fields = append(fields, slack.AttachmentField{
			Title: "Console Command",
			Value: fmt.Sprintf(CLUSTER_ROLE_REQUEST_MADE, c.ClusterRolerequest.Name),
		})
	} else if purpose == "rolerequest-made" {
		fields = append(fields, slack.AttachmentField{
			Title: "Console Command",
			Value: fmt.Sprintf(ROLE_REQUEST_MADE, c.RoleRequest.Name, c.RoleRequest.Namespace),
		})
	} else if purpose == "tenant-request-made" {
		fields = append(fields, slack.AttachmentField{
			Title: "Console Command",
			Value: fmt.Sprintf(TENANT_REQUEST_MADE, c.TenantRequest.Tenant),
		})
	}

	// Set edgenet colors
	attachment := slack.Attachment{
		Pretext: c.Subject,
		Text:    purpose,
		Color:   "#3e7fb8",
		Fields:  fields,
	}

	_, timestamp, err := client.PostMessage(
		c.ChannelId,
		slack.MsgOptionAttachments(attachment),
	)

	if err != nil {
		return err
	}

	klog.V(4).Infof("Slack notification sent on %q", timestamp)

	return nil
}
