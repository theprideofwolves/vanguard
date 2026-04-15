package evaluator

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"vanguard/internal/models"
)

// Policy represents a security policy loaded from a YAML configuration file.
// It defines the maximum number of allowed findings per severity level.
type Policy struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   PolicyMetadata    `yaml:"metadata"`
	Thresholds map[models.Severity]int `yaml:"thresholds"`
}

// PolicyMetadata holds descriptive metadata for a policy.
type PolicyMetadata struct {
	Name string `yaml:"name"`
}

// LoadPolicy reads and parses a YAML policy file from disk.
// It validates that the required apiVersion and kind fields are present.
func LoadPolicy(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("evaluator: unable to read policy file %q: %w", path, err)
	}

	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("evaluator: failed to parse policy YAML: %w", err)
	}

	if policy.APIVersion == "" {
		return nil, fmt.Errorf("evaluator: policy file is missing required field \"apiVersion\"")
	}
	if policy.Kind == "" {
		return nil, fmt.Errorf("evaluator: policy file is missing required field \"kind\"")
	}

	return &policy, nil
}
