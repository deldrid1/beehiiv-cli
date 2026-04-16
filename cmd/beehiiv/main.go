package main

import (
	"context"
	"os"
	"strings"

	"github.com/deldrid1/beehiiv-cli/internal/cmd"
)

func main() {
	os.Exit(cmd.ExecuteContext(context.Background(), os.Args[1:], cmd.Options{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    loadEnv(),
	}))
}

// loadEnv builds a map of BEEHIIV_* environment variables so the rest of the
// CLI can read them without calling os.Getenv directly (keeps options testable).
func loadEnv() map[string]string {
	env := make(map[string]string)
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok && strings.HasPrefix(key, "BEEHIIV_") {
			env[key] = value
		}
	}
	return env
}
