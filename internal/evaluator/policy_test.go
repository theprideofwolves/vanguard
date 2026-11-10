package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyEdgeCases(t *testing.T) {
	base := "apiVersion: vanguard/v1\nkind: SecurityPolicy\n"
	for _, body := range []string{"", "null", "[]", base + "thresholds: null", base + "thresholds: {high: 0, high: 1}", base + "metadata: {name: x, typo: y}\nthresholds: {high: 0}", base + "thresholds: {HIGH: 0}", base + "thresholds: {high: 999999999999999999999999}", base + "thresholds: {high: .nan}", base + "thresholds: {high: [1]}", base + "kind: Other\nthresholds: {high: 1}"} {
		if _, err := ParsePolicy([]byte(body)); err == nil {
			t.Errorf("accepted invalid policy %q", body)
		}
	}
	p := base + "thresholds: {high: -1}\n#"
	p += strings.Repeat(" ", (1<<20)-len(p))
	if _, err := ParsePolicy([]byte(p)); err != nil {
		t.Fatal("exact size limit rejected", err)
	}
	if _, err := ParsePolicy([]byte(p + " ")); err == nil {
		t.Fatal("oversized policy accepted")
	}
}

func FuzzPolicy(f *testing.F) {
	for _, s := range []string{"null", "{}", "apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds: {high: 0}", "a: &a [*a]"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 1<<20 {
			t.Skip()
		}
		p, err := ParsePolicy([]byte(input))
		if err == nil && (p == nil || len(p.Thresholds) == 0) {
			t.Fatal("accepted empty policy")
		}
	})
}

func TestRejectNonIntegerThresholds(t *testing.T) {
	for _, value := range []string{"null", "~", "1.5", "true", `"2"`} {
		t.Run(value, func(t *testing.T) {
			body := "apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds: {high: " + value + "}\n"
			if _, err := ParsePolicy([]byte(body)); err == nil {
				t.Fatal("accepted non-integer threshold", value)
			}
		})
	}
}

func TestRejectInvalidPolicies(t *testing.T) {
	for _, body := range []string{
		"apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds: {critical: -2}",
		"apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds: {urgent: 0}",
		"apiVersion: wrong\nkind: SecurityPolicy\nthresholds: {high: 1}",
		"apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds: {}",
		"apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds: {high: 1}\ntypo: true",
		"apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds: {high: 1}\n---\n{}",
	} {
		p := filepath.Join(t.TempDir(), "policy.yaml")
		if err := os.WriteFile(p, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadPolicy(p); err == nil {
			t.Fatalf("accepted invalid policy: %s", body)
		}
	}
}
