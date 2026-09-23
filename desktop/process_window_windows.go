//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

const createNoWindowFlag = 0x08000000

// hideExternalProcessWindow impede que executáveis auxiliares como curl.exe e
// powershell.exe criem uma janela de console visível sobre a interface Wails.
func hideExternalProcessWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindowFlag,
	}
}
