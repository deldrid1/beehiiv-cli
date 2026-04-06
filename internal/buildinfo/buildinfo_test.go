package buildinfo

import "testing"

func TestSummaryUsesFallbackValues(t *testing.T) {
	t.Parallel()

	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
	})

	Version = ""
	Commit = ""
	Date = ""

	got := Summary("beehiiv")
	want := "beehiiv version dev (none, unknown)"
	if got != want {
		t.Fatalf("Summary fallback = %q, want %q", got, want)
	}
}

func TestSummaryIncludesConfiguredValues(t *testing.T) {
	t.Parallel()

	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
	})

	Version = "v1.2.3"
	Commit = "abc123"
	Date = "2026-04-03T12:00:00Z"

	got := Summary("beehiiv")
	want := "beehiiv version v1.2.3 (abc123, 2026-04-03T12:00:00Z)"
	if got != want {
		t.Fatalf("Summary configured = %q, want %q", got, want)
	}
}
