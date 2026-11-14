package subcmd

import (
	"fmt"

	"github.com/m0090-dev/eec/internal/core"
	"github.com/spf13/cobra"
)

var whichFlags resolutionFlags

// whichCmd represents the which command
var whichCmd = &cobra.Command{
	Use:   "which <name>",
	Short: "Print where a program resolves in the managed environment",
	Long: `Resolves the environment from a config file or tag (like run and
dump do) and prints the absolute path of the named program as found
in that environment's PATH.

On Windows the PATHEXT extensions (.COM;.EXE;.BAT;.CMD, or the
PATHEXT variable of the resolved environment) are tried for bare
names. On Unix a file must carry the executable bit. A name
containing a path separator is checked directly instead of being
searched. The current directory is never searched implicitly.

Exits with status 1 when the program is not found.

Examples:
  eec which go
  eec which -t dev ninja
  eec which -c test.toml ./tool`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		e := core.NewEngine(nil)
		found, err := e.Which(whichFlags.runOptions(), args[0])
		if err != nil {
			return err
		}
		fmt.Println(found)
		return nil
	},
}

func init() {
	addResolutionFlags(whichCmd, &whichFlags)
	registerResolutionCompletion(whichCmd)
	rootCmd.AddCommand(whichCmd)
}
