"""Local verification evidence. Standard library only; no real Slack or scanners.

Run from any directory: python .github/scripts/verify.py --output <new-directory>
Each output directory is immutable: choose a new name for another run.
"""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shutil
import statistics
import subprocess
import tempfile
import textwrap
import time


ROOT = Path(__file__).resolve().parents[2]
MATRIX = [
    ("Threshold equality, exceedance, omitted/-1", ["TestDecisionBoundaries", "TestEffectiveLimitChanges"]),
    ("Schema, UTF-8, null/malformed input, duplicate keys/findings", ["TestReportEdgeCases", "TestBadInput", "TestRejectAmbiguousJSONFieldCasing", "TestRejectInvalidPolicies", "TestRejectNonIntegerThresholds"]),
    ("Exact input/policy hashes, counts and violations", ["TestDecisionEvidenceMatchesExactBytes", "TestPolicyImpactTransitionsAndEvidence"]),
    ("Report/policy/corpus count and byte boundaries", ["TestReportSizeBoundary", "TestPolicyEdgeCases", "TestImpactCorpusByteBoundary", "TestInvalidImpactIsAtomic", "TestImpactFileIntake"]),
    ("All four transitions, mixed changes and no-flip relaxations", ["TestPolicyImpactTransitionsAndEvidence", "TestEffectiveLimitChanges"]),
    ("Seeded specification-model agreement", ["TestGeneratedDecisionsAgainstReference", "TestGeneratedComparisonsAgainstReference"]),
    ("Upload precedence, invalid input clears current export", ["TestUploadOverridesPasteAndInvalidInputClearsExport", "TestRejectedUploadClearsCurrentDecision", "TestPolicyImpactUISession"]),
    ("Welcome, downloads, valid FAIL/comparison completes guide", ["TestUIEvaluationAndExport", "TestPolicyImpactUISession", "TestFirstRunChoiceAndPersistence"]),
    ("Session inputs, appearance cookie and guide navigation", ["TestAppearanceAndSessionInputs", "TestThemeRequestGuards", "TestPolicyImpactUISession"]),
    ("Escaping, concurrency and local request guards", ["TestAppearanceAndSessionInputs", "TestConcurrentEvaluationAndExport", "TestRequestBoundaries", "TestRequestAndSampleBoundaries"]),
    ("CLI exits and output failures in both modes", ["TestCLI", "TestCLIOutputFailure", "TestCompareCLI"]),
    ("Slack summary/error secrecy, no retries/redirects, verdict unchanged (mocked)", ["TestSlackSummaryAndFailuresUseMockTransport", "TestCLIMockedNotificationPreservesExitStatus"]),
    ("Guide preference failures and data-directory override", ["TestPreferenceFailures", "TestVanguardDataDirectoryOverride"]),
]


def fingerprint():
    """Hash text source inputs after CRLF normalization; evidence/docs excluded."""
    selected = []
    for base in [ROOT / "cmd", ROOT / "internal", ROOT / "testdata", ROOT / ".github", ROOT / "examples"]:
        selected.extend(p for p in base.rglob("*") if p.is_file() and p.suffix in {".go", ".py", ".yaml", ".yml", ".json"})
    selected.extend(ROOT / p for p in ["go.mod", "go.sum", "VERSION", "policy.yaml", "test-report.json"])
    h = hashlib.sha256()
    for p in sorted(set(selected)):
        h.update(p.relative_to(ROOT).as_posix().encode() + b"\0")
        h.update(p.read_bytes().replace(b"\r\n", b"\n") + b"\0")
    return h.hexdigest()


def scrub(text):
    # Launcher URLs must never enter published evidence, even on a failed smoke.
    text = re.sub(r"http://(?:127\.0\.0\.1|localhost):\d+/[^\s\"<>]*", "<PRIVATE_LAUNCHER_URL>", text)
    for p, label in [(str(ROOT), "<WORKSPACE>"), (str(Path.home()), "<USERPROFILE>")]:
        text = text.replace(p, label).replace(p.replace("\\", "/"), label)
    return text


def parse_tests(path):
    events = [json.loads(line) for line in path.read_text().splitlines() if line.startswith("{")]
    passed = [e["Test"] for e in events if e.get("Action") == "pass" and e.get("Test")]
    output = "".join(e.get("Output", "") for e in events)
    return {
        "top_level_tests_passed": sum(n.startswith("Test") and "/" not in n for n in passed),
        "subtests_passed": sum(n.startswith("Test") and "/" in n for n in passed),
        "fuzz_seed_invocations_passed": sum(n.startswith("Fuzz") and "/" in n for n in passed),
        "failed_invocations": [e.get("Test", e.get("Package")) for e in events if e.get("Action") == "fail"],
        "skipped_invocations": [e.get("Test", e.get("Package")) for e in events if e.get("Action") == "skip"],
        "generated_evaluation_cases": int(re.search(r"REFERENCE_EVALUATION_CASES=(\d+)", output)[1]),
        "generated_comparison_cases": int(re.search(r"REFERENCE_COMPARISON_CASES=(\d+)", output)[1]),
        "passed_test_names": sorted(set(passed)),
    }


def parse_benchmarks(path, count):
    groups = {}
    for line in path.read_text().splitlines():
        if not line.startswith("Benchmark"):
            continue
        parts = line.split()
        name = re.sub(r"-\d+$", "", parts[0])
        metrics = {parts[i + 1]: float(parts[i]) for i in range(2, len(parts) - 1, 2)}
        groups.setdefault(name, []).append(metrics)
    if len(groups) != 6 or any(len(values) != count for values in groups.values()):
        raise RuntimeError("Incomplete benchmark samples")
    return [{"name": name, "samples": len(values),
             "median_ms_per_op": statistics.median(v["ns/op"] for v in values) / 1e6,
             "min_ms_per_op": min(v["ns/op"] for v in values) / 1e6,
             "max_ms_per_op": max(v["ns/op"] for v in values) / 1e6,
             "median_allocated_bytes_per_op": statistics.median(v["B/op"] for v in values),
             "median_allocations_per_op": statistics.median(v["allocs/op"] for v in values)}
            for name, values in groups.items()]


def check_ci_examples(executable, env, work):
    bash = Path("C:/Program Files/Git/bin/bash.exe") if os.name == "nt" else Path(shutil.which("bash") or "")
    if not bash.is_file():
        raise RuntimeError("Git Bash / Bash is required to verify CI example status preservation")
    root = work / "ci-examples"
    root.mkdir()
    shutil.copyfile(ROOT / "policy.yaml", root / "policy.yaml")
    shutil.copytree(ROOT / "testdata/impact", root / "testdata/impact")
    env = dict(env, PATH=str(executable.parent) + os.pathsep + env["PATH"])
    lines = []
    for example in ["github-action.yml", "github-policy-review.yml"]:
        source = (ROOT / "examples" / example).read_text()
        script = textwrap.dedent(source.split("    run: |\n", 1)[1].split("  - uses:", 1)[0])
        for expected in [0, 1, 2]:
            if example == "github-action.yml":
                artifact = root / "decision.json"
                if expected < 2:
                    shutil.copyfile(ROOT / "testdata" / ("report-pass.json" if expected == 0 else "report-fail.json"), root / "report.json")
                else:
                    (root / "report.json").write_text("null")
            else:
                artifact = root / "policy-impact.json"
                target = root / "testdata/impact/candidate.yaml"
                if expected < 2:
                    shutil.copyfile(ROOT / "testdata/impact" / ("base.yaml" if expected == 0 else "candidate.yaml"), target)
                else:
                    target.write_text("null")
            result = subprocess.run([str(bash), "-e", "-c", script], cwd=root, env=env, capture_output=True, timeout=60)
            if result.returncode != expected:
                raise RuntimeError(f"{example}: expected {expected}, got {result.returncode}: {result.stderr.decode(errors='replace')}")
            data = artifact.read_bytes()
            if expected == 2:
                if data:
                    raise RuntimeError("Invalid input emitted artifact evidence")
            else:
                value = json.loads(data)
                field = "Passed" if example == "github-action.yml" else "ReviewRequired"
                if value[field] != (expected == (0 if field == "Passed" else 1)):
                    raise RuntimeError("Incorrect artifact verdict")
            lines.append(f"{example}: exit {expected} and artifact verified")
    return "\n".join(lines) + "\n"


def write_report(output, summary):
    (output / "summary.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    lines = ["# Vanguard verification results", "", f"Overall: **{summary['status']}**", "",
             f"Source commit: `{summary['source_commit']}`; normalized source fingerprint: `{summary['source_fingerprint']}`.",
             f"Recorded UTC: {summary['started_utc']}. Platform: {summary['platform']}. Go: {summary['go_version']}.", "",
             "## Commands", "", "| Step | Result | Wall seconds |", "| --- | --- | ---: |"]
    for step in summary["steps"]:
        lines.append(f"| {step['name']} | {step['status']} | {step['wall_seconds']:.2f} |")
    tests = summary.get("tests")
    if tests:
        lines += ["", "## Functional evidence", "",
                  f"{tests['top_level_tests_passed']} top-level tests and {tests['subtests_passed']} subtests passed; "
                  f"{len(tests['failed_invocations'])} failed, {len(tests['skipped_invocations'])} skipped. "
                  f"Fuzz seed invocations are counted separately: {tests['fuzz_seed_invocations_passed']}.",
                  f"Generated reference-model cases: {tests['generated_evaluation_cases']} evaluations + "
                  f"{tests['generated_comparison_cases']} comparisons. These seeded cases are not additional named tests.",
                  f"Whole-project statement coverage: **{summary.get('coverage_percent', 'unavailable')}%**, using `-coverpkg=./...`.",
                  "", "| Functionality | Result | Named evidence |", "| --- | --- | --- |"]
        passed = set(tests["passed_test_names"])
        for label, names in MATRIX:
            status = "PASS" if all(n in passed for n in names) else "NOT VERIFIED"
            lines.append(f"| {label} | {status} | {', '.join('`'+n+'`' for n in names)} |")
        smoke_ok = any(s["name"] == "compiled-smoke" and s["status"] == "PASS" for s in summary["steps"])
        lines.append(f"| Compiled CLI file/stdin, guide restart, app restart/session loss, theme and export | {'PASS' if smoke_ok else 'NOT VERIFIED'} | `smoke.txt` |")
    if summary.get("fuzz"):
        lines += ["", "## Bounded fuzz runs", "", "| Target | Recorded executions | Failure found |", "| --- | ---: | --- |"]
        for item in summary["fuzz"]:
            lines.append(f"| {item['target']} | {item['executions']} | No |")
    if summary.get("benchmarks"):
        lines += ["", "## Synthetic in-process benchmarks", "",
                  "Medians and min–max are across independent Go benchmark samples, not request-latency percentiles. "
                  "Includes parsing, validation, evaluation and hashing. Excludes process startup, file I/O, "
                  "export serialization, network and input generation. Comparison findings are per report.", "",
                  "| Workload | Samples | Median ms/op | Min–max ms/op | Allocated bytes/op | Allocs/op |",
                  "| --- | ---: | ---: | ---: | ---: | ---: |"]
        for b in summary["benchmarks"]:
            lines.append(f"| {b['name']} | {b['samples']} | {b['median_ms_per_op']:.3f} | {b['min_ms_per_op']:.3f}–{b['max_ms_per_op']:.3f} | {b['median_allocated_bytes_per_op']:,.0f} | {b['median_allocations_per_op']:,.0f} |")
        lines += ["", "Allocated bytes/op measure cumulative allocations, not peak process memory. "
                  "These are local synthetic measurements, not production throughput, detection accuracy or a comparison against other tools."]
    lines += ["", "## Limits", "",
              "Manual visual/keyboard/narrow-screen checks, normal OS browser opening and clean-machine shutdown "
              "are not exercised here. Real Slack delivery is not tested; all Slack transport tests are mocked. "
              "No scanner, paid service or deployment is invoked. Benchmarks do not measure sustained load, "
              "real-world adoption, false positives/negatives or incident reduction. Coverage and bounded fuzzing "
              "are not exhaustive correctness/security proofs.", "",
              "Raw commands, toolchain, test counts and workload samples are in `summary.json`, `tests.jsonl`, "
              "`coverage.txt`, `coverage-functions.txt`, `benchmarks.txt`, `fuzz-*.txt`, `smoke.txt` and `ci-examples.txt`. "
              "Private launcher URLs and local user/workspace paths are scrubbed. Source fingerprint normalizes "
              "CRLF to LF and covers Go code/tests, dependency files, launcher/version inputs, testdata and .github scripts/workflows; "
              "it excludes documentation and generated evidence."]
    (output / "RESULTS.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True)
    parser.add_argument("--fuzz-seconds", type=int, default=15)
    parser.add_argument("--benchmark-count", type=int, default=5)
    parser.add_argument("--benchmark-time", default="500ms")
    args = parser.parse_args()
    if args.fuzz_seconds < 1 or args.benchmark_count < 3 or not re.fullmatch(r"[1-9]\d*(?:ms|s)", args.benchmark_time):
        parser.error("Use positive fuzz seconds, >=3 benchmark samples, and a benchmark duration such as 500ms")
    output = Path(args.output).resolve()
    if output.exists():
        parser.error("Output already exists; choose a new directory to preserve prior evidence")
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    clean = not subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT).strip()
    summary = {"status": "RUNNING", "started_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
               "source_commit": commit, "worktree_clean_at_start": clean, "source_fingerprint": fingerprint(),
               "platform": platform.platform(), "python": platform.python_version(),
               "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
               "benchmark_count": args.benchmark_count, "benchmark_time": args.benchmark_time,
               "fuzz_seconds_per_target": args.fuzz_seconds, "steps": []}
    output.mkdir(parents=True)
    temp_parent = ROOT / "tmp" / "_verification"
    temp_parent.mkdir(parents=True, exist_ok=True)
    failure = None
    with tempfile.TemporaryDirectory(prefix="run-", dir=temp_parent) as temp:
        work = Path(temp)
        env = dict(os.environ, TEMP=temp, TMP=temp, TMPDIR=temp, VANGUARD_SMOKE_TEMP=temp,
                   VANGUARD_NO_BROWSER="1", VANGUARD_SLACK_WEBHOOK="", VANGUARD_RUN_URL="")

        def step(name, command=None, filename=None, action=None):
            print(f"Running {name}...", flush=True)
            start = time.perf_counter()
            item = {"name": name, "command": command, "status": "FAIL"}
            summary["steps"].append(item)
            try:
                if action:
                    text = action()
                else:
                    result = subprocess.run(command, cwd=ROOT, env=env, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=600)
                    text = result.stdout + ("\nSTDERR:\n" + result.stderr if result.stderr else "")
                    if filename:
                        (output / filename).write_text(scrub(text), encoding="utf-8")
                    if result.returncode:
                        raise RuntimeError(f"{name} failed, exit {result.returncode}; see {filename}")
                if filename:
                    (output / filename).write_text(scrub(text), encoding="utf-8")
                item["status"] = "PASS"
            finally:
                item["wall_seconds"] = round(time.perf_counter() - start, 3)
                item["command"] = [scrub(str(x)) for x in command] if command else "local example verification"
            print(f"PASS {name} ({item['wall_seconds']:.2f}s)", flush=True)
            return text

        try:
            step("vet", ["go", "vet", "./..."], "vet.txt")
            step("race-and-coverage", ["go", "test", "-race", "-count=1", "-coverpkg=./...", "-coverprofile=" + str(output / "coverage.txt"), "-json", "./..."], "tests.jsonl")
            summary["tests"] = parse_tests(output / "tests.jsonl")
            text = step("coverage-summary", ["go", "tool", "cover", "-func=" + str(output / "coverage.txt")], "coverage-functions.txt")
            summary["coverage_percent"] = float(re.search(r"total:\s+\(statements\)\s+([\d.]+)%", text)[1])
            executable = work / ("vanguard.exe" if os.name == "nt" else "vanguard")
            step("build", ["go", "build", "-trimpath", "-o", str(executable), "./cmd/vanguard"], "build.txt")
            summary["binary_sha256"] = hashlib.sha256(executable.read_bytes()).hexdigest()
            step("compiled-smoke", [os.sys.executable, str(ROOT / ".github/scripts/smoke.py"), str(executable)], "smoke.txt")
            step("ci-examples", filename="ci-examples.txt", action=lambda: check_ci_examples(executable, env, work))
            summary["fuzz"] = []
            for package, target in [("parser", "FuzzParse"), ("evaluator", "FuzzPolicy"), ("gate", "FuzzPolicyImpact")]:
                text = step("fuzz-" + package, ["go", "test", "./internal/" + package, "-run", "^$", "-fuzz", target, "-fuzztime", str(args.fuzz_seconds) + "s", "-parallel", "2"], "fuzz-" + package + ".txt")
                counts = re.findall(r"execs:\s+(\d+)", text)
                summary["fuzz"].append({"target": target, "executions": int(counts[-1]) if counts else 0})
            step("benchmarks", ["go", "test", "./internal/gate", "-run", "^$", "-bench", "Benchmark", "-benchmem", "-benchtime", args.benchmark_time, "-count", str(args.benchmark_count)], "benchmarks.txt")
            summary["benchmarks"] = parse_benchmarks(output / "benchmarks.txt", args.benchmark_count)
            if fingerprint() != summary["source_fingerprint"]:
                raise RuntimeError("Source changed during verification")
            summary["status"] = "PASS"
        except Exception as exc:
            failure = scrub(str(exc))
            summary["status"] = "FAIL"
            summary["error"] = failure
        finally:
            summary["finished_utc"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
            write_report(output, summary)
    if failure:
        raise SystemExit(failure)
    print(f"Evidence recorded: {output}")


if __name__ == "__main__":
    main()
