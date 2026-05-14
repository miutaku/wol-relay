package firewall

import (
	"fmt"
	"net"
	"runtime"
	"sort"
	"strings"

	"github.com/miutaku/wol-relay/internal/config"
)

type PortRule struct {
	Protocol string
	Port     string
	Purpose  string
}

func Rules(cfg config.Config) ([]PortRule, error) {
	var rules []PortRule
	httpPort, err := portFromAddr(cfg.ListenHTTP)
	if err != nil {
		return nil, fmt.Errorf("listen_http: %w", err)
	}
	rules = append(rules, PortRule{Protocol: "tcp", Port: httpPort, Purpose: "REST API"})

	for _, addr := range cfg.ListenMagic {
		port, err := portFromAddr(addr)
		if err != nil {
			return nil, fmt.Errorf("listen_magic %q: %w", addr, err)
		}
		rules = append(rules, PortRule{Protocol: "udp", Port: port, Purpose: "magic packet listener"})
	}

	return uniqueRules(rules), nil
}

func Plan(cfg config.Config) (string, error) {
	return planForOS(runtime.GOOS, cfg)
}

func planForOS(osName string, cfg config.Config) (string, error) {
	rules, err := Rules(cfg)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Firewall commands for %s\n", osName)
	fmt.Fprintf(&b, "# Review before running. Administrator privileges may be required.\n\n")

	switch osName {
	case "windows":
		writeWindows(&b, rules)
	case "linux":
		writeLinux(&b, rules)
	case "darwin":
		writeDarwin(&b, rules)
	default:
		return "", fmt.Errorf("unsupported os %q", osName)
	}
	return b.String(), nil
}

func writeWindows(b *strings.Builder, rules []PortRule) {
	for _, rule := range rules {
		fmt.Fprintf(
			b,
			"netsh advfirewall firewall add rule name=\"wol-relay %s %s\" dir=in action=allow protocol=%s localport=%s\n",
			strings.ToUpper(rule.Protocol),
			rule.Port,
			strings.ToUpper(rule.Protocol),
			rule.Port,
		)
	}
}

func writeLinux(b *strings.Builder, rules []PortRule) {
	fmt.Fprintln(b, "# ufw")
	for _, rule := range rules {
		fmt.Fprintf(b, "sudo ufw allow %s/%s comment 'wol-relay %s'\n", rule.Port, rule.Protocol, rule.Purpose)
	}

	fmt.Fprintln(b, "\n# firewalld")
	for _, rule := range rules {
		fmt.Fprintf(b, "sudo firewall-cmd --permanent --add-port=%s/%s\n", rule.Port, rule.Protocol)
	}
	fmt.Fprintln(b, "sudo firewall-cmd --reload")

	fmt.Fprintln(b, "\n# nftables example")
	for _, rule := range rules {
		fmt.Fprintf(b, "sudo nft add rule inet filter input %s dport %s accept\n", rule.Protocol, rule.Port)
	}
}

func writeDarwin(b *strings.Builder, rules []PortRule) {
	fmt.Fprintln(b, "# macOS usually prompts for inbound access per application.")
	fmt.Fprintln(b, "# For a built binary, allow and unblock it in the Application Firewall:")
	fmt.Fprintln(b, "sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /path/to/wol-relay")
	fmt.Fprintln(b, "sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblockapp /path/to/wol-relay")
	fmt.Fprintln(b, "\n# Ports used by this config:")
	for _, rule := range rules {
		fmt.Fprintf(b, "# - %s/%s: %s\n", rule.Port, rule.Protocol, rule.Purpose)
	}
}

func portFromAddr(addr string) (string, error) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	if port == "" {
		return "", fmt.Errorf("missing port in %q", addr)
	}
	return port, nil
}

func uniqueRules(rules []PortRule) []PortRule {
	seen := map[string]PortRule{}
	for _, rule := range rules {
		key := rule.Protocol + "/" + rule.Port
		if _, ok := seen[key]; !ok {
			seen[key] = rule
		}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]PortRule, 0, len(keys))
	for _, key := range keys {
		out = append(out, seen[key])
	}
	return out
}
