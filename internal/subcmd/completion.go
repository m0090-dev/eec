package subcmd

import (
	"path/filepath"
	"strings"

	"github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
	"github.com/spf13/cobra"
)

// Shell completion: cobra's built-in `completion` command (bash,
// zsh, fish, powershell) is enabled by default in the root command;
// the functions below teach it about eec's dynamic values.

// completeTagNames completes tag names from the tag store (the
// resolved data directory). It is used for -t/--tag flag values and
// for positional tag-name arguments (tree).
func completeTagNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	e := core.NewEngine(nil)
	dir := types.ResolveDataDir(e.Runtime)
	if dir == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	files, err := general.GetFilesWithExtension(dir, ".tag")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".tag")
		if strings.HasPrefix(name, toComplete) {
			names = append(names, name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeConfigFile leaves -c/--config-file to the shell's default
// file completion.
func completeConfigFile(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

// registerResolutionCompletion wires tag/config completion into a
// command that carries the shared resolution flags. It must be
// called after the flags are registered.
func registerResolutionCompletion(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("tag", completeTagNames)
	_ = cmd.RegisterFlagCompletionFunc("config-file", completeConfigFile)
}
