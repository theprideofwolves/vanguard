package main

import (
	"encoding/json"
	"github.com/theprideofwolves/vanguard/internal/desktop"
	"github.com/theprideofwolves/vanguard/internal/gate"
	"html/template"
	"io"
	"net/http"
	"strings"
	"sync"
)

const samplePolicy = "apiVersion: vanguard/v1\nkind: SecurityPolicy\nmetadata:\n  name: example-baseline\nthresholds:\n  critical: 0\n  high: 2\n  medium: 10\n"
const sampleReport = `{"tool":"synthetic-demo","target":"fictional-app","findings":[{"id":"DEMO-1","severity":"high"},{"id":"DEMO-2","severity":"critical"}]}`

func launchUI() error {
	dir, e := desktop.DataDir("Vanguard")
	if e != nil {
		return e
	}
	return desktop.Serve("Vanguard", dir, func(token string) http.Handler { return newUI(token) })
}
func newUI(token string) http.Handler {
	var mu sync.Mutex
	var decision *gate.Decision
	tmpl := template.Must(template.New("page").Parse(strings.ReplaceAll(vanguardHTML, "{{.Style}}", desktop.Style)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		message := ""
		if r.URL.Path == "/export" {
			if r.Method != "GET" || decision == nil {
				http.Error(w, "Evaluate a report first", 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", "attachment; filename=vanguard-decision.json")
			json.NewEncoder(w).Encode(decision)
			return
		}
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		policy, report := samplePolicy, sampleReport
		if r.Method == "POST" {
			if e := r.ParseMultipartForm(1 << 20); e != nil {
				http.Error(w, "Invalid upload", 400)
				return
			}
			defer r.MultipartForm.RemoveAll()
			policy = r.FormValue("policy")
			report = r.FormValue("report")
			if f, _, e := r.FormFile("reportfile"); e == nil {
				b, e := io.ReadAll(io.LimitReader(f, (16<<20)+1))
				f.Close()
				if e != nil {
					http.Error(w, "Cannot read file", 400)
					return
				}
				report = string(b)
			}
			d, e := gate.Evaluate([]byte(report), []byte(policy))
			decision = nil
			if e != nil {
				message = e.Error()
			} else {
				decision = d
				message = "Decision complete. Demo input is synthetic; replace it with your normalized report."
			}
			if len(report) > 16<<20 {
				report = ""
			}
			if len(policy) > 1<<20 {
				policy = ""
			}
		}
		tmpl.Execute(w, struct {
			Token, Style, Message, Policy, Report string
			Decision                              *gate.Decision
		}{token, desktop.Style, message, policy, report, decision})
	})
}

const vanguardHTML = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Vanguard</title><style>{{.Style}}</style><h1>Vanguard</h1><p>A transparent security policy decision for your CI workflow.</p><p class="notice">Counts normalized findings against severity limits. Does not discover vulnerabilities or certify compliance. Omitted thresholds are unlimited. Exact duplicate finding objects are rejected; distinct occurrences count separately.</p>{{if .Message}}<p role="status">{{.Message}}</p>{{end}}<section><h2>Evaluate a report</h2><form method="post" enctype="multipart/form-data" action="/?token={{.Token}}"><label>Policy YAML<textarea name="policy" rows="10">{{.Policy}}</textarea></label><label>Normalized report JSON (synthetic example)<textarea name="report">{{.Report}}</textarea></label><label>Or upload a JSON report (replaces text)<input type="file" name="reportfile" accept=".json"></label><button>Evaluate policy</button></form></section>{{if .Decision}}<section><h2>{{if .Decision.Passed}}PASS{{else}}FAIL{{end}}</h2><table><tr><th>Severity</th><th>Observed</th><th>Allowed</th><th>Exceeded</th></tr>{{range .Decision.Rows}}<tr><td>{{.Severity}}</td><td>{{.Observed}}</td><td>{{if .Limited}}{{.Allowed}}{{else}}unlimited{{end}}</td><td>{{.Exceeded}}</td></tr>{{end}}</table><p>Policy: {{.Decision.PolicyName}}</p><small>Policy SHA256: {{.Decision.PolicyHash}}<br>Input SHA256: {{.Decision.InputHash}}</small><p><a href="/export?token={{.Token}}">Download decision JSON</a></p></section>{{end}}<section><h2>Use in CI</h2><pre>vanguard -p policy.yaml -json report.json</pre><p>Exit 0: pass. Exit 1: policy violation. Exit 2: tool/input/output failure. Optional <code>-notify</code> sends a failure-only Slack summary using VANGUARD_SLACK_WEBHOOK. Slack failure never changes the policy verdict. No source, finding descriptions, or credentials are sent. A run URL can be configured through VANGUARD_RUN_URL.</p></section><small>Session results are not saved. Export before closing. No internet is required for evaluation. Ctrl+C in the launcher stops the app.</small></html>`
