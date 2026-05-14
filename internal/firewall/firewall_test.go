package firewall

import (
	"strings"
	"testing"

	"github.com/miutaku/wol-relay/internal/config"
)

func TestRulesDeduplicatesPorts(t *testing.T) {
	rules, err := Rules(config.Config{
		ListenHTTP:  "0.0.0.0:8080",
		ListenMagic: []string{":9", "0.0.0.0:9"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}
	if rules[0].Protocol != "tcp" || rules[0].Port != "8080" {
		t.Fatalf("unexpected first rule: %+v", rules[0])
	}
	if rules[1].Protocol != "udp" || rules[1].Port != "9" {
		t.Fatalf("unexpected second rule: %+v", rules[1])
	}
}

func TestPlanWindows(t *testing.T) {
	plan, err := planForOS("windows", config.Config{
		ListenHTTP:  "0.0.0.0:8080",
		ListenMagic: []string{":9"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"netsh advfirewall firewall add rule",
		"protocol=TCP localport=8080",
		"protocol=UDP localport=9",
	} {
		if !strings.Contains(plan, want) {
			t.Fatalf("plan does not contain %q:\n%s", want, plan)
		}
	}
}
