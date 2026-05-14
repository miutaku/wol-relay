package gui

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/miutaku/wol-relay/internal/agent"
	"github.com/miutaku/wol-relay/internal/config"
	"github.com/miutaku/wol-relay/internal/wol"
)

//go:embed index.html
var indexHTML string

type Server struct {
	Agent      *agent.Agent
	ConfigPath string
	Addr       string
	Open       bool
}

func (s Server) Run(ctx context.Context) error {
	addr := s.Addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/config-path", s.handleConfigPath)
	mux.HandleFunc("POST /api/open-config-dir", s.handleOpenConfigDir)
	mux.HandleFunc("PUT /api/config", s.handleUpdateConfig)
	mux.HandleFunc("POST /api/hosts", s.handleAddHost)
	mux.HandleFunc("DELETE /api/hosts", s.handleDeleteHost)
	mux.HandleFunc("POST /api/wake", s.handleWake)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	url := "http://" + listener.Addr().String()
	log.Printf("GUI listening on %s", url)
	if s.Open {
		openBrowser(url)
	}

	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		wg.Wait()
		return nil
	case err := <-errCh:
		return err
	}
}

func (s Server) handleAddHost(w http.ResponseWriter, req *http.Request) {
	var host config.HostConfig
	if err := json.NewDecoder(req.Body).Decode(&host); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	host.Name = strings.TrimSpace(host.Name)
	host.MAC = strings.TrimSpace(host.MAC)
	host.IP = strings.TrimSpace(host.IP)
	host.Broadcast = strings.TrimSpace(host.Broadcast)
	host.Relay = strings.TrimSpace(host.Relay)
	host.Check.Method = strings.TrimSpace(host.Check.Method)
	host.Check.Timeout = strings.TrimSpace(host.Check.Timeout)
	host.Check.Interval = strings.TrimSpace(host.Check.Interval)
	if host.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if _, err := wol.ParseMAC(host.MAC); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := s.Agent.UpsertHost(host)
	if s.ConfigPath != "" {
		if err := config.Save(s.ConfigPath, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) handleDeleteHost(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	payload.Target = strings.TrimSpace(payload.Target)
	if payload.Target == "" {
		http.Error(w, "target is required", http.StatusBadRequest)
		return
	}
	cfg, ok := s.Agent.DeleteHost(payload.Target)
	if !ok {
		http.Error(w, "target is not registered", http.StatusNotFound)
		return
	}
	if s.ConfigPath != "" {
		if err := config.Save(s.ConfigPath, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHTML)
}

func (s Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Agent.Config())
}

func (s Server) handleConfigPath(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"path": s.ConfigPath, "dir": filepath.Dir(s.ConfigPath)})
}

func (s Server) handleOpenConfigDir(w http.ResponseWriter, _ *http.Request) {
	if s.ConfigPath == "" {
		http.Error(w, "config path is unknown", http.StatusBadRequest)
		return
	}
	if err := openPath(filepath.Dir(s.ConfigPath)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) handleUpdateConfig(w http.ResponseWriter, req *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(req.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	normalizeConfig(&cfg)
	if s.ConfigPath != "" {
		if err := config.Save(s.ConfigPath, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		saved, err := config.Load(s.ConfigPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cfg = saved
	}
	s.Agent.UpdateConfig(cfg)
	writeJSON(w, http.StatusOK, cfg)
}

func (s Server) handleWake(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	payload.Target = strings.TrimSpace(payload.Target)
	if payload.Target == "" {
		http.Error(w, "target is required", http.StatusBadRequest)
		return
	}
	result, err := s.Agent.Wake(req.Context(), payload.Target, agent.SourceCLI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func normalizeConfig(cfg *config.Config) {
	cfg.NodeName = strings.TrimSpace(cfg.NodeName)
	cfg.ListenHTTP = strings.TrimSpace(cfg.ListenHTTP)
	cfg.DefaultRelay = strings.TrimSpace(cfg.DefaultRelay)
	cfg.DefaultTarget = strings.TrimSpace(cfg.DefaultTarget)
	cfg.Auth.SharedSecret = strings.TrimSpace(cfg.Auth.SharedSecret)
	cfg.ListenMagic = trimStrings(cfg.ListenMagic)
	cfg.AllowedMagicSources = trimStrings(cfg.AllowedMagicSources)
	for i := range cfg.Hosts {
		cfg.Hosts[i].Name = strings.TrimSpace(cfg.Hosts[i].Name)
		cfg.Hosts[i].MAC = strings.TrimSpace(cfg.Hosts[i].MAC)
		cfg.Hosts[i].IP = strings.TrimSpace(cfg.Hosts[i].IP)
		cfg.Hosts[i].Broadcast = strings.TrimSpace(cfg.Hosts[i].Broadcast)
		cfg.Hosts[i].Relay = strings.TrimSpace(cfg.Hosts[i].Relay)
		cfg.Hosts[i].AllowedBy = trimStrings(cfg.Hosts[i].AllowedBy)
		cfg.Hosts[i].Check.Method = strings.TrimSpace(cfg.Hosts[i].Check.Method)
		cfg.Hosts[i].Check.Timeout = strings.TrimSpace(cfg.Hosts[i].Check.Timeout)
		cfg.Hosts[i].Check.Interval = strings.TrimSpace(cfg.Hosts[i].Check.Interval)
	}
}

func trimStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func openBrowser(url string) {
	_ = openPath(url)
}

func openPath(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
