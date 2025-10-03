//go:build windows
// +build windows

// executor_windows.go
package impl

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	//"strings"
	"syscall"
	"time"
	"github.com/aymanbagabas/go-pty"
)

// DefaultExecutor uses os/exec
type DefaultExecutor struct{}


func (d DefaultExecutor) StartProcess(path string, args []string, env []string, stdin, stdout, stderr *os.File, hideWindow bool) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		if _, err := exec.LookPath("cmd.exe"); err == nil {
			cmdArgs := append([]string{"/C", path}, args...)
			cmd = exec.Command("cmd.exe", cmdArgs...)
			if hideWindow {
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			} else {
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false, CreationFlags: 0x00000010}
			}
		} else {
			return nil, fmt.Errorf("cmd.exe not found in PATH")
		}
	case "linux", "darwin":
		if _, err := exec.LookPath("sh"); err == nil {
			cmdArgs := append([]string{"-c", path}, args...)
			cmd = exec.Command("sh", cmdArgs...)
		} else {
			return nil, fmt.Errorf("sh not found in PATH")
		}
	default:
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
    /*case "windows":*/
        /*// Windows の場合、cmd.exe が存在するか確認*/
        /*if _, err := exec.LookPath("cmd.exe"); err == nil {*/
            /*cmd = exec.Command("cmd.exe", "/C", path+" "+strings.Join(args, " "))*/
            /*//cmd = exec.Command(path,args...)*/
           /*}*/
        /*if hideWindow {*/
            /*cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}*/
        /*}else{*/
            /*cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false,*/
	    /*CreationFlags:  0x00000010}*/
	/*}*/
    /*case "linux", "darwin":*/
        /*// Unix系の場合、sh が存在するか確認*/
        /*if _, err := exec.LookPath("sh"); err == nil {*/
            /*cmd = exec.Command("sh", "-c", path+" "+strings.Join(args, " "))*/
        /*}*/
    /*}*/

    /*// どちらも存在しない場合はそのまま実行*/
    /*if cmd == nil {*/
        /*cmd = exec.Command(path, args...)*/
    /*}*/

    /*cmd.Env = env*/
    /*cmd.Stdin = stdin*/
    /*cmd.Stdout = stdout*/
    /*cmd.Stderr = stderr*/

    /*if err := cmd.Start(); err != nil {*/
        /*return 0, nil, cmd, err*/
    /*}*/
    /*return cmd.Process.Pid, cmd.Process, cmd, nil*/
/*}*/

/*func (d DefaultExecutor) StartProcessPty(p *pty.Pty,cmd *pty.Cmd,shell string, args []string) (int,*os.Process,*pty.Cmd,error) {*/
    /*newPty, err := pty.New()*/
    /*if err != nil {*/
        /*return 0, nil, nil, err*/
    /*}*/
    /*p = &newPty*/
    /*newCmd := newPty.Command(shell, args...)*/
    /*if err := newCmd.Start(); err != nil {       */
        /*return 0, nil, cmd, err*/
    /*}*/
    /*cmd = newCmd*/
    /*return cmd.Process.Pid,cmd.Process,cmd,nil*/
/*}*/

/*
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
*/

func (d DefaultExecutor) StartProcessPty(shell string, args []string, env []string) (*pty.Cmd, pty.Pty, error) {
    // 新しい PTY 作成
    p, err := pty.New()
    if err != nil {
        return nil, nil, err
    }

    // コマンド生成
    cmd := p.Command(shell, args...)

    // 環境変数セット
    if env != nil {
        cmd.Env = env
    }

    // プロセス起動
    if err := cmd.Start(); err != nil {
        return nil, nil, err
    }

    return cmd, p, nil
}

func (d DefaultExecutor) RestartProcessPty(p pty.Pty, shell string, args []string, env []string) (*pty.Cmd, error) {
    cmd := p.Command(shell, args...)
    cmd.Env = env // 環境変数をセット
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
