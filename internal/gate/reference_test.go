package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// This small specification model uses accepted-count sets, independently of the
// production parser/evaluator. Generated policies and reports traverse the public
// gate APIs; it is not a claim of exhaustive correctness or scanner accuracy.
var referenceSeverities = []string{"critical", "high", "medium", "low", "info"}

func referencePolicy(rng *rand.Rand) ([]byte, [5]int) {
	var b strings.Builder
	b.WriteString("apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds:\n")
	var limits [5]int
	fields := 0
	for i, severity := range referenceSeverities {
		limits[i] = rng.Intn(5) - 2 // -2 omitted, -1 unlimited, 0/1/2 finite
		if limits[i] != -2 {
			fmt.Fprintf(&b, "  %s: %d\n", severity, limits[i])
			fields++
		}
	}
	if fields == 0 {
		b.WriteString("  info: -1\n")
		limits[4] = -1
	}
	return []byte(b.String()), limits
}

func referenceReport(rng *rand.Rand) ([]byte, [5]int) {
	findings := make([]map[string]string, 0)
	var counts [5]int
	for i, severity := range referenceSeverities {
		counts[i] = rng.Intn(4)
		for j := 0; j < counts[i]; j++ {
			findings = append(findings, map[string]string{"id": fmt.Sprintf("%s-%d", severity, j), "severity": strings.ToUpper(severity)})
		}
	}
	b, _ := json.Marshal(map[string]any{"findings": findings})
	return b, counts
}

func referenceAccepts(limit, count int) bool {
	allowed := map[int][]int{0: {0}, 1: {0, 1}, 2: {0, 1, 2}}
	if limit < 0 {
		return true
	}
	for _, n := range allowed[limit] {
		if n == count {
			return true
		}
	}
	return false
}

func referencePass(limits, counts [5]int) bool {
	for i := range limits {
		if !referenceAccepts(limits[i], counts[i]) {
			return false
		}
	}
	return true
}

func TestGeneratedDecisionsAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20261110))
	for i := 0; i < 2048; i++ {
		p, limits := referencePolicy(rng)
		input, counts := referenceReport(rng)
		d, err := Evaluate(input, p)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if d.Passed != referencePass(limits, counts) || len(d.Rows) != 5 {
			t.Fatalf("case %d: wrong verdict", i)
		}
		violations := 0
		for j, row := range d.Rows {
			if row.Observed != counts[j] || string(row.Severity) != referenceSeverities[j] || row.Exceeded != !referenceAccepts(limits[j], counts[j]) {
				t.Fatalf("case %d: wrong row %+v", i, row)
			}
			if row.Exceeded {
				violations++
			}
		}
		if len(d.Violations) != violations {
			t.Fatalf("case %d: wrong violations", i)
		}
		sum := sha256.Sum256(input)
		if d.InputHash != hex.EncodeToString(sum[:]) {
			t.Fatalf("case %d: wrong input hash", i)
		}
		sum = sha256.Sum256(p)
		if d.PolicyHash != hex.EncodeToString(sum[:]) {
			t.Fatalf("case %d: wrong policy hash", i)
		}
	}
	t.Log("REFERENCE_EVALUATION_CASES=2048 seed=20261110")
}

func TestGeneratedComparisonsAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20261111))
	for i := 0; i < 1024; i++ {
		base, from := referencePolicy(rng)
		candidate, to := referencePolicy(rng)
		input, counts := referenceReport(rng)
		impact, err := ComparePolicies(base, candidate, []ReportInput{{"synthetic.json", input}})
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		review := false
		for j, change := range impact.Limits {
			direction := "unchanged"
			for n := 0; n <= 3; n++ {
				before, after := referenceAccepts(from[j], n), referenceAccepts(to[j], n)
				if !before && after {
					direction = "relaxed"
					break
				}
				if before && !after {
					direction = "tightened"
					break
				}
			}
			if direction == "relaxed" {
				review = true
			}
			if change.Direction != direction {
				t.Fatalf("case %d: wrong direction", i)
			}
		}
		before, after := referencePass(from, counts), referencePass(to, counts)
		if impact.ReviewRequired != review || impact.Reports[0].Base.Passed != before || impact.Reports[0].Candidate.Passed != after {
			t.Fatalf("case %d: wrong impact", i)
		}
		want := "fail_to_fail"
		if before && after {
			want = "pass_to_pass"
		} else if before {
			want = "pass_to_fail"
		} else if after {
			want = "fail_to_pass"
		}
		if impact.Reports[0].Transition != want {
			t.Fatalf("case %d: wrong transition", i)
		}
	}
	t.Log("REFERENCE_COMPARISON_CASES=1024 seed=20261111")
}
