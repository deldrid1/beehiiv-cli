package workflows

import "strings"

func exampleBlock(value string) string {
	return strings.TrimSpace(value)
}

func advertisementOpportunitiesSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"ad-opps", "ad-opportunities"},
		Short:   "List Beehiiv advertisement opportunities",
		Long:    "Inspect advertisement opportunities available to the active publication.",
		Example: exampleBlock(`
beehiiv advertisement-opportunities list
beehiiv ad-opps list --query limit=25
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List advertisement opportunities for the active publication",
				Example: exampleBlock(`
beehiiv advertisement-opportunities list
beehiiv ad-opps list --query limit=25
`),
			},
		},
	}
}

func authorsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"author"},
		Short:   "Inspect publication authors",
		Long:    "List authors for the active publication and show an individual author by ID.",
		Example: exampleBlock(`
beehiiv authors list
beehiiv authors show author_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List authors for the active publication",
				Example: exampleBlock(`
beehiiv authors list
beehiiv author list --output table
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show an author by ID",
				Example: exampleBlock(`
beehiiv authors get author_123
beehiiv authors show author_123
`),
			},
		},
	}
}

func bulkSubscriptionUpdatesSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"bulk-updates", "bulk-update"},
		Short:   "Inspect bulk subscription update jobs",
		Long:    "Track bulk subscription update jobs after you submit a bulk change request.",
		Example: exampleBlock(`
beehiiv bulk-subscription-updates list
beehiiv bulk-subscription-updates show bulk_subscription_update_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List bulk subscription update jobs",
				Example: exampleBlock(`
beehiiv bulk-subscription-updates list
beehiiv bulk-updates list --output table
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a bulk subscription update job",
				Example: exampleBlock(`
beehiiv bulk-subscription-updates get bulk_subscription_update_123
beehiiv bulk-updates show bulk_subscription_update_123
`),
			},
		},
	}
}

func bulkSubscriptionsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"bulk-subscription", "bulk-imports"},
		Short:   "Import subscribers in bulk",
		Long:    "Create subscriptions in bulk for the active publication from a JSON request body.",
		Example: exampleBlock(`
beehiiv bulk-subscriptions create --body @bulk-subscriptions.json
beehiiv bulk-imports import --body @bulk-subscriptions.json
`),
		Actions: map[string]ActionSpec{
			"create": {
				Aliases: []string{"import", "add"},
				Short:   "Create subscriptions in bulk",
				Example: exampleBlock(`
beehiiv bulk-subscriptions create --body @bulk-subscriptions.json
beehiiv bulk-imports import --body @bulk-subscriptions.json
`),
			},
		},
	}
}

func conditionSetsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"condition-set", "conditions"},
		Short:   "Inspect condition sets",
		Long:    "List condition sets for the active publication and inspect an individual set by ID.",
		Example: exampleBlock(`
beehiiv condition-sets list
beehiiv condition-sets show cond_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List condition sets",
				Example: exampleBlock(`
beehiiv condition-sets list
beehiiv conditions list --output table
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a condition set by ID",
				Example: exampleBlock(`
beehiiv condition-sets get cond_123
beehiiv conditions show cond_123
`),
			},
		},
	}
}

func customFieldsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"custom-field", "fields", "field"},
		Short:   "Manage publication custom fields",
		Long:    "Create, inspect, update, replace, and delete custom fields for the active publication.",
		Example: exampleBlock(`
beehiiv custom-fields list
beehiiv custom-fields show custom_field_123
beehiiv custom-fields create --body '{"kind":"string","display":"Favorite Airport"}'
beehiiv fields add --body @custom-field.json
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List custom fields for the active publication",
				Example: exampleBlock(`
beehiiv custom-fields list
beehiiv fields list --output table
`),
			},
			"create": {
				Aliases: []string{"add"},
				Short:   "Create a custom field",
				Example: exampleBlock(`
beehiiv custom-fields create --body '{"kind":"string","display":"Favorite Airport"}'
beehiiv fields add --body @custom-field.json
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a custom field by ID",
				Example: exampleBlock(`
beehiiv custom-fields get custom_field_123
beehiiv fields show custom_field_123
`),
			},
			"update": {
				Aliases: []string{"edit"},
				Short:   "Update a custom field",
				Example: exampleBlock(`
beehiiv custom-fields update custom_field_123 --body @custom-field.json
beehiiv fields edit custom_field_123 --body '{"display":"Favorite Airport"}'
`),
			},
			"replace": {
				Aliases: []string{"set"},
				Short:   "Replace a custom field",
				Example: exampleBlock(`
beehiiv custom-fields replace custom_field_123 --body @custom-field.json
beehiiv fields set custom_field_123 --body @custom-field.json
`),
			},
			"delete": {
				Aliases: []string{"remove"},
				Short:   "Delete a custom field",
				Example: exampleBlock(`
beehiiv custom-fields delete custom_field_123
beehiiv fields remove custom_field_123
`),
			},
		},
	}
}

func emailBlastsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"blast", "blasts"},
		Short:   "Inspect email blasts",
		Long:    "List email blasts for the active publication and inspect an individual blast by ID.",
		Example: exampleBlock(`
beehiiv email-blasts list
beehiiv email-blasts show blast_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List email blasts for the active publication",
				Example: exampleBlock(`
beehiiv email-blasts list
beehiiv blasts list --query limit=25
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show an email blast by ID",
				Example: exampleBlock(`
beehiiv email-blasts get blast_123
beehiiv blasts show blast_123
`),
			},
		},
	}
}

func engagementsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"engagement"},
		Short:   "Inspect publication engagement events",
		Long:    "List engagement events returned by the Beehiiv API for the active publication.",
		Example: exampleBlock(`
beehiiv engagements list
beehiiv engagement list --query limit=50
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List engagement events",
				Example: exampleBlock(`
beehiiv engagements list
beehiiv engagement list --query limit=50
`),
			},
		},
	}
}

func newsletterListsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"newsletter-list", "nlists"},
		Short:   "Inspect newsletter lists",
		Long:    "List newsletter lists for the active publication and inspect a single list by ID.",
		Example: exampleBlock(`
beehiiv newsletter-lists list
beehiiv newsletter-lists show list_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List newsletter lists for the active publication",
				Example: exampleBlock(`
beehiiv newsletter-lists list
beehiiv nlists list --output table
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a newsletter list by ID",
				Example: exampleBlock(`
beehiiv newsletter-lists get list_123
beehiiv nlists show list_123
`),
			},
		},
	}
}

func pollsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"poll"},
		Short:   "Inspect publication polls",
		Long:    "List polls, inspect a specific poll, and jump straight into the responses for that poll.",
		Example: exampleBlock(`
beehiiv polls list --query expand=stats
beehiiv polls show poll_123 --query expand=stats
beehiiv polls responses poll_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List polls for the active publication",
				Example: exampleBlock(`
beehiiv polls list
beehiiv polls list --query expand=stats
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a poll by ID",
				Example: exampleBlock(`
beehiiv polls get poll_123
beehiiv polls show poll_123 --query expand=stats,poll_responses
`),
			},
		},
	}
}

func pollResponsesSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"poll-response"},
		Short:   "List individual responses for a Beehiiv poll",
		Long:    "List paginated individual poll responses for a specific poll.",
		Example: exampleBlock(`
beehiiv poll-responses list poll_123
beehiiv polls responses poll_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List responses for a poll",
				Example: exampleBlock(`
beehiiv poll-responses list poll_123
beehiiv polls responses poll_123 --query expand=post
`),
			},
		},
	}
}

func postTemplatesSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"post-template", "templates"},
		Short:   "List post templates",
		Long:    "Inspect post templates available to the active publication.",
		Example: exampleBlock(`
beehiiv post-templates list
beehiiv templates list --output table
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List post templates",
				Example: exampleBlock(`
beehiiv post-templates list
beehiiv templates list --output table
`),
			},
		},
	}
}

func referralProgramSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"referrals", "referral"},
		Short:   "Inspect the publication referral program",
		Long:    "Show referral program configuration and rewards for the active publication.",
		Example: exampleBlock(`
beehiiv referral-program show
beehiiv referrals show
`),
		Actions: map[string]ActionSpec{
			"get": {
				Aliases: []string{"show"},
				Short:   "Show the referral program",
				Example: exampleBlock(`
beehiiv referral-program get
beehiiv referrals show
`),
			},
		},
	}
}

func segmentsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"segment"},
		Short:   "Inspect and manage publication segments",
		Long:    "List segments, inspect a single segment, recalculate it, and jump into segment members or lightweight results.",
		Example: exampleBlock(`
beehiiv segments list
beehiiv segments show segment_123
beehiiv segments members segment_123 --query expand=stats,custom_fields
beehiiv segments results segment_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List segments for the active publication",
				Example: exampleBlock(`
beehiiv segments list
beehiiv segment list --output table
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a segment by ID",
				Example: exampleBlock(`
beehiiv segments get segment_123
beehiiv segments show segment_123
`),
			},
			"recalculate": {
				Aliases: []string{"refresh", "recalc"},
				Short:   "Recalculate a segment",
				Example: exampleBlock(`
beehiiv segments recalculate segment_123
beehiiv segments refresh segment_123
`),
			},
			"delete": {
				Aliases: []string{"remove"},
				Short:   "Delete a segment",
				Example: exampleBlock(`
beehiiv segments delete segment_123
beehiiv segments remove segment_123
`),
			},
		},
	}
}

func segmentMembersSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"segment-member"},
		Short:   "List full subscriber records for a segment",
		Long:    "List segment members with full subscription data for a specific segment.",
		Example: exampleBlock(`
beehiiv segment-members list segment_123
beehiiv segments members segment_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List full subscriber records for a segment",
				Example: exampleBlock(`
beehiiv segment-members list segment_123
beehiiv segments members segment_123 --query expand=stats,custom_fields
`),
			},
		},
	}
}

func segmentResultsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"segment-result"},
		Short:   "List lightweight segment results",
		Long:    "List lightweight segment results when you only need IDs or a lighter-weight response than full segment members.",
		Example: exampleBlock(`
beehiiv segment-results list segment_123
beehiiv segments results segment_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List lightweight results for a segment",
				Example: exampleBlock(`
beehiiv segment-results list segment_123
beehiiv segments results segment_123
`),
			},
		},
	}
}

func subscriptionBulkActionsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"subscription-bulk", "sub-bulk-actions"},
		Short:   "Update subscription bulk-action settings",
		Long:    "Update or replace bulk-action settings for subscriptions in the active publication.",
		Example: exampleBlock(`
beehiiv subscription-bulk-actions update --body @bulk-action.json
beehiiv subscription-bulk-actions set --body @bulk-action.json
`),
		Actions: map[string]ActionSpec{
			"update": {
				Aliases: []string{"edit"},
				Short:   "Update subscription bulk-action settings",
				Example: exampleBlock(`
beehiiv subscription-bulk-actions update --body @bulk-action.json
beehiiv subscription-bulk-actions edit --body @bulk-action.json
`),
			},
			"replace": {
				Aliases: []string{"set"},
				Short:   "Replace subscription bulk-action settings",
				Example: exampleBlock(`
beehiiv subscription-bulk-actions replace --body @bulk-action.json
beehiiv subscription-bulk-actions set --body @bulk-action.json
`),
			},
		},
	}
}

func subscriptionTagsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"subscription-tag", "tags"},
		Short:   "Create subscription tags",
		Long:    "Create tags that can be attached to subscriptions in the active publication.",
		Example: exampleBlock(`
beehiiv subscription-tags create --body '{"name":"vip"}'
beehiiv tags add --body '{"name":"vip"}'
`),
		Actions: map[string]ActionSpec{
			"create": {
				Aliases: []string{"add"},
				Short:   "Create a subscription tag",
				Example: exampleBlock(`
beehiiv subscription-tags create --body '{"name":"vip"}'
beehiiv tags add --body '{"name":"vip"}'
`),
			},
		},
	}
}

func tiersSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"tier"},
		Short:   "Manage premium tiers",
		Long:    "Create, inspect, update, and replace premium tiers for the active publication.",
		Example: exampleBlock(`
beehiiv tiers list
beehiiv tiers show tier_123
beehiiv tiers create --body @tier.json
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List premium tiers for the active publication",
				Example: exampleBlock(`
beehiiv tiers list
beehiiv tier list --output table
`),
			},
			"create": {
				Aliases: []string{"add"},
				Short:   "Create a premium tier",
				Example: exampleBlock(`
beehiiv tiers create --body @tier.json
beehiiv tier add --body @tier.json
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a premium tier by ID",
				Example: exampleBlock(`
beehiiv tiers get tier_123
beehiiv tiers show tier_123
`),
			},
			"update": {
				Aliases: []string{"edit"},
				Short:   "Update a premium tier",
				Example: exampleBlock(`
beehiiv tiers update tier_123 --body @tier.json
beehiiv tier edit tier_123 --body @tier.json
`),
			},
			"replace": {
				Aliases: []string{"set"},
				Short:   "Replace a premium tier",
				Example: exampleBlock(`
beehiiv tiers replace tier_123 --body @tier.json
beehiiv tier set tier_123 --body @tier.json
`),
			},
		},
	}
}

func workspacesSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"workspace"},
		Short:   "Inspect workspace-level data",
		Long:    "Inspect workspace identity and look up publications by subscriber email across the workspace tied to your API key.",
		Example: exampleBlock(`
beehiiv workspaces identify
beehiiv workspaces publications person@example.com --query expand=publication,subscription
`),
		Actions: map[string]ActionSpec{
			"identify": {
				Aliases: []string{"whoami"},
				Short:   "Identify the current workspace",
				Example: exampleBlock(`
beehiiv workspaces identify
beehiiv workspace whoami
`),
			},
			"publications-by-subscription-email": {
				Aliases: []string{"publications", "lookup"},
				Short:   "Find workspace publications by subscriber email",
				Example: exampleBlock(`
beehiiv workspaces publications person@example.com
beehiiv workspaces lookup person@example.com --query expand=publication,subscription
`),
			},
		},
	}
}
