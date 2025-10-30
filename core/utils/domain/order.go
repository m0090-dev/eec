package domain
import "github.com/m0090-dev/eec-go/core/types"
import "github.com/m0090-dev/eec-go/core/interfaces"
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
