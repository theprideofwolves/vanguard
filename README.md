# Vanguard

A small Go policy gate: normalized findings + YAML thresholds -> explained
pass/fail decision. It does not scan vulnerabilities or certify compliance.

## Two workflows
1. Local UI: double-click, inspect the synthetic example, replace the report
and policy, evaluate, and download a decision JSON.
2. CI: run the same evaluation with explicit exit codes and optional Slack.

```
vanguard -p policy.yaml report.json
vanguard -p policy.yaml -json report.json
vanguard -p policy.yaml -json -notify report.json
vanguard -p policy.yaml -json -
```
No arguments opens the UI; `-` reads stdin explicitly. Reports are capped at
16 MiB and policies at 1 MiB. Input is a normalized report, not arbitrary native
Trivy/Snyk output. See testdata and policy.yaml. Exactly equal counts pass.
Omitted thresholds and -1 are unlimited. Exact duplicate normalized finding
objects are rejected; distinct normalized finding objects count separately. Unknown
severity, invalid/null findings, multiple documents and invalid policy fields fail.
Ambiguous duplicate JSON keys are rejected even when their casing differs.

Decision JSON includes policy/input SHA256 hashes, UTC time, policy name,
observed counts, limits, violations, and schema version. Hashes identify input;
they do not prove it was genuine or complete. An empty array passes applicable
thresholds but does not establish that a scanner actually ran successfully.
Session results are not persisted: export before closing.
Larger file uploads can use OS temporary files during the request; those files
are removed after processing. Reports are not saved as application history.

Exit 0: pass. Exit 1: policy violation. Exit 2: invalid input/configuration,
read/write failure, or tool error. CLI stdout is JSON only when -json is selected.

## Slack (optional CLI feature)
Set VANGUARD_SLACK_WEBHOOK in the process environment; never commit it.
Use -notify to send a failure-only summary. Optional VANGUARD_RUN_URL must be
an HTTPS CI-run link. Only the policy hash, exceeded severity counts/limits and run information
are sent, not finding descriptions or source. Official Slack/GovSlack webhook
origins only; redirects disabled; no automatic retries after uncertain delivery.
A notification failure is printed to stderr and never changes the policy verdict.
Successes are silent. See examples/github-action.yml for preserving exit status
while retaining the JSON report as an artifact.

## Installation and privacy
Download the platform ZIP from GitHub Releases and extract it. On Windows,
double-click the executable. On Linux, run the binary; if no desktop opener is
installed, copy the printed localhost URL into a browser. No development tools,
Docker, database server, or paid API is needed for the offline workflow.
The executable is unsigned: OS warnings are possible. Close the application with
Ctrl+C in its console. Closing the browser does not stop it.

The server binds to an ephemeral localhost port and uses a private per-launch
token, exact Host and Origin checks, CSP, HTML escaping and request limits.
Single-user OS trust boundary: no roles or enterprise multi-tenancy. Local
privileged processes can read memory and data. Do not expose it to a network.
No telemetry, background uploads, automatic remediation or AI dependency.

The OS user configuration directory / Vanguard stores only the guide preference
and launcher lock. VANGUARD_DATA_DIR changes its parent; VANGUARD_NO_BROWSER=1
suppresses automatic browser opening. Only one launcher may use that directory.
After a crash, confirm the process stopped before removing application.lock.
Upgrades replace extracted binaries. Uninstall by deleting those files; remove
the Vanguard data directory separately if you also want to reset guide progress.
Vanguard has no saved decision history or database to restore.

## Release verification
CI runs unit/integration tests, race detection, vet, and builds on Windows/Linux.
Version tags or an explicit [release] commit on main publish the VERSION-selected portable ZIPs and SHA256 checksums after tests pass.
Dependency metadata is included using go version -m; it is not a full license
inventory or a signed supply-chain attestation. No unsupported scale or
enterprise-certification claim is made.

## Developer checks
`go test -race ./...`, `go vet ./...`, `go build ./cmd/vanguard`.
See architecture.md, interviewprep.md and explaining-each-decision-taken-over-others.md.

## Guided first run

Choose **Try the guided demo** or **Use my data**. The offline demo intentionally
FAILS: one critical finding exceeds the allowed zero. FAIL is a valid policy
decision, not an application error. The guide completes after any valid
evaluation, whether PASS or FAIL. Invalid input does not mark completion.

Setup offers a normalized JSON report and YAML policy download. Paste your
report or upload it in the workspace, review the policy thresholds, and evaluate.
There are no accounts, credentials or connection checks for local evaluation.
The browser workflow never sends Slack notifications.

Only the version/completion flag is saved in onboarding.json. **Restart the
guide** and **Return to workspace** preserve the current session decision.
Decisions and report/policy inputs do not survive application restart; export
before closing. Invalid evaluations clear the previous decision to avoid a
stale export. Missing/corrupt guide preferences show the guide again; a progress
save failure is displayed without discarding the current valid decision.

Portable ZIPs include policy.yaml, passing/failing reports under testdata, and
examples/github-action.yml for trying CLI evaluation and adapting a CI step.
The CI example expects a verified Vanguard binary already on PATH and a report
normalized by your own upstream process.
