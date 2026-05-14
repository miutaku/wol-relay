//go:build !darwin && !linux && !windows

package notify

import (
	"context"
	"errors"
)

func notify(_ context.Context, _, _ string) error {
	return errors.New("desktop notification is unsupported on this OS")
}
