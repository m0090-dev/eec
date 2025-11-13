package domain

import "github.com/m0090-dev/eec/internal/ext/types"
import "github.com/m0090-dev/eec/internal/ext/interfaces"
import "path/filepath"
/*
// ResolveRunOptionsは、RunOptions、TagData、Config、Importsを考慮して
// 最終的な ConfigFile, Program, ProgramArgs を返す
func ResolveRunOptions(opts types.RunOptions, tagData types.TagData, config types.Config, os types.OS, logger interfaces.Logger) (configFile string, program string, programArgs []string, finalEnv []string) {
    // 基本は RunOptions の値
    configFile = opts.ConfigFile
    program = opts.Program
    programArgs = opts.ProgramArgs

    // タグ値で上書き
    if tagData.ConfigFile != "" {
        configFile = tagData.ConfigFile
    }
    if tagData.Program != "" {
        program = tagData.Program
    }
    if len(tagData.ProgramArgs) != 0 {
        programArgs = tagData.ProgramArgs
    }

    // config 値で補完
    if configFile == "" {
    }
    if program == "" {
        program = config.Program.Path
    }
    if len(programArgs) == 0 {
        programArgs = config.Program.Args
    }

    // imports も考慮して最終環境変数を作成
    allConfigs := []types.Config{}

    // opts で指定された imports
    for _, imp := range opts.Imports {
        if cfg, err := ReadOrFallbackRecursive(opts,os, logger, imp); err == nil {
            allConfigs = append(allConfigs, cfg)
        }
    }

    // タグで指定された imports
    for _, imp := range tagData.ImportConfigFiles {
        if cfg, err := ReadOrFallbackRecursive(opts,os, logger, imp); err == nil {
            allConfigs = append(allConfigs, cfg)
        }
    }

    // メイン config を最後に追加
    allConfigs = append(allConfigs, config)

    // 現在の環境変数をベースにマージ
    finalEnv = os.Env.Environ()
    for _, cfg := range allConfigs {
        finalEnv = cfg.BuildEnvs(os, logger, finalEnv,opts.Separator)
    }

    return
}
*/
/*[>*/
/*// ResolveRunOptionsは、RunOptions、TagData、Config、Importsを考慮して*/
/*// 最終的な ConfigFile, Program, ProgramArgs, 最終環境変数 を返す*/
/*func ResolveRunOptions(*/
	/*opts types.RunOptions,*/
	/*tagData types.TagData,*/
	/*//config types.Config,*/
	/*os types.OS,*/
	/*logger interfaces.Logger,*/
/*) (configFile string, program string, programArgs []string, finalEnv []string) {*/

	/*var config types.Config*/
	/*var err error*/

	/*// ------------------------*/
	/*// ConfigFile の決定*/
	/*// ------------------------*/
	/*switch {*/
	/*case opts.ConfigFile != "":*/
		/*configFile = opts.ConfigFile // CLI優先*/
	/*case tagData.ConfigFile != "":*/
		/*configFile = tagData.ConfigFile // タグ*/
	/*}*/
	/*// ----------------------*/
	/*// メイン config 読み込み*/
	/*// -----------------------*/
	/*if configFile != "" && os.FS.FileExists(configFile) {*/
		/*config, err = types.ReadConfig(os, logger, configFile)*/
		/*if err != nil {*/
			/*logger.Error().Err(err).Str("configFile", configFile).Msg("failed to read config")*/
		/*}*/
	/*}*/

	/*// ------------------------*/
	/*// Program の決定*/
	/*// ------------------------*/
	/*switch {*/
	/*case opts.Program != "":*/
		/*program = opts.Program*/
	/*case tagData.Program != "":*/
		/*program = tagData.Program*/
	/*default:*/
		/*program = config.Program.Path*/
	/*}*/

	/*// ------------------------*/
	/*// ProgramArgs の決定*/
	/*// ------------------------*/
	/*switch {*/
	/*case len(opts.ProgramArgs) != 0:*/
		/*programArgs = opts.ProgramArgs*/
	/*case len(tagData.ProgramArgs) != 0:*/
		/*programArgs = tagData.ProgramArgs*/
	/*default:*/
		/*programArgs = config.Program.Args*/
	/*}*/

	/*// ------------------------*/
	/*// imports も考慮して最終環境変数を作成*/
	/*// ------------------------*/
	/*allConfigs := []types.Config{}*/

	/*// CLI で指定された imports*/
	/*for _, imp := range opts.Imports {*/
		/*if cfg, err := ReadOrFallbackRecursive(opts, os, logger, imp); err == nil {*/
			/*allConfigs = append(allConfigs, cfg)*/
		/*}*/
	/*}*/

	/*// タグで指定された imports*/
	/*for _, imp := range tagData.ImportConfigFiles {*/
		/*if cfg, err := ReadOrFallbackRecursive(opts, os, logger, imp); err == nil {*/
			/*allConfigs = append(allConfigs, cfg)*/
		/*}*/
	/*}*/

	/*// メイン Config を最後に追加*/
	/*allConfigs = append(allConfigs, config)*/

	/*// 現在の環境変数をベースにマージ*/
	/*finalEnv = os.Env.Environ()*/
	/*for _, cfg := range allConfigs {*/
		/*finalEnv = cfg.BuildEnvs(os, logger, finalEnv, opts.Separator)*/
	/*}*/

	/*return*/
/*}*/
/**/





func ResolveRunOptions(
	opts types.RunOptions,
	tagData types.TagData,
	os types.OS,
	logger interfaces.Logger,
) (configFile string, program string, programArgs []string, finalEnv []string) {

	var config types.Config
	var err error
	allConfigs := []types.Config{}

	// ------------------------
	// ConfigFile の決定
	// ------------------------
	switch {
	case opts.ConfigFile != "":
		configFile = opts.ConfigFile // CLI優先
	case tagData.ConfigFile != "":
		configFile = tagData.ConfigFile // タグ
	}

	// 絶対パス化はこのタイミングで行う
	if configFile != "" {
		if abs, err := filepath.Abs(configFile); err == nil {
			configFile = abs
		} else {
			logger.Warn().Err(err).Msg("failed to resolve absolute path for configFile")
		}
	}

	// ----------------------
	// imports（優先度：低 → 高 の順）
	// ----------------------

	// タグで指定された imports（最も低い）
	for _, imp := range tagData.ImportConfigFiles {
		if cfg, err := ReadOrFallbackRecursive(opts, os, logger, imp); err == nil {
			allConfigs = append(allConfigs, cfg)
		}
	}

	// CLI で指定された imports（中間）
	for _, imp := range opts.Imports {
		if cfg, err := ReadOrFallbackRecursive(opts, os, logger, imp); err == nil {
			allConfigs = append(allConfigs, cfg)
		}
	}

	// ----------------------
	// メイン config（最も高い）
	// ----------------------
	if configFile != "" && os.FS.FileExists(configFile) {
		config, err = types.ReadConfig(os, logger, configFile)
		if err != nil {
			logger.Error().Err(err).Str("configFile", configFile).Msg("failed to read config")
		} else {
			allConfigs = append(allConfigs, config)
		}
	}

	// ------------------------
	// Program の決定
	// ------------------------
	switch {
	case opts.Program != "":
		program = opts.Program
	case tagData.Program != "":
		program = tagData.Program
	default:
		program = config.Program.Path
	}

	// ------------------------
	// ProgramArgs の決定
	// ------------------------
	switch {
	case len(opts.ProgramArgs) != 0:
		programArgs = opts.ProgramArgs
	case len(tagData.ProgramArgs) != 0:
		programArgs = tagData.ProgramArgs
	default:
		programArgs = config.Program.Args
	}

	// ------------------------
	// 環境変数を構築
	// ------------------------
	finalEnv = os.Env.Environ()
	for _, cfg := range allConfigs {
		finalEnv = cfg.BuildEnvs(os, logger, finalEnv, opts.Separator)
	}

	return configFile, program, programArgs, finalEnv
}
