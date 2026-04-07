package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHelpListsCoreCommands(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"--help"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "auth") {
		t.Fatalf("help output missing auth command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "reports") {
		t.Fatalf("help output missing reports command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "publications") {
		t.Fatalf("help output missing publications group: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "version") {
		t.Fatalf("help output missing version command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--output") {
		t.Fatalf("help output missing global output flag: %s", stdout.String())
	}
}

func TestVersionCommandPrintsBuildSummary(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"version"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "beehiiv version") {
		t.Fatalf("version output missing summary: %s", stdout.String())
	}
}

func TestAuthStatusShowsConfiguredSessionWithoutSecrets(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"auth", "status"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
		Env: map[string]string{
			"BEEHIIV_API_KEY":        "test-token",
			"BEEHIIV_PUBLICATION_ID": "pub_test",
		},
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if strings.Contains(stdout.String(), "test-token") {
		t.Fatalf("stdout unexpectedly leaked token: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"configured": true`) {
		t.Fatalf("stdout missing configured state: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"publication_id": "pub_test"`) {
		t.Fatalf("stdout missing publication_id: %s", stdout.String())
	}
}

func TestGeneratedActionHelpUsesCobraCommandHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"publications", "list", "--help"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Retrieve all publications") && !strings.Contains(stdout.String(), "List publications") {
		t.Fatalf("stdout missing action help summary: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "API path: /publications") {
		t.Fatalf("stdout missing generated API path help: %s", stdout.String())
	}
}

func TestGlobalFlagsPropagateThroughCobra(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"--output", "table", "auth", "status"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
		Env: map[string]string{
			"BEEHIIV_API_KEY":        "test-token",
			"BEEHIIV_PUBLICATION_ID": "pub_test",
		},
	})
	if exitCode != 0 {
		t.Fatalf("ExecuteContext exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "| field") {
		t.Fatalf("stdout missing table header: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "publication_id") {
		t.Fatalf("stdout missing publication_id row: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "test-token") {
		t.Fatalf("stdout unexpectedly leaked token: %s", stdout.String())
	}
}

func TestLegacyAuthCurrentAliasIsRemoved(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteContext(context.Background(), []string{"auth", "current"}, Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if exitCode == 0 {
		t.Fatalf("ExecuteContext exit code = %d, want non-zero", exitCode)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr missing unknown command error: %s", stderr.String())
	}
}
