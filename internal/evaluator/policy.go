package evaluator

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
	"vanguard/internal/models"
)

// Policy represents a security policy loaded from a YAML configuration file.
// It defines the maximum number of allowed findings per severity level.
type Policy struct {
	APIVersion string                  `yaml:"apiVersion"`
	Kind       string                  `yaml:"kind"`
	Metadata   PolicyMetadata          `yaml:"metadata"`
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
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&policy); err != nil {
		return nil, fmt.Errorf("evaluator: failed to parse policy YAML: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("evaluator: expected one YAML policy")
	}

	if policy.APIVersion == "" {
		return nil, fmt.Errorf("evaluator: policy file is missing required field \"apiVersion\"")
	}
	if policy.Kind == "" {
		return nil, fmt.Errorf("evaluator: policy file is missing required field \"kind\"")
	}

	if policy.APIVersion != "vanguard/v1" || policy.Kind != "SecurityPolicy" {
		return nil, fmt.Errorf("evaluator: unsupported policy version or kind")
	}
	if len(policy.Thresholds) == 0 {
		return nil, fmt.Errorf("evaluator: at least one threshold is required")
	}
	for severity, limit := range policy.Thresholds {
		if !models.ValidSeverity(severity) || limit < -1 {
			return nil, fmt.Errorf("evaluator: invalid threshold %q=%d", severity, limit)
		}
	}
	return &policy, nil
}
