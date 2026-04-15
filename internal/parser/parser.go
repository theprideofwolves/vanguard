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

	// Normalize severity values to lowercase for consistent evaluation.
	for i := range report.Findings {
		report.Findings[i].Severity = models.Severity(
			strings.ToLower(string(report.Findings[i].Severity)),
		)
	}

	return &report, nil
}
