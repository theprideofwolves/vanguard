package main

import (
	"encoding/json"
	"github.com/theprideofwolves/vanguard/internal/desktop"
	"github.com/theprideofwolves/vanguard/internal/gate"
	"github.com/theprideofwolves/vanguard/internal/onboarding"
	"html/template"
	"io"
	"net/http"
	"sync"
)

const samplePolicy = "apiVersion: vanguard/v1\nkind: SecurityPolicy\nmetadata:\n  name: example-baseline\nthresholds:\n  critical: 0\n  high: 2\n  medium: 10\n"
const sampleReport = `{"tool":"synthetic-demo","target":"fictional-app","findings":[{"id":"DEMO-1","severity":"high"},{"id":"DEMO-2","severity":"critical"}]}`
const sampleCandidatePolicy = "apiVersion: vanguard/v1\nkind: SecurityPolicy\nmetadata:\n  name: proposed-baseline\nthresholds:\n  critical: 1\n  high: 2\n  medium: 10\n"

func launchUI() error {
	dir, e := desktop.DataDir("Vanguard")
	if e != nil {
		return e
	}
	return desktop.Serve("Vanguard", dir, func(token string) http.Handler { return newUIAt(token, dir) })
}
func newUI(token string) http.Handler { return newUIAt(token, "") }
func newUIAt(token, dir string) http.Handler {
	guide := onboarding.New(onboarding.Config{
		Name: "Vanguard", Token: token, Dir: dir,
		Intro:    "Evaluate normalized findings or review the impact of a policy change.",
		Expected: "The example intentionally FAILS: one critical finding exceeds the allowed zero. A failed policy is a successful evaluation, not a broken app.",
		Next:     "Download the decision, or compare the proposed policy to see why the same report would pass.",
		Steps: []string{
			"Evaluation and policy comparison work offline; no accounts are required.",
			"Download a normalized report, current policy and proposed policy. Native scanner reports need normalization first.",
			"Upload or paste a report. Evaluate the current policy, or compare the proposed policy using the same report.",
			"Optional Slack is an evaluation-only CLI feature. Policy comparison and this guide never send notifications.",
		},
		DemoFields: map[string]string{"policy": samplePolicy, "report": sampleReport, "candidate": sampleCandidatePolicy},
		Samples:    []onboarding.Sample{{Name: "report.json", Content: sampleReport}, {Name: "policy.yaml", Content: samplePolicy}, {Name: "proposed-policy.yaml", Content: sampleCandidatePolicy}},
	})
	var mu sync.Mutex
	var decision *gate.Decision
	var impact *gate.PolicyImpact
	policy, report := samplePolicy, sampleReport
	candidate := sampleCandidatePolicy
	tmpl := template.Must(template.New("page").Funcs(template.FuncMap{"onboarding": guide.Panel, "limit": limitText}).Parse(vanguardHTML))
	return guide.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		message := ""
		if r.URL.Path == "/export" {
			if r.Method != "GET" || (decision == nil && impact == nil) {
				http.Error(w, "Evaluate a report first", 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			var evidence any = decision
			filename := "vanguard-decision.json"
			if impact != nil {
				evidence = impact
				filename = "vanguard-policy-impact.json"
			}
			w.Header().Set("Content-Disposition", "attachment; filename="+filename)
			json.NewEncoder(w).Encode(evidence)
			return
		}
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method == "POST" {
			// An attempted evaluation invalidates old evidence, even if upload
			// parsing or reading fails before the gate can evaluate it.
			decision = nil
			impact = nil
			if e := r.ParseMultipartForm(1 << 20); e != nil {
				http.Error(w, "Invalid upload", 400)
				return
			}
			defer r.MultipartForm.RemoveAll()
			policy = r.FormValue("policy")
			report = r.FormValue("report")
			candidate = r.FormValue("candidate")
			if f, _, e := r.FormFile("reportfile"); e == nil {
				b, e := io.ReadAll(io.LimitReader(f, (16<<20)+1))
				f.Close()
				if e != nil {
					http.Error(w, "Cannot read file", 400)
					return
				}
				report = string(b)
			}
			var e error
			switch r.FormValue("action") {
			case "", "evaluate":
				decision, e = gate.Evaluate([]byte(report), []byte(policy))
			case "compare":
				impact, e = gate.ComparePolicies([]byte(policy), []byte(candidate), []gate.ReportInput{{Name: "workspace-report.json", Data: []byte(report)}})
			default:
				message = "Choose evaluation or policy comparison."
			}
			if e != nil {
				message = e.Error()
			} else if decision != nil || impact != nil {
				guide.Complete()
				message = "Evaluation complete. Review the counts and download your decision. A FAIL is a valid policy result."
				if impact != nil {
					message = "Comparison complete. Review changed limits and verdicts. A relaxation requires review even if this report does not change verdict."
				}
			}
			if len(report) > 16<<20 {
				report = ""
			}
			if len(policy) > 1<<20 {
				policy = ""
			}
			if len(candidate) > 1<<20 {
				candidate = ""
			}
		}
		tmpl.Execute(w, struct {
			Token, Message, Policy, Candidate, Report, Theme string
			Control                                          template.HTML
			Decision                                         *gate.Decision
			Impact                                           *gate.PolicyImpact
		}{token, message, policy, candidate, report, desktop.Theme(r), desktop.ThemeControl(token, desktop.Theme(r), "/"), decision, impact})
	}))
}

const vanguardHTML = `<!doctype html>
<html lang="en" data-theme="{{.Theme}}">
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Vanguard</title>
<style>` + desktop.Style + `</style>
<header>
<h1>Vanguard</h1>{{.Control}}</header>
<main>
<p class="muted">Normalized findings. Explicit thresholds. Explained decisions.</p>{{onboarding}}<p class="notice">Counts normalized findings against severity limits. Does not discover vulnerabilities or certify compliance. Omitted thresholds are unlimited. Exact duplicate finding objects are rejected; distinct normalized finding objects count separately.</p>{{if .Message}}<p role="status">{{.Message}}</p>{{end}}<section>
<h2>Evaluate a report</h2>
<form method="post" enctype="multipart/form-data" action="/?token={{.Token}}">
<label>Current policy YAML<textarea name="policy" rows="10">{{.Policy}}</textarea>
</label>
<label>Normalized report JSON<textarea name="report">{{.Report}}</textarea>
</label>
<label>Or upload a JSON report (replaces text)<input type="file" name="reportfile" accept=".json">
</label>
<button name="action" value="evaluate">Evaluate policy</button>
<details>
<summary>Compare a proposed policy</summary>
<p>Keep the report fixed and see how a policy change affects its verdict. The example raises the critical limit from 0 to 1. Use the CLI to compare a directory of reports.</p>
<label>Proposed policy YAML<textarea name="candidate" rows="10">{{.Candidate}}</textarea></label>
<button name="action" value="compare">Compare policies</button>
</details>
</form>
</section>{{if .Decision}}<section id="results">
<h2>{{if .Decision.Passed}}PASS{{else}}FAIL{{end}}</h2>
<div class="scroll">
<table>
<tr>
<th>Severity</th>
<th>Observed</th>
<th>Allowed</th>
<th>Exceeded</th>
</tr>{{range .Decision.Rows}}<tr>
<td>{{.Severity}}</td>
<td>{{.Observed}}</td>
<td>{{if .Limited}}{{.Allowed}}{{else}}unlimited{{end}}</td>
<td>{{.Exceeded}}</td>
</tr>{{end}}</table>
</div>
<p>Policy: {{.Decision.PolicyName}}</p>
<small>Policy SHA256: {{.Decision.PolicyHash}}<br>Input SHA256: {{.Decision.InputHash}}</small>
<p>
<a href="/export?token={{.Token}}">Download decision JSON</a>
</p>
</section>{{end}}
{{if .Impact}}<section id="results">
<h2>Policy impact: {{if .Impact.ReviewRequired}}REVIEW REQUIRED{{else}}NO RELAXED LIMITS{{end}}</h2>
<p class="notice">A relaxed limit may admit more findings. Results describe only the supplied report; they do not approve the policy or predict unseen reports.</p>
<div class="scroll"><table>
<tr><th>Severity</th><th>Current limit</th><th>Proposed limit</th><th>Change</th></tr>
{{range .Impact.Limits}}<tr><td>{{.Severity}}</td><td>{{limit .Base}}</td><td>{{limit .Candidate}}</td><td>{{.Direction}}</td></tr>{{end}}
</table></div>
<p>Newly passing: {{.Impact.Summary.NewlyPassing}} · Newly failing: {{.Impact.Summary.NewlyFailing}} · Unchanged PASS: {{.Impact.Summary.UnchangedPassed}} · Unchanged FAIL: {{.Impact.Summary.UnchangedFailed}}</p>
{{range .Impact.Reports}}
<h3>{{.Name}}</h3><p>Current: {{if .Base.Passed}}PASS{{else}}FAIL{{end}} → Proposed: {{if .Candidate.Passed}}PASS{{else}}FAIL{{end}}</p>
{{if eq .Transition "fail_to_pass"}}<p>This report passes only because the proposed policy relaxes its previously exceeded limits.</p>{{end}}
{{$candidateRows := .Candidate.Rows}}<div class="scroll"><table><tr><th>Severity</th><th>Observed</th><th>Current exceeded</th><th>Proposed exceeded</th></tr>
{{range $i, $row := .Base.Rows}}<tr><td>{{$row.Severity}}</td><td>{{$row.Observed}}</td><td>{{$row.Exceeded}}</td><td>{{(index $candidateRows $i).Exceeded}}</td></tr>{{end}}
</table></div>
<small>Input SHA256: {{.Base.InputHash}}</small>
{{end}}
<p>Current policy: {{.Impact.BasePolicyName}} · Proposed policy: {{.Impact.CandidatePolicyName}}</p>
<small>Current policy SHA256: {{.Impact.BasePolicyHash}}<br>Proposed policy SHA256: {{.Impact.CandidatePolicyHash}}</small>
<p><a href="/export?token={{.Token}}">Download policy impact JSON</a></p>
</section>{{end}}<details>
<summary>CLI and CI usage</summary>
<pre>vanguard -p policy.yaml -json report.json</pre>
<pre>vanguard compare -base current.yaml -candidate proposed.yaml -json reports/</pre>
<p>Comparison exits: 0 no relaxed limits; 1 relaxed limits require review; 2 tool/input/output failure. Comparison never sends Slack.</p>
<p>Exit 0: pass. Exit 1: policy violation. Exit 2: tool/input/output failure. Optional <code>-notify</code> sends a failure-only Slack summary using VANGUARD_SLACK_WEBHOOK. Slack failure never changes the policy verdict. No source, finding descriptions, or credentials are sent. A run URL can be configured through VANGUARD_RUN_URL.</p>
</details>
</main>
<footer>Session results are not saved. Export before closing. No internet is required for evaluation. Ctrl+C in the launcher stops the app.</footer>
</html>`
