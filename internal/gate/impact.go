package gate

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/theprideofwolves/vanguard/internal/evaluator"
	"github.com/theprideofwolves/vanguard/internal/models"
	"github.com/theprideofwolves/vanguard/internal/parser"
)

const (
	MaxImpactReports = 32
	MaxImpactBytes   = 64 << 20
)

type ReportInput struct {
	Name string
	Data []byte
}

// Allowed is canonicalized to -1 for both omitted and explicit unlimited limits.
type Limit struct {
	Limited bool
	Allowed int
}

type LimitChange struct {
	Severity        models.Severity
	Base, Candidate Limit
	Direction       string // unchanged, relaxed or tightened
}

type ReportImpact struct {
	Name            string
	Transition      string // pass_to_pass, fail_to_pass, pass_to_fail or fail_to_fail
	Base, Candidate *Decision
}

type ImpactSummary struct {
	Reports, BasePassed, CandidatePassed                         int
	NewlyPassing, NewlyFailing, UnchangedPassed, UnchangedFailed int
}

type PolicyImpact struct {
	Version                             string
	At                                  time.Time
	BasePolicyHash, CandidatePolicyHash string
	BasePolicyName, CandidatePolicyName string
	ReviewRequired                      bool
	Limits                              []LimitChange
	Summary                             ImpactSummary
	Reports                             []ReportImpact
}

func effectiveLimit(p *evaluator.Policy, severity models.Severity) Limit {
	if n, ok := p.Thresholds[severity]; ok && n >= 0 {
		return Limit{true, n}
	}
	return Limit{false, -1}
}

func limitDirection(base, candidate Limit) string {
	if base == candidate {
		return "unchanged"
	}
	if !candidate.Limited || (base.Limited && candidate.Allowed > base.Allowed) {
		return "relaxed"
	}
	return "tightened"
}

// ComparePolicies evaluates identical report bytes against both policies. It is
// atomic: any invalid input returns no evidence. Limits are compared independently
// of the sample corpus; a relaxation needs review even when no verdict flips.
func ComparePolicies(baseBytes, candidateBytes []byte, inputs []ReportInput) (*PolicyImpact, error) {
	base, err := evaluator.ParsePolicy(baseBytes)
	if err != nil {
		return nil, fmt.Errorf("base policy: %w", err)
	}
	candidate, err := evaluator.ParsePolicy(candidateBytes)
	if err != nil {
		return nil, fmt.Errorf("candidate policy: %w", err)
	}
	if len(inputs) == 0 || len(inputs) > MaxImpactReports {
		return nil, fmt.Errorf("provide 1 to %d reports", MaxImpactReports)
	}
	ordered := append([]ReportInput(nil), inputs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	total := 0
	for i, input := range ordered {
		if input.Name == "" || len(input.Name) > 255 || (i > 0 && ordered[i-1].Name == input.Name) {
			return nil, fmt.Errorf("report names must be unique and 1 to 255 bytes")
		}
		if len(input.Data) > MaxImpactBytes-total {
			return nil, fmt.Errorf("report corpus exceeds 64 MiB")
		}
		total += len(input.Data)
	}
	impact := &PolicyImpact{
		Version: "vanguard-impact/v1", At: time.Now().UTC(),
		BasePolicyHash: Hash(baseBytes), CandidatePolicyHash: Hash(candidateBytes),
		BasePolicyName: base.Metadata.Name, CandidatePolicyName: candidate.Metadata.Name,
	}
	for _, severity := range []models.Severity{"critical", "high", "medium", "low", "info"} {
		from, to := effectiveLimit(base, severity), effectiveLimit(candidate, severity)
		direction := limitDirection(from, to)
		impact.Limits = append(impact.Limits, LimitChange{severity, from, to, direction})
		impact.ReviewRequired = impact.ReviewRequired || direction == "relaxed"
	}
	for _, input := range ordered {
		report, err := parser.Parse(bytes.NewReader(input.Data))
		if err != nil {
			return nil, fmt.Errorf("report %q: %w", input.Name, err)
		}
		hash := Hash(input.Data)
		before := decisionFor(report, base, hash, impact.BasePolicyHash, impact.At)
		after := decisionFor(report, candidate, hash, impact.CandidatePolicyHash, impact.At)
		transition := "fail_to_fail"
		switch {
		case before.Passed && after.Passed:
			transition = "pass_to_pass"
			impact.Summary.UnchangedPassed++
		case !before.Passed && after.Passed:
			transition = "fail_to_pass"
			impact.Summary.NewlyPassing++
		case before.Passed && !after.Passed:
			transition = "pass_to_fail"
			impact.Summary.NewlyFailing++
		default:
			impact.Summary.UnchangedFailed++
		}
		impact.Summary.Reports++
		if before.Passed {
			impact.Summary.BasePassed++
		}
		if after.Passed {
			impact.Summary.CandidatePassed++
		}
		impact.Reports = append(impact.Reports, ReportImpact{input.Name, transition, before, after})
	}
	return impact, nil
}
