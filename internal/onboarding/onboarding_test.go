package onboarding

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func req(h http.Handler, method, path string, form url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	out := httptest.NewRecorder()
	h.ServeHTTP(out, r)
	return out
}
func TestFirstRunChoiceAndPersistence(t *testing.T) {
	dir := t.TempDir()
	w := New(Config{Name: "Vanguard", Token: "private", Dir: dir, Expected: "one synthetic result", DemoFields: map[string]string{"action": "demo"}})
	calls := 0
	h := w.Wrap(http.HandlerFunc(func(out http.ResponseWriter, r *http.Request) { calls++; io.WriteString(out, "workspace") }))
	out := req(h, "GET", "/?token=private", nil)
	if out.Header().Get("Referrer-Policy") != "same-origin" {
		t.Fatal("form-compatible referrer policy missing")
	}
	if out.Code != 200 || !strings.Contains(out.Body.String(), "Try the guided demo") || calls != 0 {
		t.Fatal(out.Body.String(), calls)
	}
	out = req(h, "POST", "/onboarding/start?token=private", url.Values{"mode": {"own"}})
	if out.Code != 303 {
		t.Fatal(out.Code)
	}
	out = req(h, "GET", "/?token=private", nil)
	if out.Body.String() != "workspace" {
		t.Fatal(out.Body.String())
	}
	if _, e := os.Stat(filepath.Join(dir, "onboarding.json")); !os.IsNotExist(e) {
		t.Fatal("choice falsely marked successful")
	}
	w.Complete()
	next := New(w.c)
	if !next.done || !next.started {
		t.Fatal("completion not remembered")
	}
	out = req(h, "GET", "/welcome?token=private", nil)
	if !strings.Contains(out.Body.String(), "Return to your workspace") {
		t.Fatal("guide not restartable")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "onboarding.json"))
	var state map[string]any
	if json.Unmarshal(b, &state) != nil || len(state) != 2 {
		t.Fatal(string(b))
	}
}
func TestRequestAndSampleBoundaries(t *testing.T) {
	w := New(Config{Token: "t", Host: "localhost", Samples: []Sample{{"example.txt", "synthetic"}}})
	h := w.Wrap(http.HandlerFunc(func(out http.ResponseWriter, r *http.Request) { out.WriteHeader(204) }))
	for _, tc := range []struct {
		method, path string
		code         int
	}{{"GET", "/", 403}, {"DELETE", "/?token=t", 405}, {"GET", "/onboarding/start?token=t", 405}, {"POST", "/welcome?token=t", 405}, {"GET", "/onboarding/sample?token=t&name=../../secret", 404}} {
		if got := req(h, tc.method, tc.path, nil).Code; got != tc.code {
			t.Fatalf("%s %s: %d", tc.method, tc.path, got)
		}
	}
	out := req(h, "GET", "/onboarding/sample?token=t&name=example.txt", nil)
	if out.Body.String() != "synthetic" || !strings.Contains(out.Header().Get("Content-Disposition"), "example.txt") {
		t.Fatal(out)
	}
	if req(h, "POST", "/onboarding/start?token=t", url.Values{"mode": {"wrong"}}).Code != 400 {
		t.Fatal("bad choice")
	}
	r := httptest.NewRequest("POST", "http://localhost/onboarding/start?token=t", nil)
	r.Header.Set("Origin", "https://evil.test")
	out = httptest.NewRecorder()
	h.ServeHTTP(out, r)
	if out.Code != 403 {
		t.Fatal("origin")
	}
	r = httptest.NewRequest("POST", "http://localhost/onboarding/start?token=t", nil)
	r.Header.Set("Origin", "null")
	out = httptest.NewRecorder()
	h.ServeHTTP(out, r)
	if out.Code != 403 {
		t.Fatal("opaque origin allowed")
	}
}
func TestPreferenceFailures(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "onboarding.json"), []byte("corrupt"), 0600)
	w := New(Config{Dir: dir})
	if w.done || w.started {
		t.Fatal("corrupt preference trusted")
	}
	w.c.Dir = filepath.Join(dir, "missing")
	w.Complete()
	if w.warning == "" {
		t.Fatal("preference write error hidden")
	}
}
