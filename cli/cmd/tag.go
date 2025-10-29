/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/

package cmd

import (
	"github.com/m0090-dev/eec-go/core"
	"github.com/m0090-dev/eec-go/core/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// ---------------------------
// 位置引数やフラグ 格納
// ---------------------------
var (
	//tagNameTagFlag     string
	configFileTagFlag  string
	programTagFlag     string
	programArgsTagFlag []string
	importConfigFilesTagFlag []string
)

func add(name string){
	e := core.NewEngine(nil,nil)
	data := types.TagData{
		ConfigFile: configFileTagFlag,
		Program: programTagFlag,
		ProgramArgs: programArgsTagFlag,
		ImportConfigFiles: importConfigFilesTagFlag,
	}
	if err:=e.TagAdd(name,data);err!=nil{
		log.Fatal().Err(err).Msg("Failed to tag add")
	}
}
func read(name string){
	e := core.NewEngine(nil,nil)
	if err:=e.TagRead(name);err != nil{
		log.Fatal().Err(err).Msg("Failed to tag read")
	}
}
func list(){
	e := core.NewEngine(nil,nil)
	if err:=e.TagList();err!=nil{
		log.Fatal().Err(err).Msg("Failed to tag list")
	}
}
func remove(name string){
	e := core.NewEngine(nil,nil)
	if err:=e.TagRemove(name);err!=nil{
		log.Fatal().Err(err).Msg("Failed tp tag remove")
	}
}




// ---------------------------
// Cobra コマンド定義
// ---------------------------
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
}

var addTagCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new tag",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		add(args[0])	
	},
}

var readTagCmd = &cobra.Command{
	Use:   "read [name]",
	Short: "Read a tag",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		read(args[0])	
	},
}
var listTagCmd = &cobra.Command{
	Use:   "list",
	Short: "List a tags",
	Run: func(cmd *cobra.Command, args []string) {
		list()	
	},
}

var removeTagCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a tag",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		remove(args[0])
	},
}

func init() {
	//addTagCmd.Flags().StringVar(&tagNameTagFlag, "name", "", "Tag name")
	addTagCmd.Flags().StringVar(&configFileTagFlag, "config-file", "", "Config file")
	addTagCmd.Flags().StringVar(&programTagFlag, "program", "", "Program name")
	addTagCmd.Flags().StringSliceVar(&programArgsTagFlag, "program-args", []string{}, "Program args")
	addTagCmd.Flags().StringSliceVar(&importConfigFilesTagFlag, "import", []string{}, "Import config files")

	tagCmd.AddCommand(addTagCmd)
	tagCmd.AddCommand(readTagCmd)
	tagCmd.AddCommand(listTagCmd)
	tagCmd.AddCommand(removeTagCmd)
	rootCmd.AddCommand(tagCmd)
}
