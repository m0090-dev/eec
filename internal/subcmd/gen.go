/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package subcmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/m0090-dev/eec-go/core"
)

func genScript(){
		//domain.GenUtilsScript()
		//domain.GenWrapScript()
		e := core.NewEngine(nil,nil)
		if err := e.GenScript();err != nil {
			e.Logger.Fatal().Err(err).Msg("Failed to gen script")
		}
}

// ---------------------------
// Cobra Command Definition - gen
// ---------------------------
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate utility scripts or other helper assets for eec",
	Long: `The 'gen' command provides utilities to generate scripts or helper assets
that make eec usage more convenient. 

Currently, it supports generating quick-launch scripts for registered tags
so that you can easily start environments without typing long commands.

Examples:
  eec gen script
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use a subcommand such as 'eec gen script'")
	},
}

// ---------------------------
// Cobra Command Definition - script
// ---------------------------
var scriptCmd = &cobra.Command{
	Use:   "script",
	Short: "Generate quick-launch scripts for registered tags",
	Long: `Generates utility scripts that make it easy to launch programs 
in specific environments using registered tags.

On Windows, it creates .bat files (e.g., tdev.bat).
On Linux or macOS, it creates shell scripts (e.g., tdev).

For example, if a tag named 'dev' exists:
  tdev cmd     → runs 'cmd' within the 'dev' environment

Examples:
  eec gen script
Effect:
  • Scans registered tags
  • Creates platform-appropriate launcher scripts
  • Simplifies frequent environment switching`,
	Run: func(cmd *cobra.Command, args []string) {
		genScript()
	},
}



func init() {
	rootCmd.AddCommand(genCmd)
	genCmd.AddCommand(scriptCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// genCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// genCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
