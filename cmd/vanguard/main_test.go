package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		code        int
	}{
		{"pass", "report-pass.json", 0}, {"fail", "report-fail.json", 1}, {"missing", "missing.json", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errs bytes.Buffer
			got := run([]string{"-p", "../../policy.yaml", "../../testdata/" + tc.input}, &out, &errs)
			if got != tc.code {
				t.Fatalf("code=%d want %d: %s", got, tc.code, errs.String())
			}
			if tc.code < 2 && !strings.Contains(out.String(), "Result:") {
				t.Fatal("missing verdict")
			}
		})
	}
}

func TestRejectInvalidInput(t *testing.T) {
	for _, input := range []string{
		`{"findings":[{"severity":"urgent"}]}`,
		`{"findings":[]} {"findings":[]}`,
		`{}`, `null`,
	} {
		path := filepath.Join(t.TempDir(), "input.json")
		if err := os.WriteFile(path, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		var out, errs bytes.Buffer
		if got := run([]string{"-p", "../../policy.yaml", path}, &out, &errs); got != 2 {
			t.Fatalf("input %s: code=%d", input, got)
		}
	}
}
