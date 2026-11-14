package domain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
)

// WatchPollInterval is how often watch mode stats the config files.
// WatchSettleDelay is the debounce delay after a first observed
// change: the new state must additionally survive one more poll
// unchanged before the child is restarted, so editors that rewrite
// a file in several quick steps trigger a single restart. Both are
// variables so tests can shorten them.
var (
	WatchPollInterval = 500 * time.Millisecond
	WatchSettleDelay  = 200 * time.Millisecond
)

// FileState is the snapshot of one watched file: whether it exists
// and, if it does, its modification time.
type FileState struct {
	Exists  bool
	ModTime time.Time
}

// SnapshotFiles stats every path and returns its FileState. Files
// that cannot be stat'ed are recorded as missing.
func SnapshotFiles(fs interfaces.FS, paths []string) map[string]FileState {
	snap := make(map[string]FileState, len(paths))
	for _, p := range paths {
		info, err := fs.Stat(p)
		if err != nil || info == nil {
			snap[p] = FileState{Exists: false}
			continue
		}
		snap[p] = FileState{Exists: true, ModTime: info.ModTime()}
	}
	return snap
}

// ChangedPaths returns the sorted list of paths whose FileState
// differs between two snapshots, including paths present in only
// one of them.
func ChangedPaths(prev, cur map[string]FileState) []string {
	var changed []string
	for p, cs := range cur {
		ps, ok := prev[p]
		if !ok || ps != cs {
			changed = append(changed, p)
		}
	}
	for p := range prev {
		if _, ok := cur[p]; !ok {
			changed = append(changed, p)
		}
	}
	sort.Strings(changed)
	return changed
}

// CollectWatchPaths returns the deduplicated, sorted SourcePath of
// every resolved config — the files watch mode monitors. Inline
// configs have no SourcePath and are skipped.
func CollectWatchPaths(allConfigs []types.Config) []string {
	seen := make(map[string]bool)
	var paths []string
	for _, cfg := range allConfigs {
		if cfg.SourcePath == "" || seen[cfg.SourcePath] {
			continue
		}
		seen[cfg.SourcePath] = true
		paths = append(paths, cfg.SourcePath)
	}
	sort.Strings(paths)
	return paths
}

// RunWatch runs the program for opts under a supervisor: whenever a
// config file that fed the resolution changes, the child is killed,
// everything is re-resolved from scratch, and the child restarts.
// When the child exits on its own (with no change pending) its exit
// code is returned, exactly like Run. Cancelling ctx kills the
// child and returns exit code 130, following the usual SIGINT
// convention.
//
// The restart kill targets the child process only — same
// granularity as WaitProcess timeouts; detached grandchildren are
// out of scope.
func RunWatch(ctx context.Context, rt interfaces.Runtime, opts types.RunOptions, tagData types.TagData, stdin, stdout, stderr *os.File) error {
	for {
		_, program, programArgs, finalEnv, allConfigs, err := ResolveRunOptions(opts, tagData, rt)
		if err != nil {
			return err
		}
		if program == "" {
			return errors.New("no program specified")
		}
		paths := CollectWatchPaths(allConfigs)
		if len(paths) == 0 {
			return errors.New("watch requires file-based configs: no config file to watch (inline configs cannot be watched)")
		}

		cmd, err := rt.Executor().StartProcess(program, programArgs, finalEnv, stdin, stdout, stderr, opts.HideWindow)
		if err != nil {
			return fmt.Errorf("failed to start process: %w", err)
		}
		done := make(chan error, 1)
		go func() { done <- rt.Executor().WaitProcess(cmd, opts.WaitTimeout) }()

		restart, result := watchChild(ctx, rt, done, cmd.Process, paths)
		if !restart {
			return result
		}
		rt.Logger().Info().Msg("watch: config changed, restarting")
	}
}

// watchChild watches paths while one child runs. It reports
// restart=true after killing the child because a stable config
// change was confirmed; otherwise it returns the child's own wait
// result (or exit code 130 on ctx cancellation).
func watchChild(ctx context.Context, rt interfaces.Runtime, done chan error, proc *os.Process, paths []string) (restart bool, result error) {
	base := SnapshotFiles(rt.FS(), paths)
	var pending map[string]FileState

	kill := func() {
		if proc != nil {
			_ = proc.Kill()
		}
		<-done // reap; the killed child's exit error is expected here
	}

	ticker := time.NewTicker(WatchPollInterval)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return false, err
		case <-ctx.Done():
			kill()
			return false, &types.ProcessExitError{ExitCode: 130}
		case <-ticker.C:
			snap := SnapshotFiles(rt.FS(), paths)
			if pending == nil {
				if len(ChangedPaths(base, snap)) == 0 {
					continue
				}
				// First observed change: debounce. Wait the settle
				// delay (still reacting to the child and ctx), then
				// demand one unchanged confirmation snapshot.
				pending = snap
				select {
				case err := <-done:
					return false, err
				case <-ctx.Done():
					kill()
					return false, &types.ProcessExitError{ExitCode: 130}
				case <-time.After(WatchSettleDelay):
				}
				continue
			}
			if len(ChangedPaths(pending, snap)) == 0 {
				// The new state survived a full extra poll: it is
				// stable, restart with it.
				kill()
				return true, nil
			}
			// Still changing; track the newest state and keep
			// waiting for it to settle.
			pending = snap
		}
	}
}
