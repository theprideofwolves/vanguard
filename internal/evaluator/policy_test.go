package evaluator

import (
	"os"
	"path/filepath"
	"testing"
)

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
