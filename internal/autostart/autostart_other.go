//go:build !darwin && !linux && !windows

package autostart

import "errors"

var errUnsupported = errors.New("このOSではアプリ内からの自動起動登録にまだ対応していません")

func IsEnabled(configPath string) (bool, error) {
	return false, errUnsupported
}

func SetEnabled(enabled bool, configPath string) error {
	if !enabled {
		return nil
	}
	return errUnsupported
}
