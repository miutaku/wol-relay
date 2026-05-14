//go:build !nativegui

package nativegui

import (
	"context"

	"github.com/miutaku/wol-relay/internal/agent"
	"github.com/miutaku/wol-relay/internal/gui"
)

type Options struct {
	Agent       *agent.Agent
	ConfigPath  string
	AgentErrors <-chan error
}

func Run(ctx context.Context, opts Options) error {
	return gui.Server{
		Agent:      opts.Agent,
		ConfigPath: opts.ConfigPath,
		Addr:       "127.0.0.1:0",
		Open:       true,
	}.Run(ctx)
}
