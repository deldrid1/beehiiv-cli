package workflows

import "strings"

func automationsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"automation", "autos"},
		Short:   "Inspect Beehiiv automations and related workflow activity",
		Long: "List automation definitions, inspect a single automation with optional stats, and " +
			"jump into the emails and journeys that belong to a specific automation.",
		Example: strings.TrimSpace(`
beehiiv automations list
beehiiv automations list --query expand=stats
beehiiv automations show aut_123 --query expand=stats
beehiiv automations emails aut_123
beehiiv automations journeys aut_123
beehiiv automations enroll aut_123 --body '{"email":"person@example.com"}'
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List automations for the active publication",
				Example: strings.TrimSpace(`
beehiiv automations list
beehiiv automations list --query expand=stats --output table
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a single automation by ID",
				Example: strings.TrimSpace(`
beehiiv automations get aut_123
beehiiv automations show aut_123 --query expand=stats
`),
			},
		},
	}
}

func automationEmailsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"automation-email"},
		Short:   "List the emails that belong to a Beehiiv automation",
		Long: "Inspect the emails attached to a specific automation, including the per-email " +
			"engagement statistics Beehiiv returns for that automation.",
		Example: strings.TrimSpace(`
beehiiv automation-emails list aut_123
beehiiv automations emails aut_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List the emails inside an automation",
				Example: strings.TrimSpace(`
beehiiv automation-emails list aut_123
beehiiv automations emails aut_123 --all
`),
			},
		},
	}
}

func automationJourneysSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"automation-journey"},
		Short:   "Inspect and manage automation journeys",
		Long: "List journeys inside an automation, inspect a specific journey, or enroll an " +
			"existing subscriber when the automation has an active Add by API trigger.",
		Example: strings.TrimSpace(`
beehiiv automation-journeys list aut_123
beehiiv automation-journeys get aut_123 journey_123
beehiiv automation-journeys create aut_123 --body '{"email":"person@example.com"}'
beehiiv automations journeys aut_123
beehiiv automations enroll aut_123 --body '{"subscription_id":"sub_123"}'
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List journeys that occurred inside an automation",
				Example: strings.TrimSpace(`
beehiiv automation-journeys list aut_123
beehiiv automations journeys aut_123 --all
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a single automation journey by ID",
				Example: strings.TrimSpace(`
beehiiv automation-journeys get aut_123 journey_123
beehiiv automation-journeys show aut_123 journey_123
beehiiv automations journey aut_123 journey_123
`),
			},
			"create": {
				Aliases: []string{"enroll"},
				Short:   "Enroll an existing subscriber into an automation",
				Example: strings.TrimSpace(`
beehiiv automation-journeys create aut_123 --body '{"email":"person@example.com"}'
beehiiv automation-journeys enroll aut_123 --body '{"subscription_id":"sub_123"}'
beehiiv automations enroll aut_123 --body @journey.json
`),
			},
		},
	}
}
