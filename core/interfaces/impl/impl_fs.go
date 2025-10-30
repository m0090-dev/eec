package impl
import "os"
import "github.com/m0090-dev/eec-go/core/utils/general"

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
func (OSFS) Open(name string) (*os.File,error) {return os.Open(name)}
func (OSFS) FileBase(path string) string {return general.FileBase(path)}
