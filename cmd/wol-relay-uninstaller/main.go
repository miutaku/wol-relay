package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	runWizard()
}

func uninstall(removeConfig bool) error {
	if runtime.GOOS != "windows" {
		return errors.New("this uninstaller is for Windows")
	}

	installDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "wol-relay")
	desktop, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	paths := []string{
		filepath.Join(desktop, "Desktop", "wol-relay.exe"),
		filepath.Join(installDir, "wol-relay.exe"),
	}
	for _, path := range paths {
		if err := removeIfExists(path); err != nil {
			return err
		}
	}

	if removeConfig {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		if err := removeIfExists(filepath.Join(configDir, "wol-relay")); err != nil {
			return err
		}
	}

	if err := os.Remove(installDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove install directory: %w", err)
	}
	return nil
}

func removeIfExists(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}
