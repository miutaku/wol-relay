//go:build linux

package autostart

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func IsEnabled(configPath string) (bool, error) {
	path, err := desktopEntryPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func SetEnabled(enabled bool, configPath string) error {
	if enabled {
		return enable(configPath)
	}
	return disable()
}

func enable(configPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}
	path, err := desktopEntryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	args := []string{exe, "gui"}
	if configPath != "" {
		args = append(args, "-config", configPath)
	}
	return os.WriteFile(path, []byte(desktopEntry(args)), 0o600)
}

func disable() error {
	path, err := desktopEntryPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

func desktopEntryPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", "wol-relay.desktop"), nil
}

func desktopEntry(args []string) string {
	var b bytes.Buffer
	b.WriteString("[Desktop Entry]\n")
	b.WriteString("Type=Application\n")
	b.WriteString("Name=wol-relay Wake on LAN\n")
	b.WriteString("Comment=Wake on LAN relay agent and GUI\n")
	b.WriteString("Exec=" + desktopExec(args) + "\n")
	b.WriteString("Terminal=false\n")
	b.WriteString("X-GNOME-Autostart-enabled=true\n")
	return b.String()
}

func desktopExec(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, strconv.Quote(strings.ReplaceAll(arg, "\n", "")))
	}
	return strings.Join(quoted, " ")
}
