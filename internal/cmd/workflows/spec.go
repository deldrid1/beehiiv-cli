package workflows

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
	switch group {
	case "automations":
		return automationsSpec(), true
	case "automation-emails":
		return automationEmailsSpec(), true
	case "automation-journeys":
		return automationJourneysSpec(), true
	case "publications":
		return publicationsSpec(), true
	case "subscriptions":
		return subscriptionsSpec(), true
	case "posts":
		return postsSpec(), true
	case "webhooks":
		return webhooksSpec(), true
	default:
		return GroupSpec{}, false
	}
}

func ActionFor(group, action string) (ActionSpec, bool) {
	groupSpec, ok := Lookup(group)
	if !ok {
		return ActionSpec{}, false
	}
	actionSpec, ok := groupSpec.Actions[action]
	return actionSpec, ok
}
