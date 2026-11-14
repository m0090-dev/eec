package core

import (
	"context"
	"fmt"
	"io"

	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// loadTagData reads the tag file referenced by opts, mirroring the
// tag handling of Run and Dump. It returns an empty TagData when no
// tag is requested.
func (e *Engine) loadTagData(opts types.RunOptions) (types.TagData, error) {
	if opts.Tag == "" {
		return types.TagData{}, nil
	}
	tagData, err := types.ReadTagData(e.Runtime, opts.Tag)
	if err != nil {
		e.Runtime.Logger().Error().Err(err).Str("tag", opts.Tag).Msg("failed to read tag")
		return types.TagData{}, fmt.Errorf("failed to read tag %s: %w", opts.Tag, err)
	}
	return tagData, nil
}

// ResolveEnv resolves the environment for opts without requiring a
// program (the same resolution dump performs). It also returns the
// tag data that fed the resolution.
func (e *Engine) ResolveEnv(opts types.RunOptions) (finalEnv []string, tagData types.TagData, err error) {
	tagData, err = e.loadTagData(opts)
	if err != nil {
		return nil, types.TagData{}, err
	}
	_, _, _, finalEnv, _, err = domain.ResolveRunOptions(opts, tagData, e.Runtime)
	if err != nil {
		return nil, tagData, err
	}
	return finalEnv, tagData, nil
}

// Which resolves the environment for opts and returns the absolute
// path of name within that environment's PATH.
func (e *Engine) Which(opts types.RunOptions, name string) (string, error) {
	finalEnv, _, err := e.ResolveEnv(opts)
	if err != nil {
		return "", err
	}
	if found, ok := domain.FindInPath(e.Runtime, finalEnv, name); ok {
		return found, nil
	}
	return "", fmt.Errorf("%s: not found in PATH of the resolved environment", name)
}

// Validate validates every config source referenced by opts. An
// unreadable tag file is reported as a single failed source rather
// than aborting the whole validation.
func (e *Engine) Validate(opts types.RunOptions) []domain.ValidationResult {
	tagData, err := e.loadTagData(opts)
	if err != nil {
		return []domain.ValidationResult{{Source: "tag " + opts.Tag, Err: err}}
	}
	return domain.ValidateSources(e.Runtime, opts, tagData)
}

// Shell resolves the environment for opts and starts an interactive
// shell with it on the console stdio. The shell's exit code is
// returned as a *types.ProcessExitError.
func (e *Engine) Shell(ctx context.Context, opts types.RunOptions) error {
	tagData, err := e.loadTagData(opts)
	if err != nil {
		return err
	}
	return domain.RunShell(e.Runtime, opts, tagData,
		e.Console().Stdin(), e.Console().Stdout(), e.Console().Stderr())
}

// RunWatch runs the program for opts under the watch supervisor:
// config-file changes restart the child with a freshly resolved
// environment. Cancelling ctx stops the child and yields exit
// code 130.
func (e *Engine) RunWatch(ctx context.Context, opts types.RunOptions) error {
	if opts.Verbose {
		e.Runtime.SetLogger(e.Runtime.Logger().EnableDebug())
	}
	tagData, err := e.loadTagData(opts)
	if err != nil {
		return err
	}
	return domain.RunWatch(ctx, e.Runtime, opts, tagData,
		e.Console().Stdin(), e.Console().Stdout(), e.Console().Stderr())
}

// RunREPL runs an interactive session in the environment resolved
// from opts, reading lines from in and writing to out.
func (e *Engine) RunREPL(ctx context.Context, opts types.RunOptions, in io.Reader, out io.Writer, interactive bool) error {
	return domain.RunREPL(ctx, e.Runtime, opts, in, out, interactive)
}

// Doctor runs the resolution sanity checks for opts; program may be
// empty to skip the program lookup check.
func (e *Engine) Doctor(opts types.RunOptions, program string) ([]domain.DoctorCheck, error) {
	tagData, err := e.loadTagData(opts)
	if err != nil {
		return nil, err
	}
	return domain.RunDoctor(e.Runtime, opts, tagData, program), nil
}
