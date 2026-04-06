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
