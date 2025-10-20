/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/m0090-dev/eec-go/core"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)
func info(){
		//fmt.Printf("version: %s\n",ext.VERSION)
		e := core.NewEngine(nil,nil)
		if err := e.Info();err!=nil{
			log.Fatal().Err(err).Msg("Failed to info")
		}
}

// ---------------------------
// Cobra Command Definition - info
// ---------------------------
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show detailed information about the eec environment and build",
	Long: `Displays detailed information about the eec installation, 
including version, build mode, commit hash, Go runtime, and platform details.

This command helps verify the running version of eec, 
the Go environment it was built with, and other diagnostic data.

Examples:
  eec info

Effect:
  • Prints eec version, build type, commit hash, and Go runtime info
  • Useful for debugging, support, or verifying installed builds`,
	Run: func(cmd *cobra.Command, args []string) {
		info()
	},
}


func init() {
	rootCmd.AddCommand(infoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// infoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// infoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
