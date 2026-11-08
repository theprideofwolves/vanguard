# Vanguard v1.2.1

Portable single-user Windows/Linux policy evaluation: normalized JSON findings,
YAML severity thresholds, explained decisions with hashes, session UI/export,
CI exit codes and optional failure-only CLI Slack notification.

This patch removes unrelated project docs, unused connection/folder setup and
the old table reporter. It replaces diagrams claiming native scanner adapters
with the actual evaluation flow. Only VANGUARD_DATA_DIR (parent directory) and
VANGUARD_NO_BROWSER control the launcher; the default guide directory is unchanged.
Decisions are session-only. No database or saved assessment history is implemented.
Ambiguous case-variant duplicate JSON keys now fail validation instead of silently
replacing an earlier findings array or severity value.

Portable ZIPs include the executable, docs, policy.yaml, passing/failing reports,
the CI step example and dependency build metadata. ZIP checksums are published.
No developer tools, account, database server or paid API are needed offline.

Automated gates: race-enabled Go tests, vet, Windows/Linux builds and packaged
welcome/setup/demo/PASS/FAIL/export/session-restart/invalid-input checks. Tests
exercise policy boundaries, schema validation, exit codes and Slack contracts.
Live Slack delivery, human accessibility, clean-machine interactive launch and
sustained resource measurements remain separate validation. Test success does
not establish production deployment or exhaustive edge-case coverage.

The binaries are unsigned. The CI example expects an installed verified binary
and an already normalized upstream report; it does not run a scanner itself.
