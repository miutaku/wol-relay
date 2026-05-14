package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	NodeName            string             `json:"node_name"`
	ListenHTTP          string             `json:"listen_http"`
	ListenMagic         []string           `json:"listen_magic"`
	AllowedMagicSources []string           `json:"allowed_magic_sources"`
	DefaultRelay        string             `json:"default_relay"`
	DefaultTarget       string             `json:"default_target"`
	Lightweight         bool               `json:"lightweight"`
	Auth                AuthConfig         `json:"auth"`
	Notifications       NotificationConfig `json:"notifications"`
	Hosts               []HostConfig       `json:"hosts"`
}

type AuthConfig struct {
	SharedSecret  string `json:"shared_secret"`
	AllowInsecure bool   `json:"allow_insecure"`
}

type NotificationConfig struct {
	Enabled bool `json:"enabled"`
}

type HostConfig struct {
	Name      string      `json:"name"`
	MAC       string      `json:"mac"`
	IP        string      `json:"ip"`
	Broadcast string      `json:"broadcast"`
	Relay     string      `json:"relay"`
	AllowedBy []string    `json:"allowed_by"`
	Check     CheckConfig `json:"check"`
}

type CheckConfig struct {
	Enabled  bool   `json:"enabled"`
	Method   string `json:"method"`
	Port     int    `json:"port"`
	Timeout  string `json:"timeout"`
	Interval string `json:"interval"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg.setDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg.setDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func LoadOrCreate(path string) (Config, error) {
	cfg, err := Load(path)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	cfg, err = Example()
	if err != nil {
		return Config{}, err
	}
	if err := Save(path, cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "wol-relay", "wol-relay.json"), nil
}

func (c *Config) setDefaults() {
	if c.NodeName == "" {
		hostname, err := os.Hostname()
		if err == nil && hostname != "" {
			c.NodeName = hostname
		} else {
			c.NodeName = "wol-relay"
		}
	}
	if c.ListenHTTP == "" {
		c.ListenHTTP = "127.0.0.1:8080"
	}
	if len(c.ListenMagic) == 0 {
		c.ListenMagic = []string{":9"}
	}
	if c.DefaultTarget == "" {
		c.DefaultTarget = "255.255.255.255:9"
	}
	if c.DefaultRelay == "http://raspberrypi.local:8080" {
		c.DefaultRelay = ""
	}
	if c.Lightweight {
		c.Notifications.Enabled = false
	}
	for i := range c.Hosts {
		if c.Hosts[i].Broadcast == "" {
			c.Hosts[i].Broadcast = c.DefaultTarget
		}
		if c.Hosts[i].Relay == "http://raspberrypi.local:8080" {
			c.Hosts[i].Relay = ""
		}
		if c.Hosts[i].Check.Enabled {
			if c.Hosts[i].Check.Method == "" {
				c.Hosts[i].Check.Method = "tcp"
			}
			if c.Hosts[i].Check.Interval == "" {
				c.Hosts[i].Check.Interval = "3s"
			}
			if c.Hosts[i].Check.Timeout == "" {
				c.Hosts[i].Check.Timeout = "2m"
			}
		}
	}
}

func (c Config) Validate() error {
	if c.NodeName == "" {
		return errors.New("node_name is required")
	}
	if _, _, err := net.SplitHostPort(c.ListenHTTP); err != nil {
		return fmt.Errorf("listen_http must be host:port: %w", err)
	}
	for _, addr := range c.ListenMagic {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			return fmt.Errorf("listen_magic address %q must be host:port: %w", addr, err)
		}
	}
	for _, source := range c.AllowedMagicSources {
		if ip := net.ParseIP(source); ip != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(source); err != nil {
			return fmt.Errorf("allowed_magic_sources entry %q must be an IP address or CIDR: %w", source, err)
		}
	}
	if c.Auth.SharedSecret == "" && !c.Auth.AllowInsecure {
		return errors.New("auth.shared_secret is required unless auth.allow_insecure is true")
	}
	seen := map[string]struct{}{}
	for _, host := range c.Hosts {
		if host.Name == "" && host.MAC == "" {
			return errors.New("each host requires at least name or mac")
		}
		if host.MAC != "" {
			if _, err := net.ParseMAC(host.MAC); err != nil {
				return fmt.Errorf("host %q has invalid mac: %w", host.Name, err)
			}
			seen[normalize(host.MAC)] = struct{}{}
		}
		if host.Name != "" {
			key := strings.ToLower(host.Name)
			if _, ok := seen[key]; ok {
				return fmt.Errorf("duplicate host key %q", host.Name)
			}
			seen[key] = struct{}{}
		}
		if host.Check.Enabled {
			if host.IP == "" {
				return fmt.Errorf("host %q online check requires ip", host.Name)
			}
			if host.Check.Method != "tcp" && host.Check.Method != "icmp" {
				return fmt.Errorf("host %q has unsupported check method %q", host.Name, host.Check.Method)
			}
			if host.Check.Method == "tcp" && host.Check.Port < 0 {
				return fmt.Errorf("host %q has invalid check port %d", host.Name, host.Check.Port)
			}
			if _, err := time.ParseDuration(host.Check.Timeout); err != nil {
				return fmt.Errorf("host %q has invalid check timeout: %w", host.Name, err)
			}
			if _, err := time.ParseDuration(host.Check.Interval); err != nil {
				return fmt.Errorf("host %q has invalid check interval: %w", host.Name, err)
			}
		}
	}
	return nil
}

func (c Config) FindHost(value string) (HostConfig, bool) {
	needle := normalize(value)
	for _, host := range c.Hosts {
		if strings.EqualFold(host.Name, value) || normalize(host.MAC) == needle {
			return host, true
		}
	}
	return HostConfig{}, false
}

func WriteExample(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	example, err := Example()
	if err != nil {
		return err
	}
	return Save(path, example)
}

func Example() (Config, error) {
	secret, err := randomSecret()
	if err != nil {
		return Config{}, err
	}
	example := Config{
		NodeName:      "desktop",
		ListenHTTP:    "127.0.0.1:8080",
		ListenMagic:   []string{":9"},
		DefaultRelay:  "",
		DefaultTarget: "255.255.255.255:9",
		Auth: AuthConfig{
			SharedSecret: secret,
		},
		Notifications: NotificationConfig{
			Enabled: true,
		},
		Hosts: []HostConfig{},
	}
	return example, nil
}

func normalize(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(value, ":", ""), "-", ""))
}

func randomSecret() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
