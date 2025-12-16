//go:build darwin

package main

import (
	"os/exec"
)

// 检查进程是否运行 (macOS - 用于开发测试)
func isProcessRunning(processName string) bool {
	cmd := exec.Command("pgrep", "-x", processName)
	err := cmd.Run()
	return err == nil
}
