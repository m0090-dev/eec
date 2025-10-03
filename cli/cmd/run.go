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

//import "github.com/m0090-dev/eec-go/core"

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
var ptyRunFlag bool

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
		Pty: ptyRunFlag,
	}
	if err := e.Run(context.Background(), opts); err != nil {
		log.Fatal().Err(err).Msg("Failed to run")
	}
}

/*func run() {*/
/*log.Debug().Str("configFileRunFlag", configFileRunFlag).Msg("")*/
/*log.Debug().Str("programRunFlag", programRunFlag).Msg("")*/
/*log.Debug().Str("programArgsRunFlag", strings.Join(programArgsRunFlag, ", ")).Msg("")*/
/*// -----------------------*/
/*// 開始時環境変数表示*/
/*// -----------------------*/
/*{*/
/*envs := os.Environ()*/
/*envStr := strings.Join(envs, ", ")*/
/*log.Debug().Str("Started envs", envStr).Msg("")*/
/*}*/

/*selfProgram := os.Args[0]*/
/*tempData := ext.TempData{}*/

/*var tagData ext.TagData*/
/*var configFile, program string*/
/*var pArgs []string*/
/*var config ext.Config*/
/*var err error*/

/*// -----------------------*/
/*// タグ優先で読み込み*/
/*// -----------------------*/
/*if tagRunFlag != "" {*/
/*tagData, err = ext.ReadTagData(tagRunFlag)*/
/*if err != nil {*/
/*log.Error().Err(err).Str("tagRunFlag", tagRunFlag).*/
/*Msg("タグデータの読み込みに失敗しました")*/
/*os.Exit(1)*/
/*}*/
/*configFile = tagData.ConfigFile*/
/*program = tagData.Program*/
/*pArgs = tagData.ProgramArgs*/
/*} else {*/
/*configFile = configFileRunFlag*/
/*program = programRunFlag*/
/*pArgs = programArgsRunFlag*/
/*}*/

/*// 引数で上書き*/
/*if configFileRunFlag != "" {*/
/*configFile = configFileRunFlag*/
/*}*/
/*if programRunFlag != "" {*/
/*program = programRunFlag*/
/*}*/
/*if len(programArgsRunFlag) != 0 {*/
/*pArgs = programArgsRunFlag*/
/*}*/

/*// 設定ファイル読み込み*/
/*if configFile != "" && general.FileExists(configFile) {*/
/*config, err = ext.ReadConfig(configFile)*/
/*if err != nil {*/
/*log.Error().Err(err).Str("configFile", configFile).*/
/*Msg("tomlファイルの読み込みに失敗しました")*/
/*}*/
/*}*/

/*// program 補完*/
/*if (tagRunFlag == "" || tagData.Program == "") && config.Program.Path != "" {*/
/*program = config.Program.Path*/
/*}*/
/*if (tagRunFlag == "" || len(tagData.ProgramArgs) == 0) && len(config.Program.Args) != 0 {*/
/*pArgs = config.Program.Args*/
/*}*/

/*if programRunFlag != "" {*/
/*program = programRunFlag*/
/*}*/
/*if len(programArgsRunFlag) != 0 {*/
/*pArgs = programArgsRunFlag*/
/*}*/

/*// -----------------------*/
/*// 一時ファイル & マニフェスト*/
/*// -----------------------*/
/*tmpDir := os.TempDir()*/
/*tmpPrefix := fmt.Sprintf(*/
/*"%s_%s_%s.tmp",*/
/*general.RemoveExtension(filepath.Base(selfProgram)),*/
/*general.RemoveExtension(filepath.Base(program)),*/
/*uuid.New().String(),*/
/*)*/
/*tmpPath := filepath.Join(tmpDir, tmpPrefix)*/
/*tmpFile, err := os.Create(tmpPath)*/
/*if err != nil {*/
/*log.Error().Err(err).Str("prefix", tmpPrefix).Msg("一時ファイルの作成に失敗しました")*/
/*os.Exit(1)*/
/*}*/
/*log.Info().Str("Temp file", tmpPath).Msg("Created temp file")*/

/*manifest := ext.Manifest{*/
/*TempFilePath: tmpFile.Name(),*/
/*EECPID:       os.Getpid(),*/
/*}*/
/*manifestPath, err := manifest.WriteToManifest()*/
/*if err != nil {*/
/*log.Error().Err(err).Str("manifestPath", manifestPath).Msg("マニフェストファイルの作成に失敗しました")*/
/*os.Exit(1)*/
/*}*/
/*log.Info().Str("Manifest file", manifestPath).Msg("Created manifest file")*/

/*// -----------------------*/
/*// 環境変数マージ*/
/*// -----------------------*/
/*allConfigs := []ext.Config{}*/

/*// importsRunFlag の順に追加*/
/*for _, importEnvFile := range importsRunFlag {*/
/*cfg, err := domain.ReadOrFallback(importEnvFile)*/
/*if err == nil {*/
/*allConfigs = append(allConfigs, cfg)*/
/*}*/
/*}*/

/*// タグの ImportConfigFiles を順に追加*/
/*for _, importEnvFile := range tagData.ImportConfigFiles {*/
/*cfg, err := domain.ReadOrFallback(importEnvFile)*/
/*if err == nil {*/
/*allConfigs = append(allConfigs, cfg)*/
/*}*/
/*}*/

/*// メイン config を最後に追加*/
/*allConfigs = append(allConfigs, config)*/

/*// baseEnv に os.Environ() をセット*/
/*finalEnv := os.Environ()*/
/*for _, cfg := range allConfigs {*/
/*finalEnv = cfg.BuildEnvs(finalEnv)*/
/*}*/

/*for _, e := range finalEnv {*/
/*if strings.HasPrefix(strings.ToUpper(e), "PATH=") {*/
/*log.Debug().Str("Final PATH", e).Msg("Checking PATH")*/
/*}*/
/*}*/

/*log.Debug().Str("LookPath PATH", os.Getenv("PATH")).Msg("")*/

/*// -----------------------*/
/*// exec.LookPath で確実に実行可能ファイルを解決*/
/*// -----------------------*/
/*//progPath, err := exec.LookPath(program)*/
/*//if err != nil {*/
/*//log.Error().Err(err).Str("program", program).Msg("実行ファイルが見つかりません")*/
/*//os.Exit(1)*/
/*//}*/

/*//executeCommand := exec.Command(progPath, pArgs...)*/
/*executeCommand := exec.Command("cmd.exe", "/C", program+" "+strings.Join(pArgs, " "))*/

/*executeCommand.Env = finalEnv*/
/*executeCommand.Stdin = os.Stdin*/
/*executeCommand.Stdout = os.Stdout*/
/*executeCommand.Stderr = os.Stderr*/

/*if err := executeCommand.Start(); err != nil {*/
/*log.Error().Err(err).Msg("プログラムの起動に失敗しました")*/
/*os.Exit(1)*/
/*}*/
/*childPID := executeCommand.Process.Pid*/
/*log.Info().Int("PID", childPID).Msg("Sub process started ppid")*/

/*// -----------------------*/
/*// 一時データ保存*/
/*// -----------------------*/
/*tempData.ParentPID = os.Getpid()*/
/*tempData.ChildPID = childPID*/
/*tempData.ConfigFile = configFile*/
/*tempData.Program = program*/
/*tempData.ProgramArgs = pArgs*/

/*var tempDataBin bytes.Buffer*/
/*encoder := gob.NewEncoder(&tempDataBin)*/
/*if err := encoder.Encode(tempData); err != nil {*/
/*log.Error().Err(err).Msg("一時ファイル使用データのエンコードに失敗しました")*/
/*os.Exit(1)*/
/*}*/
/*if _, err := tmpFile.Write(tempDataBin.Bytes()); err != nil {*/
/*log.Error().Err(err).Msg("一時ファイルの書き込みに失敗しました")*/
/*os.Exit(1)*/
/*}*/
/*log.Info().*/
/*Int("ParentPID", tempData.ParentPID).*/
/*Int("ChildPID", tempData.ChildPID).*/
/*Str("ConfigFile", tempData.ConfigFile).*/
/*Str("Program", tempData.Program).*/
/*Str("Program Args", strings.Join(tempData.ProgramArgs, ", ")).*/
/*Msg("Temp file written successfully")*/

/*// -----------------------*/
/*// プロセス待機*/
/*// -----------------------*/
/*if err := executeCommand.Wait(); err != nil {*/
/*log.Error().Err(err).Msg("プログラム終了時にエラーが発生しました")*/
/*os.Exit(1)*/
/*}*/

/*// -----------------------*/
/*// 終了時環境変数表示*/
/*// -----------------------*/
/*{*/
/*envs := os.Environ()*/
/*envStr := strings.Join(envs, ", ")*/
/*log.Debug().Str("Finished envs", envStr).Msg("")*/
/*}*/

/*}*/

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

	runCmd.Flags().BoolVarP(
		&ptyRunFlag,
		"pty",
		"",
		false,
		"Run the command inside a pseudo terminal (PTY). On Unix/macOS it uses native PTY. On Windows, this requires ConPTY (Windows 10 or later).",
	)
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
