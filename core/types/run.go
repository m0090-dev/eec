package types
import "time"

// RunOptions contains all inputs that were previously taken from flags / tag file.
type RunOptions struct {
	ConfigFile  string
	Program     string
	ProgramArgs []string
	Tag         string
	Imports     []string
	// Timeout for waiting program; zero means wait indefinitely
	WaitTimeout       time.Duration
	HideWindow        bool
	DeleterPath       string
	DeleterHideWindow bool
	Pty		  bool
}


