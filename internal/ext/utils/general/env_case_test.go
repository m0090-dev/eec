package general_test

import (
	"testing"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
)

// fakeGoosEnv / fakeGoosRuntime simulate a foreign GOOS so the
// OS-specific key case rules can be exercised on a Linux host.
type fakeGoosEnv struct {
	goos   string
	lookup map[string]string
}

func (e *fakeGoosEnv) Environ() []string                 { return nil }
func (e *fakeGoosEnv) Unsetenv(string) error             { return nil }
func (e *fakeGoosEnv) Setenv(k, v string) error          { e.lookup[k] = v; return nil }
func (e *fakeGoosEnv) UserHomeDir() (string, error)      { return "", nil }
func (e *fakeGoosEnv) GOOS() string                      { return e.goos }
func (e *fakeGoosEnv) LookupEnv(k string) (string, bool) { v, ok := e.lookup[k]; return v, ok }
func (e *fakeGoosEnv) PathListSeparator() string {
	if e.goos == "windows" {
		return ";"
	}
	return ":"
}

type fakeGoosRuntime struct {
	*impl.DefaultRuntime
	env *fakeGoosEnv
}

func (r *fakeGoosRuntime) Env() interfaces.Env { return r.env }

func newFakeGoosRuntime(goos string) *fakeGoosRuntime {
	return &fakeGoosRuntime{
		DefaultRuntime: &impl.DefaultRuntime{},
		env:            &fakeGoosEnv{goos: goos, lookup: map[string]string{}},
	}
}

// Phase 3-1: expansion lookup must use the runtime GOOS case rules.
func TestExpandEnvAndCommands_CaseRulesByGOOS(t *testing.T) {
	// Windows: ${path} finds the canonical PATH entry.
	win := newFakeGoosRuntime("windows")
	envMap := map[string][]string{"PATH": {`C:\bin`}}
	if got := general.ExpandEnvAndCommands(win, "${path}", envMap); got != `C:\bin` {
		t.Errorf("windows: ${path} = %q, want %q", got, `C:\bin`)
	}

	// Linux: exact case resolves, different case stays literal and separate.
	linux := newFakeGoosRuntime("linux")
	envMap = map[string][]string{"http_proxy": {"http://x"}}
	if got := general.ExpandEnvAndCommands(linux, "${http_proxy}", envMap); got != "http://x" {
		t.Errorf("linux: ${http_proxy} = %q, want %q", got, "http://x")
	}
	if got := general.ExpandEnvAndCommands(linux, "${HTTP_PROXY}", envMap); got != "${HTTP_PROXY}" {
		t.Errorf("linux: ${HTTP_PROXY} = %q, want it left literal", got)
	}
}
