/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/

package cmd

import (
	"github.com/m0090-dev/eec-go/core"
	"github.com/m0090-dev/eec-go/core/ext"
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
	data := ext.TagData{
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







/*func add(args[] string){*/
	/*tagName := args[0]*/
		/*// -- デバッグ用 --*/
		/*log.Debug().*/
			/*Str("tagName", tagName).*/
			/*Msg("")*/
		/*log.Debug().*/
			/*Str("configFileFlag", configFileTagFlag).*/
			/*Msg("")*/
		/*log.Debug().*/
			/*Str("programFlag", programTagFlag).*/
			/*Msg("")*/
		/*log.Debug().*/
			/*Str("programArgsFlag", strings.Join(programArgsTagFlag, ", ")).*/
			/*Msg("")*/
		/*log.Debug().*/
			/*Str("Import config files", strings.Join(importConfigFilesTagFlag, ", ")).*/
			/*Msg("")	*/
		/*//*/

		/*data := ext.TagData{*/
			/*ConfigFile:  configFileTagFlag,*/
			/*Program:     programTagFlag,*/
			/*ProgramArgs: programArgsTagFlag,*/
			/*ImportConfigFiles: importConfigFilesTagFlag,*/
		/*}*/
		/*if err := data.Write(tagName); err != nil {*/
			/*log.Error().Err(err).Msg("タグファイルの書き込みに失敗しました")*/
			/*os.Exit(1)*/
		/*}*/
		/*fmt.Println("Tag added:", tagName)*/

/*}*/


/*func read(args[] string){*/
	/*tagName := args[0]*/
		/*data, err := ext.ReadTagData(tagName)*/
		/*if err != nil {*/
/*log.Error().Err(err).Msg("タグファイルの読み込みに失敗しました")*/
			/*os.Exit(1)*/
		/*}*/
		/*fmt.Printf("Tag: %s\n  Config: %s\n  Program: %s\n  Args: %v\n  Import config files: %v\n",*/
			/*tagName, data.ConfigFile, data.Program, data.ProgramArgs,data.ImportConfigFiles)*/

/*}*/


/*func list(args[] string){*/
	/*homeDir, err := os.UserHomeDir()*/
		/*if homeDir == "" {*/
			/*log.Error().Err(err).Msg(fmt.Sprintf("homeDir(%s)が設定されていません", homeDir))*/
			/*os.Exit(1)*/
		/*}*/
		/*tagDir := filepath.Join(homeDir, ext.DEFAULT_TAG_DIR)*/
		/*fileLists, err := general.GetFilesWithExtension(tagDir, ".tag")*/
		/*if err != nil {*/
			/*log.Error().Err(err).Msg("タグファイルが見つかりませんでした")*/
			/*os.Exit(1)*/
		/*}*/
		/*fmt.Printf("-- current tag lists  --\n")*/
		/*fmt.Printf("%s\n", strings.Join(fileLists, "\n"))*/
/*}*/

/*func remove(args[] string){*/
	/*tagName := args[0]*/
		/*homeDir, err := os.UserHomeDir()*/
		/*if homeDir == "" {*/
			/*log.Error().Err(err).Msg(fmt.Sprintf("homeDir(%s)が設定されていません", homeDir))*/
			/*os.Exit(1)*/
		/*}*/
		/*tagDir := filepath.Join(homeDir, ext.DEFAULT_TAG_DIR)*/
		/*tagPath := filepath.Join(tagDir, fmt.Sprintf("%s.tag", tagName))*/
		/*err = os.Remove(tagPath)*/
		/*if err != nil {*/
			/*log.Error().Err(err).Msg(fmt.Sprintf("タグ名: %sの削除に失敗しました", tagName))*/
			/*os.Exit(1)*/
		/*}*/
		/*fmt.Printf("タグ名: %sを削除しました\n", tagName)*/
	
		/*fileLists, err := general.GetFilesWithExtension(tagDir, ".tag")*/
		/*if err != nil {*/
			/*log.Error().Err(err).Msg("タグファイルが見つかりませんでした")*/
			/*os.Exit(1)*/
		/*}*/
		/*fmt.Printf("-- current tag lists  --\n")*/
		/*fmt.Printf("%s\n", strings.Join(fileLists, "\n"))*/

/*}*/



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
