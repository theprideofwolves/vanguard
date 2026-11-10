package desktop

import (
	"bytes"
	"html/template"
	"net/http"
)

// Theme is an allowlisted browser preference, independent of evaluation state.
func Theme(r *http.Request) string {
	if cookie, err := r.Cookie("vanguard-theme"); err == nil && cookie.Value == "dark" {
		return "dark"
	}
	return "light"
}

// SetTheme handles a protected form POST. Call only after token/origin checks.
func SetTheme(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid appearance preference", http.StatusBadRequest)
		return
	}
	theme := r.PostForm.Get("theme")
	if theme != "light" && theme != "dark" {
		http.Error(w, "Choose light or dark", http.StatusBadRequest)
		return
	}
	path := r.PostForm.Get("return")
	if path != "/welcome" && path != "/setup" {
		path = "/"
	}
	http.SetCookie(w, &http.Cookie{Name: "vanguard-theme", Value: theme, Path: "/", MaxAge: 31536000, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, path+"?token="+template.URLQueryEscaper(token), http.StatusSeeOther)
}

var themeControl = template.Must(template.New("theme").Parse(`<form class="theme-control" method="post" action="/theme?token={{.Token}}"><input type="hidden" name="return" value="{{.Path}}"><button class="secondary" name="theme" value="{{if eq .Theme "dark"}}light{{else}}dark{{end}}">{{if eq .Theme "dark"}}Light mode{{else}}Dark mode{{end}}</button></form>`))

func ThemeControl(token, theme, path string) template.HTML {
	var b bytes.Buffer
	themeControl.Execute(&b, struct{ Token, Theme, Path string }{token, theme, path})
	return template.HTML(b.String())
}

// Shared by the guide and workspace. No scripts, external assets or web fonts.
const Style = `
:root{color-scheme:light;--bg:#fff;--fg:#171717;--muted:#595959;--line:#d4d4d4;--accent:#087f8c;--on-accent:#fff}
html[data-theme="dark"]{color-scheme:dark;--bg:#000;--fg:#ededed;--muted:#aaa;--line:#383838;--accent:#38bdc9;--on-accent:#000}
*{box-sizing:border-box}body{font:16px/1.6 system-ui,sans-serif;background:var(--bg);color:var(--fg);max-width:1040px;margin:0 auto;padding:28px 24px 48px}
header{display:flex;align-items:center;justify-content:space-between;gap:16px;border-bottom:2px solid var(--accent);padding-bottom:18px;margin-bottom:28px}
h1{font-size:28px;line-height:1.2;margin:0}h2{font-size:20px;line-height:1.3;margin:0 0 16px}p{margin:12px 0}small,.muted{color:var(--muted)}small{font-size:13px}section{margin:28px 0;padding:0 0 24px;border-bottom:1px solid var(--line)}
a{color:var(--accent);text-underline-offset:3px}button{font:inherit;font-weight:600;background:var(--accent);color:var(--on-accent);border:1px solid var(--accent);border-radius:3px;padding:8px 16px;cursor:pointer}.secondary{background:var(--bg);color:var(--accent)}
button:hover{filter:brightness(.9)}a:focus-visible,button:focus-visible,input:focus-visible,textarea:focus-visible{outline:2px solid var(--accent);outline-offset:3px}
label{display:block;font-weight:600;margin:18px 0}input:not([type="hidden"]),textarea{display:block;width:100%;background:var(--bg);color:var(--fg);border:1px solid var(--line);border-radius:3px;padding:10px;margin-top:8px;font:14px/1.5 ui-monospace,monospace}textarea{min-height:150px;resize:vertical}
input[type="file"]{font:inherit}pre,code{font-family:ui-monospace,monospace;font-size:14px}pre{white-space:pre-wrap;overflow-wrap:anywhere;border-left:3px solid var(--accent);padding:12px 16px}small,td{overflow-wrap:anywhere}
table{border-collapse:collapse;width:100%;font-variant-numeric:tabular-nums}th,td{padding:10px 8px;text-align:left;border-bottom:1px solid var(--line)}th{font-size:13px;color:var(--muted)}.scroll{overflow-x:auto}.notice,[role="status"]{border-left:3px solid var(--accent);padding:8px 16px}.choices{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,280px),1fr));gap:28px}.theme-control{margin:0;flex-shrink:0}details{margin:24px 0}summary{cursor:pointer;color:var(--accent)}footer{color:var(--muted);font-size:13px;margin-top:24px}
@media(max-width:540px){body{padding:20px 16px}h1{font-size:24px}th,td{padding:8px 5px;font-size:13px}.theme-control button{padding:6px 10px}}
`
