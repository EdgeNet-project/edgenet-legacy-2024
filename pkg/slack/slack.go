package slack

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
	// TODO: Implement 'send slack notification'

	return nil
}
