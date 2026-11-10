package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func evaluateUI(t *testing.T, h http.Handler, policy, report string) string {
	t.Helper()
	var b bytes.Buffer
	f := multipart.NewWriter(&b)
	f.WriteField("policy", policy)
	f.WriteField("report", report)
	f.Close()
	r := httptest.NewRequest("POST", "/?token=token", &b)
	r.Header.Set("Content-Type", f.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("evaluation status %d", w.Code)
	}
	return w.Body.String()
}

func TestAppearanceAndSessionInputs(t *testing.T) {
	h := newUI("token")
	policy := strings.Replace(samplePolicy, "example-baseline", "my-policy", 1)
	report := `{"target":"</textarea><script>alert(1)</script>","findings":[]}`
	page := evaluateUI(t, h, policy, report)
	if strings.Contains(page, "<script>") || !strings.Contains(page, "&lt;/textarea&gt;") {
		t.Fatal("report escaped its textarea")
	}
	before := httptest.NewRecorder()
	h.ServeHTTP(before, httptest.NewRequest("GET", "/export?token=token", nil))
	for _, path := range []string{"/", "/welcome", "/setup"} {
		form := url.Values{"theme": {"dark"}, "return": {path}}
		r := httptest.NewRequest("POST", "/theme?token=token", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		if out.Code != 303 || out.Header().Get("Location") != path+"?token=token" {
			t.Fatal(out.Code, out.Header())
		}
		cookies := out.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
			t.Fatal("unsafe theme cookie")
		}
		r = httptest.NewRequest("GET", path+"?token=token", nil)
		r.AddCookie(cookies[0])
		out = httptest.NewRecorder()
		h.ServeHTTP(out, r)
		if !strings.Contains(out.Body.String(), `data-theme="dark"`) || !strings.Contains(out.Body.String(), "Light mode") || strings.Contains(out.Body.String(), "ZgotmplZ") {
			t.Fatal("theme missing on ", path)
		}
	}
	for _, path := range []string{"/welcome", "/"} {
		out := httptest.NewRecorder()
		h.ServeHTTP(out, httptest.NewRequest("GET", path+"?token=token", nil))
		if path == "/" && (!strings.Contains(out.Body.String(), "my-policy") || !strings.Contains(out.Body.String(), "&lt;/textarea&gt;")) {
			t.Fatal("GET reset evaluated inputs")
		}
	}
	after := httptest.NewRecorder()
	h.ServeHTTP(after, httptest.NewRequest("GET", "/export?token=token", nil))
	if before.Body.String() != after.Body.String() {
		t.Fatal("appearance/navigation changed decision")
	}
}

func TestThemeRequestGuards(t *testing.T) {
	h := newUI("token")
	for _, tc := range []struct {
		method, query, body, origin string
		want                        int
	}{
		{"GET", "?token=token", "", "", 405},
		{"POST", "", "theme=dark", "", 403},
		{"POST", "?token=token", "theme=dark", "https://example.test", 403},
		{"POST", "?token=token", "theme=other", "", 400},
		{"POST", "?token=token", "theme=dark&extra=" + strings.Repeat("x", 1024), "", 400},
		{"POST", "?token=token", "theme=light&return=https://example.test", "", 303},
	} {
		r := httptest.NewRequest(tc.method, "http://localhost/theme"+tc.query, strings.NewReader(tc.body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("status %d want %d", w.Code, tc.want)
		}
		if tc.want == 303 && w.Header().Get("Location") != "/?token=token" {
			t.Fatal("open redirect")
		}
	}
}

func TestConcurrentEvaluationAndExport(t *testing.T) {
	h := newUI("token")
	evaluateUI(t, h, samplePolicy, sampleReport)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%3 == 0 {
				evaluateUI(t, h, samplePolicy, `{"findings":[]}`)
				return
			}
			path := "/?token=token"
			if i%3 == 1 {
				path = "/export?token=token"
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			if w.Code != 200 {
				t.Errorf("concurrent request: %d", w.Code)
			}
		}(i)
	}
	wg.Wait()
}
