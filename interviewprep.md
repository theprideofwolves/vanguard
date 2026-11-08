# Vanguard: interview study guide

## Explain it in 30 seconds

Vanguard accepts a normalized findings report and YAML severity thresholds,
validates them, and produces an explained PASS/FAIL decision for local review or
CI. The CLI has distinct policy-failure and tool-error exit codes. Optional Slack
reports failures without changing the verdict. Vanguard does not scan source
code, discover vulnerabilities, or certify compliance.

## Trace the implementation

Follow main.go or ui.go into gate.Evaluate, ParsePolicy, parser.Parse and
evaluator.Evaluate. Trace the fixed severity order into decision rows, exact-byte
hashes, JSON export and CLI exit status. Then trace Notify separately.

## Questions and grounded answers

1. What is normalized? Findings follow internal/models/finding.go. Native scanner
   conversion must happen upstream; severity text is lowercased here.
2. When does a threshold fail? Observed count > allowed. Equal counts pass.
   Omitted thresholds and -1 are unlimited; at least one threshold is required.
3. Why reject duplicates? Identical normalized finding objects can double-count
   accidental input duplication. Distinct modeled finding objects count separately;
   this is not CVE deduplication across different scanners or an occurrence-ID system.
4. Why reject unknown fields/severities? Silently ignored input could hide a
   schema mismatch or let an unsupported severity escape the fixed counting loop.
5. What do hashes prove? Exact input/policy bytes can be identified. They do not
   prove report authenticity, successful scanning, completeness or tamper resistance.
6. What does an empty report prove? It passes applicable thresholds. It does not
   prove the upstream scanner succeeded; CI must check upstream execution separately.
7. Why does the demo FAIL? It deliberately exceeds critical:0. Evaluation worked;
   FAIL completes onboarding, whereas malformed input does not.
8. What persists? Only guide completion and the launcher lock. Current decisions
   and inputs live in memory; invalid evaluation clears the old decision and restart
   starts a new session. Export before closing if evidence is needed.
   Large uploaded files may be staged in temporary storage during the request
   and removed by multipart cleanup; there is no saved report history.
9. What goes to Slack? Policy hash, exceeded severity counts/limits, generic failure
   text and optional run URL. No source/findings descriptions or credentials.
   The webhook itself is a secret and must be supplied privately by CI.
10. What if Slack fails? Stderr reports failure, the computed verdict/exit status
    stays unchanged, and there is no blind delivery retry or exactly-once claim.

## Practice with source

Read policy_test.go, gate_test.go and the CLI/UI tests. Predict equality and
one-above-limit cases, a missing severity, a duplicate record, malformed input
after a valid decision and a failed notification. Compare the exported hashes
with the exact input bytes. Use the packaged CLI to demonstrate 0/1/2 exits.
examples/github-action.yml is a reusable step fragment, not an installed CI pipeline.

Describe AI assistance honestly and explain the source/tests you personally
reviewed. Do not claim measured production impact or real deployment without evidence.
