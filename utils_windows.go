//go:build windows
// +build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// 检查进程是否运行 (Windows)
func isProcessRunning(processName string) bool {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snapshot)

	var procEntry windows.ProcessEntry32
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))

	if err := windows.Process32First(snapshot, &procEntry); err != nil {
		return false
	}

	for {
		exeName := windows.UTF16ToString(procEntry.ExeFile[:])
		if exeName == processName {
			return true
		}

		if err := windows.Process32Next(snapshot, &procEntry); err != nil {
			break
		}
	}

	return false
}
