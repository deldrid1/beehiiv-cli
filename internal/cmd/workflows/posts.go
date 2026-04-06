package workflows

import "strings"

func postsSpec() GroupSpec {
	return GroupSpec{
		Aliases: []string{"post"},
		Short:   "Create and inspect Beehiiv posts",
		Long: "Work with publication posts, including listing, reading, creating, deleting, and " +
			"retrieving aggregate post statistics.",
		Example: strings.TrimSpace(`
beehiiv posts list --query limit=25
beehiiv posts show post_123
beehiiv posts stats
beehiiv posts create --body @post.json
`),
		Actions: map[string]ActionSpec{
			"list": {
				Short: "List posts for the active publication",
				Example: strings.TrimSpace(`
beehiiv posts list --query limit=25
beehiiv post list --output table
`),
			},
			"create": {
				Aliases: []string{"add"},
				Short:   "Create a post",
				Example: strings.TrimSpace(`
beehiiv posts create --body @post.json
beehiiv posts add --body '{"title":"Launch update"}'
`),
			},
			"get": {
				Aliases: []string{"show"},
				Short:   "Show a post by ID",
				Example: strings.TrimSpace(`
beehiiv posts get post_123
beehiiv posts show post_123
`),
			},
			"aggregate-stats": {
				Aliases: []string{"stats"},
				Short:   "Show aggregate statistics for posts",
				Example: strings.TrimSpace(`
beehiiv posts aggregate-stats
beehiiv posts stats
`),
			},
			"delete": {
				Aliases: []string{"remove"},
				Short:   "Delete a post by ID",
				Example: strings.TrimSpace(`
beehiiv posts delete post_123
beehiiv posts remove post_123
`),
			},
		},
	}
}
