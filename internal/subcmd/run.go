/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package subcmd

import (
	"context"
	"github.com/m0090-dev/eec-go/core"
	"github.com/m0090-dev/eec-go/core/types"
	"github.com/spf13/cobra"
	"time"
)

// ---------------------------
// 位置引数やフラグ 格納
// ---------------------------
var configFileRunFlag string
var programRunFlag string
var programArgsRunFlag []string
var tagRunFlag string
var importsRunFlag []string
var waitTimeoutRunFlag int
var HideWindowRunFlag bool
var deleterPathRunFlag string
var DeleterHideWindowRunFlag bool
var SeparatorRunFlag string

func run() {
	e := core.NewEngine(nil, nil)
	opts := types.RunOptions{
		ConfigFile:        configFileRunFlag,
		Program:           programRunFlag,
		ProgramArgs:       programArgsRunFlag,
		Tag:               tagRunFlag,
		Imports:           importsRunFlag,
		WaitTimeout:       time.Duration(waitTimeoutRunFlag),
		HideWindow:        HideWindowRunFlag,
		DeleterPath:       deleterPathRunFlag,
		DeleterHideWindow: DeleterHideWindowRunFlag,
		Separator: SeparatorRunFlag,
	}
	if err := e.Run(context.Background(), opts); err != nil {
		e.Logger.Fatal().Err(err).Msg("Failed to run")
	}
}


// ---------------------------
// Cobra Command Definition
// ---------------------------
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a program in a temporary environment defined by a config or tag",
	Long: `Runs a program within an isolated environment loaded from a configuration file 
(TOML / YAML / JSON) or a registered tag. This allows you to execute programs safely 
without polluting the global system environment.

Examples:
  eec run -c test.toml -p bash
  eec run -t dev -p powershell --program-args="-NoExit","-Command","Write-Output 'hello world'"

Effect:
  • Loads environment variables from the specified config file or tag
  • Launches the target program within that temporary environment
  • Cleans up after execution without affecting the global system

This command is useful for testing, isolated development, and running tools 
in clean, reproducible environments.`,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	runCmd.Flags().StringVarP(&configFileRunFlag, "config-file","c", "", "Config file")

	runCmd.Flags().StringVarP(&programRunFlag, "program","p", "", "Program name")
	runCmd.Flags().StringSliceVarP(&programArgsRunFlag, "program-args","a", []string{}, "Program args")

	runCmd.Flags().StringVarP(&tagRunFlag, "tag","t", "", "Tag name")
	runCmd.Flags().StringSliceVarP(&importsRunFlag, "import","i", []string{}, "Import config files")
	runCmd.Flags().Int("wait-time-out", waitTimeoutRunFlag, "Time to wait before timeout in seconds")
	runCmd.Flags().BoolVarP(&HideWindowRunFlag, "hide-window", "", false, "Hide the console window when running")
	runCmd.Flags().StringVar(&deleterPathRunFlag, "deleter-path", "", "Deleter path")
	
	runCmd.Flags().BoolVarP(&DeleterHideWindowRunFlag, "deleter-hide-window", "", false, "hide the console window when runnning  deleter")
	
	runCmd.Flags().StringVarP(&SeparatorRunFlag,"separator","s","","Separator Value")


	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
