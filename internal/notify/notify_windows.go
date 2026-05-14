package notify

import (
	"context"

	"github.com/go-toast/toast"
)

func notify(_ context.Context, title, message string) error {
	notification := toast.Notification{
		AppID:    "wol-relay",
		Title:    title,
		Message:  message,
		Audio:    toast.Default,
		Duration: toast.Short,
	}
	return notification.Push()
}
