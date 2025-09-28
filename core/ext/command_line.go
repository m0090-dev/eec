package ext

import "os"

type CommandLine interface {
	Args() []string
}
type DefaultCommandLine struct {
}

func (d DefaultCommandLine) Args() []string { return os.Args }

