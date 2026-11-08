// Package onboarding provides a local, explicit first-task guide without changing domain results.
package onboarding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
)

type Sample struct{ Name, Content string }
type Config struct {
	Name, Token, Host, Dir, Intro, Expected, Next string
	Steps                                         []string
	Samples                                       []Sample
	DemoFields                                    map[string]string
}
type Wizard struct {
	mu                  sync.Mutex
	c                   Config
	started, done, demo bool
	ready               bool
	warning             string
}

func New(c Config) *Wizard {
	w := &Wizard{c: c}
	if c.Dir != "" {
		b, e := os.ReadFile(filepath.Join(c.Dir, "onboarding.json"))
		var state struct {
			Version   int
			Completed bool
		}
		if e == nil && json.Unmarshal(b, &state) == nil && state.Version == 1 && state.Completed {
			w.started = true
			w.done = true
		}
	}
	return w
}
func (w *Wizard) Complete() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.started = true
	w.done = true
	w.ready = true
	if w.c.Dir == "" {
		return
	}
	b := []byte(`{"Version":1,"Completed":true}`)
	f, e := os.CreateTemp(w.c.Dir, "onboarding-*.tmp")
	if e == nil {
		name := f.Name()
		defer os.Remove(name)
		e = f.Chmod(0600)
		if e == nil {
			_, e = f.Write(b)
		}
		if e == nil {
			e = f.Sync()
		}
		closeErr := f.Close()
		if e == nil {
			e = closeErr
		}
		if e == nil {
			e = os.Rename(name, filepath.Join(w.c.Dir, "onboarding.json"))
		}
	}
	if e != nil {
		w.warning = "Guide progress could not be saved. Your result is still available; export it before closing."
	}
}
func (w *Wizard) Panel() template.HTML {
	w.mu.Lock()
	data := struct {
		Name, Token, Expected, Next, Warning string
		Done, Demo, Ready                    bool
	}{w.c.Name, w.c.Token, w.c.Expected, w.c.Next, w.warning, w.done, w.demo, w.ready}
	w.mu.Unlock()
	var b bytes.Buffer
	panel.Execute(&b, data)
	return template.HTML(b.String())
}

var panel = template.Must(template.New("panel").Parse(`<section aria-label="Getting started"><h2>{{if .Ready}}Your first result is ready{{else if .Done}}Welcome back{{else}}Your first task{{end}}</h2>{{if .Demo}}<p><strong>Guided demo - synthetic data.</strong> {{.Expected}}</p>{{else}}<p>Choose your input, run the task, then review the result and export it.</p>{{end}}<ol><li>Choose demo or your own data.</li><li>Run the task and inspect its explanation.</li><li>Export your result, then choose the next action.</li></ol>{{if .Ready}}<p><strong>Next:</strong> {{.Next}}</p><p><a href="#results">Review my result</a></p>{{else if .Done}}<p>Your guide progress is remembered. Evaluate a new report, or review the decision from this session. Decisions are not saved across application restarts.</p>{{end}}{{if .Warning}}<p role="status">{{.Warning}}</p>{{end}}<a href="/welcome?token={{.Token}}">Restart the guide</a> | <a href="/setup?token={{.Token}}">Report and policy format</a></section>`))

func (w *Wizard) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(out http.ResponseWriter, r *http.Request) {
		out.Header().Set("Cache-Control", "no-store")
		out.Header().Set("Referrer-Policy", "same-origin")
		out.Header().Set("X-Content-Type-Options", "nosniff")
		out.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		if r.URL.Query().Get("token") != w.c.Token || (w.c.Host != "" && r.Host != w.c.Host) {
			http.Error(out, "Open the private launcher URL", 403)
			return
		}
		if r.Method != "GET" && r.Method != "POST" {
			http.Error(out, "Method not allowed", 405)
			return
		}
		if r.Method == "POST" {
			if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
				http.Error(out, "Invalid origin", 403)
				return
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(out, "Cross-site request blocked", 403)
				return
			}
		}
		path := r.URL.Path
		if path == "/onboarding/sample" {
			if r.Method != "GET" {
				http.Error(out, "GET required", 405)
				return
			}
			for _, s := range w.c.Samples {
				if s.Name == r.URL.Query().Get("name") {
					out.Header().Set("Content-Type", "text/plain; charset=utf-8")
					out.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", s.Name))
					io.WriteString(out, s.Content)
					return
				}
			}
			http.NotFound(out, r)
			return
		}
		if path == "/onboarding/start" {
			if r.Method != "POST" {
				http.Error(out, "POST required", 405)
				return
			}
			r.Body = http.MaxBytesReader(out, r.Body, 16<<10)
			if e := r.ParseForm(); e != nil {
				http.Error(out, "Invalid request", 400)
				return
			}
			if r.FormValue("mode") != "own" && r.FormValue("mode") != "resume" {
				http.Error(out, "Choose your own data or resume", 400)
				return
			}
			w.mu.Lock()
			w.started = true
			w.demo = false
			w.mu.Unlock()
			http.Redirect(out, r, "/?token="+url.QueryEscape(w.c.Token), http.StatusSeeOther)
			return
		}
		w.mu.Lock()
		started := w.started
		w.mu.Unlock()
		if path == "/welcome" || path == "/setup" || (path == "/" && r.Method == "GET" && !started) {
			if r.Method != "GET" {
				http.Error(out, "GET required", 405)
				return
			}
			w.render(out, path == "/setup")
			return
		}
		if path == "/" && r.Method == "POST" {
			w.mu.Lock()
			w.started = true
			w.ready = false
			w.demo = r.URL.Query().Get("onboarding") == "demo"
			w.mu.Unlock()
		}
		next.ServeHTTP(out, r)
	})
}
func (w *Wizard) render(out http.ResponseWriter, setup bool) {
	w.mu.Lock()
	data := struct {
		C     Config
		Setup bool
	}{w.c, setup}
	w.mu.Unlock()
	out.Header().Set("Content-Type", "text/html; charset=utf-8")
	welcome.Execute(out, data)
}

const style = `body{font:16px system-ui;background:#0b1627;color:#e4edf8;max-width:1000px;margin:40px auto;padding:0 24px}section{background:#18293e;padding:24px;border:1px solid #344a64;border-radius:12px;margin:20px 0}h1{font-size:38px}p,li{line-height:1.7}small{color:#b8cce0}input{display:block;width:92%;max-width:700px;background:#0b1627;color:white;border:1px solid #8ba3bd;padding:12px;border-radius:6px;margin:8px 0}label{display:block;margin:16px 0}button{background:#92e5c0;color:#102332;font-weight:700;padding:13px 18px;border:0;border-radius:6px;cursor:pointer;margin:8px 8px 8px 0}a{color:#a3d2ff}.notice{padding:18px;background:#29415b;border-radius:8px}.choices{display:grid;grid-template-columns:repeat(auto-fit,minmax(250px,1fr));gap:18px}pre{white-space:pre-wrap;overflow-wrap:anywhere}`

var welcome = template.Must(template.New("welcome").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Get started - {{.C.Name}}</title><style>` + style + `</style><main><small>GETTING STARTED</small><h1>Welcome to {{.C.Name}}</h1><p>{{.C.Intro}}</p><p class="notice">Start with a safe offline example, or use data you are authorized to review. Evaluation works offline and does not contact external services.</p>{{if not .Setup}}<div class="choices"><section><h2>Try the demo</h2><p>No accounts or credentials needed. {{.C.Expected}}</p><form method="post" enctype="multipart/form-data" action="/?token={{.C.Token}}&onboarding=demo">{{range $k,$v:=.C.DemoFields}}<input type="hidden" name="{{$k}}" value="{{$v}}">{{end}}<button>Try the guided demo</button></form></section><section><h2>Use my data</h2><p>Read the report format and policy requirements before evaluating your data.</p><a href="/setup?token={{.C.Token}}">Set up my workspace</a></section></div>{{else}}<section><h2>Set up your first task</h2><ol>{{range .C.Steps}}<li>{{.}}</li>{{end}}</ol>{{range .C.Samples}}<p><a href="/onboarding/sample?token={{$.C.Token}}&name={{.Name}}">Download example {{.Name}}</a></p>{{end}}<form method="post" action="/onboarding/start?token={{.C.Token}}"><input type="hidden" name="mode" value="own"><button>Open my workspace</button><small>No account or connection is required for local evaluation.</small></form></section>{{end}}<section><h2>Return to your workspace</h2><form method="post" action="/onboarding/start?token={{.C.Token}}"><input type="hidden" name="mode" value="resume"><button>Return to workspace</button></form><p>Restarting this guide preserves your current session decision. Decisions are not saved across application restarts; export before closing.</p></section><p>Close the application with Ctrl+C in its launcher console. Closing only the browser leaves it running.</p></main></html>`))
