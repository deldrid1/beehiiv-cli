package runtime

import (
	"context"
	"io"
	"os"

	"github.com/deldrid1/beehiiv-cli/internal/cli"
	"github.com/deldrid1/beehiiv-cli/internal/client"
)

type Executor struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	env        map[string]string
	httpClient client.HTTPClient
}

func NewExecutor(stdin io.Reader, stdout, stderr io.Writer, env map[string]string, httpClient client.HTTPClient) *Executor {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	return &Executor{
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		env:        env,
		httpClient: httpClient,
	}
}

func (e *Executor) Run(ctx context.Context, args []string) int {
	app := cli.NewApp(e.stdin, e.stdout, e.stderr, e.env, e.httpClient)
	return app.Run(ctx, args)
}
