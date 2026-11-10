# Hosted verification

[GitHub Actions run 42851012838](https://github.com/theprideofwolves/vanguard/actions/runs/42851012838)
completed successfully for source commit `feb6d3e8d1d614cd9e86c9d47708d0c92cf9dc67`.

| Job | Result |
| --- | --- |
| Ubuntu test/build | PASS |
| Windows test/build | PASS |
| Binary release | SKIPPED |

Both test jobs ran `.github/scripts/verify.py`: race-enabled functional tests,
whole-project coverage, compiled CLI/UI HTTP smoke, local CI example checks,
three 10-second fuzz budgets, and six benchmark workloads with three 250ms samples.
They then built the portable binary. Verification and binary artifacts are attached
to the run; GitHub retention and login requirements apply to artifact downloads.
The job/step outcomes and artifact metadata are preserved in `summary.json`.

This records actual hosted Linux/Windows execution, not just cross-compilation.
Hosted benchmark timing is environment-specific; it should not be mixed with
the [five-sample local Windows measurements](../2026-11-10-windows/RESULTS.md).
The local report and raw evidence remain committed after hosted artifacts expire.

No binary release, scanner, real webhook or deployment was executed. The HTTP
smoke uses localhost, isolated guide data and disabled browser opening. Slack
transport tests are mocked. Human appearance/accessibility, clean-machine desktop
launch, live delivery and production workload behavior remain separate checks.

This documentation-only snapshot is committed after the validated source; it
does not change the implementation or the measured workload.
