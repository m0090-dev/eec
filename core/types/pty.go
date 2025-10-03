package types

import "github.com/aymanbagabas/go-pty"

type PtyData struct{
	P pty.Pty
	Cmd *pty.Cmd
	Env []string
}
