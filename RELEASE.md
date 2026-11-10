# Vanguard v1.3.0 — local release candidate

Policy-change impact analysis adds review of current/proposed severity limits and
per-report verdict transitions. It identifies newly passing reports, including
cases where aggregate PASS counts hide changed admissions. Any effective relaxation
requires review even if supplied samples show no verdict change.

- CLI: `vanguard compare -base current.yaml -candidate proposed.yaml [-json] reports-dir`
  (or one report file). Exits 0=no relaxed limits, 1=review required, 2=error.
- UI: compare a proposed policy on the current pasted/uploaded report. Export only
  the latest valid evaluation or comparison; invalid input clears stale evidence.
- Evidence: versioned impact JSON, exact-byte policy/report hashes, ordered limit
  changes and paired ordinary decisions. Omitted/-1 limits compare as unlimited.
- Resources: 32 reports / 64 MiB corpus, 16 MiB per report, 1 MiB per policy.
- Existing evaluation/stdin/0-1-2 exits and failure-only CLI Slack are preserved.
  Comparison never sends Slack. White and true-black themes share a teal accent.

Portable packaging includes policy/report fixtures, the four-report impact demo,
CI step fragments, testing instructions and interview/architecture notes. The
workflow is configured for Windows/Linux; local verification is Windows unless
TESTING-NOTES.md records otherwise. Builds are unsigned.

This version is not published by this local change. Unit/integration/race/fuzz and
packaged smoke checks provide bounded evidence, not exhaustive coverage or proof
of enterprise readiness. Human UI/accessibility, clean-machine launch, live Slack
and sustained performance remain separate checks. Vanguard is a normalized policy
gate and review aid; it does not scan, approve policy changes or certify compliance.
