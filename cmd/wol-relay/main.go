package main

import (
	"os"

	"github.com/miutaku/wol-relay/internal/app"
)

const version = "0.1.0"

func main() {
	os.Exit(app.Main(os.Args, version))
}
