// Package scenario loads, validates, and runs test scenarios.
package scenario

import (
	"fmt"
	"os"

	"github.com/klabo/tinyclaw/internal/plugin"
	"gopkg.in/yaml.v3"
)

// Scenario defines a test scenario for an agent run.
type Scenario struct {
	Name          string           `yaml:"name"`
	Description   string           `yaml:"description"`
	InboundEvents []InboundEvent   `yaml:"inbound_events"`
	HarnessEvents []plugin.RunEvent `yaml:"harness_events"`
	ExpectedOps   []ExpectedOp     `yaml:"expected_transport_ops"`
}

// InboundEvent is a scripted inbound event in a scenario.
type InboundEvent struct {
	Type      plugin.InboundEventType `yaml:"type"`
	Content   string `yaml:"content"`
	ChannelID string `yaml:"channel_id"`
	AuthorID  string `yaml:"author_id"`
}

// ExpectedOp is an expected outbound transport operation.
type ExpectedOp struct {
	Kind    plugin.OutboundOpKind `yaml:"kind"`
	Content string                `yaml:"content,omitempty"`
}

// LoadFile loads a scenario from a YAML file.
func LoadFile(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("scenario: read %s: %w", path, err)
	}
	return Parse(data)
}

// Parse parses a scenario from YAML bytes.
func Parse(data []byte) (*Scenario, error) {
	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("scenario: parse: %w", err)
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Scenario) validate() error {
	if s.Name == "" {
		return fmt.Errorf("scenario: name is required")
	}
	if len(s.InboundEvents) == 0 {
		return fmt.Errorf("scenario: at least one inbound_event is required")
	}
	if len(s.HarnessEvents) == 0 {
		return fmt.Errorf("scenario: at least one harness_event is required")
	}
	return nil
}
