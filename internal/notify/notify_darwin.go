package notify

import (
	"context"
	"os/exec"
	"strings"
)

func notify(ctx context.Context, title, message string) error {
	return exec.CommandContext(ctx, "osascript", "-e", `display notification `+osaQuote(message)+` with title `+osaQuote(title)).Run()
}

func osaQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}
