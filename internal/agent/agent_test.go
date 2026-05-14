package agent

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/miutaku/wol-relay/internal/config"
	"github.com/miutaku/wol-relay/internal/wol"
)

type recordingNotifier struct {
	titles []string
}

func (n *recordingNotifier) Notify(title, _ string) {
	n.titles = append(n.titles, title)
}

type channelNotifier struct {
	ch chan string
}

func (n channelNotifier) Notify(title, _ string) {
	n.ch <- title
}

func TestHandleDetectedMagicIgnoresUnknownTarget(t *testing.T) {
	n := &recordingNotifier{}
	a := New(config.Config{
		NodeName:      "test",
		ListenHTTP:    "127.0.0.1:0",
		ListenMagic:   []string{"127.0.0.1:0"},
		DefaultTarget: "255.255.255.255:9",
		Auth:          config.AuthConfig{SharedSecret: "secret"},
	})
	a.notifier = n

	hw, err := net.ParseMAC("00:11:22:33:44:55")
	if err != nil {
		t.Fatal(err)
	}
	a.handleDetectedMagic(context.Background(), &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9}, hw)
	if len(n.titles) != 0 {
		t.Fatalf("unexpected notification for unknown target: %v", n.titles)
	}
}

func TestHandleDetectedMagicDoesNotResendLocalPacket(t *testing.T) {
	n := &recordingNotifier{}
	a := New(config.Config{
		NodeName:      "test",
		ListenHTTP:    "127.0.0.1:0",
		ListenMagic:   []string{"127.0.0.1:0"},
		DefaultTarget: "255.255.255.255:9",
		Auth:          config.AuthConfig{SharedSecret: "secret"},
		Hosts: []config.HostConfig{
			{
				Name: "local",
				MAC:  "00:11:22:33:44:55",
				IP:   "127.0.0.1",
			},
		},
	})
	a.notifier = n

	hw, err := net.ParseMAC("00:11:22:33:44:55")
	if err != nil {
		t.Fatal(err)
	}
	a.handleDetectedMagic(context.Background(), &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9}, hw)
	if len(n.titles) != 1 || n.titles[0] != "Wake on LAN detected" {
		t.Fatalf("unexpected notifications: %v", n.titles)
	}
}

func TestListenMagicDetectsUDPMagicPacket(t *testing.T) {
	portConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	listenAddr := portConn.LocalAddr().String()
	if err := portConn.Close(); err != nil {
		t.Fatal(err)
	}

	notifier := channelNotifier{ch: make(chan string, 1)}
	a := New(config.Config{
		NodeName:      "test",
		ListenHTTP:    "127.0.0.1:0",
		ListenMagic:   []string{listenAddr},
		DefaultTarget: "255.255.255.255:9",
		Auth:          config.AuthConfig{SharedSecret: "secret"},
		Hosts: []config.HostConfig{
			{
				Name: "local",
				MAC:  "00:11:22:33:44:55",
				IP:   "127.0.0.1",
			},
		},
	})
	a.notifier = notifier

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.listenMagic(ctx, listenAddr)
	}()
	time.Sleep(50 * time.Millisecond)

	hw, err := net.ParseMAC("00:11:22:33:44:55")
	if err != nil {
		t.Fatal(err)
	}
	packet, err := wol.BuildMagicPacket(hw)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		t.Fatal(err)
	}
	var sent bool
	for i := 0; i < 20; i++ {
		conn, err := net.DialUDP("udp", nil, addr)
		if err == nil {
			_, err = conn.Write(packet)
			_ = conn.Close()
		}
		if err == nil {
			sent = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !sent {
		t.Fatal("failed to send magic packet to listener")
	}

	select {
	case title := <-notifier.ch:
		if title != "Wake on LAN detected" {
			t.Fatalf("got notification %q, want Wake on LAN detected", title)
		}
	case err := <-errCh:
		t.Fatalf("listener exited early: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not detect magic packet")
	}
}
