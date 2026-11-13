package domain

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
)

// DoctorStatus is the outcome class of a single doctor check.
type DoctorStatus int

const (
	DoctorOK DoctorStatus = iota
	DoctorWarn
	DoctorFail
)

// Symbol returns the checklist glyph for the status.
func (s DoctorStatus) Symbol() string {
	switch s {
	case DoctorOK:
		return "✓"
	case DoctorWarn:
		return "⚠"
	default:
		return "✗"
	}
}

// DoctorCheck is one line of the doctor report.
type DoctorCheck struct {
	Status DoctorStatus
	Label  string
	Detail string
}

// DoctorFailed reports whether any check is a hard failure. Warnings
// never fail a doctor run.
func DoctorFailed(checks []DoctorCheck) bool {
	for _, c := range checks {
		if c.Status == DoctorFail {
			return true
		}
	}
	return false
}

// RunDoctor performs the resolution sanity checks behind `eec
// doctor`:
//
//   - the configs referenced by opts/tagData resolve without error;
//   - every absolute PATH entry of the resolved environment names an
//     existing directory (missing entries are warnings, not
//     failures: the host PATH routinely contains stale entries);
//   - when program is non-empty, it is found in the resolved PATH
//     (a missing program is a failure — the run would fail too);
//   - when a tag is used, each of its imports resolves to an existing
//     file or to another tag (a missing import is a failure).
func RunDoctor(rt interfaces.Runtime, opts types.RunOptions, tagData types.TagData, program string) []DoctorCheck {
	goos := rt.Env().GOOS()

	_, _, _, finalEnv, allConfigs, err := ResolveRunOptions(opts, tagData, rt)
	if err != nil {
		return []DoctorCheck{{
			Status: DoctorFail,
			Label:  "configs resolve",
			Detail: err.Error(),
		}}
	}
	checks := []DoctorCheck{{
		Status: DoctorOK,
		Label:  "configs resolve",
		Detail: fmt.Sprintf("%d config(s), %d variables", len(allConfigs), len(finalEnv)),
	}}

	// PATH entries: only absolute entries can be checked portably.
	pathVal, _ := GetEnvValue(finalEnv, goos, "PATH")
	checked := 0
	var missing []string
	for _, dir := range strings.Split(pathVal, rt.Env().PathListSeparator()) {
		if dir == "" || !filepath.IsAbs(dir) {
			continue
		}
		checked++
		if info, statErr := rt.FS().Stat(dir); statErr != nil || info == nil || !info.IsDir() {
			missing = append(missing, dir)
		}
	}
	if len(missing) == 0 {
		checks = append(checks, DoctorCheck{
			Status: DoctorOK,
			Label:  "PATH entries exist",
			Detail: fmt.Sprintf("%d absolute entries checked", checked),
		})
	} else {
		for _, dir := range missing {
			checks = append(checks, DoctorCheck{
				Status: DoctorWarn,
				Label:  "PATH entry does not exist",
				Detail: dir,
			})
		}
	}

	if program != "" {
		if found, ok := FindInPath(rt, finalEnv, program); ok {
			checks = append(checks, DoctorCheck{
				Status: DoctorOK,
				Label:  "program found: " + program,
				Detail: found,
			})
		} else {
			checks = append(checks, DoctorCheck{
				Status: DoctorFail,
				Label:  "program not found: " + program,
				Detail: "not in the resolved PATH",
			})
		}
	}

	if opts.Tag != "" {
		for _, imp := range tagData.ImportConfigFiles {
			resolved := ResolveDataDirRelative(rt, imp)
			switch {
			case rt.FS().FileExists(resolved):
				checks = append(checks, DoctorCheck{
					Status: DoctorOK,
					Label:  "tag import resolves",
					Detail: imp,
				})
			default:
				if _, tagErr := types.ReadTagData(rt, imp); tagErr == nil {
					checks = append(checks, DoctorCheck{
						Status: DoctorOK,
						Label:  "tag import resolves (tag)",
						Detail: imp,
					})
				} else {
					checks = append(checks, DoctorCheck{
						Status: DoctorFail,
						Label:  "tag import does not resolve",
						Detail: imp,
					})
				}
			}
		}
	}

	return checks
}
