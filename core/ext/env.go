package ext
import (
	"os"
	_ "github.com/m0090-dev/eec-go/core/utils/general"
)

// Env is minimal env-system abstraction.
type Env interface {
	Environ() []string
	Unsetenv(key string) error
	Setenv(key string, value string) error
	UserHomeDir() (string,error)
}


// OSFS is a thin wrapper over os calls.
type OSEnv struct{}

func (OSEnv) Environ() []string {return os.Environ()}
func (OSEnv) Unsetenv(key string) error{return os.Unsetenv(key)}
func (OSEnv) Setenv(key string ,value string) error{return os.Setenv(key,value)}
func (OSEnv) UserHomeDir() (string,error) {return os.UserHomeDir()}


