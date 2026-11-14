package domain

import "fmt"
import "github.com/m0090-dev/eec/internal/ext/types"
import "github.com/m0090-dev/eec/internal/ext/interfaces"
import "github.com/m0090-dev/eec/internal/ext/utils/general"
import "path/filepath"

// resolveExpandOptions computes the effective expansion options for a
// run: defaults are both enabled; each config's [configs] expand_vars /
// exec_commands overrides the previous value when specified (later
// configs win, like env vars); the CLI NoExpand / NoExec flags then
// force-disable.
func resolveExpandOptions(allConfigs []types.Config, opts types.RunOptions) general.ExpandOptions {
	eo := general.DefaultExpandOptions()
	for _, cfg := range allConfigs {
		for _, mc := range cfg.Configs {
			if mc.ExpandVars != nil {
				eo.ExpandVars = *mc.ExpandVars
			}
			if mc.ExecCommands != nil {
				eo.ExecCommands = *mc.ExecCommands
			}
		}
	}
	if opts.NoExpand {
		eo.ExpandVars = false
	}
	if opts.NoExec {
		eo.ExecCommands = false
	}
	return eo
}

// ResolveRunOptions は戻り値に []types.Config (allConfigs) を追加。
func ResolveRunOptions(
	opts types.RunOptions,
	tagData types.TagData,
	rt interfaces.Runtime,
) (configFile string, program string, programArgs []string, finalEnv []string, allConfigs []types.Config, err error) {

	var config types.Config
	allConfigs = []types.Config{}

	switch {
	case opts.ConfigFile != "":
		configFile = opts.ConfigFile
	case tagData.ConfigFile != "":
		configFile = tagData.ConfigFile
	}

	if configFile != "" {
		// タグにデータディレクトリ相対で保存された ConfigFile を
		// 解決する (CWD 相対で見つからない場合のみ)。
		configFile = ResolveDataDirRelative(rt, configFile)
		if abs, absErr := filepath.Abs(configFile); absErr == nil {
			configFile = abs
		} else {
			rt.Logger().Warn().Err(absErr).Msg("failed to resolve absolute path for configFile")
		}
	}

	if configFile != "" && rt.FS().FileExists(configFile) {
		config, err = types.ReadConfig(rt, configFile)
		if err != nil {
			rt.Logger().Error().Err(err).Str("configFile", configFile).Msg("failed to read config")
			return "", "", nil, nil, nil, err
		}
		allConfigs = append(allConfigs, config)

	}

	for _, imp := range tagData.ImportConfigFiles {
		cfg, impErr := ReadOrFallbackRecursive(opts, rt, imp)
		if impErr != nil {
			return "", "", nil, nil, nil, impErr
		}
		allConfigs = append(allConfigs, cfg)
	}

	for _, imp := range opts.Imports {
		cfg, impErr := ReadOrFallbackRecursive(opts, rt, imp)
		if impErr != nil {
			return "", "", nil, nil, nil, impErr
		}
		allConfigs = append(allConfigs, cfg)
	}

	inlineCfg, inlineErr := resolveInlineFromOptions(opts, rt)
	if inlineErr != nil {
		return "", "", nil, nil, nil, fmt.Errorf("failed to parse inline config: %w", inlineErr)
	}
	// envs/program が無くても、[configs] のトグル (expand_vars /
	// exec_commands) だけを持つ inline 設定は取り込む。
	if isConfigPresent(inlineCfg) {
		allConfigs = append(allConfigs, inlineCfg)
		if inlineCfg.Program.Path != "" && opts.Program == "" {
			program = inlineCfg.Program.Path
			if len(inlineCfg.Program.Args) > 0 && len(opts.ProgramArgs) == 0 {
				programArgs = inlineCfg.Program.Args
			}
		}
	}

	switch {
	case opts.Program != "":
		program = opts.Program
	case tagData.Program != "":
		program = tagData.Program
	default:
		program = config.Program.Path
	}

	switch {
	case len(opts.ProgramArgs) != 0:
		programArgs = opts.ProgramArgs
	case len(tagData.ProgramArgs) != 0:
		programArgs = tagData.ProgramArgs
	default:
		programArgs = config.Program.Args
	}

	expandOpts := resolveExpandOptions(allConfigs, opts)
	finalEnv = rt.Env().Environ()
	for _, cfg := range allConfigs {
		finalEnv = cfg.BuildEnvs(rt, finalEnv, opts.Separator, expandOpts)
	}

	// Phase 4: [capture] コマンドを最終環境に対して実行し、結果の
	// 環境差分を取り込む (exec_commands が無効ならスキップ)。
	finalEnv, err = applyCaptures(rt, allConfigs, finalEnv, expandOpts)
	if err != nil {
		return "", "", nil, nil, nil, err
	}

	return configFile, program, programArgs, finalEnv, allConfigs, nil
}

func resolveInlineFromOptions(opts types.RunOptions, rt interfaces.Runtime) (types.Config, error) {
	if opts.InlineConfigToml != "" {
		return types.ReadInlineConfig(rt, opts.InlineConfigToml, "toml")
	}
	if opts.InlineConfigYaml != "" {
		return types.ReadInlineConfig(rt, opts.InlineConfigYaml, "yaml")
	}
	if opts.InlineConfigJson != "" {
		return types.ReadInlineConfig(rt, opts.InlineConfigJson, "json")
	}
	if opts.InlineConfig != "" {
		return types.ReadInlineConfig(rt, opts.InlineConfig, "")
	}
	return types.Config{}, nil
}

func isConfigPresent(cfg types.Config) bool {
	return cfg.RawEnvs != nil || len(cfg.Configs) > 0 || cfg.Program.Path != "" || len(cfg.CaptureCommands) > 0
}
