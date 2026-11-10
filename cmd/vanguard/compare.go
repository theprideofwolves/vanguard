package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/theprideofwolves/vanguard/internal/gate"
)

func readBoundedFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%q must be a regular file (no symlinks)", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%q exceeds its %d-byte limit", path, limit)
	}
	return data, nil
}

func loadImpactReports(path string) ([]gate.ReportInput, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	paths := []string{path}
	if info.IsDir() {
		paths = nil
		dir, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer dir.Close()
		// Bound directory enumeration too, rather than allocating every entry.
		entries, err := dir.ReadDir(4097)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if len(entries) > 4096 {
			return nil, fmt.Errorf("report directory exceeds 4096 entries")
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
				paths = append(paths, filepath.Join(path, entry.Name()))
			}
		}
	}
	if len(paths) == 0 || len(paths) > gate.MaxImpactReports {
		return nil, fmt.Errorf("provide 1 to %d top-level JSON reports", gate.MaxImpactReports)
	}
	inputs := make([]gate.ReportInput, 0, len(paths))
	total := 0
	for _, name := range paths {
		limit := min(16<<20, gate.MaxImpactBytes-total)
		data, err := readBoundedFile(name, int64(limit))
		if err != nil {
			return nil, err
		}
		total += len(data)
		inputs = append(inputs, gate.ReportInput{Name: filepath.Base(name), Data: data})
	}
	return inputs, nil
}

func runCompare(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("vanguard compare", flag.ContinueOnError)
	fs.SetOutput(errOut)
	basePath := fs.String("base", "", "Current YAML policy path (required)")
	candidatePath := fs.String("candidate", "", "Proposed YAML policy path (required)")
	asJSON := fs.Bool("json", false, "Machine-readable policy impact on stdout")
	fs.Usage = func() {
		fmt.Fprintln(errOut, "Usage: vanguard compare -base current.yaml -candidate proposed.yaml [-json] report.json|reports-dir\nExits: 0 no relaxed limits, 1 review required, 2 error. No Slack notifications.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *basePath == "" || *candidatePath == "" || fs.NArg() != 1 {
		fmt.Fprintln(errOut, "usage: vanguard compare -base current.yaml -candidate proposed.yaml [-json] report.json|reports-dir")
		return 2
	}
	base, err := readBoundedFile(*basePath, 1<<20)
	if err != nil {
		fmt.Fprintln(errOut, "base policy:", err)
		return 2
	}
	candidate, err := readBoundedFile(*candidatePath, 1<<20)
	if err != nil {
		fmt.Fprintln(errOut, "candidate policy:", err)
		return 2
	}
	inputs, err := loadImpactReports(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	impact, err := gate.ComparePolicies(base, candidate, inputs)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	if *asJSON {
		err = json.NewEncoder(out).Encode(impact)
	} else {
		err = writeImpact(out, impact)
	}
	if err != nil {
		fmt.Fprintln(errOut, "cannot write policy impact")
		return 2
	}
	if impact.ReviewRequired {
		return 1
	}
	return 0
}

func limitText(limit gate.Limit) string {
	if !limit.Limited {
		return "unlimited"
	}
	return fmt.Sprint(limit.Allowed)
}

func writeImpact(out io.Writer, impact *gate.PolicyImpact) error {
	var b strings.Builder
	status := "NO RELAXED LIMITS"
	if impact.ReviewRequired {
		status = "REVIEW REQUIRED"
	}
	fmt.Fprintln(&b, "Policy impact:", status)
	for _, change := range impact.Limits {
		fmt.Fprintf(&b, "%s: %s -> %s (%s)\n", change.Severity, limitText(change.Base), limitText(change.Candidate), change.Direction)
	}
	s := impact.Summary
	fmt.Fprintf(&b, "Reports=%d newly_passing=%d newly_failing=%d unchanged_pass=%d unchanged_fail=%d\n", s.Reports, s.NewlyPassing, s.NewlyFailing, s.UnchangedPassed, s.UnchangedFailed)
	for _, report := range impact.Reports {
		fmt.Fprintf(&b, "%q: %s\n", report.Name, report.Transition)
	}
	fmt.Fprintf(&b, "Base SHA256: %s\nCandidate SHA256: %s\nSample results do not predict unseen reports or approve a policy.\n", impact.BasePolicyHash, impact.CandidatePolicyHash)
	_, err := io.WriteString(out, b.String())
	return err
}
