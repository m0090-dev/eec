package domain

import (
	"errors"
	"fmt"
	"os"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
)

// InteractiveShell picks the interactive shell for the current OS:
// on Windows the ComSpec variable (default "cmd"); on Unix the
// SHELL variable (default "sh"). The resolved environment env is
// consulted first, then the process environment.
func InteractiveShell(rt interfaces.Runtime, env []string) string {
	goos := rt.Env().GOOS()
	if goos == "windows" {
		if v, ok := GetEnvValue(env, goos, "ComSpec"); ok && v != "" {
			return v
		}
		if v, ok := rt.Env().LookupEnv("ComSpec"); ok && v != "" {
			return v
		}
		return "cmd"
	}
	if v, ok := GetEnvValue(env, goos, "SHELL"); ok && v != "" {
		return v
	}
	if v, ok := rt.Env().LookupEnv("SHELL"); ok && v != "" {
		return v
	}
	return "sh"
}

// RunShell resolves the environment for opts and starts an
// interactive shell as a child process with that environment,
// inheriting stdio. The shell's exit code is propagated as a
// *types.ProcessExitError, exactly like Run. Note that the shell is
// a NEW process: nothing it does (cd, set, exports) can affect the
// shell that launched eec.
func RunShell(rt interfaces.Runtime, opts types.RunOptions, tagData types.TagData, stdin, stdout, stderr *os.File) error {
	_, _, _, finalEnv, _, err := ResolveRunOptions(opts, tagData, rt)
	if err != nil {
		return err
	}

	shell := InteractiveShell(rt, finalEnv)
	cmd, err := rt.Executor().StartProcess(shell, nil, finalEnv, stdin, stdout, stderr, opts.HideWindow)
	if err != nil {
		return fmt.Errorf("failed to start shell %q: %w", shell, err)
	}

	if cmd.Process != nil {
		rt.Logger().Info().
			Int("ChildPID", cmd.Process.Pid).
			Str("Shell", shell).
			Msg("shell started")
	}

	if err := rt.Executor().WaitProcess(cmd, opts.WaitTimeout); err != nil {
		var exitErr *types.ProcessExitError
		if errors.As(err, &exitErr) {
			return exitErr
		}
		return fmt.Errorf("shell wait error: %w", err)
	}
	return nil
}
