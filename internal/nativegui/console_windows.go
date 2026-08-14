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
	// Hide it completely so it does not leave a minimized taskbar button.
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		const swHide = 0
		procShowWindow.Call(hwnd, swHide)
	}
}
