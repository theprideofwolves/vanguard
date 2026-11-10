# A policy change that hides behind unchanged totals

All reports here are synthetic. No scanner, account or external service is used.
Build from the repository root (`-o vanguard` on Linux):

```sh
go build -o vanguard.exe ./cmd/vanguard
./vanguard.exe compare -base testdata/impact/base.yaml -candidate testdata/impact/candidate.yaml testdata/impact/reports
```

The current policy allows critical `0`, high `1`. The proposed policy allows
critical `1`, high `0`: it relaxes critical while tightening high.

| Report | Critical / high | Current | Proposed |
| --- | --- | --- | --- |
| `01-clean.json` | 0 / 0 | PASS | PASS |
| `02-critical.json` | 1 / 0 | FAIL | PASS |
| `03-high.json` | 0 / 1 | PASS | FAIL |
| `04-mixed.json` | 1 / 2 | FAIL | FAIL |

Both policies pass two reports, but they pass different reports. The critical
report becomes passing without any change to its findings. The output identifies
that transition rather than reducing the review to the aggregate PASS count.
It returns **REVIEW REQUIRED**, exit `1`, because critical was relaxed.

To inspect the evidence, repeat with `-json` before the report directory. The
`vanguard-impact/v1` JSON contains policy hashes, ordered limit changes, the four
transition counts, and full ordinary decisions under both policies for each report.
`Base.InputHash` and `Candidate.InputHash` must match within each report pair.

Compare a stricter proposal instead:

```sh
./vanguard.exe compare -base testdata/impact/base.yaml -candidate testdata/impact/tightened.yaml testdata/impact/reports
```

This returns **NO RELAXED LIMITS**, exit `0`, with one newly failing report.
Exit `0` does not mean every report passes or that the policy is approved.

An unchanged policy also returns `0`. A relaxation returns `1` even when a clean
sample report passes both policies. Malformed input returns `2` with no valid
impact JSON. The sample corpus does not establish effects on unseen reports,
report authenticity, actual exploitability or compliance.

The UI's **Compare a proposed policy** option performs the same analysis on one
pasted or uploaded report. Its built-in example changes only critical `0` to `1`.
The CLI accepts one report file or a directory of top-level `.json` files,
bounded to 32 reports and 64 MiB total. It does not recurse or follow symlink
entries. File labels are sorted; duplicate report bytes with different labels
remain separate samples, not a combined vulnerability count.
