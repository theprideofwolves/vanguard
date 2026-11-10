# Verification and defensible metrics

Vanguard evaluates supplied normalized findings; it is not a malware detector.
Report correctness, measured cost and tested behaviors. Do not describe synthetic
tests as detection accuracy, production adoption, prevented incidents or time saved.

## Reproduce the evidence

Requires Go 1.25.5+, a C compiler for race detection, Python 3 and Bash (Git Bash
on Windows). From the repository root, choose a new output directory:

```sh
python .github/scripts/verify.py --output tmp/verification-review
```

The default run performs vet, a race-enabled full test suite with whole-project
statement coverage, a compiled CLI/UI smoke test, local execution of both CI
example blocks, three 15-second fuzz runs, and six in-process benchmark workloads
with five samples each. It disables browser opening and Slack, uses isolated guide
and temporary data, and records scrubbed logs, JSON metrics and a functionality
matrix in `RESULTS.md`. It invokes no scanner or deployment. Existing output
directories are never overwritten; use a new name for each run.

GitHub Actions runs the same verifier on Windows/Linux with shorter 10-second fuzz
budgets and three 250ms benchmark samples, and uploads `verification-*` artifacts.
Check each run's status before claiming a platform passed. Timing variation is
expected on shared runners; benchmark timing is not a CI acceptance threshold.
Source/binary release publication remains a separate workflow condition.

## What the metrics mean

| Metric | Evidence and interpretation |
| --- | --- |
| Functional checks | Named Go tests/subtests and compiled workflows; generated model-check cases are counted separately. |
| Model agreement | 2,048 evaluation and 1,024 comparison cases with fixed seeds, checked against an independent accepted-count-set model. Bounded input space, not exhaustive proof. |
| Statement coverage | `go test -race -coverpkg=./... -coverprofile=...`; numerator/denominator include production code across packages, including launcher code. Not a correctness score. |
| Fuzz executions | Recorded from each actual run. Counts depend on caches/hardware; no crash found in a bounded run does not mean no vulnerabilities exist. |
| Evaluation cost | 100, 1,000 and 10,000 unique synthetic findings/report. Includes parse, schema/duplicate validation, counting and hashing. |
| Comparison cost | 1, 8 and 32 reports, each with 1,000 findings. Includes both policy decisions and report ordering. |
| Memory allocations | Go `B/op` and `allocs/op`, not peak RSS or maximum supported deployment memory. |
| Reproducibility | Source commit plus normalized source fingerprint, commands, toolchain, platform and raw samples. Documentation/evidence are excluded from the fingerprint to avoid circular hashes. |

Benchmark fixtures are generated before timing. Measurements exclude process
startup, disk I/O, export encoding and network; benchmark sample medians and ranges
are not per-request p95/p99 latency. There is no comparison against a competing tool.
The four-report policy demo is a case study, not a statistical accuracy benchmark.

## Remaining human checks

Follow [TESTING.md](../TESTING.md) for visual white/black theme review, keyboard and
narrow-screen usability, normal browser opening and clean-machine shutdown. Live
Slack delivery is deliberately not exercised; mocked transport behavior is recorded.
No sustained load test, external assessment authenticity check or production-impact
study is included. Do not claim all possible inputs or environments were tested.
