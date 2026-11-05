# Vanguard
### Policy evaluation components for security findings
A Go prototype that compares normalized JSON security findings against YAML severity thresholds and produces a consistent pass/fail summary.

## What I built
- File and stdin reader components for normalized reports.
- JSON parsing with lowercase severity normalization.
- YAML policy loading and per-severity finding counts.
- Threshold evaluation with explicit policy violations.
- A table reporter with a fixed severity order and sample pass/fail reports.

## Architecture
`normalized JSON -> parser -> finding counts + YAML policy -> verdict -> table reporter`

The repository contains reusable internal packages. A command-line entry point and CI exit-code handling are not currently included. The diagrams illustrate the intended pipeline.

![Architecture](assets/architecture.png)

## Inspect the components
Requires Go 1.25.5 or newer, matching `go.mod`.

```bash
git clone https://github.com/theprideofwolves/vanguard.git
cd vanguard
go mod download
go build ./...
```

This compiles the packages; it does not create a CLI executable.

| Package | Role |
| --- | --- |
| `internal/reader` | Open a file or stdin |
| `internal/parser` | Decode the normalized report and lowercase severities |
| `internal/models` | Report, finding, and severity types |
| `internal/evaluator` | Load YAML policy, count findings, return verdict |
| `internal/reporter` | Print severity table and violations |

## Policy example
```yaml
apiVersion: vanguard/v1
kind: SecurityPolicy
metadata:
  name: default-baseline
thresholds:
  critical: 0
  high: 2
  medium: 10
  low: -1
  info: -1
```

A count greater than its threshold fails evaluation. A threshold of `-1`, or an omitted threshold, is unlimited. Review `testdata/report-pass.json` and `testdata/report-fail.json` for sample inputs.

## Current scope
Input must follow the repository's normalized `Report` schema. Direct Snyk/Trivy adapters, automated remediation, and an executable CI gate are not implemented. Unknown severities are not rejected and fall outside the fixed five-level evaluation loop; policy validation is limited to required metadata fields.

## Resume bullets
- Implemented Go components that parse normalized JSON security findings, normalize severity values, and evaluate finding counts against YAML policy thresholds.
- Built deterministic severity summaries and policy-violation reporting, with sample pass/fail inputs for a security pipeline gate prototype.
