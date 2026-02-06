// Package router provides deterministic routing of inbound events to agent profiles.
package router

import (
	"fmt"
	"sort"
	"strings"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Rule maps a condition to a target profile.
type Rule struct {
	Channel string `json:"channel"` // exact channel match (most specific)
	Prefix  string `json:"prefix"`  // message prefix match
	Profile string `json:"profile"` // target agent profile
}

// Config holds the routing configuration.
type Config struct {
	Rules   []Rule `json:"rules"`
	Default string `json:"default"` // fallback profile
}

// Router routes inbound events to agent profiles using "most specific wins".
type Router struct {
	cfg Config
}

// New creates a Router from the given config.
func New(cfg Config) *Router {
	return &Router{cfg: cfg}
}

// Route determines the target agent profile for an event.
// Priority: exact channel > prefix > default.
func (r *Router) Route(event plugin.InboundEvent) (string, error) {
	channel, _ := event.Data["channel"].(string)
	text, _ := event.Data["text"].(string)

	// Collect matching rules with specificity scores.
	type scored struct {
		rule        Rule
		specificity int
	}
	var matches []scored

	for _, rule := range r.cfg.Rules {
		if rule.Channel != "" && rule.Channel == channel {
			matches = append(matches, scored{rule, 2})
		} else if rule.Prefix != "" && strings.HasPrefix(text, rule.Prefix) {
			matches = append(matches, scored{rule, 1})
		}
	}

	if len(matches) > 0 {
		// Sort by specificity descending for "most specific wins".
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].specificity > matches[j].specificity
		})
		return matches[0].rule.Profile, nil
	}

	if r.cfg.Default != "" {
		return r.cfg.Default, nil
	}

	return "", fmt.Errorf("router: no matching rule for event")
}
