package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprideofwolves/vanguard/internal/gate"
)

func TestPolicyImpactUISession(t *testing.T) {
	dir := t.TempDir()
	h := newUIAt("token", dir)
	post := func(candidate, report, upload string) *httptest.ResponseRecorder {
		t.Helper()
		var body bytes.Buffer
		f := multipart.NewWriter(&body)
		f.WriteField("policy", samplePolicy)
		f.WriteField("candidate", candidate)
		f.WriteField("report", report)
		f.WriteField("action", "compare")
		if upload != "" {
			p, err := f.CreateFormFile("reportfile", "report.json")
			if err != nil {
				t.Fatal(err)
			}
			p.Write([]byte(upload))
		}
		f.Close()
		r := httptest.NewRequest("POST", "/?token=token", &body)
		r.Header.Set("Content-Type", f.FormDataContentType())
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	export := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/export?token=token", nil))
		return w
	}
	// Invalid first comparison does not complete onboarding; valid comparison does.
	post("null", sampleReport, "")
	if _, err := os.Stat(filepath.Join(dir, "onboarding.json")); !os.IsNotExist(err) {
		t.Fatal("invalid comparison completed guide", err)
	}
	post(sampleCandidatePolicy, sampleReport, "")
	if _, err := os.Stat(filepath.Join(dir, "onboarding.json")); err != nil {
		t.Fatal("valid comparison did not complete guide", err)
	}
	// Comparison replaces an existing ordinary decision in the single export slot.
	evaluateUI(t, h, samplePolicy, sampleReport)
	w := post(sampleCandidatePolicy, "malformed paste", sampleReport)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "REVIEW REQUIRED") || !strings.Contains(w.Body.String(), "Newly passing: 1") || strings.Contains(w.Body.String(), "ZgotmplZ") {
		t.Fatal("comparison UI incomplete", w.Body.String())
	}
	out := export()
	var result gate.PolicyImpact
	if out.Code != 200 || !strings.Contains(out.Header().Get("Content-Disposition"), "vanguard-policy-impact.json") || json.Unmarshal(out.Body.Bytes(), &result) != nil {
		t.Fatal("wrong export")
	}
	if result.Reports[0].Base.InputHash != gate.Hash([]byte(sampleReport)) || !result.Reports[0].Candidate.Passed {
		t.Fatal("comparison ignored upload")
	}
	before := out.Body.String()
	for _, path := range []string{"/welcome", "/setup", "/"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path+"?token=token", nil))
	}
	if export().Body.String() != before {
		t.Fatal("guide restart changed comparison evidence")
	}
	// Malformed comparison clears both kinds of evidence, including an older gate result.
	post("null", sampleReport, "")
	if export().Code != 400 {
		t.Fatal("invalid comparison left stale evidence")
	}
	post(sampleCandidatePolicy, "null", "")
	if export().Code != 400 {
		t.Fatal("invalid report left comparison evidence")
	}
	post(sampleCandidatePolicy, sampleReport, "")
	evaluateUI(t, h, samplePolicy, `{"findings":[]}`)
	var decision gate.Decision
	if json.Unmarshal(export().Body.Bytes(), &decision) != nil || decision.Version != "vanguard-decision/v1" || !decision.Passed {
		t.Fatal("evaluation did not replace impact")
	}
	post(sampleCandidatePolicy, sampleReport, "")
	evaluateUI(t, h, samplePolicy, "null")
	if export().Code != 400 {
		t.Fatal("invalid evaluation retained comparison")
	}
	// Policy names, including malicious markup, remain escaped in the impact view.
	malicious := strings.Replace(sampleCandidatePolicy, "proposed-baseline", `'<script>alert(1)</script>'`, 1)
	w = post(malicious, sampleReport, "")
	if strings.Contains(w.Body.String(), "<script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;") {
		t.Fatal("policy name escaped its context")
	}
}
