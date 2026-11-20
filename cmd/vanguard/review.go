package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/theprideofwolves/vanguard/internal/review"
)

func runReview(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("vanguard review", flag.ContinueOnError)
	fs.SetOutput(errOut)
	repo := fs.String("repo", ".", "Local Git repository (no fetch or checkout)")
	baseRef := fs.String("base-ref", "", "Full trusted baseline commit ID (required)")
	basePath := fs.String("base-policy", "policy.yaml", "Policy path inside the pinned commit")
	candidatePath := fs.String("candidate", "", "Proposed local YAML policy (required)")
	mode := fs.String("mode", "review", "review flags relaxations; shadow keeps the pinned baseline gate")
	format := fs.String("format", "markdown", "stdout format: markdown or json")
	output := fs.String("output", "", "Optional NEW evidence-pack directory; parent must exist")
	fs.Usage = func() {
		fmt.Fprintln(errOut, "Usage: vanguard review -base-ref FULL_COMMIT -candidate proposed.yaml [flags] report.json|reports-dir\nNo AI API, scanner, policy modification or notification.\nReview exits: 0 no relaxed limits, 1 review required, 2 error.\nShadow exits: 0 all reports pass the pinned baseline, 1 baseline FAIL, 2 error.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *baseRef == "" || *candidatePath == "" || fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	if (*mode != "review" && *mode != "shadow") || (*format != "markdown" && *format != "json") {
		fmt.Fprintln(errOut, "choose mode review/shadow and format markdown/json")
		return 2
	}
	source, base, err := review.LoadBase(*repo, *baseRef, *basePath)
	if err != nil {
		fmt.Fprintln(errOut, "baseline:", err)
		return 2
	}
	candidate, err := readBoundedFile(*candidatePath, 1<<20)
	if err != nil {
		fmt.Fprintln(errOut, "candidate:", err)
		return 2
	}
	reports, err := loadImpactReports(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	result, err := review.Analyze(source, base, candidate, reports, *mode, filepath.Base(*candidatePath))
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	if *output != "" {
		if err := review.WritePack(*output, result, base, candidate); err != nil {
			fmt.Fprintln(errOut, "cannot complete review pack:", err)
			return 2
		}
	}
	if *format == "json" {
		err = json.NewEncoder(out).Encode(result)
	} else {
		_, err = io.WriteString(out, review.Markdown(result))
	}
	if err != nil {
		fmt.Fprintln(errOut, "cannot write review output")
		return 2
	}
	return result.ResultExitCode
}
