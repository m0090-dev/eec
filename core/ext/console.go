package ext
import "os"

type Console interface {
    Stdin() *os.File
    Stdout() *os.File
    Stderr() *os.File
}

type DefaultConsole struct{}

func (d DefaultConsole) Stdin() *os.File{return os.Stdin}
func (d DefaultConsole) Stdout() *os.File{return os.Stdout}
func (d DefaultConsole) Stderr() *os.File{return os.Stderr}



/*package ext*/

/*import "os"*/

/*type Console interface {*/
    /*Stdin() *os.File*/
    /*Stdout() *os.File*/
    /*Stderr() *os.File*/

    /*SetStdin(f *os.File)*/
    /*SetStdout(f *os.File)*/
    /*SetStderr(f *os.File)*/
/*}*/

/*type DefaultConsole struct{*/
    /*stdin  *os.File*/
    /*stdout *os.File*/
    /*stderr *os.File*/
/*}*/

/*// コンストラクタ的に初期値を設定*/
/*func NewDefaultConsole() *DefaultConsole {*/
    /*return &DefaultConsole{*/
        /*stdin:  os.Stdin,*/
        /*stdout: os.Stdout,*/
        /*stderr: os.Stderr,*/
    /*}*/
/*}*/

/*// Getter*/
/*func (d *DefaultConsole) Stdin() *os.File  { return d.stdin }*/
/*func (d *DefaultConsole) Stdout() *os.File { return d.stdout }*/
/*func (d *DefaultConsole) Stderr() *os.File { return d.stderr }*/

/*// Setter*/
/*func (d *DefaultConsole) SetStdin(f *os.File)  { d.stdin = f }*/
/*func (d *DefaultConsole) SetStdout(f *os.File) { d.stdout = f }*/
/*func (d *DefaultConsole) SetStderr(f *os.File) { d.stderr = f }*/
