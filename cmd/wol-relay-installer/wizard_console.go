//go:build !nativegui

package main

import (
	"fmt"
	"os"
)

func runWizard() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		waitForEnter()
		os.Exit(1)
	}
	fmt.Println("wol-relay was installed successfully.")
	waitForEnter()
}
