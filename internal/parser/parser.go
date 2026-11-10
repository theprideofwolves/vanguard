package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/theprideofwolves/vanguard/internal/models"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Parse reads JSON from r and decodes it into a models.Report.
// It retains a capped report in memory, validates keys and schema, and normalizes
// severity values before checking for identical finding objects.
func Parse(r io.Reader) (*models.Report, error) {
	var report models.Report

	data, err := io.ReadAll(io.LimitReader(r, (16<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 16<<20 {
		return nil, fmt.Errorf("report exceeds 16 MiB")
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("report must be valid UTF-8")
	}
	if err := uniqueKeys(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return nil, fmt.Errorf("parser: failed to decode report JSON: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("parser: expected one JSON report")
	}
	if report.Findings == nil {
		return nil, fmt.Errorf("parser: findings array is required (use [] for no findings)")
	}

	// Normalize severity values to lowercase for consistent evaluation.
	for i := range report.Findings {
		report.Findings[i].Severity = models.Severity(
			strings.ToLower(string(report.Findings[i].Severity)),
		)

		if !models.ValidSeverity(report.Findings[i].Severity) {
			return nil, fmt.Errorf("parser: finding %d has unsupported severity %q", i, report.Findings[i].Severity)
		}
	}

	seen := map[string]bool{}
	for _, f := range report.Findings {
		identity, _ := json.Marshal(f)
		key := string(identity)
		if seen[key] {
			return nil, fmt.Errorf("duplicate identical finding; fix input rather than double-counting")
		}
		seen[key] = true
	}
	return &report, nil
}

// Reject duplicate object keys instead of silently allowing last-key-wins input.
func uniqueKeys(d *json.Decoder) error {
	token, e := d.Token()
	if e != nil {
		return e
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, e := d.Token()
			if e != nil {
				return e
			}
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("object key must be a string")
			}
			// encoding/json matches struct fields case-insensitively, including
			// Unicode simple folds. Use the same equivalence for duplicate keys
			// so a case variant cannot overwrite a previously validated field.
			name = strings.Map(func(r rune) rune {
				canonical := r
				for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
					if next < canonical {
						canonical = next
					}
				}
				return canonical
			}, name)
			if seen[name] {
				return fmt.Errorf("duplicate JSON object key")
			}
			seen[name] = true
			if e = uniqueKeys(d); e != nil {
				return e
			}
		}
		_, e = d.Token()
		return e
	case '[':
		for d.More() {
			if e = uniqueKeys(d); e != nil {
				return e
			}
		}
		_, e = d.Token()
		return e
	default:
		return fmt.Errorf("unexpected JSON delimiter")
	}
}
