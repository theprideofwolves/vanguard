# Manual policy-review results

Status: **NOT RUN. Template only.**

- Source commit: not recorded
- Build metadata / Go version: not recorded
- Git version, OS, date: not recorded
- Baseline commit and policy path: not recorded
- Dataset: not recorded; label synthetic or real normalized data
- Tester: not recorded
- Commands and exit statuses: not recorded
- Screenshots/video: none

| Case | Expected | Observed | Status | Evidence |
| --- | --- | --- | --- | --- |
| Mixed policy review | Exit 1, one newly passing / one newly failing | Not recorded | NOT RUN | None |
| Tightened review | Exit 0, no relaxation | Not recorded | NOT RUN | None |
| Shadow baseline PASS / candidate FAIL | Exit 0 | Not recorded | NOT RUN | None |
| Shadow baseline FAIL / candidate PASS | Exit 1 | Not recorded | NOT RUN | None |
| Pinned baseline despite dirty clone | Reads committed policy | Not recorded | NOT RUN | None |
| Invalid ref/path/object/type | Error 2, no fallback | Not recorded | NOT RUN | None |
| Missing/malformed/oversized input | Error 2 | Not recorded | NOT RUN | None |
| Evidence pack and independent hashes | Complete manifest, hashes match | Not recorded | NOT RUN | None |
| Existing destination / failed output | Error 2, no overwrite | Not recorded | NOT RUN | None |
| Markdown labels | Text only, no injected structure | Not recorded | NOT RUN | None |
| UI JSON/Markdown clearing and session behavior | Only current valid comparison exported | Not recorded | NOT RUN | None |
| Consuming CI workflow | Baseline/tool/corpus protected, candidate read as data | Not recorded | NOT RUN | None |

## Failures, fixes and remaining limits

Not recorded. Do not replace NOT RUN with PASS without the corresponding evidence.
Never include webhook URLs, API tokens, private launcher URLs or unauthorized data.
