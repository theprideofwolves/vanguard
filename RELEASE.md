# Vanguard v1.4.0 source candidate

Status: new pinned-review functionality is implemented and compilation checked.
Manual verification is pending. Tests, benchmarks, live integrations and CI were
not run for this extension, at the owner's request. No binary release is published.

- `review` loads a bounded baseline policy from an exact local Git commit/blob.
  It does not fetch, check out or execute candidate code. Commit trust is supplied
  by the operator/CI context, not established through hashing alone.
- Review mode flags effective relaxations; shadow mode preserves the pinned
  baseline's PASS/FAIL across supplied reports while showing candidate outcomes.
- JSON/Markdown and an optional new-directory evidence pack carry source context,
  paired decisions, policy snapshots, report hashes and artifact checksums.
- Ordinary compare and UI comparisons can export readable Markdown, explicitly
  labeled as supplied policies without pinned-Git provenance.
- The evaluator, original exit contracts, resource caps and failure-only CLI
  Slack scope are retained. New review/shadow workflows cannot send Slack.
- No AI dependency, provider account, database, hosted service or API-key field.

The new CI file is a template under examples, not an installed workflow. Existing
CI configuration remains available for later verification. This publication uses
[skip ci] and creates no release tag. Historical November 10 measurements remain
scoped to the commits recorded with them and do not verify this candidate.

Start with docs/MANUAL-REVIEW.md and record actual results before marking the new
workflow verified. Portable packaging includes the new guides/template when a
future explicit release is requested; builds remain unsigned.
