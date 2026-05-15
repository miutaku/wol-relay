package autostart

import "runtime"

const launchID = "io.github.miutaku.wol-relay"

func IsSupported() bool {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		return true
	default:
		return false
	}
}
