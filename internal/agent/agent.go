package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/miutaku/wol-relay/internal/config"
	"github.com/miutaku/wol-relay/internal/netutil"
	"github.com/miutaku/wol-relay/internal/notify"
	"github.com/miutaku/wol-relay/internal/online"
	"github.com/miutaku/wol-relay/internal/wol"
)

const (
	SourceCLI   = "cli"
	SourceHTTP  = "http"
	SourceMagic = "magic"
)

type Agent struct {
	cfg      config.Config
	client   *http.Client
	notifier notify.Notifier
	mu       sync.RWMutex

	burstMu           sync.Mutex
	burstMap          map[string]*magicBurst
	burstDebounceWindow time.Duration
}

type WakeResult struct {
	Message string `json:"message"`
	Relayed bool   `json:"relayed"`
	Target  string `json:"target"`
	Online  bool   `json:"online"`
}

type wakeRequest struct {
	Host      string `json:"host,omitempty"`
	MAC       string `json:"mac"`
	IP        string `json:"ip,omitempty"`
	Broadcast string `json:"broadcast,omitempty"`
}

func New(cfg config.Config) *Agent {
	return &Agent{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		notifier:            notify.OSNotifier{Enabled: cfg.Notifications.Enabled && !cfg.Lightweight},
		burstDebounceWindow: 30 * time.Second,
	}
}

func (a *Agent) Config() config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

func (a *Agent) UpdateConfig(cfg config.Config) config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg = cfg
	a.notifier = notify.OSNotifier{Enabled: cfg.Notifications.Enabled && !cfg.Lightweight}
	return a.cfg
}

func (a *Agent) UpsertHost(host config.HostConfig) config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, current := range a.cfg.Hosts {
		if host.Name != "" && strings.EqualFold(current.Name, host.Name) {
			a.cfg.Hosts[i] = mergeHost(current, host)
			return a.cfg
		}
		if host.MAC != "" && normalizeMAC(current.MAC) == normalizeMAC(host.MAC) {
			a.cfg.Hosts[i] = mergeHost(current, host)
			return a.cfg
		}
	}
	a.cfg.Hosts = append(a.cfg.Hosts, host)
	return a.cfg
}

func (a *Agent) DeleteHost(target string) (config.Config, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	needle := normalizeMAC(target)
	for i, host := range a.cfg.Hosts {
		if strings.EqualFold(host.Name, target) || normalizeMAC(host.MAC) == needle {
			a.cfg.Hosts = append(a.cfg.Hosts[:i], a.cfg.Hosts[i+1:]...)
			return a.cfg, true
		}
	}
	return a.cfg, false
}

func (a *Agent) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.handleHealthz)
	mux.HandleFunc("POST /v1/wake", a.handleWake)

	server := &http.Server{
		Addr:              a.cfg.ListenHTTP,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, len(a.cfg.ListenMagic)+1)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("HTTP agent listening on %s", a.cfg.ListenHTTP)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	for _, addr := range a.cfg.ListenMagic {
		addr := addr
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := a.listenMagic(ctx, addr); err != nil && !errors.Is(err, net.ErrClosed) {
				errCh <- err
			}
		}()
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		wg.Wait()
		return nil
	case err := <-errCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		wg.Wait()
		return err
	}
}

func (a *Agent) Wake(ctx context.Context, target string, source string) (WakeResult, error) {
	cfg := a.Config()
	host, ok := cfg.FindHost(target)
	if !ok {
		if source == SourceHTTP || source == SourceMagic {
			return WakeResult{}, fmt.Errorf("target %q is not registered", target)
		}
		host = config.HostConfig{MAC: target, Broadcast: cfg.DefaultTarget, Relay: cfg.DefaultRelay}
	}
	return a.wakeHost(ctx, host, source)
}

func (a *Agent) wakeHost(ctx context.Context, host config.HostConfig, source string) (WakeResult, error) {
	cfg := a.Config()
	hw, err := wol.ParseMAC(host.MAC)
	if err != nil {
		return WakeResult{}, err
	}

	if source == SourceHTTP && !a.allowedFromCaller(ctx, host) {
		return WakeResult{}, fmt.Errorf("node %q is not allowed to wake %q", callerNode(ctx), displayHost(host))
	}

	if a.shouldWakeLocally(host) {
		target := host.Broadcast
		if target == "" {
			target = cfg.DefaultTarget
		}
		if err := wol.SendMagicPacket(hw, target); err != nil {
			a.notify("Wake on LAN failed", err.Error())
			return WakeResult{}, err
		}
		result := WakeResult{
			Message: fmt.Sprintf("sent magic packet to %s via %s", displayHost(host), target),
			Target:  hw.String(),
		}
		a.finishOnlineCheck(ctx, host, &result)
		return result, nil
	}

	relay := host.Relay
	if relay == "" {
		relay = cfg.DefaultRelay
	}
	if relay == "" {
		target := host.Broadcast
		if target == "" {
			target = cfg.DefaultTarget
		}
		if err := wol.SendMagicPacket(hw, target); err != nil {
			a.notify("Wake on LAN failed", err.Error())
			return WakeResult{}, err
		}
		result := WakeResult{
			Message: fmt.Sprintf("sent magic packet to %s via %s (no relay configured)", displayHost(host), target),
			Target:  hw.String(),
		}
		a.finishOnlineCheck(ctx, host, &result)
		return result, nil
	}

	res, err := a.relayWake(ctx, relay, host)
	if err != nil {
		a.notify("Wake on LAN relay failed", err.Error())
		return WakeResult{}, err
	}
	res.Relayed = true
	if source == SourceMagic {
		// Notification for "sent" is handled by the burst debounce timer.
		// Only notify on definitive failure so the user sees one error at the end.
		if !res.Online {
			a.notify("Wake on LAN not confirmed", res.Message)
		}
	} else {
		a.notify("Wake on LAN relayed", res.Message)
	}
	return res, nil
}

func (a *Agent) shouldWakeLocally(host config.HostConfig) bool {
	if host.IP == "" {
		return host.Relay == ""
	}
	ip := net.ParseIP(host.IP)
	return netutil.IsLocalIP(ip)
}

func (a *Agent) finishOnlineCheck(ctx context.Context, host config.HostConfig, result *WakeResult) {
	check, ok, err := onlineCheck(host)
	if err != nil {
		result.Message += "; online check is invalid: " + err.Error()
		a.notify("Wake on LAN check failed", result.Message)
		return
	}
	if !ok {
		a.notify("Wake on LAN sent", result.Message)
		return
	}
	if err := online.Wait(ctx, host.IP, check); err != nil {
		result.Message += "; online check timed out"
		a.notify("Wake on LAN not confirmed", result.Message)
		return
	}
	result.Online = true
	result.Message += "; host is online"
	a.notify("Wake on LAN succeeded", result.Message)
}

func (a *Agent) relayWake(ctx context.Context, relay string, host config.HostConfig) (WakeResult, error) {
	cfg := a.Config()
	reqBody := wakeRequest{
		Host:      host.Name,
		MAC:       host.MAC,
		IP:        host.IP,
		Broadcast: host.Broadcast,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return WakeResult{}, err
	}
	url := strings.TrimRight(relay, "/") + "/v1/wake"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return WakeResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	signRequest(req, cfg.NodeName, cfg.Auth.SharedSecret, body)

	client := a.client
	if host.Check.Enabled && host.Check.Timeout != "" {
		if checkTimeout, err := time.ParseDuration(host.Check.Timeout); err == nil {
			client = &http.Client{Timeout: checkTimeout + 15*time.Second}
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return WakeResult{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return WakeResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return WakeResult{}, fmt.Errorf("relay %s returned %s: %s", relay, resp.Status, strings.TrimSpace(string(respBody)))
	}
	var result WakeResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return WakeResult{}, err
	}
	return result, nil
}

func (a *Agent) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	cfg := a.Config()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "node": cfg.NodeName})
}

func (a *Agent) handleWake(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := a.Config()
	if !cfg.Auth.AllowInsecure {
		if err := verifyRequest(req, cfg.Auth.SharedSecret, body); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	}

	var payload wakeRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	host := config.HostConfig{
		Name:      payload.Host,
		MAC:       payload.MAC,
		IP:        payload.IP,
		Broadcast: payload.Broadcast,
	}
	known := false
	if payload.Host != "" {
		if knownHost, ok := cfg.FindHost(payload.Host); ok {
			host = mergeHost(knownHost, host)
			known = true
		}
	}
	if !known && payload.MAC != "" {
		if knownHost, ok := cfg.FindHost(payload.MAC); ok {
			host = mergeHost(knownHost, host)
			known = true
		}
	}
	if !known {
		http.Error(w, "target is not registered", http.StatusForbidden)
		return
	}

	ctx := context.WithValue(req.Context(), callerNodeKey{}, req.Header.Get(headerNode))
	result, err := a.wakeHost(ctx, host, SourceHTTP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	status := http.StatusAccepted
	if result.Online {
		status = http.StatusOK
	}
	writeJSON(w, status, result)
}

func (a *Agent) listenMagic(ctx context.Context, listenAddr string) error {
	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("magic packet listener active on %s", listenAddr)

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buf := make([]byte, 2048)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		hw, ok := wol.ParseMagicPacket(buf[:n])
		if !ok {
			continue
		}
		if !a.magicSourceAllowed(remote) {
			log.Printf("ignored magic packet from unauthorized source %s for %s", remote, hw)
			continue
		}
		a.handleDetectedMagic(ctx, remote, hw)
	}
}

func (a *Agent) magicSourceAllowed(remote *net.UDPAddr) bool {
	if remote == nil {
		return false
	}
	cfg := a.Config()
	if len(cfg.AllowedMagicSources) == 0 {
		return true
	}
	for _, allowed := range cfg.AllowedMagicSources {
		if ip := net.ParseIP(allowed); ip != nil && ip.Equal(remote.IP) {
			return true
		}
		if _, ipNet, err := net.ParseCIDR(allowed); err == nil && ipNet.Contains(remote.IP) {
			return true
		}
	}
	return false
}

type magicBurst struct {
	timer *time.Timer
	count int
}

// trackMagicBurst records a magic packet for the given MAC and returns whether
// this is the first packet in the current burst window (i.e. should wake).
// On the first call for a MAC, a 30-second timer is started; when it fires,
// notifyFn is called with the total packet count for that burst.
func (a *Agent) trackMagicBurst(mac string, notifyFn func(count int)) (first bool) {
	a.burstMu.Lock()
	defer a.burstMu.Unlock()
	if a.burstMap == nil {
		a.burstMap = make(map[string]*magicBurst)
	}
	if b, exists := a.burstMap[mac]; exists {
		b.count++
		return false
	}
	b := &magicBurst{count: 1}
	b.timer = time.AfterFunc(a.burstDebounceWindow, func() {
		a.burstMu.Lock()
		count := a.burstMap[mac].count
		delete(a.burstMap, mac)
		a.burstMu.Unlock()
		notifyFn(count)
	})
	a.burstMap[mac] = b
	return true
}

func (a *Agent) handleDetectedMagic(ctx context.Context, remote net.Addr, hw net.HardwareAddr) {
	cfg := a.Config()
	host, known := cfg.FindHost(hw.String())
	if !known {
		log.Printf("ignored magic packet from %s for unregistered target %s", remote, hw)
		return
	}
	if a.shouldWakeLocally(host) {
		log.Printf("detected local magic packet from %s for %s; no relay needed", remote, hw)
		a.trackMagicBurst(hw.String(), func(count int) {
			msg := fmt.Sprintf("%s 宛てのマジックパケットは、すでにこのLANに届いています。リレーは不要です。", displayHost(host))
			if count > 1 {
				msg += fmt.Sprintf(" (%d件検知)", count)
			}
			a.notify("Wake on LANを検知しました", msg)
		})
		return
	}
	first := a.trackMagicBurst(hw.String(), func(count int) {
		msg := fmt.Sprintf("%s へのリレーを送信しました。", displayHost(host))
		if count > 1 {
			msg += fmt.Sprintf(" (%d件のパケットを集約)", count)
		}
		a.notify("Wake on LAN 送信", msg)
	})
	if !first {
		log.Printf("burst: additional magic packet from %s for %s", remote, hw)
		return
	}
	go func() {
		res, err := a.wakeHost(ctx, host, SourceMagic)
		if err != nil {
			log.Printf("failed to handle magic packet from %s for %s: %v", remote, hw, err)
			return
		}
		log.Printf("handled magic packet from %s for %s: %s", remote, hw, res.Message)
	}()
}

type callerNodeKey struct{}

func callerNode(ctx context.Context) string {
	if value, ok := ctx.Value(callerNodeKey{}).(string); ok {
		return value
	}
	return ""
}

func (a *Agent) allowedFromCaller(ctx context.Context, host config.HostConfig) bool {
	if len(host.AllowedBy) == 0 {
		return true
	}
	caller := callerNode(ctx)
	for _, node := range host.AllowedBy {
		if node == caller {
			return true
		}
	}
	return false
}

func mergeHost(base, override config.HostConfig) config.HostConfig {
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.MAC != "" {
		base.MAC = override.MAC
	}
	if override.IP != "" {
		base.IP = override.IP
	}
	if override.Broadcast != "" {
		base.Broadcast = override.Broadcast
	}
	if override.Relay != "" {
		base.Relay = override.Relay
	}
	if len(override.AllowedBy) > 0 {
		base.AllowedBy = override.AllowedBy
	}
	return base
}

func displayHost(host config.HostConfig) string {
	if host.Name != "" {
		return host.Name
	}
	return host.MAC
}

func normalizeMAC(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(value, ":", ""), "-", ""))
}

func onlineCheck(host config.HostConfig) (online.Check, bool, error) {
	if !host.Check.Enabled {
		return online.Check{}, false, nil
	}
	timeout, err := time.ParseDuration(host.Check.Timeout)
	if err != nil {
		return online.Check{}, true, err
	}
	interval, err := time.ParseDuration(host.Check.Interval)
	if err != nil {
		return online.Check{}, true, err
	}
	return online.Check{
		Enabled:  true,
		Method:   host.Check.Method,
		Port:     host.Check.Port,
		Timeout:  timeout,
		Interval: interval,
	}, true, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (a *Agent) notify(title, message string) {
	if a.notifier != nil {
		a.notifier.Notify(title, message)
	}
}
