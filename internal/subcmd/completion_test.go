package subcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Cobra's built-in completion command must be reachable: the root
// command does not disable it.
func TestCompletionCommandGeneratesScript(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "bash"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("eec completion bash: %v", err)
	}
	if !strings.Contains(buf.String(), "bash completion") || !strings.Contains(buf.String(), "eec") {
		t.Errorf("unexpected completion script output:\n%.200s", buf.String())
	}
}

func TestCompleteTagNames(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("EEC_HOME", dir)
	for _, name := range []string{"dev.tag", "prod.tag", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	names, directive := completeTagNames(nil, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"dev", "prod"}) {
		t.Errorf("names = %v, want [dev prod]", names)
	}

	names, _ = completeTagNames(nil, nil, "d")
	if !slices.Equal(names, []string{"dev"}) {
		t.Errorf("prefix-filtered names = %v, want [dev]", names)
	}
}

func TestCompleteConfigFileUsesDefault(t *testing.T) {
	_, directive := completeConfigFile(nil, nil, "")
	if directive != cobra.ShellCompDirectiveDefault {
		t.Errorf("directive = %v, want Default (file completion)", directive)
	}
}
