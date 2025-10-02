package impl
import "os"
type DefaultCommandLine struct {
}

func (d DefaultCommandLine) Args() []string { return os.Args }

