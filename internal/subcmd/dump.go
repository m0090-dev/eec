package subcmd

import (
	"github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/spf13/cobra"
	"time"
)

var configFileDumpFlag string
var inlineConfigDumpFlag string
var inlineConfigTomlDumpFlag string
var inlineConfigYamlDumpFlag string
var inlineConfigJsonDumpFlag string
var tagDumpFlag string
var importsDumpFlag []string
var separatorDumpFlag string
var shellDumpFlag string
var noExpandDumpFlag bool
var noExecDumpFlag bool

func dump() {
	e := core.NewEngine(nil)
	opts := types.RunOptions{
		ConfigFile:       configFileDumpFlag,
		InlineConfig:     inlineConfigDumpFlag,
		InlineConfigToml: inlineConfigTomlDumpFlag,
		InlineConfigYaml: inlineConfigYamlDumpFlag,
		InlineConfigJson: inlineConfigJsonDumpFlag,
		Tag:              tagDumpFlag,
		Imports:          importsDumpFlag,
		Separator:        separatorDumpFlag,
		WaitTimeout:      time.Duration(0),
		NoExpand:         noExpandDumpFlag,
		NoExec:           noExecDumpFlag,
	}
	if err := e.Dump(opts, shellDumpFlag); err != nil {
		e.Runtime.Logger().Fatal().Err(err).Msg("Failed to dump")
	}
}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump resolved environment variables from a config or tag",
	Long: `Resolves and prints all environment variables from a config file or tag
after full expansion (variable substitution, command execution, etc.).

Output formats (--shell flag):
  default  KEY=VALUE  (compatible with .env)
  unix     export KEY=VALUE
  win      set KEY=VALUE

Examples:
  eec dump -c test.toml
  eec dump -t dev --shell unix
  eec dump -t dev --shell win`,
	Run: func(cmd *cobra.Command, args []string) {
		dump()
	},
}

func init() {
	dumpCmd.Flags().StringVarP(&configFileDumpFlag, "config-file", "c", "", "Config file")
	dumpCmd.Flags().StringVarP(&inlineConfigDumpFlag, "inline", "l", "", "Inline config string (auto-detect TOML/JSON/YAML)")
	dumpCmd.Flags().StringVar(&inlineConfigTomlDumpFlag, "inline-toml", "", "Inline TOML config string")
	dumpCmd.Flags().StringVar(&inlineConfigYamlDumpFlag, "inline-yaml", "", "Inline YAML config string")
	dumpCmd.Flags().StringVar(&inlineConfigJsonDumpFlag, "inline-json", "", "Inline JSON config string")
	dumpCmd.Flags().StringVarP(&tagDumpFlag, "tag", "t", "", "Tag name")
	dumpCmd.Flags().StringSliceVarP(&importsDumpFlag, "import", "i", []string{}, "Import config files")
	dumpCmd.Flags().StringVarP(&separatorDumpFlag, "separator", "s", "", "Separator value")
	dumpCmd.Flags().StringVar(&shellDumpFlag, "shell", "", "Output format: unix (export KEY=VALUE) or win (set KEY=VALUE)")
	dumpCmd.Flags().BoolVar(&noExpandDumpFlag, "no-expand", false, "Disable ${} variable expansion")
	dumpCmd.Flags().BoolVar(&noExecDumpFlag, "no-exec", false, "Disable $() command substitution (left literal)")

	rootCmd.AddCommand(dumpCmd)
}
