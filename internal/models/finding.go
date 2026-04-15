package models

// Severity represents the severity level of a security finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Finding represents a single normalized vulnerability from a security scan.
type Finding struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Severity       Severity `json:"severity"`
	PackageName    string   `json:"package_name"`
	PackageVersion string   `json:"package_version,omitempty"`
	FixedVersion   string   `json:"fixed_version,omitempty"`
	CVSSScore      float64  `json:"cvss_score,omitempty"`
	CVE            string   `json:"cve,omitempty"`
	Description    string   `json:"description,omitempty"`
	References     []string `json:"references,omitempty"`
}

// Report is the top-level structure representing a normalized security scan report.
type Report struct {
	Tool        string    `json:"tool"`
	GeneratedAt string    `json:"generated_at,omitempty"`
	Target      string    `json:"target"`
	Findings    []Finding `json:"findings"`
}
