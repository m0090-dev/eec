package impl
import "os"
type DefaultConsole struct{}

func (d DefaultConsole) Stdin() *os.File{return os.Stdin}
func (d DefaultConsole) Stdout() *os.File{return os.Stdout}
func (d DefaultConsole) Stderr() *os.File{return os.Stderr}


