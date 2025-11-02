/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"
	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "eec",
	Short: "Environment Execution Controller — run isolated environments safely",
	Long: `eec (env-exec) is a Go-based Environment Execution Controller.

It allows you to safely manage and execute environments defined in TOML, YAML,JSON or .env
configuration files without polluting your system environment.

With eec, you can:
  • Run programs within temporary, isolated environments
  • Group multiple configurations under tags for easy access
  • Generate utility scripts for quick launching
  • Use interactive (REPL) or restart modes for flexible workflows
  • Build CLI, GUI, and libraries via mage

Examples:
  eec run --config-file dev.toml --program cmd
  eec tag add dev --import base-dev.toml --import go-dev.toml
  eec run --tag dev
  eec gen script
  eec repl

eec is designed for developers who need clean, reproducible multi-language
development environments without touching the global system state.`,
}



// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.main.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


