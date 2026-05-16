//go:build nativegui && windows

package nativegui

import "syscall"

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procShowWindow       = user32.NewProc("ShowWindow")
)

func init() {
	// Dev builds without -H=windowsgui allocate a console window.
	// Minimize it so the tray app starts without a visible console.
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		const swShowMinNoActive = 7
		procShowWindow.Call(hwnd, swShowMinNoActive)
	}
}
