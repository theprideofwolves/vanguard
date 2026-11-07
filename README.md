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
objects are rejected; distinct locations/occurrences count separately. Unknown
severity, invalid/null findings, multiple documents and invalid policy fields fail.

Decision JSON includes policy/input SHA256 hashes, UTC time, policy name,
observed counts, limits, violations, and schema version. Hashes identify input;
they do not prove it was genuine or complete. An empty array passes applicable
thresholds but does not establish that a scanner actually ran successfully.
Session results are not persisted: export before closing.

Exit 0: pass. Exit 1: policy violation. Exit 2: invalid input/configuration,
read/write failure, or tool error. CLI stdout is JSON only when -json is selected.

## Slack (optional CLI feature)
Set VANGUARD_SLACK_WEBHOOK in the process environment; never commit it.
Use -notify to send a failure-only summary. Optional VANGUARD_RUN_URL must be
an HTTPS CI-run link. Only the policy hash and generic failure/run information
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

Data defaults to the OS user configuration directory / application name.
Set PORTFOLIO_DATA_DIR to change its parent; PORTFOLIO_NO_BROWSER=1 suppresses
browser opening. Only one launcher may use a data directory. After a crash,
confirm the process has stopped before removing application.lock.
Stop the app before backup/restore. Replace executables to upgrade; user data
stays separate. Uninstall by deleting extracted binaries; delete the data
directory separately only if you intend to remove history.

## Release verification
CI runs unit/integration tests, race detection, vet, and builds on Windows/Linux.
Version tags or an explicit [release] commit on main publish the VERSION-selected portable ZIPs and SHA256 checksums after tests pass.
Dependency metadata is included using go version -m; it is not a full license
inventory or a signed supply-chain attestation. No unsupported scale or
enterprise-certification claim is made.

## Developer checks
`go test -race ./...`, `go vet ./...`, `go build ./cmd/vanguard`.
See architecture.md, interviewprep.md and explaining-each-decision-taken-over-others.md.

## Guided first run (v1.2.0)

On a new installation, choose **Try the guided demo** or **Use my data**.
The demo runs offline and explains its expected result. Setup checks are separate
from scans/evaluations and never create tickets. After a successful task, the
workspace shows the next step and remembers guide completion in onboarding.json.
That file contains only a version and completion flag, never credentials.
Use **Restart the guide** at any time; it does not reset existing reports.
**Continue to existing workspace** opens saved work without rerunning a task.

Jira-enabled tools offer a read-only sign-in check against /rest/api/3/myself.
This verifies authentication only, not project creation/comment permissions.
The check does not retain email/API tokens. Re-enter credentials when sending
a reviewed finding. The AWS setup check in S3 Auditor verifies account identity
only; it does not claim bucket access is verified.

CredDrift offers an explicit native folder picker, a readability check, and
transfer of the selected path into the scan form. On Linux the optional picker
uses zenity; if unavailable, paste an absolute path instead. Vanguard and LogFlow
offer downloadable example inputs. Vanguard evaluation needs no connection;
optional Slack remains a CLI feature and this guide never sends Slack messages.

A corrupt/missing guide-preference file shows the guide again without changing
application results. If progress cannot be saved, the guide says so; exports and
existing domain data are not discarded. Setup is single-user and local.
