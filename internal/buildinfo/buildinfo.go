package buildinfo

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func versionValue() string {
	if Version == "" {
		return "dev"
	}
	return Version
}

func commitValue() string {
	if Commit == "" {
		return "none"
	}
	return Commit
}

func dateValue() string {
	if Date == "" {
		return "unknown"
	}
	return Date
}

func Summary(name string) string {
	return fmt.Sprintf("%s version %s (%s, %s)", name, versionValue(), commitValue(), dateValue())
}
