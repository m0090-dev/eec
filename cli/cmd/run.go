/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"github.com/m0090-dev/eec-go/core"
	"github.com/m0090-dev/eec-go/core/types"
	"github.com/rs/zerolog/log"
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
var hideWindowRunFlag bool
var deleterPathRunFlag string
var deleterHideWindowRunFlag bool

func run() {
	e := core.NewEngine(nil, nil)
	opts := types.RunOptions{
		ConfigFile:        configFileRunFlag,
		Program:           programRunFlag,
		ProgramArgs:       programArgsRunFlag,
		Tag:               tagRunFlag,
		Imports:           importsRunFlag,
		WaitTimeout:       time.Duration(waitTimeoutRunFlag),
		HideWindow:        hideWindowRunFlag,
		DeleterPath:       deleterPathRunFlag,
		DeleterHideWindow: deleterHideWindowRunFlag,
	}
	if err := e.Run(context.Background(), opts); err != nil {
		log.Fatal().Err(err).Msg("Failed to run")
	}
}

// ---------------------------
// Cobra コマンド定義
// ---------------------------
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	runCmd.Flags().StringVar(&configFileRunFlag, "config-file", "", "Config file")

	runCmd.Flags().StringVar(&programRunFlag, "program", "", "Program name")
	runCmd.Flags().StringSliceVar(&programArgsRunFlag, "program-args", []string{}, "Program args")

	runCmd.Flags().StringVar(&tagRunFlag, "tag", "", "Tag name")
	runCmd.Flags().StringSliceVar(&importsRunFlag, "import", []string{}, "Import config files")
	runCmd.Flags().Int("wait-time-out", waitTimeoutRunFlag, "Time to wait before timeout in seconds")
	runCmd.Flags().BoolVarP(&hideWindowRunFlag, "hide-window", "", false, "Hide the console window when running")
	runCmd.Flags().StringVar(&deleterPathRunFlag, "deleter-path", "", "Deleter path")
	
	runCmd.Flags().BoolVarP(&deleterHideWindowRunFlag, "deleter-hide-window", "", false, "hide the console window when runnning  deleter")



	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
