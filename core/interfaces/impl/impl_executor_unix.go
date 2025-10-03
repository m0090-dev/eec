// executor_unix.go
//go:build linux || darwin
// +build linux darwin
package impl

import (
	"runtime"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	//"syscall"
	"github.com/aymanbagabas/go-pty"
)
// DefaultExecutor uses os/exec
type DefaultExecutor struct{}





func (d DefaultExecutor) StartProcess(path string, args []string, env []string, stdin, stdout, stderr *os.File,hideWindow bool) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows の場合、cmd.exe が存在するか確認
		if _, err := exec.LookPath("cmd.exe"); err == nil {
			cmd = exec.Command("cmd.exe", "/C", path+" "+strings.Join(args, " "))
		}
	case "linux", "darwin":
		// Unix系の場合、sh が存在するか確認
		if _, err := exec.LookPath("sh"); err == nil {
			cmd = exec.Command("sh", "-c", path+" "+strings.Join(args, " "))
		}
	}

	// どちらも存在しない場合はそのまま実行
	if cmd == nil {
		cmd = exec.Command(path, args...)
	}

	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}


/*// Executor runs commands. Default uses os/exec.*/
/*func (d DefaultExecutor) StartProcessWithCmd(*/
    /*path string,*/
    /*args []string,*/
    /*env []string,*/
    /*stdin, stdout, stderr *os.File,*/
    /*hideWindow bool,*/
/*) (int, *os.Process, *exec.Cmd, error) {*/
    /*var cmd *exec.Cmd*/

    /*switch runtime.GOOS {*/
    /*case "linux", "darwin":*/
        /*if hideWindow {*/
            /*// --------------------------*/
            /*// nohup 相当 (バックグラウンド & 出力捨て)*/
            /*// --------------------------*/
            /*cmd = exec.Command(path, args...)*/

            /*// 新しいセッションで開始 → 親が落ちても終了しない*/
            /*cmd.SysProcAttr = &syscall.SysProcAttr{*/
                /*Setsid: true,*/
            /*}*/

            /*// /dev/null に捨てる*/
            /*devNull, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0)*/
            /*cmd.Stdin = devNull*/
            /*cmd.Stdout = devNull*/
            /*cmd.Stderr = devNull*/
        /*} else {*/
            /*// --------------------------*/
            /*// 通常フォアグラウンド実行*/
            /*// --------------------------*/
            /*if _, err := exec.LookPath("sh"); err == nil {*/
                /*cmd = exec.Command("sh", "-c", path+" "+strings.Join(args, " "))*/
            /*} else {*/
                /*cmd = exec.Command(path, args...)*/
            /*}*/

            /*cmd.Stdin = stdin*/
            /*cmd.Stdout = stdout*/
            /*cmd.Stderr = stderr*/
        /*}*/
    /*default:*/
        /*return 0, nil, nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)*/
    /*}*/

    /*cmd.Env = env*/

    /*if err := cmd.Start(); err != nil {*/
        /*return 0, nil, cmd, err*/
    /*}*/
    /*return cmd.Process.Pid, cmd.Process, cmd, nil*/
/*}*/


func (d DefaultExecutor) StartProcessPty(shell string, args []string) (*pty.Cmd, pty.Pty, error) {
    p, err := pty.New()
    if err != nil {
        return nil, nil, err
    }

    cmd := p.Command(shell, args...)
    if err := cmd.Start(); err != nil {
        return nil, nil, err
    }

    return cmd, p, nil
}


func (d DefaultExecutor) RestartProcessPty(p pty.Pty, shell string, args []string) (*pty.Cmd, error) {
    cmd := p.Command(shell, args...)
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    return cmd, nil
}

func (d DefaultExecutor) WaitProcess(proc *os.Process, timeout time.Duration) error {
	// We need the *Cmd to call Wait; but we only have *os.Process here.
	// Simpler: poll process state.
	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()
	if timeout == 0 {
		return <-done
	}
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("wait timeout after %s", timeout)
	}
}
func (d DefaultExecutor) Getpid() int{return os.Getpid()}

func (d DefaultExecutor) FindProcess(pid int) (*os.Process,error) {return os.FindProcess(pid)}
