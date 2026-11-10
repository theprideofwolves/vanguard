package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprideofwolves/vanguard/internal/gate"
)

func TestCompareCLI(t *testing.T) {
	dir := t.TempDir()
	base, candidate := filepath.Join(dir, "base.yaml"), filepath.Join(dir, "candidate.yaml")
	if err := os.WriteFile(base, []byte(samplePolicy), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte(sampleCandidatePolicy), 0600); err != nil {
		t.Fatal(err)
	}
	reports := filepath.Join(dir, "reports")
	if err := os.Mkdir(reports, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(reports, "report.json")
	if err := os.WriteFile(file, []byte(sampleReport), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reports, "ignored.txt"), []byte("not a report"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VANGUARD_SLACK_WEBHOOK", "https://example.test/never-used")
	for _, tc := range []struct {
		name string
		args []string
		code int
		text string
	}{
		{"relax", []string{"compare", "-base", base, "-candidate", candidate, "-json", reports}, 1, ""},
		{"tighten", []string{"compare", "-base", candidate, "-candidate", base, "-json", file}, 0, ""},
		{"text", []string{"compare", "-base", base, "-candidate", candidate, file}, 1, "REVIEW REQUIRED"},
		{"no change", []string{"compare", "-base", base, "-candidate", base, file}, 0, "NO RELAXED LIMITS"},
		{"help", []string{"compare", "-h"}, 0, ""},
		{"missing flags", []string{"compare", file}, 2, ""},
		{"missing base", []string{"compare", "-base", base + "missing", "-candidate", candidate, file}, 2, ""},
		{"missing candidate", []string{"compare", "-base", base, "-candidate", candidate + "missing", file}, 2, ""},
		{"missing report", []string{"compare", "-base", base, "-candidate", candidate, file + "missing"}, 2, ""},
		{"reject notification", []string{"compare", "-base", base, "-candidate", candidate, "-notify", file}, 2, ""},
		{"extra arg", []string{"compare", "-base", base, "-candidate", candidate, file, file}, 2, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errs bytes.Buffer
			if code := run(tc.args, &out, &errs); code != tc.code {
				t.Fatalf("code=%d want=%d: %s", code, tc.code, errs.String())
			}
			if tc.code == 2 && (out.Len() != 0 || errs.Len() == 0) {
				t.Fatal("failure emitted evidence or omitted error")
			}
			if tc.text != "" && !strings.Contains(out.String(), tc.text) {
				t.Fatal(out.String())
			}
			if tc.name == "relax" || tc.name == "tighten" {
				var result gate.PolicyImpact
				if err := json.Unmarshal(out.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.ReviewRequired != (tc.code == 1) || result.Summary.Reports != 1 {
					t.Fatal("wrong result")
				}
				if tc.name == "relax" && result.Summary.NewlyPassing != 1 {
					t.Fatal("missing newly passing report")
				}
			}
		})
	}
	for _, asJSON := range []bool{false, true} {
		args := []string{"compare", "-base", base, "-candidate", candidate}
		if asJSON {
			args = append(args, "-json")
		}
		args = append(args, file)
		var errs bytes.Buffer
		if code := run(args, failingWriter{}, &errs); code != 2 || !strings.Contains(errs.String(), "cannot write policy impact") {
			t.Fatal("output failure lost", code, errs.String())
		}
	}
	// A malformed member invalidates the whole corpus; no partial JSON is emitted.
	if err := os.WriteFile(filepath.Join(reports, "z-bad.json"), []byte("null"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if code := run([]string{"compare", "-base", base, "-candidate", candidate, "-json", reports}, &out, &errs); code != 2 || out.Len() != 0 {
		t.Fatal("partial evidence emitted", code)
	}
}

func TestImpactFileIntake(t *testing.T) {
	dir := t.TempDir()
	if _, err := loadImpactReports(dir); err == nil {
		t.Fatal("empty directory accepted")
	}
	for i := 0; i <= gate.MaxImpactReports; i++ {
		if err := os.WriteFile(filepath.Join(dir, strings.Repeat("r", i+1)+".json"), []byte(`{"findings":[]}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := loadImpactReports(dir); err == nil {
		t.Fatal("report count exceeded")
	}
	if _, err := readBoundedFile(dir, 1<<20); err == nil {
		t.Fatal("directory used as policy")
	}
	file := filepath.Join(t.TempDir(), "policy")
	if err := os.WriteFile(file, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedFile(file, 4); err == nil {
		t.Fatal("file size cap exceeded")
	}
	if b, err := readBoundedFile(file, 5); err != nil || string(b) != "12345" {
		t.Fatal(b, err)
	}
}
