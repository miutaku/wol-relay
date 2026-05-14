package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/miutaku/wol-relay/internal/agent"
	"github.com/miutaku/wol-relay/internal/config"
	"github.com/miutaku/wol-relay/internal/firewall"
	"github.com/miutaku/wol-relay/internal/nativegui"
	"github.com/miutaku/wol-relay/internal/wol"
)

func Main(args []string, version string) int {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("wol-relay: ")

	if err := Run(args, version); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func Run(args []string, version string) error {
	if len(args) < 2 {
		return runGUI("")
	}

	switch args[1] {
	case "agent":
		fs := flag.NewFlagSet("agent", flag.ExitOnError)
		configPath := fs.String("config", "wol-relay.json", "path to config file")
		light := fs.Bool("light", false, "run headless without GUI or desktop notifications")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		if *light {
			cfg.Lightweight = true
			cfg.Notifications.Enabled = false
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		return agent.New(cfg).Run(ctx)
	case "gui":
		fs := flag.NewFlagSet("gui", flag.ExitOnError)
		defaultPath, err := config.DefaultPath()
		if err != nil {
			return err
		}
		configPath := fs.String("config", defaultPath, "path to config file")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		return runGUIWithOptions(*configPath)
	case "wake":
		fs := flag.NewFlagSet("wake", flag.ExitOnError)
		configPath := fs.String("config", "wol-relay.json", "path to config file")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return errors.New("usage: wol-relay wake [-config wol-relay.json] <host-name|mac-address>")
		}
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		res, err := agent.New(cfg).Wake(context.Background(), fs.Arg(0), agent.SourceCLI)
		if err != nil {
			return err
		}
		fmt.Println(res.Message)
		return nil
	case "send":
		fs := flag.NewFlagSet("send", flag.ExitOnError)
		mac := fs.String("mac", "", "target MAC address")
		broadcast := fs.String("broadcast", "255.255.255.255:9", "broadcast address")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *mac == "" {
			return errors.New("-mac is required")
		}
		hw, err := wol.ParseMAC(*mac)
		if err != nil {
			return err
		}
		if err := wol.SendMagicPacket(hw, *broadcast); err != nil {
			return err
		}
		fmt.Printf("sent magic packet to %s via %s\n", hw, *broadcast)
		return nil
	case "init":
		fs := flag.NewFlagSet("init", flag.ExitOnError)
		configPath := fs.String("config", "wol-relay.json", "path to write")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		return config.WriteExample(*configPath)
	case "firewall":
		fs := flag.NewFlagSet("firewall", flag.ExitOnError)
		configPath := fs.String("config", "wol-relay.json", "path to config file")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		plan, err := firewall.Plan(cfg)
		if err != nil {
			return err
		}
		fmt.Print(plan)
		return nil
	case "version":
		fmt.Println(version)
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[1])
	}
}

func runGUI(configPath string) error {
	if configPath == "" {
		defaultPath, err := config.DefaultPath()
		if err != nil {
			return err
		}
		configPath = defaultPath
	}
	return runGUIWithOptions(configPath)
}

func runGUIWithOptions(configPath string) error {
	cfg, err := config.LoadOrCreate(configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.Lightweight {
		return agent.New(cfg).Run(ctx)
	}

	app := agent.New(cfg)
	agentErrCh := make(chan error, 1)
	go func() {
		if err := app.Run(ctx); err != nil {
			agentErrCh <- err
		}
	}()
	err = nativegui.Run(ctx, nativegui.Options{Agent: app, ConfigPath: configPath, AgentErrors: agentErrCh})
	stop()
	return err
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage:
  wol-relay init  [-config wol-relay.json]
  wol-relay agent [-config wol-relay.json] [-light]
  wol-relay gui   [-config wol-relay.json]
  wol-relay wake  [-config wol-relay.json] <host-name|mac-address>
  wol-relay send  -mac <mac-address> [-broadcast 255.255.255.255:9]
  wol-relay firewall [-config wol-relay.json]
  wol-relay version`)
}
