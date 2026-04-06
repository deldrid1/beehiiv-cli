package cli

import (
	"github.com/deldrid1/beehiiv-cli/internal/config"
	clioutput "github.com/deldrid1/beehiiv-cli/internal/output"
)

func (a *App) writeOutput(value any, rawBody []byte, runtime config.Runtime) {
	if err := clioutput.Write(a.stdout, value, rawBody, runtime); err != nil {
		a.writeError(err)
	}
}
