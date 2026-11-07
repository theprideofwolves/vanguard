# Interview walkthrough: Policy evaluation for security findings

## Explain it in 45 seconds
I built Go components to normalize security findings and compare their severity counts with YAML thresholds. The original resume describes that prototype. I subsequently connected it into a runnable CLI and added input validation, exit codes, and tests.

## Demonstrate it
1. Build: `go build -o vanguard ./cmd/vanguard` (Windows: `-o vanguard.exe`).
2. Run `./vanguard -p policy.yaml testdata/report-pass.json`: PASS, exit 0.
3. Run `./vanguard -p policy.yaml testdata/report-fail.json`: FAIL, exit 1.
4. Supply a missing file or unknown severity: error, exit 2.
5. Run `go test ./...` to check the decision paths.

On Windows use `.\vanguard.exe` instead of `./vanguard`. `go run` does not preserve a program's nonzero exit code exactly; build the binary when demonstrating exit codes.

## Questions to be ready for
- Why normalized input? Different scanners have different schemas; a stable internal model keeps policy evaluation separate from adapters.
- Why reject unknown severities? Otherwise an unrecognized finding can escape the fixed severity loop.
- What does -1 mean? Unlimited; an omitted threshold is also unlimited. Empty policies are rejected.
- Determinism? Evaluation and reporting use the same fixed severity order.
- Limits? No native Snyk/Trivy adapters, automatic remediation, or claim of compliance certification.

## Timeline and resume consistency
The submitted resume describes the original project's core behavior. These subsequent improvements strengthen the same project. If asked, distinguish the original implementation from later hardening. Practice the demo and inspect the source before describing decisions as your own. Do not claim production deployment, measured business impact, or autonomous agents from this repository.
