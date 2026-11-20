package review

import (
	"fmt"
	"html"
	"strings"
	"unicode"

	"github.com/theprideofwolves/vanguard/internal/gate"
)

// Untrusted labels must remain text, not headings, links, images or raw HTML.
func markdownText(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteByte(' ')
		case unicode.IsControl(r) || unicode.Is(unicode.Cf, r):
			fmt.Fprintf(&b, `\u%04x`, r)
		case r == '|':
			b.WriteString("&#124;")
		case r == ':' || r == '.' || r == '@':
			// Prevent GFM from turning a plain URL/email label into an autolink.
			fmt.Fprintf(&b, "&#%d;", r)
		case strings.ContainsRune("\\`*_[]()#!~", r):
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteString(html.EscapeString(string(r)))
		}
	}
	return b.String()
}

func limitText(limit gate.Limit) string {
	if !limit.Limited {
		return "unlimited"
	}
	return fmt.Sprint(limit.Allowed)
}

func verdict(passed bool) string {
	if passed {
		return "PASS"
	}
	return "FAIL"
}

func writeImpact(b *strings.Builder, impact *gate.PolicyImpact) {
	status := "NO RELAXED LIMITS"
	if impact.ReviewRequired {
		status = "REVIEW REQUIRED"
	}
	fmt.Fprintf(b, "## Policy change: %s\n\n", status)
	fmt.Fprintf(b, "Current policy: %s. Proposed policy: %s.\n\n", markdownText(impact.BasePolicyName), markdownText(impact.CandidatePolicyName))
	b.WriteString("| Severity | Current limit | Proposed limit | Change |\n| --- | ---: | ---: | --- |\n")
	for _, row := range impact.Limits {
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", row.Severity, limitText(row.Base), limitText(row.Candidate), row.Direction)
	}
	s := impact.Summary
	fmt.Fprintf(b, "\nReports: %d. Current PASS: %d. Proposed PASS: %d.\n\nNewly passing: %d. Newly failing: %d. Unchanged PASS: %d. Unchanged FAIL: %d.\n\n", s.Reports, s.BasePassed, s.CandidatePassed, s.NewlyPassing, s.NewlyFailing, s.UnchangedPassed, s.UnchangedFailed)
	b.WriteString("| Report | Current | Proposed | Transition |\n| --- | --- | --- | --- |\n")
	for _, report := range impact.Reports {
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", markdownText(report.Name), verdict(report.Base.Passed), verdict(report.Candidate.Passed), report.Transition)
	}
	for _, report := range impact.Reports {
		fmt.Fprintf(b, "\n### Report: %s\n\n", markdownText(report.Name))
		if report.Transition == "fail_to_pass" {
			b.WriteString("This report becomes passing without changes to its findings. Previously exceeded limits are now permitted.\n\n")
		}
		b.WriteString("| Severity | Observed | Current exceeded | Proposed exceeded |\n| --- | ---: | --- | --- |\n")
		for i, row := range report.Base.Rows {
			fmt.Fprintf(b, "| %s | %d | %t | %t |\n", row.Severity, row.Observed, row.Exceeded, report.Candidate.Rows[i].Exceeded)
		}
		fmt.Fprintf(b, "\nInput SHA256: `%s`\n", report.Base.InputHash)
	}
	fmt.Fprintf(b, "\n## Evidence\n\nUTC: %s\n\nCurrent policy SHA256: `%s`\n\nProposed policy SHA256: `%s`\n\n", impact.At.Format("2006-01-02T15:04:05.999999999Z07:00"), impact.BasePolicyHash, impact.CandidatePolicyHash)
	b.WriteString("## Limits of this review\n\nResults cover only the supplied reports. Missing findings, failed upstream assessments and incomplete corpora cannot be inferred from a valid JSON array. Hashes identify bytes, not authorship, completeness or approval. A relaxed limit needs human review; this report does not approve, merge or deploy a policy. No scanner, notification or AI service is invoked.\n")
}

func Markdown(result *Result) string {
	var b strings.Builder
	b.WriteString("# Vanguard policy review\n\n")
	fmt.Fprintf(&b, "Schema: `%s`. Mode: **%s**. Result: **%s**. Result exit code: **%d**.\n\n", result.Version, result.Mode, result.Status, result.ResultExitCode)
	if result.Mode == "shadow" {
		b.WriteString("The proposal is advisory. The exit status follows the pinned current policy across all supplied reports. Candidate results do not replace that gate.\n\n")
	} else {
		b.WriteString("The exit status flags relaxed limits. It is not the candidate policy's PASS/FAIL and does not grant approval.\n\n")
	}
	fmt.Fprintf(&b, "## Pinned baseline\n\nCommit: `%s`\n\nPolicy path: %s\n\nGit blob: `%s`\n\nCandidate file label: %s\n\n", result.BaseSource.Commit, markdownText(result.BaseSource.Path), result.BaseSource.Blob, markdownText(result.CandidateLabel))
	b.WriteString("The commit was supplied by the caller and read from local Git objects. The caller must select it from a trusted review context. This is not a signature or branch-protection check.\n\n")
	writeImpact(&b, result.Impact)
	return b.String()
}

// MarkdownImpact is for ordinary file/UI comparisons without Git provenance.
func MarkdownImpact(impact *gate.PolicyImpact) string {
	var b strings.Builder
	b.WriteString("# Vanguard policy comparison\n\nBaseline source: supplied file or current browser session, not pinned Git. Use the review command when CI needs an exact baseline revision.\n\n")
	writeImpact(&b, impact)
	return b.String()
}
