package gate

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

const policy = "apiVersion: vanguard/v1\nkind: SecurityPolicy\nthresholds: {high: 2, critical: 0, low: -1}\n"

func TestDecisionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		body string
		pass bool
	}{{`{"findings":[]}`, true}, {`{"findings":[{"id":"1","severity":"HIGH"},{"id":"2","severity":"high"}]}`, true}, {`{"findings":[{"id":"1","severity":"high"},{"id":"2","severity":"high"},{"id":"3","severity":"high"}]}`, false}, {`{"findings":[{"severity":"info"}]}`, true}, {`{"findings":[{"severity":"critical"}]}`, false}} {
		d, e := Evaluate([]byte(tc.body), []byte(policy))
		if e != nil || d.Passed != tc.pass || len(d.PolicyHash) != 64 || len(d.InputHash) != 64 {
			t.Fatal(d, e)
		}
	}
}
func TestBadInput(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"findings":[],"findings":[]}`, `{"findings":[],"unexpected":true}`, `{"findings":[]} {}`, `{"findings":[{"severity":"urgent"}]}`, `{"findings":[{"severity":"high"},{"severity":"HIGH"}]}`, strings.Repeat(" ", 16<<20) + `{}`} {
		if _, e := Evaluate([]byte(body), []byte(policy)); e == nil {
			t.Fatal("accepted invalid or duplicate input")
		}
	}
	if _, e := Evaluate([]byte(`{"findings":[]}`), []byte(strings.Repeat("x", (1<<20)+1))); e == nil {
		t.Fatal("oversized policy")
	}
}

func TestRejectAmbiguousJSONFieldCasing(t *testing.T) {
	for _, input := range []string{
		`{"findings":[{"severity":"critical"}],"Findings":[]}`,
		`{"findings":[{"severity":"critical","Severity":"low"}]}`,
		`{"findings":[{"severity":"critical","ſeverity":"low"}]}`,
	} {
		if d, err := Evaluate([]byte(input), []byte(policy)); err == nil {
			t.Fatalf("ambiguous field casing accepted: %s -> %+v", input, d)
		}
	}
}

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestSlackFailureDoesNotMutateVerdict(t *testing.T) {
	d, _ := Evaluate([]byte(`{"findings":[{"severity":"critical"}]}`), []byte(policy))
	calls := 0
	client := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		calls++
		b, _ := io.ReadAll(r.Body)
		if strings.Contains(string(b), "findings") || strings.Contains(string(b), "private-source") {
			t.Fatal("source disclosed")
		}
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("rate limited")), Header: http.Header{}}, nil
	})}
	if e := Notify(context.Background(), client, "https://hooks.slack.com/services/TEST/DEMO/TOKEN", d, "https://github.com/example/actions/runs/1"); e == nil {
		t.Fatal("failure hidden")
	}
	if d.Passed || calls != 1 {
		t.Fatal("verdict changed")
	}
	d.Passed = true
	Notify(context.Background(), client, "https://hooks.slack.com/services/TEST/DEMO/TOKEN", d, "")
	if calls != 1 {
		t.Fatal("pass notified")
	}
	if e := Notify(context.Background(), client, "https://evil.test/services/x", d, ""); e == nil {
		t.Fatal("invalid webhook")
	}
}
