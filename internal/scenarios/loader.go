// Package scenarios loads and validates scenario definitions from YAML files.
package scenarios

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Scenario defines a test scenario for an agent run.
type Scenario struct {
	Name                string              `yaml:"name"`
	Description         string              `yaml:"description"`
	InboundEvents       []InboundEvent      `yaml:"inbound_events"`
	HarnessEvents       []HarnessEvent      `yaml:"harness_events"`
	ExpectedOps         []ExpectedOp        `yaml:"expected_transport_ops"`
	ExpectedContext     *ExpectedContext     `yaml:"expected_context"`
	ExpectedFailures    []ExpectedFailure   `yaml:"expected_failures"`
}

// InboundEvent is a scripted inbound event in a scenario.
type InboundEvent struct {
	Type  string         `yaml:"type"`
	Data  map[string]any `yaml:"data"`
	Delay int            `yaml:"delay"`
}

// HarnessEvent is a scripted harness event in a scenario.
type HarnessEvent struct {
	Kind string         `yaml:"kind"`
	Data map[string]any `yaml:"data"`
}

// ExpectedOp is an expected outbound transport operation.
type ExpectedOp struct {
	Kind string `yaml:"kind"`
}

// ExpectedContext defines context expectations for a scenario.
type ExpectedContext struct {
	MustInclude []ContextExpectation `yaml:"must_include"`
}

// ContextExpectation is a single expected context item.
type ContextExpectation struct {
	Name string `yaml:"name"`
}

// ExpectedFailure is an expected error condition.
type ExpectedFailure struct {
	Kind            string `yaml:"kind"`
	MessageContains string `yaml:"message_contains"`
}

// LoadFile loads a scenario from a YAML file.
func LoadFile(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("scenarios: read %s: %w", path, err)
	}
	return Parse(data)
}

// Parse parses a scenario from YAML bytes.
func Parse(data []byte) (*Scenario, error) {
	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("scenarios: parse: %w", err)
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Scenario) validate() error {
	if s.Name == "" {
		return fmt.Errorf("scenarios: name is required")
	}
	if len(s.InboundEvents) == 0 {
		return fmt.Errorf("scenarios: at least one inbound_event is required")
	}
	if len(s.HarnessEvents) == 0 {
		return fmt.Errorf("scenarios: at least one harness_event is required")
	}
	return nil
}
