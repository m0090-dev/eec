package domain

import (
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
)

// ValidationResult is the outcome of validating one config source.
type ValidationResult struct {
	Source string
	Err    error
}

// ConfigSources lists the individual config sources referenced by
// opts and tagData, in resolution order: the -c config file (or the
// tag's config file) first, then the tag's imports, then -i imports.
// Inline configs are not listed: they have no source to point at.
func ConfigSources(opts types.RunOptions, tagData types.TagData) []string {
	var sources []string
	switch {
	case opts.ConfigFile != "":
		sources = append(sources, opts.ConfigFile)
	case tagData.ConfigFile != "":
		sources = append(sources, tagData.ConfigFile)
	}
	sources = append(sources, tagData.ImportConfigFiles...)
	sources = append(sources, opts.Imports...)
	return sources
}

// ValidateSource loads one config source exactly the way run does —
// import recursion, tag fallback and circular-import detection
// included — and builds its environment against the current process
// environment. Parse errors, circular references and dependency-sort
// errors all surface here; nothing is swallowed.
func ValidateSource(rt interfaces.Runtime, opts types.RunOptions, source string) error {
	cfg, err := ReadOrFallbackRecursive(opts, rt, source)
	if err != nil {
		return err
	}
	eo := general.DefaultExpandOptions()
	if opts.NoExpand {
		eo.ExpandVars = false
	}
	if opts.NoExec {
		eo.ExecCommands = false
	}
	_ = cfg.BuildEnvs(rt, rt.Env().Environ(), opts.Separator, eo)
	return nil
}

// ValidateSources validates every source referenced by opts/tagData
// and returns one result per source, in ConfigSources order.
func ValidateSources(rt interfaces.Runtime, opts types.RunOptions, tagData types.TagData) []ValidationResult {
	sources := ConfigSources(opts, tagData)
	results := make([]ValidationResult, 0, len(sources))
	for _, s := range sources {
		results = append(results, ValidationResult{Source: s, Err: ValidateSource(rt, opts, s)})
	}
	return results
}
