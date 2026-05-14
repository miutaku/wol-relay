package notify

import (
	"context"
	"os/exec"
)

func notify(ctx context.Context, title, message string) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return err
	}
	return exec.CommandContext(ctx, "notify-send", title, message).Run()
}
