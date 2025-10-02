package interfaces
import "os"
// FS is minimal file-system abstraction.
type FS interface {
	Create(name string) (*os.File, error)
	TempDir() string
	FileExists(name string) bool
	MkdirAll(path string,perm uint32) error
	WriteFile(name string,data[] byte,perm uint32) error
	ReadFile(name string) ([]byte,error)
	Remove(name string)error
	FileExt(path string) string
}


