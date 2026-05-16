//go:build nativegui && windows

package nativegui

import "golang.org/x/sys/windows"

func init() {
	// Dev builds without -H=windowsgui allocate a console window.
	// Minimize it so the tray app starts without a visible console.
	if hwnd := windows.GetConsoleWindow(); hwnd != 0 {
		windows.ShowWindow(hwnd, windows.SW_SHOWMINNOACTIVE)
	}
}
