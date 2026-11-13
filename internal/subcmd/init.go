package subcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initForceFlag bool

// starterConfig is the template written by `eec init`. Everything
// functional is commented out except a minimal [envs] example, so
// the file is valid as written and grows by uncommenting.
const starterConfig = `# eec configuration file.
#
# Run a program with this environment:
#   eec run -c eec.toml -p <program>
# Or register it as a tag:
#   eec tag add dev -c eec.toml

[configs]
description = "My environment"

[envs]
# Plain variables. Values may reference other variables with
# ${VAR} (resolved in dependency order, regardless of the order
# they are written in) and embed command output with $(...).
GREETING = "hello"

# A list value is joined with the OS path separator. For PATH-like
# variables the entries are combined with the inherited value.
# PATH = ["./bin", "./tools"]

# OS-specific sections apply only on that OS and override [envs].
[envs.windows]
# TOOLS_DIR = "C:\\tools"

[envs.linux]
# TOOLS_DIR = "/opt/tools"

[envs.darwin]
# TOOLS_DIR = "/opt/homebrew"

# Capture: run generic setup commands (a batch file, a shell
# script, ...) before the program starts and merge the environment
# they leave behind — added/changed variables are taken over,
# removed ones disappear. On Unix the command must affect the
# shell itself, so source your script: . "${CONFIG_DIR}/setup.sh"
# [capture]
# commands = ["setup-env.bat"]

# Default program for ` + "`eec run`" + ` without -p.
# [program]
# path = "cmd"
# args = ["/k"]
`

// writeStarterConfig writes the starter config to path. It refuses
// to overwrite an existing file unless force is set.
func writeStarterConfig(path string, force bool) error {
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists (use --force to overwrite)", path)
		}
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(starterConfig); err != nil {
		return err
	}
	return nil
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Write a starter config file (default: eec.toml)",
	Long: `Writes a commented starter TOML configuration with an [envs]
section, OS-specific section examples and a commented-out
[capture] example. Refuses to overwrite an existing file unless
--force is given.

Examples:
  eec init
  eec init dev.toml
  eec init --force`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "eec.toml"
		if len(args) == 1 {
			path = args[0]
		}
		if err := writeStarterConfig(path, initForceFlag); err != nil {
			return err
		}
		fmt.Printf("Wrote %s\n", path)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForceFlag, "force", false, "Overwrite the file if it already exists")
	rootCmd.AddCommand(initCmd)
}
