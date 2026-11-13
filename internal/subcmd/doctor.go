package subcmd

import (
	"errors"
	"fmt"

	"github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
	"github.com/spf13/cobra"
)

var doctorFlags resolutionFlags
var doctorProgramFlag string

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check that a config or tag resolves to a sane environment",
	Long: `Runs a checklist against the environment resolved from -c/-t/-i:

  ✓/✗  the configs resolve without error
  ✓/⚠  every absolute PATH entry exists (missing entries are
       warnings — the host PATH routinely contains stale entries)
  ✓/✗  the program given with -p is found in the resolved PATH
  ✓/✗  every import of the tag given with -t resolves to a file
       or another tag

The exit status is non-zero only when a ✗ check fails; warnings do
not fail the run.

Examples:
  eec doctor -t dev
  eec doctor -c test.toml -p ninja`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		e := core.NewEngine(nil)
		checks, err := e.Doctor(doctorFlags.runOptions(), doctorProgramFlag)
		if err != nil {
			return err
		}
		for _, c := range checks {
			if c.Detail != "" {
				fmt.Printf("%s %s — %s\n", c.Status.Symbol(), c.Label, c.Detail)
			} else {
				fmt.Printf("%s %s\n", c.Status.Symbol(), c.Label)
			}
		}
		if domain.DoctorFailed(checks) {
			return errors.New("doctor found failing checks")
		}
		return nil
	},
}

func init() {
	addResolutionFlags(doctorCmd, &doctorFlags)
	registerResolutionCompletion(doctorCmd)
	doctorCmd.Flags().StringVarP(&doctorProgramFlag, "program", "p", "", "Program to look up in the resolved PATH")
	rootCmd.AddCommand(doctorCmd)
}
