package domain

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
)

// replPrompt is shown before each input line in interactive mode.
const replPrompt = "eec> "

// replSession is the mutable state of one REPL session: the most
// recently resolved environment, the session overrides applied on
// top of it, and the session working directory.
//
// Session state model: the effective environment is always
// recomputed as (resolved base env) − (unset tombstones) +
// (set overrides). Overrides and tombstones live outside the base
// env, so a reload — which replaces the base env with a fresh
// resolution — re-applies them automatically.
type replSession struct {
	rt          interfaces.Runtime
	opts        types.RunOptions
	tagData     types.TagData
	out         io.Writer
	interactive bool
	stdinFile   *os.File // stdin handed to child commands; may be nil

	baseEnv    []string
	overrides  map[string]string // canonical key -> "KEY=VALUE" entry
	tombstones map[string]bool   // canonical keys removed by unset
	cwd        string
}

// RunREPL runs an interactive session in the environment resolved
// from opts. Lines are read from in; output goes to out; prompts are
// printed only when interactive is true.
//
// Builtins: env, set, unset, cd, reload, help, exit/quit. Any other
// line is executed through the OS shell (cmd /c on Windows, sh -c on
// Unix) with the session environment and working directory; a
// non-zero exit prints "[exit N]" but keeps the REPL alive.
//
// reload re-resolves the original sources — the config files are
// re-read — and re-applies the session's set/unset state on top.
// That is the supported way to pick up config edits in a live
// session (the workflow the original "restart" idea wanted): edit
// the file, reload, keep working.
//
// EOF on in ends the session cleanly.
func RunREPL(ctx context.Context, rt interfaces.Runtime, opts types.RunOptions, in io.Reader, out io.Writer, interactive bool) error {
	var tagData types.TagData
	if opts.Tag != "" {
		var err error
		tagData, err = types.ReadTagData(rt, opts.Tag)
		if err != nil {
			return fmt.Errorf("failed to read tag %s: %w", opts.Tag, err)
		}
	}

	s := &replSession{
		rt:          rt,
		opts:        opts,
		tagData:     tagData,
		out:         out,
		interactive: interactive,
		overrides:   make(map[string]string),
		tombstones:  make(map[string]bool),
	}
	if cwd, err := os.Getwd(); err == nil {
		s.cwd = cwd
	}
	if f, ok := in.(*os.File); ok {
		s.stdinFile = f
	} else if interactive {
		s.stdinFile = rt.Console().Stdin()
	}

	configFile, err := s.resolve()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "eec repl — %s, %d variables\n", describeSources(opts, configFile), len(s.sessionEnv()))
	fmt.Fprintln(out, `Type "help" for builtins, "exit" to quit.`)

	reader := bufio.NewReader(in)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if interactive {
			fmt.Fprint(out, replPrompt)
		}
		line, readErr := reader.ReadString('\n')
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			if s.handleLine(trimmed) {
				return nil
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

// describeSources summarizes where the session environment came
// from, for the banner.
func describeSources(opts types.RunOptions, configFile string) string {
	var parts []string
	if configFile != "" {
		parts = append(parts, "config "+configFile)
	}
	if opts.Tag != "" {
		parts = append(parts, "tag "+opts.Tag)
	}
	for _, imp := range opts.Imports {
		parts = append(parts, "import "+imp)
	}
	if opts.InlineConfig != "" || opts.InlineConfigToml != "" || opts.InlineConfigYaml != "" || opts.InlineConfigJson != "" {
		parts = append(parts, "inline config")
	}
	if len(parts) == 0 {
		return "process environment only"
	}
	return strings.Join(parts, ", ")
}

// resolve re-runs the full resolution and replaces the base env.
func (s *replSession) resolve() (string, error) {
	configFile, _, _, finalEnv, _, err := ResolveRunOptions(s.opts, s.tagData, s.rt)
	if err != nil {
		return "", err
	}
	s.baseEnv = finalEnv
	return configFile, nil
}

// sessionEnv computes the effective environment: base env minus
// tombstones, with overrides replacing their key (appended at the
// end when the key was tombstoned or absent).
func (s *replSession) sessionEnv() []string {
	goos := s.rt.Env().GOOS()
	env := make([]string, 0, len(s.baseEnv)+len(s.overrides))
	for _, kv := range s.baseEnv {
		k, _, ok := strings.Cut(kv, "=")
		if !ok {
			env = append(env, kv)
			continue
		}
		ck := types.CanonicalEnvKey(goos, k)
		if s.tombstones[ck] {
			continue
		}
		if _, overridden := s.overrides[ck]; overridden {
			continue // appended below, exactly once
		}
		env = append(env, kv)
	}
	keys := make([]string, 0, len(s.overrides))
	for ck := range s.overrides {
		keys = append(keys, ck)
	}
	sort.Strings(keys)
	for _, ck := range keys {
		env = append(env, s.overrides[ck])
	}
	return env
}

// handleLine executes one input line and reports whether the
// session should end. "exit"/"quit" are builtins only when they are
// the whole line, so "exit 42" still runs as a shell command (and
// prints "[exit 42]") instead of leaving the REPL.
func (s *replSession) handleLine(line string) (quit bool) {
	if line == "exit" || line == "quit" {
		return true
	}
	fields := strings.Fields(line)
	switch fields[0] {
	case "help":
		s.cmdHelp()
	case "env":
		s.cmdEnv(fields[1:])
	case "set":
		s.cmdSet(line)
	case "unset":
		if len(fields) < 2 {
			fmt.Fprintln(s.out, "usage: unset KEY")
		} else {
			s.cmdUnset(fields[1])
		}
	case "cd":
		s.cmdCd(line, fields)
	case "reload":
		s.cmdReload()
	default:
		s.runExternal(line)
	}
	return false
}

func (s *replSession) cmdHelp() {
	fmt.Fprintln(s.out, `Builtins:
  env [KEY]       print the session environment (or one variable)
  set KEY=VALUE   set a session variable (survives reload)
  unset KEY       remove a session variable (survives reload)
  cd [DIR]        change (or print) the session working directory
  reload          re-read the configs and rebuild the environment
  help            show this help
  exit / quit     leave the REPL
Anything else runs through the OS shell with the session
environment and working directory.`)
}

func (s *replSession) cmdEnv(args []string) {
	env := s.sessionEnv()
	if len(args) == 0 {
		sorted := append([]string(nil), env...)
		sort.Strings(sorted)
		for _, kv := range sorted {
			fmt.Fprintln(s.out, kv)
		}
		return
	}
	key := args[0]
	goos := s.rt.Env().GOOS()
	for _, kv := range env {
		k, _, ok := strings.Cut(kv, "=")
		if ok && types.EnvKeyEqual(goos, k, key) {
			fmt.Fprintln(s.out, kv)
			return
		}
	}
	fmt.Fprintf(s.out, "%s is not set\n", key)
}

func (s *replSession) cmdSet(line string) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "set"))
	key, value, ok := strings.Cut(rest, "=")
	key = strings.TrimSpace(key)
	if !ok || key == "" || strings.ContainsAny(key, " \t") {
		fmt.Fprintln(s.out, "usage: set KEY=VALUE")
		return
	}
	goos := s.rt.Env().GOOS()
	ck := types.CanonicalEnvKey(goos, key)
	s.overrides[ck] = key + "=" + strings.TrimSpace(value)
	delete(s.tombstones, ck)
}

func (s *replSession) cmdUnset(key string) {
	goos := s.rt.Env().GOOS()
	ck := types.CanonicalEnvKey(goos, key)
	delete(s.overrides, ck)
	s.tombstones[ck] = true
}

func (s *replSession) cmdCd(line string, fields []string) {
	if len(fields) < 2 {
		fmt.Fprintln(s.out, s.cwd)
		return
	}
	arg := strings.TrimSpace(strings.TrimPrefix(line, "cd"))
	target := arg
	if !filepath.IsAbs(target) {
		target = filepath.Join(s.cwd, target)
	}
	target = filepath.Clean(target)
	info, err := s.rt.FS().Stat(target)
	if err != nil || info == nil || !info.IsDir() {
		fmt.Fprintf(s.out, "cd: no such directory: %s\n", arg)
		return
	}
	s.cwd = target
}

func (s *replSession) cmdReload() {
	if _, err := s.resolve(); err != nil {
		fmt.Fprintf(s.out, "reload failed: %v\n", err)
		return
	}
	fmt.Fprintf(s.out, "Reloaded — %d variables\n", len(s.sessionEnv()))
}

// runExternal executes a line through the OS shell with the session
// environment and working directory. A non-zero exit prints
// "[exit N]"; the REPL always stays alive.
func (s *replSession) runExternal(line string) {
	goos := s.rt.Env().GOOS()
	var name string
	var args []string
	if goos == "windows" {
		name, args = "cmd", []string{"/c", line}
	} else {
		name, args = "sh", []string{"-c", line}
	}

	outFile, cleanup, err := s.childStdout()
	if err != nil {
		fmt.Fprintf(s.out, "eec: %v\n", err)
		return
	}
	cmd, err := s.rt.Executor().Command(name, args, s.sessionEnv(), s.stdinFile, outFile, outFile, false)
	if err != nil {
		cleanup()
		fmt.Fprintf(s.out, "eec: %v\n", err)
		return
	}
	if s.cwd != "" {
		cmd.Dir = s.cwd
	}
	if err := cmd.Start(); err != nil {
		cleanup()
		fmt.Fprintf(s.out, "eec: %v\n", err)
		return
	}
	waitErr := s.rt.Executor().WaitProcess(cmd, 0)
	cleanup()
	if waitErr != nil {
		var exitErr *types.ProcessExitError
		if errors.As(waitErr, &exitErr) {
			fmt.Fprintf(s.out, "[exit %d]\n", exitErr.ExitCode)
		} else {
			fmt.Fprintf(s.out, "eec: %v\n", waitErr)
		}
	}
}

// childStdout returns the file child output should be written to.
// When the REPL's own output is a file it is used directly;
// otherwise (tests, pipes) the child's output is pumped through an
// os.Pipe into out, and the returned cleanup must be called after
// the child exits to flush it.
func (s *replSession) childStdout() (*os.File, func(), error) {
	if f, ok := s.out.(*os.File); ok {
		return f, func() {}, nil
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(s.out, r)
		close(done)
	}()
	return w, func() {
		_ = w.Close()
		<-done
		_ = r.Close()
	}, nil
}
