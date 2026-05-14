package online

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

type Check struct {
	Enabled  bool
	Method   string
	Port     int
	Timeout  time.Duration
	Interval time.Duration
}

func Wait(ctx context.Context, ip string, check Check) error {
	if !check.Enabled {
		return nil
	}
	if ip == "" {
		return errors.New("online check requires host ip")
	}
	if check.Method == "" {
		check.Method = "tcp"
	}
	if check.Timeout <= 0 {
		check.Timeout = 2 * time.Minute
	}
	if check.Interval <= 0 {
		check.Interval = 3 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	var lastErr error
	for {
		switch check.Method {
		case "tcp":
			lastErr = probeTCP(ctx, ip, check.Port)
		case "icmp":
			lastErr = probeICMP(ctx, ip)
		default:
			return fmt.Errorf("unsupported online check method %q", check.Method)
		}
		if lastErr == nil {
			return nil
		}

		timer := time.NewTimer(check.Interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("online check timed out: %w", lastErr)
		case <-timer.C:
		}
	}
}

func probeTCP(ctx context.Context, ip string, port int) error {
	if port <= 0 {
		port = 22
	}
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	return conn.Close()
}

func probeICMP(ctx context.Context, ip string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.CommandContext(ctx, "ping", "-n", "1", "-w", "1000", ip).Run()
	default:
		return exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ip).Run()
	}
}
