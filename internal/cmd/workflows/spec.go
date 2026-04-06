package workflows

var specLoaders = map[string]func() GroupSpec{
	"advertisement-opportunities": advertisementOpportunitiesSpec,
	"authors":                     authorsSpec,
	"automation-emails":           automationEmailsSpec,
	"automation-journeys":         automationJourneysSpec,
	"automations":                 automationsSpec,
	"bulk-subscription-updates":   bulkSubscriptionUpdatesSpec,
	"bulk-subscriptions":          bulkSubscriptionsSpec,
	"condition-sets":              conditionSetsSpec,
	"custom-fields":               customFieldsSpec,
	"email-blasts":                emailBlastsSpec,
	"engagements":                 engagementsSpec,
	"newsletter-lists":            newsletterListsSpec,
	"poll-responses":              pollResponsesSpec,
	"polls":                       pollsSpec,
	"post-templates":              postTemplatesSpec,
	"posts":                       postsSpec,
	"publications":                publicationsSpec,
	"referral-program":            referralProgramSpec,
	"segment-members":             segmentMembersSpec,
	"segment-results":             segmentResultsSpec,
	"segments":                    segmentsSpec,
	"subscription-bulk-actions":   subscriptionBulkActionsSpec,
	"subscription-tags":           subscriptionTagsSpec,
	"subscriptions":               subscriptionsSpec,
	"tiers":                       tiersSpec,
	"webhooks":                    webhooksSpec,
	"workspaces":                  workspacesSpec,
}

type ActionSpec struct {
	Aliases []string
	Short   string
	Long    string
	Example string
}

type GroupSpec struct {
	Aliases []string
	Short   string
	Long    string
	Example string
	Actions map[string]ActionSpec
}

func Lookup(group string) (GroupSpec, bool) {
	loader, ok := specLoaders[group]
	if !ok {
		return GroupSpec{}, false
	}
	return loader(), true
}

func ActionFor(group, action string) (ActionSpec, bool) {
	groupSpec, ok := Lookup(group)
	if !ok {
		return ActionSpec{}, false
	}
	actionSpec, ok := groupSpec.Actions[action]
	return actionSpec, ok
}
