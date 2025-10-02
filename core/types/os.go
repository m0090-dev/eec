package types
import "github.com/m0090-dev/eec-go/core/interfaces"
import "github.com/m0090-dev/eec-go/core/interfaces/impl"
type OS struct {
    Executor interfaces.Executor
    FS       interfaces.FS
    Env      interfaces.Env
    CommandLine interfaces.CommandLine
    Console interfaces.Console
}

func NewOS() OS {
    return OS{
        Executor:    impl.DefaultExecutor{},
        FS:          impl.OSFS{},
        Env:         impl.OSEnv{},
        CommandLine: impl.DefaultCommandLine{},
        Console:     impl.DefaultConsole{},
    }
}
