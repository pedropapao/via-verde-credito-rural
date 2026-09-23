//go:build !windows

package main

import "os/exec"

func hideExternalProcessWindow(cmd *exec.Cmd) {
	// No-op fora do Windows. Mantém testes e builds multiplataforma.
}
