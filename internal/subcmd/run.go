/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package subcmd

import (
	"context"
	"errors"
	"github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/spf13/cobra"
	"os"
	"time"
)

// ---------------------------
// 位置引数やフラグ 格納
// ---------------------------
var configFileRunFlag string
var inlineConfigRunFlag string     // 追加: 自動判別
var inlineConfigTomlRunFlag string // 追加: TOML
var inlineConfigYamlRunFlag string // 追加: YAML
var inlineConfigJsonRunFlag string // 追加: JSON
var programRunFlag string
var programArgsRunFlag []string
var tagRunFlag string
var importsRunFlag []string
var waitTimeoutRunFlag int
var HideWindowRunFlag bool
var SeparatorRunFlag string
var verboseRunFlag bool
var duplicateStrictRunFlag bool
var duplicateNoWarnRunFlag bool

func run() {
	e := core.NewEngine(nil)
	opts := types.RunOptions{
		ConfigFile:       configFileRunFlag,
		InlineConfig:     inlineConfigRunFlag,
		InlineConfigToml: inlineConfigTomlRunFlag,
		InlineConfigYaml: inlineConfigYamlRunFlag,
		InlineConfigJson: inlineConfigJsonRunFlag,
		Program:          programRunFlag,
		ProgramArgs:      programArgsRunFlag,
		Tag:              tagRunFlag,
		Imports:          importsRunFlag,
		WaitTimeout:      time.Duration(waitTimeoutRunFlag) * time.Second,
		HideWindow:       HideWindowRunFlag,
		Separator:        SeparatorRunFlag,
		Verbose:          verboseRunFlag,
		DuplicateStrict:  duplicateStrictRunFlag,
		DuplicateNoWarn:  duplicateNoWarnRunFlag,
	}
	if err := e.Run(context.Background(), opts); err != nil {
		// If the child process exited with a non-zero code, exit with
		// that same code instead of collapsing it to 1 via Fatal.
		var exitErr *types.ProcessExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode)
		}
		e.Runtime.Logger().Error().Err(err).Msg("Failed to run")
		os.Exit(1)
	}
}

// ---------------------------
// Cobra Command Definition
// ---------------------------
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a program in a temporary environment defined by a config or tag",
	Long: `Runs a program within an isolated environment loaded from a configuration file
(TOML / YAML / JSON) or a registered tag.

Examples:
  eec run -c test.toml -p bash
  eec run -t dev -p powershell --program-args="-NoExit","-Command","Write-Output 'hello world'"
  eec run -c test.toml -p bash --verbose
  eec run -c test.toml -p bash --duplicate-strict`,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	runCmd.Flags().StringVarP(&configFileRunFlag, "config-file", "c", "", "Config file")
	runCmd.Flags().StringVarP(&inlineConfigRunFlag, "inline", "l", "", "Inline config string (auto-detect TOML/JSON/YAML)")
	runCmd.Flags().StringVar(&inlineConfigTomlRunFlag, "inline-toml", "", "Inline TOML config string")
	runCmd.Flags().StringVar(&inlineConfigYamlRunFlag, "inline-yaml", "", "Inline YAML config string")
	runCmd.Flags().StringVar(&inlineConfigJsonRunFlag, "inline-json", "", "Inline JSON config string")
	runCmd.Flags().StringVarP(&programRunFlag, "program", "p", "", "Program name")
	runCmd.Flags().StringSliceVarP(&programArgsRunFlag, "program-args", "a", []string{}, "Program args")
	runCmd.Flags().StringVarP(&tagRunFlag, "tag", "t", "", "Tag name")
	runCmd.Flags().StringSliceVarP(&importsRunFlag, "import", "i", []string{}, "Import config files")
	runCmd.Flags().IntVar(&waitTimeoutRunFlag, "wait-time-out", 0, "Time to wait before timeout in seconds")
	runCmd.Flags().BoolVarP(&HideWindowRunFlag, "hide-window", "", false, "Hide the console window when running")
	runCmd.Flags().StringVarP(&SeparatorRunFlag, "separator", "s", "", "Separator value")

	// #38: ログ制御
	runCmd.Flags().BoolVarP(&verboseRunFlag, "verbose", "v", false, "Enable verbose (debug) log output")

	// #20: 環境変数上書き制御
	runCmd.Flags().BoolVar(&duplicateStrictRunFlag, "duplicate-strict", false, "Treat env var overrides as errors")
	runCmd.Flags().BoolVar(&duplicateNoWarnRunFlag, "duplicate-no-warn", false, "Suppress env var override warnings")

	rootCmd.AddCommand(runCmd)
}
