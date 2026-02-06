package router

import (
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func event(channel, text string) plugin.InboundEvent {
	return plugin.InboundEvent{
		Type: "message",
		Data: map[string]any{"channel": channel, "text": text},
	}
}

func TestRouteExactChannel(t *testing.T) {
	r := New(Config{
		Rules: []Rule{
			{Channel: "general", Profile: "gen-bot"},
			{Prefix: "!", Profile: "cmd-bot"},
		},
		Default: "default-bot",
	})
	profile, err := r.Route(event("general", "!hello"))
	if err != nil {
		t.Fatal(err)
	}
	// Channel match (specificity 2) beats prefix match (specificity 1).
	if profile != "gen-bot" {
		t.Fatalf("profile = %q, want gen-bot", profile)
	}
}

func TestRoutePrefixMatch(t *testing.T) {
	r := New(Config{
		Rules: []Rule{
			{Channel: "general", Profile: "gen-bot"},
			{Prefix: "!", Profile: "cmd-bot"},
		},
		Default: "default-bot",
	})
	profile, err := r.Route(event("random", "!deploy"))
	if err != nil {
		t.Fatal(err)
	}
	if profile != "cmd-bot" {
		t.Fatalf("profile = %q, want cmd-bot", profile)
	}
}

func TestRouteDefault(t *testing.T) {
	r := New(Config{
		Rules:   []Rule{{Channel: "general", Profile: "gen-bot"}},
		Default: "default-bot",
	})
	profile, err := r.Route(event("random", "hello"))
	if err != nil {
		t.Fatal(err)
	}
	if profile != "default-bot" {
		t.Fatalf("profile = %q, want default-bot", profile)
	}
}

func TestRouteNoMatchNoDefault(t *testing.T) {
	r := New(Config{
		Rules: []Rule{{Channel: "general", Profile: "gen-bot"}},
	})
	_, err := r.Route(event("random", "hello"))
	if err == nil {
		t.Fatal("expected error when no rule matches and no default")
	}
}

func TestRouteEmptyEvent(t *testing.T) {
	r := New(Config{Default: "fallback"})
	profile, err := r.Route(plugin.InboundEvent{Type: "message", Data: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if profile != "fallback" {
		t.Fatalf("profile = %q, want fallback", profile)
	}
}

func TestRouteNilData(t *testing.T) {
	r := New(Config{Default: "fallback"})
	profile, err := r.Route(plugin.InboundEvent{Type: "message"})
	if err != nil {
		t.Fatal(err)
	}
	if profile != "fallback" {
		t.Fatalf("profile = %q, want fallback", profile)
	}
}

func TestRouteMostSpecificWins(t *testing.T) {
	r := New(Config{
		Rules: []Rule{
			{Prefix: "!", Profile: "prefix-bot"},
			{Channel: "ops", Profile: "ops-bot"},
		},
	})
	// Both match: channel + prefix. Channel is more specific.
	profile, err := r.Route(event("ops", "!deploy"))
	if err != nil {
		t.Fatal(err)
	}
	if profile != "ops-bot" {
		t.Fatalf("profile = %q, want ops-bot (channel beats prefix)", profile)
	}
}

func TestRouteMultipleRulesNoMatch(t *testing.T) {
	r := New(Config{
		Rules: []Rule{
			{Channel: "general", Profile: "gen-bot"},
			{Prefix: "!", Profile: "cmd-bot"},
		},
	})
	_, err := r.Route(event("random", "hello"))
	if err == nil {
		t.Fatal("expected error when no rule matches and no default")
	}
}
