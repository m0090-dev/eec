package subcmd

import (
	"errors"
	"fmt"

	"github.com/m0090-dev/eec/internal/core"
	"github.com/spf13/cobra"
)

var validateFlags resolutionFlags

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate config files and tags without running anything",
	Long: `Loads every config source selected by -c/-t/-i exactly the way
run would — import recursion, tag fallback and circular-import
detection included — and builds its environment, reporting one line
per source:

  OK <source>
  NG <source>: <error>

Parse errors, circular references and expansion failures are
reported, never swallowed. Exits with a non-zero status when any
source is NG.

Examples:
  eec validate -c test.toml
  eec validate -t dev
  eec validate -c base.toml -i extra.toml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := validateFlags.runOptions()
		if opts.ConfigFile == "" && opts.Tag == "" && len(opts.Imports) == 0 {
			return errors.New("no config source given: specify at least one of -c/--config-file, -t/--tag or -i/--import")
		}
		e := core.NewEngine(nil)
		results := e.Validate(opts)
		failed := 0
		for _, r := range results {
			if r.Err != nil {
				failed++
				fmt.Printf("NG %s: %v\n", r.Source, r.Err)
			} else {
				fmt.Printf("OK %s\n", r.Source)
			}
		}
		if failed > 0 {
			return fmt.Errorf("%d of %d config source(s) failed validation", failed, len(results))
		}
		return nil
	},
}

func init() {
	addResolutionFlags(validateCmd, &validateFlags)
	registerResolutionCompletion(validateCmd)
	rootCmd.AddCommand(validateCmd)
}
