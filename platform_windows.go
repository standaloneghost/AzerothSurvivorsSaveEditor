//go:build windows

package main

import (
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const gameExe = "AzerothSurvivors.exe"

// gameRunning lists processes through the Windows API instead of launching tasklist.
func gameRunning() bool {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(snap)
	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = syscall.Process32First(snap, &entry); err == nil; err = syscall.Process32Next(snap, &entry) {
		if strings.EqualFold(syscall.UTF16ToString(entry.ExeFile[:]), gameExe) {
			return true
		}
	}
	return false
}

// openBrowser asks Windows to open the default browser instead of launching rundll32.
func openBrowser(url string) {
	verb, _ := syscall.UTF16PtrFromString("open")
	target, _ := syscall.UTF16PtrFromString(url)
	const swShowNormal = 1
	syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW").Call(0,
		uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(target)), 0, 0, swShowNormal)
}

func fatal(msg string) {
	t, _ := syscall.UTF16PtrFromString(msg)
	c, _ := syscall.UTF16PtrFromString("Azeroth Survivors Save Editor")
	syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
	os.Exit(1)
}
