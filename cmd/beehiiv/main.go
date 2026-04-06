package main

import (
	"context"
	"os"

	"github.com/deldrid1/beehiiv-cli/internal/cmd"
)

func main() {
	os.Exit(cmd.ExecuteContext(context.Background(), os.Args[1:], cmd.Options{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}))
}
