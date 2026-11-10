package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func impactPolicy(thresholds string) []byte {
	return []byte("apiVersion: vanguard/v1\nkind: SecurityPolicy\nmetadata: {name: review}\nthresholds: {" + thresholds + "}\n")
}

func TestPolicyImpactTransitionsAndEvidence(t *testing.T) {
	base, candidate := impactPolicy("critical: 0, high: 1"), impactPolicy("critical: 1, high: 0")
	inputs := []ReportInput{
		{"z-still-fails.json", []byte(`{"findings":[{"id":"1","severity":"high"},{"id":"2","severity":"high"},{"severity":"critical"}]}`)},
		{"b-new-pass.json", []byte(" {\"findings\":[{\"severity\":\"CRITICAL\"}]}\r\n")},
		{"a-still-passes.json", []byte(`{"findings":[]}`)},
		{"c-new-fail.json", []byte(`{"findings":[{"severity":"high"}]}`)},
	}
	result, err := ComparePolicies(base, candidate, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "vanguard-impact/v1" || !result.ReviewRequired || result.At.IsZero() || result.At.Location().String() != "UTC" {
		t.Fatalf("wrong metadata: %+v", result)
	}
	if result.Summary != (ImpactSummary{4, 2, 2, 1, 1, 1, 1}) {
		t.Fatalf("wrong summary: %+v", result.Summary)
	}
	sum := sha256.Sum256(base)
	if result.BasePolicyHash != hex.EncodeToString(sum[:]) {
		t.Fatal("base hash mismatch")
	}
	sum = sha256.Sum256(candidate)
	if result.CandidatePolicyHash != hex.EncodeToString(sum[:]) {
		t.Fatal("candidate hash mismatch")
	}
	for i, transition := range []string{"pass_to_pass", "fail_to_pass", "pass_to_fail", "fail_to_fail"} {
		row := result.Reports[i]
		if row.Transition != transition {
			t.Fatalf("report order/transition: %+v", row)
		}
		var raw []byte
		for _, input := range inputs {
			if input.Name == row.Name {
				raw = input.Data
			}
		}
		before, err := Evaluate(raw, base)
		if err != nil {
			t.Fatal(err)
		}
		after, err := Evaluate(raw, candidate)
		if err != nil {
			t.Fatal(err)
		}
		before.At, after.At = result.At, result.At
		if !reflect.DeepEqual(before, row.Base) || !reflect.DeepEqual(after, row.Candidate) {
			t.Fatal("comparison diverged from ordinary evaluation")
		}
	}
	if inputs[0].Name != "z-still-fails.json" {
		t.Fatal("caller input reordered")
	}
	if result.Limits[0].Direction != "relaxed" || result.Limits[1].Direction != "tightened" || result.Limits[4].Direction != "unchanged" {
		t.Fatal(result.Limits)
	}
	if len(result.Reports[1].Base.Violations) != 1 || len(result.Reports[1].Candidate.Violations) != 0 {
		t.Fatal("wrong explained verdict flip")
	}
	encoded, err := json.Marshal(result)
	if err != nil || !json.Valid(encoded) {
		t.Fatal(err)
	}
}

func TestEffectiveLimitChanges(t *testing.T) {
	for _, tc := range []struct {
		from, to, direction string
		review              bool
	}{
		{"high: 0", "high: 1", "relaxed", true},
		{"high: 1", "high: 0", "tightened", false},
		{"high: 0", "high: -1", "relaxed", true},
		{"high: -1", "critical: 0", "unchanged", false},
		{"critical: 0", "high: -1", "unchanged", true}, // critical becomes unlimited
		{"high: -1", "high: 0", "tightened", false},
		{"high: 1", "high: 1", "unchanged", false},
	} {
		t.Run(tc.from+"->"+tc.to, func(t *testing.T) {
			r, err := ComparePolicies(impactPolicy(tc.from), impactPolicy(tc.to), []ReportInput{{"empty.json", []byte(`{"findings":[]}`)}})
			if err != nil {
				t.Fatal(err)
			}
			if r.Limits[1].Direction != tc.direction || r.ReviewRequired != tc.review {
				t.Fatalf("wrong change: %+v", r)
			}
			if r.Summary.NewlyPassing != 0 || r.Summary.NewlyFailing != 0 {
				t.Fatal("empty report flipped")
			}
		})
	}
	// Whitespace, name and omitted vs -1 can change hashes without changing limits.
	r, err := ComparePolicies(impactPolicy("high: -1"), []byte("# different bytes\n"+string(impactPolicy("high: -1, low: -1"))), []ReportInput{{"report", []byte(`{"findings":[]}`)}})
	if err != nil || r.ReviewRequired || r.BasePolicyHash == r.CandidatePolicyHash {
		t.Fatal(r, err)
	}
}

func TestInvalidImpactIsAtomic(t *testing.T) {
	p := impactPolicy("high: 0")
	valid := ReportInput{"valid.json", []byte(`{"findings":[]}`)}
	for _, tc := range []struct {
		name            string
		base, candidate []byte
		inputs          []ReportInput
	}{
		{"invalid base", []byte("null"), p, []ReportInput{valid}},
		{"invalid candidate", p, []byte("null"), []ReportInput{valid}},
		{"no reports", p, p, nil},
		{"duplicate labels", p, p, []ReportInput{valid, valid}},
		{"missing label", p, p, []ReportInput{{"", valid.Data}}},
		{"bad report after valid", p, p, []ReportInput{valid, {"z-invalid", []byte("null")}}},
		{"ambiguous report", p, p, []ReportInput{{"bad", []byte(`{"findings":[],"Findings":[]}`)}}},
		{"oversized report", p, p, []ReportInput{{"large", []byte(strings.Repeat(" ", (16<<20)+1))}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := ComparePolicies(tc.base, tc.candidate, tc.inputs)
			if err == nil || r != nil {
				t.Fatal("invalid input returned evidence", err)
			}
		})
	}
	inputs := make([]ReportInput, MaxImpactReports)
	for i := range inputs {
		inputs[i] = ReportInput{fmt.Sprint(i), valid.Data}
	}
	if _, err := ComparePolicies(p, p, inputs); err != nil {
		t.Fatal("exact report count rejected", err)
	}
	inputs = append(inputs, ReportInput{"extra", valid.Data})
	if r, err := ComparePolicies(p, p, inputs); err == nil || r != nil {
		t.Fatal("too many reports accepted")
	}
}

func TestImpactCorpusByteBoundary(t *testing.T) {
	p := impactPolicy("high: 0")
	data := []byte(`{"findings":[]}` + strings.Repeat(" ", (16<<20)-len(`{"findings":[]}`)))
	inputs := []ReportInput{{"a", data}, {"b", data}, {"c", data}, {"d", data}}
	if _, err := ComparePolicies(p, p, inputs); err != nil {
		t.Fatal("exact corpus size rejected", err)
	}
	if r, err := ComparePolicies(p, p, append(inputs, ReportInput{"e", []byte(" ")})); err == nil || r != nil {
		t.Fatal("oversized corpus accepted")
	}
}

func FuzzPolicyImpact(f *testing.F) {
	f.Add(0, 1, `{"findings":[{"severity":"high"}]}`)
	f.Add(-1, 0, `{"findings":[]}`)
	f.Fuzz(func(t *testing.T, from, to int, report string) {
		if len(report) > 1<<16 {
			t.Skip()
		}
		// Exercise valid policies with arbitrary report syntax and boundary limits.
		from, to = int(uint(from)%5)-1, int(uint(to)%5)-1
		r, err := ComparePolicies(impactPolicy(fmt.Sprintf("high: %d", from)), impactPolicy(fmt.Sprintf("high: %d", to)), []ReportInput{{"fuzz", []byte(report)}})
		if err != nil {
			if r != nil {
				t.Fatal("partial evidence")
			}
			return
		}
		if !r.ReviewRequired && r.Summary.NewlyPassing != 0 {
			t.Fatal("a non-relaxing policy admitted a previously failing report")
		}
		if r.Reports[0].Base.InputHash != r.Reports[0].Candidate.InputHash {
			t.Fatal("different input evidence")
		}
	})
}
