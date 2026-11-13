package subcmd

import (
	"context"
	"os"

	"github.com/m0090-dev/eec/internal/core"
	"github.com/spf13/cobra"
)

var replFlags resolutionFlags

// replCmd represents the repl command
var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Interactive session in the managed environment, with reload",
	Long: `Starts an interactive session in the environment resolved from a
config file or tag. Each line is executed through the OS shell with
the session environment and working directory. Builtins:

  env [KEY]       print the session environment (or one variable)
  set KEY=VALUE   set a session variable (survives reload)
  unset KEY       remove a session variable (survives reload)
  cd [DIR]        change (or print) the session working directory
  reload          re-read the configs and rebuild the environment
  help            list the builtins
  exit / quit     leave the REPL

reload is the supported way to pick up config edits in a live
session: edit the config file, run reload, and keep working in the
same session with the new environment — no process restart needed.
Session set/unset overrides are re-applied on top of the reloaded
environment.

Examples:
  eec repl -t dev
  eec repl -c test.toml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		e := core.NewEngine(nil)
		interactive := false
		if st, err := os.Stdin.Stat(); err == nil && (st.Mode()&os.ModeCharDevice) != 0 {
			interactive = true
		}
		return e.RunREPL(context.Background(), replFlags.runOptions(), os.Stdin, os.Stdout, interactive)
	},
}

func init() {
	addResolutionFlags(replCmd, &replFlags)
	registerResolutionCompletion(replCmd)
	rootCmd.AddCommand(replCmd)
}
