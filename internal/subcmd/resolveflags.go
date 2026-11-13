package subcmd

import (
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/spf13/cobra"
)

// resolutionFlags bundles the flags every command that resolves an
// environment shares with `run`: where configs come from (-c/-t/-i
// and the inline variants), the list separator, and the expansion
// toggles. Program-related flags stay per-command.
type resolutionFlags struct {
	configFile string
	inline     string
	inlineToml string
	inlineYaml string
	inlineJson string
	tag        string
	imports    []string
	separator  string
	noExpand   bool
	noExec     bool
}

// addResolutionFlags registers the shared resolution flags on cmd.
func addResolutionFlags(cmd *cobra.Command, f *resolutionFlags) {
	cmd.Flags().StringVarP(&f.configFile, "config-file", "c", "", "Config file")
	cmd.Flags().StringVarP(&f.inline, "inline", "l", "", "Inline config string (auto-detect TOML/JSON/YAML)")
	cmd.Flags().StringVar(&f.inlineToml, "inline-toml", "", "Inline TOML config string")
	cmd.Flags().StringVar(&f.inlineYaml, "inline-yaml", "", "Inline YAML config string")
	cmd.Flags().StringVar(&f.inlineJson, "inline-json", "", "Inline JSON config string")
	cmd.Flags().StringVarP(&f.tag, "tag", "t", "", "Tag name")
	cmd.Flags().StringSliceVarP(&f.imports, "import", "i", []string{}, "Import config files")
	cmd.Flags().StringVarP(&f.separator, "separator", "s", "", "Separator value")
	cmd.Flags().BoolVar(&f.noExpand, "no-expand", false, "Disable ${} variable expansion")
	cmd.Flags().BoolVar(&f.noExec, "no-exec", false, "Disable $() command substitution (left literal)")
}

// runOptions converts the flags into the shared options structure.
func (f *resolutionFlags) runOptions() types.RunOptions {
	return types.RunOptions{
		ConfigFile:       f.configFile,
		InlineConfig:     f.inline,
		InlineConfigToml: f.inlineToml,
		InlineConfigYaml: f.inlineYaml,
		InlineConfigJson: f.inlineJson,
		Tag:              f.tag,
		Imports:          f.imports,
		Separator:        f.separator,
		NoExpand:         f.noExpand,
		NoExec:           f.noExec,
	}
}
