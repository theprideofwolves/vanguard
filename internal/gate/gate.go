package gate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/theprideofwolves/vanguard/internal/evaluator"
	"github.com/theprideofwolves/vanguard/internal/models"
	"github.com/theprideofwolves/vanguard/internal/parser"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Row struct {
	Severity          models.Severity
	Observed          int
	Allowed           int
	Limited, Exceeded bool
}
type Decision struct {
	Version, PolicyHash, InputHash, PolicyName string
	At                                         time.Time
	Passed                                     bool
	Rows                                       []Row
	Violations                                 []string
}

func Hash(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func Evaluate(input, policy []byte) (*Decision, error) {
	p, e := evaluator.ParsePolicy(policy)
	if e != nil {
		return nil, e
	}
	r, e := parser.Parse(strings.NewReader(string(input)))
	if e != nil {
		return nil, e
	}
	v := evaluator.Evaluate(r, p)
	d := &Decision{Version: "vanguard-decision/v1", PolicyHash: Hash(policy), InputHash: Hash(input), PolicyName: p.Metadata.Name, At: time.Now().UTC(), Passed: v.Passed, Violations: v.Violations}
	for _, sev := range []models.Severity{"critical", "high", "medium", "low", "info"} {
		limit, ok := p.Thresholds[sev]
		limited := ok && limit >= 0
		d.Rows = append(d.Rows, Row{sev, v.Summary[sev], limit, limited, limited && v.Summary[sev] > limit})
	}
	return d, nil
}
func Notify(ctx context.Context, client *http.Client, hook string, d *Decision, runURL string) error {
	u, e := url.Parse(hook)
	if e != nil || u.Scheme != "https" || (u.Host != "hooks.slack.com" && u.Host != "hooks.slack-gov.com") || !strings.HasPrefix(u.Path, "/services/") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid Slack webhook origin/path")
	}
	if d.Passed {
		return nil
	}
	text := "Vanguard policy failed. Policy SHA256: " + d.PolicyHash + ". Inspect the decision report in CI."
	for _, row := range d.Rows {
		if row.Exceeded {
			text += fmt.Sprintf(" %s observed=%d allowed=%d.", row.Severity, row.Observed, row.Allowed)
		}
	}
	if runURL != "" {
		v, e := url.Parse(runURL)
		if e != nil || v.Scheme != "https" || v.Host == "" || v.User != nil {
			return fmt.Errorf("run URL must be HTTPS")
		}
		text += " Run: " + runURL
	}
	b, _ := json.Marshal(map[string]string{"text": text})
	req, e := http.NewRequestWithContext(ctx, "POST", hook, strings.NewReader(string(b)))
	if e != nil {
		return fmt.Errorf("Slack request invalid")
	}
	req.Header.Set("Content-Type", "application/json")
	res, e := client.Do(req)
	if e != nil {
		return fmt.Errorf("Slack transport failed; inspect channel before retrying")
	}
	defer res.Body.Close()
	body, e := io.ReadAll(io.LimitReader(res.Body, 4096))
	if e != nil || res.StatusCode != 200 || strings.TrimSpace(string(body)) != "ok" {
		return fmt.Errorf("Slack did not confirm delivery (HTTP %d)", res.StatusCode)
	}
	return nil
}
