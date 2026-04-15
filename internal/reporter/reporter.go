package reporter

import (
	"fmt"
	"io"
	"text/tabwriter"

	"vanguard/internal/evaluator"
	"vanguard/internal/models"
)

// PrintTable writes a human-readable severity breakdown to w.
// Each row shows the severity, observed count, policy threshold, and status.
func PrintTable(verdict *evaluator.Verdict, policy *evaluator.Policy, w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	fmt.Fprintln(tw, "SEVERITY\tCOUNT\tMAX\tSTATUS")
	fmt.Fprintln(tw, "--------\t-----\t---\t------")

	order := []models.Severity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
		models.SeverityInfo,
	}

	for _, sev := range order {
		count := verdict.Summary[sev]
		threshold, exists := policy.Thresholds[sev]

		maxStr := "∞"
		status := "OK"

		if exists && threshold != -1 {
			maxStr = fmt.Sprintf("%d", threshold)
			if count > threshold {
				status = "FAIL"
			}
		}

		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", sev, count, maxStr, status)
	}

	tw.Flush()

	fmt.Fprintln(w)
	if verdict.Passed {
		fmt.Fprintln(w, "Result: PASS")
	} else {
		fmt.Fprintln(w, "Result: FAIL")
		for _, v := range verdict.Violations {
			fmt.Fprintf(w, "  ✗ %s\n", v)
		}
	}
}
