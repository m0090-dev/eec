package interfaces
import "os"
type Console interface {
    Stdin() *os.File
    Stdout() *os.File
    Stderr() *os.File
}


