package subcmd

import (
	"context"
	"errors"
	"os"

	"github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/spf13/cobra"
)

var shellFlags resolutionFlags

// shellCmd represents the shell command
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start an interactive shell inside the managed environment",
	Long: `Resolves the environment from a config file or tag and starts an
interactive shell as a child process with that environment:
%ComSpec% (cmd) on Windows, $SHELL (or sh) on Unix.

IMPORTANT: this starts a NEW shell process with the resolved
environment. It cannot modify the shell that launched eec — no
command can change its parent process's environment. When you exit
the child shell, your original shell is exactly as it was before.

Use 'eec repl' if you want to run commands one by one in the
environment with the ability to reload edited configs in place.

Examples:
  eec shell -t dev
  eec shell -c test.toml`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		e := core.NewEngine(nil)
		if err := e.Shell(context.Background(), shellFlags.runOptions()); err != nil {
			var exitErr *types.ProcessExitError
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.ExitCode)
			}
			e.Runtime.Logger().Error().Err(err).Msg("Failed to run shell")
			os.Exit(1)
		}
	},
}

func init() {
	addResolutionFlags(shellCmd, &shellFlags)
	registerResolutionCompletion(shellCmd)
	rootCmd.AddCommand(shellCmd)
}
