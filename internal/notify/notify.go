package notify

import (
	"context"
	"time"
)

type Notifier interface {
	Notify(title, message string)
}

type OSNotifier struct {
	Enabled bool
}

func (n OSNotifier) Notify(title, message string) {
	if !n.Enabled {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = notify(ctx, title, message)
	}()
}
