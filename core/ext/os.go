package ext
type OS struct {
    Executor Executor
    FS       FS
    Env      Env
    CommandLine CommandLine
    Console Console
}

func NewOS() OS {
    return OS{
        Executor:    DefaultExecutor{},
        FS:          OSFS{},
        Env:         OSEnv{},
        CommandLine: DefaultCommandLine{},
        Console:     DefaultConsole{},
    }
}
