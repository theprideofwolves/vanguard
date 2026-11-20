# Pinned policy review and shadow mode

Status: implemented, compilation checked, manual verification pending. No new
test, benchmark, live-integration or CI result is claimed for this extension.

Vanguard remains a deterministic policy gate. This workflow adds a baseline from
an exact local Git commit, a readable review and an optional evidence pack. No
AI API, account, scanner, server, paid service or notification is required.

## Run it locally

Build with Go 1.25.5+ and install Git 2.45+ (including `--no-lazy-fetch` support). Use `vanguard` instead of `vanguard.exe` on
Linux. From the repository root in PowerShell:

```powershell
go build -trimpath -o vanguard.exe ./cmd/vanguard
$baseline = git rev-parse HEAD
.\vanguard.exe review -repo . -base-ref $baseline -base-policy testdata/impact/base.yaml -candidate testdata/impact/candidate.yaml testdata/impact/reports
$LASTEXITCODE
```

Expected behavior, not a recorded result: critical is relaxed, high is tightened,
one report becomes passing and another becomes failing. Both policies still pass
two reports. Review mode should return `1` because a limit is relaxed.

`HEAD` above is only a local demonstration of resolving a full commit ID. In CI,
obtain the baseline ID from a trusted event or protected configuration. Do not let
the proposed policy select its own baseline. The command accepts full lowercase
commit IDs, not branch names, tags, `HEAD`, short hashes or revision expressions.

The tool reads local commit/tree/blob objects with bounded, timed Git commands.
It does not fetch, check out, edit a worktree, follow Git symlink entries, invoke
Git text-conversion filters or execute repository scripts. Git replacement objects
and lazy fetching of missing objects are disabled for these reads. The same policy path in the working tree can differ
without changing the selected committed baseline.

Pinning identifies exact bytes. It does not verify commit signatures, repository
ownership, branch protections or human approval. The local OS user and installed
Git executable remain trusted. Git blob IDs and content SHA256 hashes serve
different purposes and are both recorded.

## Choose the exit contract

| Command/mode | Exit 0 | Exit 1 | Exit 2 |
| --- | --- | --- | --- |
| Normal evaluation | Report meets policy | Policy FAIL | Input/tool/output error |
| `compare` or `review -mode review` | No effective limit relaxed | Relaxation requires review | Error |
| `review -mode shadow` | Every report meets pinned baseline | At least one baseline FAIL | Error |

Shadow mode records candidate outcomes without allowing the candidate to replace
the active baseline gate. A stricter candidate may fail a report while shadow
still returns `0`; a relaxed candidate may pass a report while shadow returns `1`.
`ReviewRequired` remains visible in either mode. It is not approval.

This is offline shadow evaluation on supplied reports, not continuous production
traffic mirroring. Assessment success and corpus selection remain upstream.

## Export a pack

Choose a new output directory whose parent exists:

```powershell
New-Item -ItemType Directory -Force tmp | Out-Null
.\vanguard.exe review -base-ref $baseline -base-policy testdata/impact/base.yaml -candidate testdata/impact/candidate.yaml -output tmp/review-first testdata/impact/reports
```

The pack contains:

- `review.md`: readable limits, verdict transitions, observed counts and hashes.
- `review.json`: `vanguard-review/v1`, pinned source, mode, result status, build
  metadata and complete paired decisions.
- `base-policy.yaml` and `candidate-policy.yaml`: exact policy snapshots.
- `manifest.json`: artifact hashes and report-name/input-hash references.

The directory is created exclusively. Existing directories are never reused.
The manifest is renamed into place only after artifact writes complete. On an
interruption/error, retain the incomplete directory for inspection and choose a
new destination. A missing manifest or any artifact hash mismatch means the pack
must not be treated as complete. This is not a transactional database or signed
audit trail, and cannot resist a privileged local user modifying the files.

Reports and finding descriptions are not copied into the pack. File labels,
policy names and policy snapshots can still contain private information. Review
them before publishing. Retain authorized source reports separately to reproduce
evaluation. A complete pack does not prove those reports were genuine or complete.
Output failures return `2`, even if some files or stdout bytes were already written.

## Readable comparison without Git

```powershell
.\vanguard.exe compare -base policy.yaml -candidate testdata/impact/candidate.yaml -markdown testdata/report-fail.json
```

The UI also offers **Download readable review (Markdown)** after comparison.
Both exports explicitly state that their baseline is supplied data, not pinned
Git. Markdown labels are escaped so policy names and report names remain text.
Invalid evaluation/comparison clears both JSON and Markdown session exports.

## CI trust boundary

[github-trusted-policy-review.yml](../examples/github-trusted-policy-review.yml)
is an unexecuted template. It reads the baseline and tool from the PR base commit,
checks out only the proposed policy as data, uses a base-controlled corpus and
publishes an artifact/step summary without PR-comment or repository-write access.
It never builds code from the candidate checkout. It does not use secrets,
`pull_request_target`, automatic policy updates or notification APIs.

Protect the workflow, baseline selection, tool supply chain, required checks and
approval rules outside the PR author's control. A PR-editable workflow alone does
not enforce this. Pin action versions under the consuming repository's standards.
Replace the synthetic corpus with authorized, successfully assessed reports.
Select GitHub or another CI runner under your own account/compute budget; running
Vanguard locally does not require hosted CI. The template is not activated here.

## Operational boundaries

The existing limits remain: 1 MiB/policy, 16 MiB/report, 32 reports and 64 MiB total
corpus, with bounded nonrecursive intake. Git reads time out after 10 seconds.
The pure gate and existing evaluation/compare contracts are unchanged. Review and
shadow operations never send Slack. They do not modify or approve policies.

See [the manual acceptance guide](MANUAL-REVIEW.md). Do not reuse earlier coverage
percentages or timing claims as verification of this new code.
