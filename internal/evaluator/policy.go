package evaluator

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/theprideofwolves/vanguard/internal/models"
	"gopkg.in/yaml.v3"
)

// Policy represents a security policy loaded from a YAML configuration file.
// It defines the maximum number of allowed findings per severity level.
type Policy struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   PolicyMetadata `yaml:"metadata"`
	Thresholds Thresholds     `yaml:"thresholds"`
}

// Thresholds requires integer YAML scalars; yaml.v3 otherwise converts null to
// zero and truncates fractional values when decoding directly into map values.
type Thresholds map[models.Severity]int

func (t *Thresholds) UnmarshalYAML(node *yaml.Node) error {
	var values map[models.Severity]yaml.Node
	if err := node.Decode(&values); err != nil {
		return err
	}
	limits := make(Thresholds, len(values))
	for severity, value := range values {
		if value.Kind != yaml.ScalarNode || value.Tag != "!!int" {
			return fmt.Errorf("threshold %q must be an integer", severity)
		}
		var limit int
		if err := value.Decode(&limit); err != nil {
			return err
		}
		limits[severity] = limit
	}
	*t = limits
	return nil
}

// PolicyMetadata holds descriptive metadata for a policy.
type PolicyMetadata struct {
	Name string `yaml:"name"`
}

// LoadPolicy reads and parses a YAML policy file from disk.
// It validates that the required apiVersion and kind fields are present.
func LoadPolicy(path string) (*Policy, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("evaluator: unable to read policy file %q: %w", path, err)
	}

	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	return ParsePolicy(data)
}
func ParsePolicy(data []byte) (*Policy, error) {
	if len(data) > 1<<20 {
		return nil, fmt.Errorf("policy exceeds 1 MiB")
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
