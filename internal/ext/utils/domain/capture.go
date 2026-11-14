package domain

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
)

// captureSentinel separates a capture command's own stdout from the
// environment dump that follows it, so chatty commands cannot corrupt
// parsing.
const captureSentinel = "__EEC_CAPTURE_SENTINEL__"

// CaptureTimeout bounds a single capture command. It is a variable
// (exported for tests) rather than a constant so tests can shorten it.
var CaptureTimeout = 60 * time.Second

var errCaptureTimeout = errors.New("capture timed out")

// shellBookkeepingKeys are variables a POSIX shell maintains for
// itself; a capture must not import them when they were not already
// part of the environment.
var shellBookkeepingKeys = map[string]bool{
	"PWD": true, "OLDPWD": true, "SHLVL": true, "_": true,
}

// applyCaptures runs every config's [capture] commands, in config
// order, on top of finalEnv and merges each resulting environment
// back in. Captures execute arbitrary shell commands, so they obey
// the same exec_commands gate as $() substitution: when exec is
// disabled all captures are skipped with a warning.
func applyCaptures(rt interfaces.Runtime, allConfigs []types.Config, finalEnv []string, expandOpts general.ExpandOptions) ([]string, error) {
	var commands []string
	for _, cfg := range allConfigs {
		commands = append(commands, cfg.CaptureCommands...)
	}
	if len(commands) == 0 {
		return finalEnv, nil
	}
	if !expandOpts.ExecCommands {
		rt.Logger().Warn().Int("commands", len(commands)).
			Msg("capture skipped: exec_commands is disabled")
		return finalEnv, nil
	}

	env := finalEnv
	for _, rawCmd := range commands {
		// ${} in the command string expands against the current env;
		// $() inside the command string itself is never executed by
		// eec (the shell may do whatever it wants with the result).
		expanded := general.ExpandEnvAndCommandsWithOptions(rt, rawCmd, envToValueMap(rt, env),
			general.ExpandOptions{ExpandVars: expandOpts.ExpandVars, ExecCommands: false})
		captured, err := runCaptureCommand(rt, expanded, env)
		if err != nil {
			return nil, fmt.Errorf("capture command %q failed: %w", rawCmd, err)
		}
		env = mergeCapturedEnv(rt.Env().GOOS(), env, captured)
	}
	return env, nil
}

// runCaptureCommand runs command through the OS shell with env and
// returns the FULL environment in effect after the command finishes.
//
//	Windows: cmd /c "<cmd>& echo <sentinel> & set"
//	Unix:    sh -c "<cmd>; echo <sentinel>; env -0"
//
// On Unix the command runs in the capture shell itself, so it must
// source any script whose exports should be captured. On Windows a
// .bat file runs in-process of cmd, so a plain path works.
func runCaptureCommand(rt interfaces.Runtime, command string, env []string) ([]string, error) {
	var name string
	var args []string
	var shellLine string
	if rt.Env().GOOS() == "windows" {
		name = "cmd"
		// The "&" after the command is glued without a space on
		// purpose: in cmd, "set X=value & ..." makes the space part
		// of the value, so a capture command ending in a set would
		// capture a trailing space.
		shellLine = command + "& echo " + captureSentinel + " & set"
		args = []string{"/c", shellLine}
	} else {
		name = "sh"
		args = []string{"-c", command + "; echo " + captureSentinel + "; env -0"}
	}

	cmd, err := rt.Executor().Command(
		name, args, env,
		rt.Console().Stdin(),
		nil, // stdout: nil here; outputWithTimeout attaches its capture pipe
		rt.Console().Stderr(),
		true, // suppress window (Windows)
	)
	if err != nil {
		return nil, fmt.Errorf("could not start capture shell: %w", err)
	}
	if rt.Env().GOOS() == "windows" {
		// Hand the composite line to cmd.exe verbatim: Go's default
		// command-line escaping turns embedded quotes into \" which
		// cmd cannot parse (a quoted capture command never ran).
		general.UseRawCmdExeLine(cmd, shellLine)
	}

	out, runErr := outputWithTimeout(cmd, CaptureTimeout)
	if errors.Is(runErr, errCaptureTimeout) {
		return nil, runErr
	}
	captured, parseErr := parseCapturedEnv(out)
	if parseErr != nil {
		// A nonzero exit of the composite shell is only a failure
		// when it also made the output unparseable; the inner
		// command's own exit status is not visible through the
		// composite and is never treated as a capture failure.
		if runErr != nil {
			return nil, fmt.Errorf("%v (shell error: %w)", parseErr, runErr)
		}
		return nil, parseErr
	}
	return captured, nil
}

// outputWithTimeout captures cmd's stdout with a kill-on-timeout
// guard.
//
// Two concurrency traps shape this implementation:
//
//   - The process is started in this goroutine, never inside the
//     helper goroutines: Start is what writes cmd.Process, and the
//     timeout branch reads cmd.Process to kill it. Reading it while
//     a background Cmd.Output() might still be inside Start is a
//     data race.
//   - Stdout is a real pipe file rather than a buffer. With a
//     buffer, exec spawns a copy goroutine and Wait blocks until it
//     finishes — which is only at pipe EOF, i.e. when EVERY holder
//     of the write end exits. A killed shell whose child (say a
//     `sleep`) inherited stdout would keep Wait blocked for the
//     child's whole lifetime, defeating the timeout. With a file,
//     Wait returns as soon as the shell itself dies; draining the
//     pipe is our reader goroutine's job, and on the timeout path
//     closing the read end unblocks it immediately.
//
// On the success path the reader runs until pipe EOF, so a
// grandchild that inherits stdout and outlives the shell still
// delays capture — the same semantics Cmd.Output() had, since its
// Wait blocked on the copy goroutine in exactly the same way.
func outputWithTimeout(cmd *exec.Cmd, timeout time.Duration) ([]byte, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	cmd.Stdout = w
	if err := cmd.Start(); err != nil {
		w.Close()
		return nil, err
	}
	w.Close() // only the child holds the write end now

	type readResult struct {
		out []byte
		err error
	}
	readDone := make(chan readResult, 1)
	go func() {
		out, err := io.ReadAll(r)
		readDone <- readResult{out, err}
	}()
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	select {
	case werr := <-waitDone:
		res := <-readDone
		if werr != nil {
			return res.out, werr
		}
		return res.out, res.err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-waitDone
		r.Close() // unblock the reader; the output is discarded
		<-readDone
		return nil, fmt.Errorf("%w after %s", errCaptureTimeout, timeout)
	}
}

// parseCapturedEnv extracts the environment dump that follows the
// sentinel in capture output. NUL-separated output (env -0) is
// preferred; when no NUL is present (e.g. Windows `set`, or an env
// without -0 support), lines are split instead.
func parseCapturedEnv(out []byte) ([]string, error) {
	s := string(out)
	idx := strings.LastIndex(s, captureSentinel)
	if idx < 0 {
		return nil, fmt.Errorf("capture sentinel %q not found in command output", captureSentinel)
	}
	rest := strings.TrimLeft(s[idx+len(captureSentinel):], "\r\n")

	var fields []string
	if strings.Contains(rest, "\x00") {
		fields = strings.Split(rest, "\x00")
	} else {
		fields = strings.Split(strings.ReplaceAll(rest, "\r\n", "\n"), "\n")
	}

	envs := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimRight(f, "\r")
		if f == "" {
			continue
		}
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 || parts[0] == "" || strings.HasPrefix(parts[0], "=") {
			continue
		}
		envs = append(envs, f)
	}
	return envs, nil
}

// mergeCapturedEnv diffs the captured full environment against the
// pre-capture env using the OS canonical key rules: keys present in
// captured replace the old value wholesale (no list merging — a
// captured PATH replaces PATH), keys missing from captured were
// removed by the command and disappear, untouched keys stay.
func mergeCapturedEnv(goos string, before, captured []string) []string {
	type entry struct{ key, val string }
	parse := func(list []string) []entry {
		entries := make([]entry, 0, len(list))
		for _, e := range list {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) != 2 || parts[0] == "" || strings.HasPrefix(parts[0], "=") {
				continue
			}
			entries = append(entries, entry{parts[0], parts[1]})
		}
		return entries
	}

	capturedByCanon := make(map[string]entry)
	for _, e := range parse(captured) {
		capturedByCanon[types.CanonicalEnvKey(goos, e.key)] = e
	}

	result := make([]string, 0, len(before)+len(captured))
	seen := make(map[string]bool)
	for _, e := range parse(before) {
		canon := types.CanonicalEnvKey(goos, e.key)
		if seen[canon] {
			continue
		}
		seen[canon] = true
		if ce, ok := capturedByCanon[canon]; ok {
			result = append(result, ce.key+"="+ce.val)
		}
		// absent from captured: the command removed it -> drop
	}
	for _, e := range parse(captured) {
		canon := types.CanonicalEnvKey(goos, e.key)
		if seen[canon] {
			continue
		}
		seen[canon] = true
		if goos != "windows" && shellBookkeepingKeys[canon] {
			continue
		}
		result = append(result, e.key+"="+e.val)
	}
	return result
}

// envToValueMap converts KEY=VALUE entries into the map shape the
// expansion helpers expect, splitting values on the path separator.
func envToValueMap(rt interfaces.Runtime, env []string) map[string][]string {
	sep := rt.Env().PathListSeparator()
	m := make(map[string][]string, len(env))
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		m[parts[0]] = strings.Split(parts[1], sep)
	}
	return m
}
