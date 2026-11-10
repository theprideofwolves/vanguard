package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("synthetic output failure") }

func TestCLIOutputFailure(t *testing.T) {
	for _, format := range [][]string{nil, {"-json"}} {
		args := append([]string{"-p", "../../policy.yaml"}, format...)
		args = append(args, "../../testdata/report-pass.json")
		var errs bytes.Buffer
		if code := run(args, failingWriter{}, &errs); code != 2 || !strings.Contains(errs.String(), "cannot write decision") {
			t.Fatal(code, errs.String())
		}
	}
}

type mockTransport func(*http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCLIMockedNotificationPreservesExitStatus(t *testing.T) {
	t.Setenv("VANGUARD_SLACK_WEBHOOK", "https://hooks.slack.com/services/FAKE/FAKE/fake-secret")
	t.Setenv("VANGUARD_RUN_URL", "")
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	for _, outcome := range []string{"429", "redirect", "transport"} {
		t.Run(outcome, func(t *testing.T) {
			calls := 0
			http.DefaultTransport = mockTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if outcome == "transport" {
					return nil, errors.New("fake-secret transport failure")
				}
				status := 429
				if outcome == "redirect" {
					status = 302
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://example.test/never-follow"}}, Body: io.NopCloser(strings.NewReader("fake-secret"))}, nil
			})
			for _, fixture := range []struct {
				file string
				code int
			}{{"report-pass.json", 0}, {"report-fail.json", 1}} {
				var out, errs bytes.Buffer
				code := run([]string{"-p", "../../policy.yaml", "-json", "-notify", "../../testdata/" + fixture.file}, &out, &errs)
				var d struct{ Passed bool }
				if code != fixture.code || json.Unmarshal(out.Bytes(), &d) != nil || d.Passed != (fixture.code == 0) {
					t.Fatal(code, out.String(), errs.String())
				}
				if strings.Contains(errs.String(), "fake-secret") || (fixture.code == 0 && errs.Len() != 0) || (fixture.code == 1 && errs.Len() == 0) {
					t.Fatal("notification diagnostic contract", errs.String())
				}
			}
			if calls != 1 {
				t.Fatalf("unexpected retries, redirects or PASS notification: %d", calls)
			}
		})
	}
}

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
