/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package subcmd

import (
	"github.com/m0090-dev/eec-go/core"
	"github.com/m0090-dev/eec-go/core/interfaces/impl"
	"github.com/m0090-dev/eec-go/core/types"
	"github.com/spf13/cobra"
)

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		os := types.NewOS()
		log := impl.NewDefaultLogger()
		e := core.NewEngine(&os,log)
		if err := e.Restart();err != nil {
			log.Fatal().Err(err).Msg("Failed to restart")
		}
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// restartCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// restartCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
