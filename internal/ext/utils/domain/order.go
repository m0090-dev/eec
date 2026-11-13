package domain

import "fmt"
import "github.com/m0090-dev/eec/internal/ext/types"
import "github.com/m0090-dev/eec/internal/ext/interfaces"
import "path/filepath"

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
	if inlineCfg.RawEnvs != nil || inlineCfg.Program.Path != "" {
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

	finalEnv = rt.Env().Environ()
	for _, cfg := range allConfigs {
		finalEnv = cfg.BuildEnvs(rt, finalEnv, opts.Separator)
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
	return cfg.RawEnvs != nil || len(cfg.Configs) > 0 || cfg.Program.Path != ""
}
