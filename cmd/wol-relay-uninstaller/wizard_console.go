//go:build !nativegui

package main

import (
	"fmt"
	"os"
)

func runWizard() {
	if err := uninstall(false); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("wol-relay was uninstalled successfully.")
}
