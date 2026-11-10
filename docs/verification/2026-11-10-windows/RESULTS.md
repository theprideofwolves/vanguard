# Vanguard verification results

Overall: **PASS**

Source commit: `c4d356a4c6787e1560bf7a7e2b314717c89663ab`; normalized source fingerprint: `1f17625d8459c4df4dedd083c9cc87c229e337573ffc3b81e739fd1984d54032`.
Recorded UTC: 2026-11-10T15:58:31.178670+00:00. Platform: Windows-11-10.0.26300-SP0. Go: go version go1.25.5 windows/amd64.

## Commands

| Step | Result | Wall seconds |
| --- | --- | ---: |
| vet | PASS | 1.21 |
| race-and-coverage | PASS | 16.01 |
| coverage-summary | PASS | 0.78 |
| build | PASS | 1.80 |
| compiled-smoke | PASS | 15.84 |
| ci-examples | PASS | 13.31 |
| fuzz-parser | PASS | 18.78 |
| fuzz-evaluator | PASS | 19.98 |
| fuzz-gate | PASS | 29.38 |
| benchmarks | PASS | 29.74 |

## Functional evidence

37 top-level tests and 60 subtests passed; 0 failed, 0 skipped. Fuzz seed invocations are counted separately: 12.
Generated reference-model cases: 2048 evaluations + 1024 comparisons. These seeded cases are not additional named tests.
Whole-project statement coverage: **85.5%**, using `-coverpkg=./...`.

| Functionality | Result | Named evidence |
| --- | --- | --- |
| Threshold equality, exceedance, omitted/-1 | PASS | `TestDecisionBoundaries`, `TestEffectiveLimitChanges` |
| Schema, UTF-8, null/malformed input, duplicate keys/findings | PASS | `TestReportEdgeCases`, `TestBadInput`, `TestRejectAmbiguousJSONFieldCasing`, `TestRejectInvalidPolicies`, `TestRejectNonIntegerThresholds` |
| Exact input/policy hashes, counts and violations | PASS | `TestDecisionEvidenceMatchesExactBytes`, `TestPolicyImpactTransitionsAndEvidence` |
| Report/policy/corpus count and byte boundaries | PASS | `TestReportSizeBoundary`, `TestPolicyEdgeCases`, `TestImpactCorpusByteBoundary`, `TestInvalidImpactIsAtomic`, `TestImpactFileIntake` |
| All four transitions, mixed changes and no-flip relaxations | PASS | `TestPolicyImpactTransitionsAndEvidence`, `TestEffectiveLimitChanges` |
| Seeded specification-model agreement | PASS | `TestGeneratedDecisionsAgainstReference`, `TestGeneratedComparisonsAgainstReference` |
| Upload precedence, invalid input clears current export | PASS | `TestUploadOverridesPasteAndInvalidInputClearsExport`, `TestRejectedUploadClearsCurrentDecision`, `TestPolicyImpactUISession` |
| Welcome, downloads, valid FAIL/comparison completes guide | PASS | `TestUIEvaluationAndExport`, `TestPolicyImpactUISession`, `TestFirstRunChoiceAndPersistence` |
| Session inputs, appearance cookie and guide navigation | PASS | `TestAppearanceAndSessionInputs`, `TestThemeRequestGuards`, `TestPolicyImpactUISession` |
| Escaping, concurrency and local request guards | PASS | `TestAppearanceAndSessionInputs`, `TestConcurrentEvaluationAndExport`, `TestRequestBoundaries`, `TestRequestAndSampleBoundaries` |
| CLI exits and output failures in both modes | PASS | `TestCLI`, `TestCLIOutputFailure`, `TestCompareCLI` |
| Slack summary/error secrecy, no retries/redirects, verdict unchanged (mocked) | PASS | `TestSlackSummaryAndFailuresUseMockTransport`, `TestCLIMockedNotificationPreservesExitStatus` |
| Guide preference failures and data-directory override | PASS | `TestPreferenceFailures`, `TestVanguardDataDirectoryOverride` |
| Compiled CLI file/stdin, guide restart, app restart/session loss, theme and export | PASS | `smoke.txt` |

## Bounded fuzz runs

| Target | Recorded executions | Failure found |
| --- | ---: | --- |
| FuzzParse | 315168 | No |
| FuzzPolicy | 63991 | No |
| FuzzPolicyImpact | 58471 | No |

## Synthetic in-process benchmarks

Medians and min–max are across independent Go benchmark samples, not request-latency percentiles. Includes parsing, validation, evaluation and hashing. Excludes process startup, file I/O, export serialization, network and input generation. Comparison findings are per report.

| Workload | Samples | Median ms/op | Min–max ms/op | Allocated bytes/op | Allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| BenchmarkEvaluate/findings_100 | 5 | 0.676 | 0.611–0.782 | 289,843 | 5,336 |
| BenchmarkEvaluate/findings_1000 | 5 | 7.648 | 7.099–9.084 | 2,931,532 | 51,271 |
| BenchmarkEvaluate/findings_10000 | 5 | 71.882 | 64.368–87.019 | 34,741,676 | 510,364 |
| BenchmarkComparePolicies/reports_1_findings_1000_each | 5 | 9.506 | 8.274–14.289 | 2,946,720 | 51,420 |
| BenchmarkComparePolicies/reports_8_findings_1000_each | 5 | 60.005 | 57.467–102.592 | 23,391,671 | 409,548 |
| BenchmarkComparePolicies/reports_32_findings_1000_each | 5 | 196.063 | 192.437–202.056 | 93,485,229 | 1,637,380 |

Allocated bytes/op measure cumulative allocations, not peak process memory. These are local synthetic measurements, not production throughput, detection accuracy or a comparison against other tools.

## Limits

Manual visual/keyboard/narrow-screen checks, normal OS browser opening and clean-machine shutdown are not exercised here. Real Slack delivery is not tested; all Slack transport tests are mocked. No scanner, paid service or deployment is invoked. Benchmarks do not measure sustained load, real-world adoption, false positives/negatives or incident reduction. Coverage and bounded fuzzing are not exhaustive correctness/security proofs.

Raw commands, toolchain, test counts and workload samples are in `summary.json`, `tests.jsonl`, `coverage.txt`, `coverage-functions.txt`, `benchmarks.txt`, `fuzz-*.txt`, `smoke.txt` and `ci-examples.txt`. Private launcher URLs and local user/workspace paths are scrubbed. Source fingerprint normalizes CRLF to LF and covers Go code/tests, dependency files, launcher/version inputs, testdata and .github scripts/workflows; it excludes documentation and generated evidence.
