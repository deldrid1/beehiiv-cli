package workflows

import "strings"

func webhooksSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"hooks", "webhook"},
		Short:   "Manage publication webhooks",
		Long: "Create, inspect, update, test, and delete Beehiiv webhooks for the active " +
			"publication with clearer examples than the generated baseline.",
		Example: strings.TrimSpace(`
beehiiv webhooks list
beehiiv webhooks create --body @webhook.json
beehiiv webhooks ping endpoint_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List webhooks for the active publication",
				Example: strings.TrimSpace(`
beehiiv webhooks list
beehiiv hooks list --output table
`),
			},
			"create": {
				Aliases: []string{"add"},
				Short:   "Create a webhook",
				Example: strings.TrimSpace(`
beehiiv webhooks create --body @webhook.json
beehiiv hooks add --body '{"url":"https://example.com/webhook","event_types":["subscription.confirmed"]}'
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a webhook by endpoint ID",
				Example: strings.TrimSpace(`
beehiiv webhooks get endpoint_123
beehiiv webhooks show endpoint_123
`),
			},
			"test": {
				Aliases: []string{"ping"},
				Short:   "Send a test request to a webhook",
				Example: strings.TrimSpace(`
beehiiv webhooks test endpoint_123
beehiiv hooks ping endpoint_123
`),
			},
			"delete": {
				Aliases: []string{"remove"},
				Short:   "Delete a webhook by endpoint ID",
				Example: strings.TrimSpace(`
beehiiv webhooks delete endpoint_123
beehiiv webhooks remove endpoint_123
`),
			},
		},
	}
}
