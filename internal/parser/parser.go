package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"vanguard/internal/models"
)

// Parse reads JSON from r and decodes it into a models.Report.
// It uses json.Decoder for streaming so that large files are not
// buffered entirely in memory before decoding begins.
func Parse(r io.Reader) (*models.Report, error) {
	var report models.Report

	decoder := json.NewDecoder(r)
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

	return &report, nil
}
