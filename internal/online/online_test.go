package online

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestWaitTCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	portNum, err := net.LookupPort("tcp", port)
	if err != nil {
		t.Fatal(err)
	}

	err = Wait(context.Background(), "127.0.0.1", Check{
		Enabled:  true,
		Method:   "tcp",
		Port:     portNum,
		Timeout:  time.Second,
		Interval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
}
