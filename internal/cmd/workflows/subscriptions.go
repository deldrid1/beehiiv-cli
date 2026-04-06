package workflows

import "strings"

func subscriptionsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"subs", "subscription"},
		Short:   "Manage subscriptions for a publication",
		Long: "Read, create, update, and remove Beehiiv subscriptions. This is one of the " +
			"highest-traffic workflow groups in the CLI, so the help surface includes common lookup and mutation examples.",
		Example: strings.TrimSpace(`
beehiiv subscriptions list --query limit=100 --all
beehiiv subscriptions get sub_123
beehiiv subscriptions find person@example.com
beehiiv subscriptions create --body '{"email":"person@example.com","reactivate_existing":false}'
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List subscriptions for the active publication",
				Example: strings.TrimSpace(`
beehiiv subscriptions list --query limit=100
beehiiv subs list --query status=active --output table
beehiiv subscriptions list --all
`),
			},
			"create": {
				Aliases: []string{"add"},
				Short:   "Create a subscription",
				Example: strings.TrimSpace(`
beehiiv subscriptions create --body '{"email":"person@example.com","reactivate_existing":false}'
beehiiv subs add --body @subscription.json
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a subscription by ID",
				Example: strings.TrimSpace(`
beehiiv subscriptions get sub_123
beehiiv subscriptions show sub_123 --query expand=stats,custom_fields
`),
			},
			"get-by-email": {
				Aliases: []string{"find"},
				Short:   "Find a subscription by email address",
				Example: strings.TrimSpace(`
beehiiv subscriptions get-by-email person@example.com
beehiiv subscriptions find person@example.com
`),
			},
			"get-by-subscriber-id": {
				Aliases: []string{"subscriber", "by-subscriber"},
				Short:   "Find a subscription by subscriber ID",
				Example: strings.TrimSpace(`
beehiiv subscriptions get-by-subscriber-id subscriber_123
beehiiv subscriptions subscriber subscriber_123 --query expand=stats,custom_fields
`),
			},
			"jwt-token": {
				Aliases: []string{"jwt", "token"},
				Short:   "Fetch a JWT token for a subscription",
				Example: strings.TrimSpace(`
beehiiv subscriptions jwt-token sub_123
beehiiv subscriptions jwt sub_123
`),
			},
			"update": {
				Aliases: []string{"edit"},
				Short:   "Update a subscription by ID",
				Example: strings.TrimSpace(`
beehiiv subscriptions update sub_123 --body @subscription.json
beehiiv subs edit sub_123 --body '{"reactivate_existing":true}'
`),
			},
			"replace": {
				Aliases: []string{"set"},
				Short:   "Replace a subscription by ID",
				Example: strings.TrimSpace(`
beehiiv subscriptions replace sub_123 --body @subscription.json
beehiiv subs set sub_123 --body @subscription.json
`),
			},
			"update-by-email": {
				Aliases: []string{"edit-by-email"},
				Short:   "Update a subscription by email address",
				Example: strings.TrimSpace(`
beehiiv subscriptions update-by-email person@example.com --body @subscription.json
beehiiv subscriptions edit-by-email person@example.com --body @subscription.json
`),
			},
			"update-status": {
				Aliases: []string{"status"},
				Short:   "Update the status of a subscription",
				Example: strings.TrimSpace(`
beehiiv subscriptions update-status sub_123 --body '{"status":"inactive"}'
beehiiv subscriptions status sub_123 --body '{"status":"active"}'
`),
			},
			"replace-status": {
				Aliases: []string{"set-status"},
				Short:   "Replace the status of a subscription",
				Example: strings.TrimSpace(`
beehiiv subscriptions replace-status sub_123 --body '{"status":"inactive"}'
beehiiv subscriptions set-status sub_123 --body '{"status":"active"}'
`),
			},
			"delete": {
				Aliases: []string{"remove"},
				Short:   "Delete a subscription by ID",
				Example: strings.TrimSpace(`
beehiiv subscriptions delete sub_123
beehiiv subscriptions remove sub_123
`),
			},
		},
	}
}
