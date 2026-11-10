# v1.3.0 (local, unpublished)

Added policy-change impact analysis in CLI and UI: effective limit changes,
per-report verdict transitions, raw policy/report hashes and review-required
status whenever a limit is relaxed. Comparison validates the whole bounded
corpus before returning evidence. Added reproducible synthetic fixtures and a
policy-review CI example. Existing evaluation exits and Slack scope are retained.

Also adds white/black themes, stricter integer/UTF-8 validation, stale-upload
export protection and session input retention, with regression and fuzz tests.

# v1.2.1

Removed unrelated project references, multi-project smoke branches, unused
connection/folder setup and legacy table reporter. Corrected diagrams and study
notes to the active normalized policy gate. Renamed launcher overrides to
VANGUARD_*. Clarified session-only decisions and guide-only persistence.
Included CLI policy/report fixtures and the CI step example in portable ZIPs.
Fixed case-variant duplicate JSON keys silently replacing findings or severity.

# v1.2.0

Guided first run, intentionally failing offline demo, normalized report/policy
downloads, remembered guide completion and safe in-session guide restart.

# v1.1.0

Local UI, bounded input validation, deterministic decisions, explicit CLI exits,
optional Slack notification and portable release automation.
