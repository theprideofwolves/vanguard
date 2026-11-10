package gate

import (
	"encoding/json"
	"fmt"
	"testing"
)

func benchmarkReport(n int) []byte {
	findings := make([]map[string]string, n)
	for i := range findings {
		findings[i] = map[string]string{"id": fmt.Sprintf("SYN-%06d", i), "severity": referenceSeverities[i%5], "description": "Synthetic benchmark finding; no real vulnerability or source data."}
	}
	b, _ := json.Marshal(map[string]any{"tool": "synthetic-benchmark", "target": "fictional-app", "findings": findings})
	return b
}

// Measures in-process parsing, validation, evaluation and hashing; excludes file
// I/O, process startup, export serialization, network and test-data generation.
func BenchmarkEvaluate(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("findings_%d", n), func(b *testing.B) {
			input := benchmarkReport(n)
			p := impactPolicy("critical: 0, high: 2")
			b.SetBytes(int64(len(input)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Evaluate(input, p); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkComparePolicies(b *testing.B) {
	for _, n := range []int{1, 8, 32} {
		b.Run(fmt.Sprintf("reports_%d_findings_1000_each", n), func(b *testing.B) {
			input := benchmarkReport(1000)
			base, candidate := impactPolicy("critical: 0, high: 2"), impactPolicy("critical: 1, high: 0")
			reports := make([]ReportInput, n)
			for i := range reports {
				reports[i] = ReportInput{fmt.Sprintf("%02d.json", i), input}
			}
			b.SetBytes(int64(n * len(input)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := ComparePolicies(base, candidate, reports); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
