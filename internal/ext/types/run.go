package types

import "time"

// RunOptions contains all inputs that were previously taken from flags / tag file.
type RunOptions struct {
	ConfigFile       string
	InlineConfig     string // 自動判別用 (引数: --inline)
	InlineConfigToml string // TOML明示 (引数: --inline-toml)
	InlineConfigYaml string // YAML明示 (引数: --inline-yaml)
	InlineConfigJson string // JSON明示 (引数: --inline-json)
	Program          string
	ProgramArgs      []string
	Tag              string
	Imports          []string
	// Timeout for waiting program; zero means wait indefinitely
	WaitTimeout     time.Duration
	HideWindow      bool
	Separator       string
	Verbose         bool
	DuplicateStrict bool
	DuplicateNoWarn bool
}
