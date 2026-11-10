package gate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDecisionEvidenceMatchesExactBytes(t *testing.T) {
	input := []byte(" {\"findings\":[{\"id\":\"1\",\"severity\":\"HIGH\"},{\"id\":\"2\",\"severity\":\"high\"},{\"id\":\"3\",\"severity\":\"high\"},{\"severity\":\"low\"},{\"severity\":\"info\"}]}\r\n")
	p := []byte(policy + "# exact policy bytes\r\n")
	d, err := Evaluate(input, p)
	if err != nil {
		t.Fatal(err)
	}
	inputSum, policySum := sha256.Sum256(input), sha256.Sum256(p)
	if d.InputHash != hex.EncodeToString(inputSum[:]) || d.PolicyHash != hex.EncodeToString(policySum[:]) {
		t.Fatal("hashes do not identify raw bytes")
	}
	if d.Passed || d.Version != "vanguard-decision/v1" || d.At.IsZero() || d.At.Location().String() != "UTC" {
		t.Fatalf("wrong decision: %+v", d)
	}
	if len(d.Rows) != 5 || len(d.Violations) != 1 || d.Violations[0] != "high: found 3 (max allowed 2)" {
		t.Fatalf("wrong evidence: %+v", d)
	}
	for i, want := range []Row{{"critical", 0, 0, true, false}, {"high", 3, 2, true, true}, {"medium", 0, 0, false, false}, {"low", 1, -1, false, false}, {"info", 1, 0, false, false}} {
		if d.Rows[i] != want {
			t.Fatalf("row %d: %+v want %+v", i, d.Rows[i], want)
		}
	}
}

func TestMalformedReports(t *testing.T) {
	for _, input := range []string{"", "{", `[]`, `{"findings":null}`, `{"findings":[null]}`, `{"findings":[{}]}`, `{"findings":[{"severity":null}]}`, `{"findings":[{"severity":2}]}`, `{"findings":[{"severity":"high","extra":1}]}`} {
		if _, err := Evaluate([]byte(input), []byte(policy)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}

func TestSlackSummaryAndFailuresUseMockTransport(t *testing.T) {
	d, err := Evaluate([]byte(`{"tool":"private-tool","target":"private-target","findings":[{"severity":"critical","description":"private-description"},{"severity":"low"}]}`), []byte(policy))
	if err != nil {
		t.Fatal(err)
	}
	// Fake endpoint and run link only; this transport never opens a socket.
	hook := "https://hooks.slack.com/services/FAKE/FAKE/fake-secret"
	runURL := "https://example.test/run/1"
	want := fmt.Sprintf("Vanguard policy failed. Policy SHA256: %s. Inspect the decision report in CI. critical observed=1 allowed=0. Run: %s", d.PolicyHash, runURL)
	for _, outcome := range []string{"ok", "429", "redirect", "transport"} {
		t.Run(outcome, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.String() != hook || r.Header.Get("Content-Type") != "application/json" {
					t.Fatal("wrong request")
				}
				var payload map[string]string
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if len(payload) != 1 || payload["text"] != want {
					t.Fatalf("unexpected disclosure: %v", payload)
				}
				if outcome == "transport" {
					return nil, fmt.Errorf("transport error contains %s", hook)
				}
				status, body := 200, "ok"
				if outcome == "429" {
					status, body = 429, "fake-secret private-description"
				}
				if outcome == "redirect" {
					status = 302
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Location": {"https://example.test/redirect"}}}, nil
			}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			err := Notify(context.Background(), client, hook, d, runURL)
			if (err == nil) != (outcome == "ok") || calls != 1 || d.Passed {
				t.Fatalf("wrong outcome: %v calls=%d", err, calls)
			}
			if err != nil && (strings.Contains(err.Error(), "fake-secret") || strings.Contains(err.Error(), "private-description")) {
				t.Fatal("error leaked private data")
			}
		})
	}
}

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
