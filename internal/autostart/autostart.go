package autostart

import "runtime"

const launchID = "io.github.miutaku.wol-relay"

func IsSupported() bool {
	return runtime.GOOS == "darwin"
}
