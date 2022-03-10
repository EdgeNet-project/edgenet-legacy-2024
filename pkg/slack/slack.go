package slack

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
	"k8s.io/klog"
)

const (
	AUTH_TOKEN_IDENTIFIER = "SLACK_AUTH_TOKEN"
	CHANNEL_ID_IDENTIFIER = "SLACK_CHANNEL_ID"
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
	RoleRequest   *RoleRequest
	TenantRequest *TenantRequest
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
	attachment := slack.Attachment{
		Pretext: c.Subject,
		Text:    purpose,
		Color:   "#3e7fb8",
		Fields: []slack.AttachmentField{
			{
				Title: "User Information",
				Value: fmt.Sprintf("%s (%s %s)", c.User, c.FirstName, c.LastName),
			},
			{
				Title: "Cluster Information",
				Value: c.Cluster,
			},
			{
				Title: "Date",
				Value: time.Now().String(),
			},
		},
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
