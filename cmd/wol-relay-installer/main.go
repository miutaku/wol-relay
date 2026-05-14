package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/miutaku/wol-relay/internal/app"
	"github.com/miutaku/wol-relay/internal/config"
)

const version = "0.1.0"

func main() {
	if launchedAsApp() || len(os.Args) > 1 {
		os.Exit(app.Main(os.Args, version))
	}
	runWizard()
}

func run() error {
	if runtime.GOOS != "windows" {
		return errors.New("this installer is for Windows")
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	installDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "wol-relay")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}
	installedApp := filepath.Join(installDir, "wol-relay.exe")
	if err := copyFile(exe, installedApp); err != nil {
		return err
	}

	configPath, err := config.DefaultPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		if _, err := config.LoadOrCreate(configPath); err != nil {
			return err
		}
	}

	if err := writeLauncher(installedApp); err != nil {
		return err
	}

	fmt.Println("Application:", installedApp)
	fmt.Println("Config:", configPath)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func writeLauncher(appPath string) error {
	desktop, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(desktop, "Desktop", "wol-relay.exe")
	return copyFile(appPath, path)
}

func waitForEnter() {
	fmt.Println("Press Enter to close this window.")
	_, _ = fmt.Scanln()
}

func launchedAsApp() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.EqualFold(filepath.Base(exe), "wol-relay.exe")
}
