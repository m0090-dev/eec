package ext
import "os"
import "github.com/m0090-dev/eec-go/core/utils/general"
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

// OSFS is a thin wrapper over os calls.
type OSFS struct{}

func (OSFS) Create(name string) (*os.File, error) { return os.Create(name) }
func (OSFS) TempDir() string                      { return os.TempDir() }
func (OSFS) FileExists(name string) bool          { return general.FileExists(name) }
func (OSFS) MkdirAll(path string,perm uint32) error {return os.MkdirAll(path,os.FileMode(perm))}
func (OSFS) WriteFile(name string,data[] byte,perm uint32)error{return os.WriteFile(name,data,os.FileMode(perm))}
func (OSFS) ReadFile(name string) ([]byte,error) {return os.ReadFile(name)}
func (OSFS) Remove(name string) error {return os.Remove(name)}
func (OSFS) FileExt(path string) string {return general.FileExt(path)}
