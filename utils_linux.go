//go:build linux
// +build linux

package main

import (
	"os/exec"
)

// 检查进程是否运行 (Linux)
func isProcessRunning(processName string) bool {
	cmd := exec.Command("pgrep", "-x", processName)
	err := cmd.Run()
	return err == nil
}
