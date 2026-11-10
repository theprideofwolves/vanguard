# Testing Vanguard

All automated checks are local and free. Use synthetic normalized reports;
do not provide real webhook credentials. Slack transport tests never open a socket.

For recorded metrics and raw results, see [verification evidence](docs/VERIFICATION.md).
To rerun all checks and generate a new report:

```sh
python .github/scripts/verify.py --output tmp/verification-review
```

This also requires Bash (Git Bash on Windows). Choose a new output directory for
each run; the verifier preserves existing evidence. CI records the same categories
on Windows/Linux and uploads verification artifacts; consult the actual run status.

## Automated checks

Requires Go 1.25.5+, a C compiler for race detection, and Python 3 for the
compiled-binary smoke test. From the repository root:

```sh
go test -race -count=1 ./...
go vet ./...
go build -o vanguard.exe ./cmd/vanguard
python .github/scripts/smoke.py ./vanguard.exe
```

On Linux build `vanguard` and pass `./vanguard` to the smoke script.
The smoke script uses isolated temporary guide data and disables browser opening.
It starts only localhost HTTP and checks welcome/setup, sample downloads, valid
FAIL/PASS, export hashes, malformed input, guide persistence, session loss on
restart, CLI file/stdin and 0/1/2 exits. Comparison checks cover all four verdict
transitions in the compiled CLI, mixed changes, exact hashes, theme/guide
preservation and invalid-comparison export clearing. Go tests additionally cover threshold
boundaries, duplicate keys/findings, size caps, policy types, HTTP guards, escaped
HTML, appearance preferences, concurrent requests, and mocked notification failures.

Optional bounded fuzzing (run each command separately):

```sh
go test ./internal/parser -run '^$' -fuzz FuzzParse -fuzztime 15s -parallel 2
go test ./internal/gate -run '^$' -fuzz FuzzPolicyImpact -fuzztime 15s -parallel 2
go test ./internal/evaluator -run '^$' -fuzz FuzzPolicy -fuzztime 15s -parallel 2
```

## Quick manual checklist

Use the freshly compiled executable. In PowerShell, isolate guide state first:

```powershell
$env:VANGUARD_DATA_DIR = Join-Path $PWD 'tmp/manual-review'
Remove-Item Env:VANGUARD_NO_BROWSER -ErrorAction SilentlyContinue
.\vanguard.exe
```

1. **Appearance:** confirm a white page with teal accents. Switch to dark mode:
   page and inputs should be black, with the same teal hue. Refresh and visit
   setup; the selection should persist. Switch back. Check narrow window and
   keyboard Tab focus; controls and results should remain usable.
2. **Demo:** choose **Try the guided demo**. Expect FAIL, critical observed `1`,
   allowed `0`, and a decision download. This completes the guide successfully.
3. **Your data:** download sample report/policy from setup. Upload the passing
   fixture while the pasted report is invalid; expect PASS and a fresh export.
4. **Policy impact:** restart the guide and run **Try the guided demo** again
   to reset report/policies. Open **Compare a proposed policy**. The built-in
   candidate changes critical `0` to `1`.
   Expect REVIEW REQUIRED and Newly passing `1`. Download policy-impact JSON;
   paired input hashes should match. Switch theme and restart the guide; the
   comparison should remain exportable.
5. **Invalid input:** remove the upload, paste `null` and evaluate. Expect an error
   and no stale export. Also compare with a `null` proposed policy; export must
   remain unavailable. Re-evaluate valid data to restore a current result.
6. **Session:** restart the guide and return to workspace; inputs/result should
   remain. Stop with Ctrl+C, reopen: guide completion remains, decision does not.
7. **Shutdown:** confirm Ctrl+C stops the process. Closing the browser alone
   leaves the application running.

## Comparison CLI checklist

Run the commands in [examples/policy-review.md](examples/policy-review.md). In
PowerShell inspect `$LASTEXITCODE` immediately after each command. Mixed relaxation/
tightening returns `1`, base-to-tightened returns `0`, and a malformed report returns
`2` with no valid impact output. Comparison `0` does not mean every report passes.
The current `vanguard-impact/v1` JSON embeds ordinary paired decisions; this keeps
counts/violations inspectable using the existing decision schema. Duplicate report
labels, empty corpora and count/byte overflows fail atomically. Comparing policies
with identical effective limits is unchanged even if raw hashes differ.

No real Slack delivery is needed for this checklist. Live Slack, clean-machine
desktop behavior, human accessibility review and sustained resource
measurements are separate checks. Automated coverage and bounded fuzzing are
not exhaustive or a security/compliance certification.
