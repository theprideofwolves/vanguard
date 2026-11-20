# Manual acceptance: pinned policy review

Status: **NOT RUN**. Commands below are instructions for the project owner, not
evidence that these checks passed. Build compilation is the only check performed
while adding this extension. Keep earlier recorded metrics separate.

## 1. Prepare a local session

From PowerShell in the repository root:

```powershell
go build -trimpath -o vanguard.exe ./cmd/vanguard
if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
$baseline = (git rev-parse HEAD).Trim()
New-Item -ItemType Directory -Force tmp | Out-Null
```

Record the full source commit, Go/Git versions (Git 2.45+ required), date and platform. Keep screenshots
free of private paths, report details, tokens and browser launcher URLs. No API
key is needed. All examples use the existing synthetic corpus.

## 2. Review the mixed policy change

```powershell
.\vanguard.exe review -base-ref $baseline -base-policy testdata/impact/base.yaml -candidate testdata/impact/candidate.yaml testdata/impact/reports
$LASTEXITCODE
```

Expected: exit `1`, current critical `0` to proposed `1` is relaxed; high `1` to `0`
is tightened. Current and proposed PASS totals are both `2`, with one newly passing
and one newly failing report. Capture the summary and per-report table.

Repeat with `-candidate testdata/impact/tightened.yaml`. Expected: exit `0` and
NO RELAXED LIMITS, even though some reports fail the policy. Compare the exact
baseline commit/blob and policy SHA256 shown in the review.

## 3. Confirm shadow keeps the baseline gate

```powershell
.\vanguard.exe review -base-ref $baseline -base-policy testdata/impact/base.yaml -candidate testdata/impact/candidate.yaml -mode shadow testdata/impact/reports/03-high.json
$LASTEXITCODE
```

Expected: current PASS, proposed FAIL, exit `0`. Now use `02-critical.json`.
Expected: current FAIL, proposed PASS, exit `1`. The candidate cannot make the
active gate pass. Capture both outcomes. Run review mode on the same files to
confirm that review status and baseline gate status have different meanings.

## 4. Export and inspect a new evidence pack

```powershell
.\vanguard.exe review -base-ref $baseline -base-policy testdata/impact/base.yaml -candidate testdata/impact/candidate.yaml -output tmp/review-first testdata/impact/reports
$LASTEXITCODE
Get-ChildItem tmp/review-first
```

Expected: exit `1` and five files: review Markdown/JSON, two exact policy snapshots
and `manifest.json`. Input reports must not be copied. Review JSON should identify
mode, status, generator metadata, exact source and paired ordinary decisions.

Check hashes from your own generated manifest:

```powershell
$pack = 'tmp/review-first'
$manifest = Get-Content "$pack/manifest.json" -Raw | ConvertFrom-Json
$manifest.Artifacts | ForEach-Object {
    $actual = (Get-FileHash (Join-Path $pack $_.Name) -Algorithm SHA256).Hash.ToLowerInvariant()
    [pscustomobject]@{ File = $_.Name; HashMatches = ($actual -eq $_.SHA256) }
}
```

Expected: every hash matches. Repeat the export to the same directory. Expected:
exit `2`, with the original pack unchanged. Change a COPY of an artifact and check
that its hash no longer matches. A hash match alone does not establish approval.

## 5. Prove the working tree cannot replace the pinned baseline

Use an isolated clone. Do not edit or reset the original repository:

```powershell
git clone --no-hardlinks . tmp/review-lab
```

Edit only `tmp/review-lab/testdata/impact/base.yaml`, changing critical to `99`.
Run review with `-repo tmp/review-lab`, the same full `$baseline` and base-policy
path. Expected: the review still uses committed critical `0`. Save the edited
file and review screenshot together to explain the trust boundary.

The tool still depends on the operator choosing a trusted commit. Selecting an
attacker-controlled commit is not made safe by hashing it.

## 6. Exercise failure and rendering cases

Use new output destinations for each case. Inspect `$LASTEXITCODE` immediately.

| Case | Expected behavior | Recorded status |
| --- | --- | --- |
| `-base-ref HEAD`, short hash or revision expression | Error `2`; no fallback to branch/worktree | NOT RUN |
| Missing commit/object or missing policy at that commit | Error `2` | NOT RUN |
| `-base-policy ../policy.yaml` or absolute path | Error `2` | NOT RUN |
| Git symlink policy entry, where the OS/lab permits creating one | Rejected; otherwise record N/A with reason | NOT RUN |
| Missing Git executable/repository | Visible error `2`, no misleading result | NOT RUN |
| Malformed/null proposed policy or report | Error `2`, no pack created | NOT RUN |
| Empty directory, too many/oversized reports | Error `2`, no partial corpus presented as complete | NOT RUN |
| Existing output directory or missing output parent | Error `2`, no overwrite | NOT RUN |
| Invalid mode/format; `-notify` passed to review | Error `2`, no notification | NOT RUN |
| Link/HTML/table/newline content in policy/report names | Rendered as text, no injected links/headings/HTML | NOT RUN |
| Changed copy of pack file | Manifest hash mismatch | NOT RUN |
| Incomplete pack without manifest | Must not be accepted as complete | NOT RUN |

For the existing report/policy byte boundaries, original evaluation and live Slack
checklist, use [TESTING.md](../TESTING.md). A new live Slack test needs a private
test channel and your own webhook. Review/shadow mode must never send Slack.

## 7. Inspect the UI export

Launch the UI as described in TESTING.md, compare policies and download JSON and
**Download readable review (Markdown)**. Expect matching counts/hashes and an
explicit notice that the UI baseline is supplied data, not a pinned Git source.
Change theme and restart the guide: current comparison exports should remain.
Submit invalid input: neither export should remain available. A fresh application
session must not retain the decision. Check white/black modes and keyboard use.

## 8. Review CI without activating it first

Read [the template](../examples/github-trusted-policy-review.yml). Confirm the
tool/corpus/baseline come from base code, candidate policy is only read as data,
permissions are read-only, no secrets/PR comments are used, and errors stay visible.
Choose your baseline/workflow protection and authorized corpus before installing
it in a test repository. Do not claim a hosted CI pass from reading the YAML.

## 9. Record evidence

Copy [the result template](verification/manual-policy-review/TEMPLATE.md), fill in
actual outcomes and attach sanitized screenshots. Mark failed, skipped and N/A
cases explicitly. Separate synthetic cases, real normalized reports, mocked
integration checks and live delivery. Do not edit old November 10 results to imply
they tested this code. Publish new results only after reviewing the evidence.

For a short video: show the policy diff, mixed-result review, the two shadow
outcomes and the manifest. Narrate that this is deterministic policy evaluation,
not scanning, AI reasoning, approval or containment.
