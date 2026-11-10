# Vanguard: interview guide

## Thirty-second explanation

Vanguard is a Go policy gate for normalized security findings. It validates JSON
reports and YAML severity thresholds, then returns an explained PASS/FAIL with
exact-byte hashes. Its distinguishing feature is policy-change impact analysis:
it evaluates the same reports under current and proposed policies, showing
relaxed/tightened limits and reports whose verdict changes. A relaxed limit needs
review even if a small sample shows no changes. It works locally and in CI; it is
not a scanner, policy approval system or compliance certification tool.

## A three-minute demonstration

Build and run the command in [examples/policy-review.md](examples/policy-review.md).
Explain the four synthetic reports before showing the result:

| Report | Current | Proposed | Meaning |
| --- | --- | --- | --- |
| Clean | PASS | PASS | Unchanged acceptance |
| One critical | FAIL | PASS | Newly admitted through relaxation |
| One high | PASS | FAIL | Newly rejected through tightening |
| Critical + two high | FAIL | FAIL | Still rejected, despite one relaxed limit |

Both policies pass two reports, so aggregate totals alone hide the changed
admissions. Current allows critical:0/high:1; proposed allows critical:1/high:0.
The output reports one newly passing and one newly failing report, and returns
exit 1 because critical was relaxed. This is a policy decision, not an application
error. It does not say that the proposed policy is always wrong; a human must
review whether the changed risk acceptance is justified.

Then compare base.yaml to tightened.yaml. Expect exit 0 but one newly failing
report. Explain why comparison exit 0 is different from an ordinary PASS.

For a visual demo, run the UI, choose the guided demo, inspect FAIL, then open
**Compare a proposed policy**. The built-in candidate changes critical:0 to 1;
the same report becomes PASS. Download impact JSON and point to matching input
hashes within the report pair. Use synthetic data only.

## Source walkthrough

1. `cmd/vanguard/main.go:run` dispatches `compare` before ordinary flags. Normal
   evaluation keeps `-p`, `-json`, optional `-notify`, explicit file/stdin and exits.
2. `cmd/vanguard/compare.go:runCompare` parses required base/candidate paths and
   report source. `readBoundedFile` rejects nonregular inputs and reads at most
   limit+1 bytes. `loadImpactReports` selects top-level JSON files and caps intake.
3. `internal/evaluator/policy.go:ParsePolicy` validates one YAML policy and its
   fields, version/kind and integer thresholds. KnownFields catches schema typos;
   custom threshold decoding prevents null/fractional coercion to integer limits.
4. `internal/parser/parser.go:Parse` bounds bytes, rejects invalid UTF-8, checks
   unique JSON keys before typed decoding, rejects unknown fields, normalizes
   severity and detects identical normalized finding objects.
5. `internal/gate/impact.go:ComparePolicies` validates both policies, sorts labels,
   compares effective limits and parses each report once. It evaluates both
   policies through `gate.go:decisionFor`, which uses the same evaluator as normal
   evaluation. There is no separate comparison threshold engine to drift.
6. `cmd/vanguard/ui.go:newUIAt` holds a mutex around evaluations, comparisons and
   export. It clears old evidence before parsing a new evaluation request, applies
   upload-over-paste precedence and exports the appropriate current JSON type.
7. `internal/desktop` protects localhost requests. `internal/onboarding` stores
   guide completion, not evidence. Theme changes use a protected form and cookie.

Read these files and explain the control flow yourself; memorizing this guide
without understanding the code will not help in a technical interview.

## Questions you should be able to answer

**Why add policy comparison instead of another scanner?**
A scanner answers what findings it observes. Vanguard answers what the policy
permits, and how a proposed change alters that permission. Keeping scanning and
normalization upstream makes offline, reproducible evaluation possible.

**What exactly counts as relaxation?**
Finite 0 to finite 1 is relaxed. Finite to unlimited is relaxed. Unlimited to finite
is tightened. Omitted and -1 are effectively equal, so adding/removing a -1 does
not count as weakening. A policy with both tightening and relaxation still needs
review. Changes to comments or names change raw hashes, not threshold semantics.

**Why review a change when no sample flips?**
Samples are incomplete. If critical:0 becomes critical:1, a clean report still
passes both policies, but a future critical finding could become permitted.
Structural comparison detects that independently of historical report coverage.

**Why not approve policies automatically?**
A threshold increase can be intentional and justified. ReviewRequired is a
conservative signal, not a risk score, approval or statement about actual
exploitability. A stricter policy can also break releases; newly failing reports
make that impact visible without claiming the change is inappropriate.

**How do you know a verdict flip came from the policy?**
The same validated report is passed to both evaluations and both decisions carry
the same raw input hash. That isolates the changed policy for this calculation.
It does not establish whether the report was authentic, complete or recent.

**What does the JSON contain?**
Impact version, UTC timestamp, base/candidate names and hashes, ReviewRequired,
five ordered effective-limit changes, report-transition counts and sorted report
pairs. Each pair contains ordinary decisions with input hashes, counts, limits
and violations. No finding descriptions are embedded in those decisions.
Unlimited comparison limits use Allowed:-1 and Limited:false. Existing ordinary
Decision rows keep their established encoding; Limited controls interpretation.

**What are your failure semantics?**
Ordinary evaluation: 0 PASS, 1 policy FAIL, 2 tool/input/output error.
Comparison: 0 no relaxation, 1 at least one relaxation, 2 error. Both policies and
all corpus members must validate; any invalid member returns no result. Output
failure can leave partial bytes on stdout/file, so exit 2 is never usable evidence.
The UI has one export slot for either kind of result. Invalid submissions clear
it to prevent a stale decision being mistaken for the new result.

**What do hashes prove? Are they signatures?**
SHA256 identifies exact input/policy bytes. Reformatting JSON or comments in YAML
changes the hash even if effective behavior is unchanged. Hashes are not digital
signatures and do not prove who produced a report or that a scanner succeeded.
An empty findings array can pass without establishing a successful assessment.

**What about duplicate findings and report files?**
Identical normalized finding objects within a report are rejected. Distinct modeled
objects count separately even if their IDs match. This is not cross-scanner CVE
correlation. Separate report labels with identical bytes are separate samples;
summary counts are not a statistical estimate or a count of unique vulnerabilities.

**How do you avoid JSON ambiguity?**
The parser token-walks objects before decoding and rejects repeated keys using
Unicode simple-fold equivalence, matching case-insensitive struct field matching.
This prevents an alternate Findings/Severity key from silently replacing a field.
Typed decoding rejects unknown fields; severity must be one of the five supported
values. UTF-8 validation prevents silent replacement of invalid input bytes.

**What is the resource cost?**
Up to 16 MiB/report, 1 MiB/policy, 32 reports and 64 MiB total corpus. Report labels
are sorted; directory intake is nonrecursive and bounded. Work is approximately
linear in input bytes/findings plus R log R for sorting labels. Reports are parsed
sequentially, but raw corpus bytes are retained in a bounded slice. Temporary
allocations mean these input caps are not peak-RSS measurements.

**How did you test the novelty?**
`internal/gate/impact_test.go` checks all four transitions, mixed changes,
finite/unlimited/equivalent limits, exact-byte hashes, agreement with ordinary
evaluation, empty/duplicate/invalid reports and count/byte boundaries. A fuzz
property checks that a policy with no relaxed limits cannot newly admit a report.
CLI tests check 0/1/2 and output errors; UI tests check replacement/clearing/export,
upload precedence and escaped policy names. Packaged smoke exercises the real
executable, sample corpus and HTTP/session behavior. Fuzzing is bounded, not proof
of exhaustive coverage. Consult ignored TESTING-NOTES.md for actual local results.

**What is protected in the UI?**
It binds to loopback with a random token and exact Host/Origin checks, CSP,
escaped HTML and body limits. The theme is an allowlisted HttpOnly, SameSite cookie.
It is single-user software, not an enterprise authorization boundary. A privileged
local process can inspect its memory and launcher token. Reports are not saved as
history; multipart uploads may temporarily use the OS temp directory.

**What is sent to Slack?**
Only optional failed ordinary CLI evaluations send a policy-hash/count/limit/run
summary. No finding descriptions or webhook secrets appear in it. HTTP timeout,
redirect suppression and no automatic retry limit delivery surprises. Notification
failure preserves the verdict. Comparison and browser evaluation never notify.
Tests use mock transports; they do not demonstrate real Slack delivery.

## Honest resume wording

- Built a Go security policy gate with strict JSON/YAML validation, versioned
  evidence exports and explicit CI exit codes.
- Implemented policy-change impact analysis that identifies relaxed limits and
  per-report verdict transitions, including changes hidden by aggregate PASS counts.
- Added regression, race, bounded fuzz and compiled-workflow checks for malformed
  input, threshold semantics and stale evidence protection.

Use only claims you can demonstrate from the current version. Do not claim reduced
incidents, adoption, time saved, deployment or a globally unique invention without
independent evidence. Explain AI assistance and your own design/testing work honestly.

## Discussing measured results

See [recorded verification](docs/verification/2026-11-10-windows/RESULTS.md) and
[methodology](docs/VERIFICATION.md). The run passed 37 top-level tests and 60
subtests, plus 2,048 evaluation and 1,024 comparison cases checked against a seeded
accepted-count-set model. Whole-project statement coverage was 85.5%, including
launcher code in the denominator. Bounded fuzz runs recorded 437,630 executions
without a failure. None of those figures is malware-detection accuracy.

The measured 10,000-finding report took 71.9 ms median across five Go benchmark
samples on Windows 11 / Ryzen 9 5980HX / Go 1.25.5. This includes parsing, schema
and duplicate validation, counting and hashing, but excludes process startup,
disk I/O and export serialization. It is not an end-to-end service latency or
p95/p99. Cumulative allocated bytes are not peak RAM. Explain the workload and
boundaries whenever quoting the timing.

A defensible resume bullet is:

> Implemented Go policy-change impact analysis with 3,072 seeded reference-model
> checks and 85.5% whole-project statement coverage; benchmarked in-process
> evaluation of 10,000 synthetic findings at 71.9 ms median across five samples.

Use the project link for the full hardware/methodology context. Do not convert
these results into a claim about production throughput, false-positive rate,
prevented incidents, time savings or Microsoft hiring outcomes.
