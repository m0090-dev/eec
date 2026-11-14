package subcmd

import (
	"github.com/m0090-dev/eec/internal/core"
	"github.com/spf13/cobra"
)

var (
	treeKey     string
	treeFull    bool
	treeNoColor bool
)

// treeCmd represents the tree command
var treeCmd = &cobra.Command{
	Use:   "tree <tag>",
	Short: "Show the dependency tree of a tag, with values",
	Long: `Prints the full dependency tree of a tag: its config file and every
imported file or tag (recursively), with each environment variable's
value, [prepend]/[append]/[os: ...] badges where relevant, capture
commands and programs.

After the tree, variables defined more than once are listed under
"Overrides" with their definition chain in resolution order and the
winning final value, followed by a one-line summary.

With --key, only that variable is traced: every definition of it
across the tag's configs, in resolution order, plus its fully
resolved final value. Exits non-zero when the variable is not
defined by the tag.

Values longer than 60 characters are truncated unless --full is
given. Colors are used only when writing to a terminal (and are
also disabled by the NO_COLOR environment variable).

Examples:
  eec tree dev
  eec tree dev --full
  eec tree dev --key PATH`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeTagNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		e := core.NewEngine(nil)
		return e.TreeWithOptions(args[0], core.TreeOptions{
			Full:    treeFull,
			NoColor: treeNoColor,
			Key:     treeKey,
		})
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)

	treeCmd.Flags().StringVar(&treeKey, "key", "", "trace a single variable: show its definition chain and final value")
	treeCmd.Flags().BoolVar(&treeFull, "full", false, "do not truncate long values")
	treeCmd.Flags().BoolVar(&treeNoColor, "no-color", false, "disable colors")
	_ = treeCmd.RegisterFlagCompletionFunc("key", cobra.NoFileCompletions)
}
