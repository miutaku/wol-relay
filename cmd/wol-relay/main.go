package main

import (
	"os"

	"github.com/miutaku/wol-relay/internal/app"
)

const version = "0.0.15"

func main() {
	os.Exit(app.Main(os.Args, version))
}
