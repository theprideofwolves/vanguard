package evaluator

import (
	"fmt"
	"vanguard/internal/models"
)

// Verdict holds the result of evaluating a security report against a policy.
type Verdict struct {
	// Passed is true when all severity counts are within policy thresholds.
	Passed bool

	// Summary maps each severity level to the number of findings observed.
	Summary map[models.Severity]int

	// Violations contains human-readable descriptions of breached thresholds.
	Violations []string
}

// Evaluate checks every finding in the report against the policy thresholds.
// A threshold of -1 means "unlimited" — that severity is never a violation.
func Evaluate(report *models.Report, policy *Policy) *Verdict {
	verdict := &Verdict{
		Passed:  true,
		Summary: make(map[models.Severity]int),
	}

	// Count findings per severity.
	for _, f := range report.Findings {
		verdict.Summary[f.Severity]++
	}

	// Ordered list so output is deterministic.
	order := []models.Severity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
		models.SeverityInfo,
	}

	for _, sev := range order {
		threshold, exists := policy.Thresholds[sev]
		if !exists {
			// No threshold defined for this severity — treat as unlimited.
			continue
		}
		if threshold == -1 {
			// Explicitly unlimited.
			continue
		}

		count := verdict.Summary[sev]
		if count > threshold {
			verdict.Passed = false
			verdict.Violations = append(verdict.Violations, fmt.Sprintf(
				"%s: found %d (max allowed %d)",
				sev, count, threshold,
			))
		}
	}

	return verdict
}
