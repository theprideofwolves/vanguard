// Package review adds pinned-source review context without changing the evaluator.
package review

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/theprideofwolves/vanguard/internal/gate"
)

const Schema = "vanguard-review/v1"

type Build struct {
	GoVersion      string
	SourceRevision string
	Modified       bool
}

type Result struct {
	Version        string
	Mode           string
	Status         string
	ResultExitCode int
	BaselinePassed bool
	BaseSource     GitSource
	CandidateLabel string
	Generator      Build
	Impact         *gate.PolicyImpact
}

func Analyze(source GitSource, base, candidate []byte, reports []gate.ReportInput, mode, label string) (*Result, error) {
	if mode != "review" && mode != "shadow" {
		return nil, fmt.Errorf("mode must be review or shadow")
	}
	if !objectID.MatchString(source.Commit) || !objectID.MatchString(source.Blob) || !validPolicyPath(source.Path) {
		return nil, fmt.Errorf("review requires pinned Git source metadata")
	}
	impact, err := gate.ComparePolicies(base, candidate, reports)
	if err != nil {
		return nil, err
	}
	r := &Result{Version: Schema, Mode: mode, Status: "no_relaxed_limits", BaselinePassed: true, BaseSource: source, CandidateLabel: label, Impact: impact, Generator: Build{GoVersion: runtime.Version()}}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				r.Generator.SourceRevision = setting.Value
			case "vcs.modified":
				r.Generator.Modified = setting.Value == "true"
			}
		}
	}
	for _, report := range impact.Reports {
		r.BaselinePassed = r.BaselinePassed && report.Base.Passed
	}
	if mode == "shadow" {
		r.Status = "baseline_pass"
		if !r.BaselinePassed {
			r.Status = "baseline_fail"
			r.ResultExitCode = 1
		}
	} else if impact.ReviewRequired {
		r.Status = "review_required"
		r.ResultExitCode = 1
	}
	return r, nil
}
