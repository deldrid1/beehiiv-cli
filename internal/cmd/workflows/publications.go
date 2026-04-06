package workflows

import "strings"

func publicationsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"pubs", "publication"},
		Short:   "Work with Beehiiv publications",
		Long: "List the publications available to your current Beehiiv session and inspect a " +
			"single publication when you already know the publication ID.",
		Example: strings.TrimSpace(`
beehiiv publications list
beehiiv publications get pub_123
beehiiv publications show pub_123
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List publications available to the current account",
				Example: strings.TrimSpace(`
beehiiv publications list
beehiiv pubs list --output table
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a publication by ID",
				Example: strings.TrimSpace(`
beehiiv publications get pub_123
beehiiv publications show pub_123
`),
			},
		},
	}
}
