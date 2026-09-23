package main

import (
	"context"
	"os/exec"
	"testing"
)

func TestHideExternalProcessWindowDoesNotReplaceCommand182(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "go", "version")
	hideExternalProcessWindow(cmd)
	if cmd.Path == "" {
		t.Fatal("comando foi alterado indevidamente")
	}
}
