package main

import (
	"bytes"
	"encoding/json"
	"github.com/theprideofwolves/vanguard/internal/gate"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadOverridesPasteAndInvalidInputClearsExport(t *testing.T) {
	dir := t.TempDir()
	h := newUIAt("token", dir)
	post := func(text, file string) *httptest.ResponseRecorder {
		t.Helper()
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		form.WriteField("policy", samplePolicy)
		form.WriteField("report", text)
		if file != "" {
			part, err := form.CreateFormFile("reportfile", "own-report.json")
			if err != nil {
				t.Fatal(err)
			}
			part.Write([]byte(file))
		}
		form.Close()
		r := httptest.NewRequest("POST", "http://localhost/?token=token", &body)
		r.Header.Set("Content-Type", form.FormDataContentType())
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		return out
	}
	// An invalid first evaluation must not save guide completion.
	if out := post(`{"findings":[{"severity":"urgent"}]}`, ""); out.Code != 200 || !strings.Contains(out.Body.String(), "unsupported severity") {
		t.Fatal(out.Code, out.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "onboarding.json")); !os.IsNotExist(err) {
		t.Fatal("invalid evaluation completed guide", err)
	}
	upload := "{\"findings\":[]}\n"
	if out := post("not-json", upload); out.Code != 200 || !strings.Contains(out.Body.String(), "<h2>PASS</h2>") {
		t.Fatal(out.Code, out.Body.String())
	}
	out := httptest.NewRecorder()
	h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost/export?token=token", nil))
	var decision gate.Decision
	if err := json.Unmarshal(out.Body.Bytes(), &decision); err != nil || !decision.Passed || decision.InputHash != gate.Hash([]byte(upload)) {
		t.Fatal("wrong uploaded evidence", out.Body.String(), err)
	}
	if _, err := os.Stat(filepath.Join(dir, "onboarding.json")); err != nil {
		t.Fatal("valid evaluation not remembered", err)
	}
	post(sampleReport, `{"findings":[{"severity":"critical","Severity":"low"}]}`)
	out = httptest.NewRecorder()
	h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost/export?token=token", nil))
	if out.Code != 400 {
		t.Fatal("invalid upload left stale export", out.Body.String())
	}
	for _, path := range []string{"/setup/check", "/missing"} {
		out = httptest.NewRecorder()
		h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost"+path+"?token=token", nil))
		if out.Code != 404 {
			t.Fatal("unsupported route available", path, out.Code)
		}
	}
}

func TestUIEvaluationAndExport(t *testing.T) {
	h := newUI("token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/?token=token", nil))
	if !strings.Contains(w.Body.String(), "background:#0b1627") {
		t.Fatal("CSS not rendered")
	}
	var b bytes.Buffer
	f := multipart.NewWriter(&b)
	f.WriteField("policy", samplePolicy)
	f.WriteField("report", sampleReport)
	f.Close()
	r := httptest.NewRequest("POST", "http://localhost/?token=token", &b)
	r.Header.Set("Content-Type", f.FormDataContentType())
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "FAIL") {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/export?token=token", nil))
	if !strings.Contains(w.Body.String(), "PolicyHash") {
		t.Fatal(w.Body.String())
	}
}
