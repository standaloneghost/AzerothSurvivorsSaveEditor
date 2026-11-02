//go:build windows

package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

const createNoWindow = 0x08000000

func hidden(c *exec.Cmd) *exec.Cmd {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return c
}

func gameRunning() bool {
	out, err := hidden(exec.Command("tasklist", "/FI", "IMAGENAME eq AzerothSurvivors.exe", "/NH", "/FO", "CSV")).Output()
	return err == nil && strings.Contains(strings.ToLower(string(out)), "azerothsurvivors.exe")
}

func openBrowser(url string) {
	hidden(exec.Command("rundll32", "url.dll,FileProtocolHandler", url)).Start()
}

func fatal(msg string) {
	t, _ := syscall.UTF16PtrFromString(msg)
	c, _ := syscall.UTF16PtrFromString("Azeroth Survivors Save Editor")
	syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
	os.Exit(1)
}
