package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/deldrid1/beehiiv-cli/internal/cmd"
)

func main() {
	referenceDir := flag.String("reference-dir", cmd.DefaultReferenceDocsDir, "Directory for generated markdown command reference docs")
	manDir := flag.String("man-dir", cmd.DefaultManpagesDir, "Directory for generated manpages")
	completionDir := flag.String("completion-dir", cmd.DefaultCompletionsDir, "Directory for generated shell completions")
	flag.Parse()

	if err := cmd.GenerateDocs(cmd.DocsOptions{
		ReferenceDir:  *referenceDir,
		ManDir:        *manDir,
		CompletionDir: *completionDir,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
