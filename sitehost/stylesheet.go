package sitehost

import (
	"bytes"
	"net/http"
	"time"
)

// siteCSS styles the public site and the back-office chrome the default host
// renders. Studio's own studio.css covers the editor; this covers everything
// around it, so a fresh install looks like a site rather than unstyled markup.
const siteCSS = `
:root {
  color-scheme: light dark;
  --site-ground: #ffffff;
  --site-ink: #16201d;
  --site-muted: #566762;
  --site-rule: #dce3e0;
  --site-accent: #0e6b59;
  --site-surface: #f6f8f7;
  --site-measure: 68ch;
}
@media (prefers-color-scheme: dark) {
  :root {
    --site-ground: #0e1413;
    --site-ink: #e6edea;
    --site-muted: #97a8a1;
    --site-rule: #24302d;
    --site-accent: #4cbba0;
    --site-surface: #151e1c;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0;
  background: var(--site-ground);
  color: var(--site-ink);
  font: 16px/1.6 ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
  -webkit-font-smoothing: antialiased;
}
img { max-width: 100%; height: auto; }
a { color: var(--site-accent); }
.site-shell { display: flex; flex-direction: column; min-height: 100vh; }
.site-header {
  display: flex; flex-wrap: wrap; align-items: baseline; gap: 12px 28px;
  padding: 20px max(16px, 5vw);
  border-bottom: 1px solid var(--site-rule);
}
.site-brand { font-weight: 600; font-size: 18px; text-decoration: none; color: var(--site-ink); }
.site-nav { display: flex; flex-wrap: wrap; gap: 8px 20px; }
.site-nav a { text-decoration: none; }
.site-nav a[aria-current="page"] { text-decoration: underline; text-underline-offset: 4px; }
.site-main { flex: 1 1 auto; padding: clamp(28px, 6vw, 64px) max(16px, 5vw); }
.site-article, .site-main > * { max-width: var(--site-measure); }
.site-title { font-size: clamp(30px, 5vw, 44px); line-height: 1.1; margin: 0 0 20px; text-wrap: balance; }
.site-lede { color: var(--site-muted); }
.site-article h2 { font-size: clamp(21px, 3vw, 27px); line-height: 1.2; margin: 32px 0 12px; text-wrap: balance; }
.site-article h3 { font-size: 19px; margin: 26px 0 10px; }
.site-article p { margin: 0 0 16px; }
.site-article blockquote {
  margin: 22px 0; padding: 4px 0 4px 18px;
  border-left: 3px solid var(--site-accent); color: var(--site-muted);
}
.site-article figure { margin: 24px 0; }
.site-button, .admin-button {
  display: inline-block; padding: 10px 18px; border-radius: 2px;
  background: var(--site-accent); color: var(--site-ground);
  text-decoration: none; font-weight: 500; border: 0; cursor: pointer;
  font-size: 15px;
}
.site-footer {
  padding: 24px max(16px, 5vw);
  border-top: 1px solid var(--site-rule);
  color: var(--site-muted); font-size: 14px;
}

/* --- back office --- */
.admin-shell { display: flex; flex-direction: column; min-height: 100vh; }
.admin-header {
  display: flex; flex-wrap: wrap; align-items: center; gap: 10px 24px;
  padding: 16px max(16px, 4vw);
  border-bottom: 1px solid var(--site-rule); background: var(--site-surface);
}
.admin-header .admin-brand { font-weight: 600; text-decoration: none; color: var(--site-ink); }
.admin-nav { display: flex; flex-wrap: wrap; gap: 6px 18px; }
.admin-nav a { text-decoration: none; font-size: 15px; }
.admin-nav a[aria-current="page"] { font-weight: 600; text-decoration: underline; text-underline-offset: 4px; }
.admin-main { flex: 1 1 auto; padding: clamp(24px, 4vw, 40px) max(16px, 4vw); }
.admin-main h1 { font-size: clamp(25px, 4vw, 33px); line-height: 1.15; margin: 0 0 6px; text-wrap: balance; }
.admin-lede { color: var(--site-muted); margin: 0 0 28px; max-width: var(--site-measure); }
.admin-panel {
  border: 1px solid var(--site-rule); background: var(--site-surface);
  padding: 20px 22px; margin: 0 0 24px; max-width: 900px;
}
.admin-panel h2 { font-size: 19px; margin: 0 0 14px; }
.admin-empty {
  border: 1px dashed var(--site-rule); padding: 26px 22px; text-align: left;
  max-width: 900px; margin: 0 0 24px;
}
.admin-empty p { color: var(--site-muted); margin: 0 0 16px; max-width: 60ch; }
.admin-table { width: 100%; border-collapse: collapse; font-size: 15px; }
.admin-table th, .admin-table td {
  text-align: left; padding: 10px 12px; border-bottom: 1px solid var(--site-rule); vertical-align: top;
}
.admin-table th { font-size: 12px; letter-spacing: .06em; text-transform: uppercase; color: var(--site-muted); }
.admin-table tr:last-child td { border-bottom: 0; }
.admin-field { display: block; margin: 0 0 16px; max-width: 640px; }
.admin-field span { display: block; font-size: 14px; margin: 0 0 5px; color: var(--site-muted); }
.admin-field input, .admin-field textarea, .admin-field select {
  width: 100%; padding: 9px 11px; font: inherit; font-size: 15px;
  border: 1px solid var(--site-rule); background: var(--site-ground); color: var(--site-ink); border-radius: 2px;
}
.admin-field textarea { min-height: 220px; resize: vertical; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 14px; }
.admin-hint { font-size: 13px; color: var(--site-muted); margin: 5px 0 0; }
.admin-actions { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; margin-top: 18px; }
.admin-secondary {
  display: inline-block; padding: 10px 18px; border-radius: 2px; font-size: 15px;
  border: 1px solid var(--site-rule); background: var(--site-ground);
  color: var(--site-ink); text-decoration: none; cursor: pointer;
}
.admin-status { padding: 12px 16px; margin: 0 0 22px; max-width: 900px; border-left: 3px solid var(--site-accent); background: var(--site-surface); }
.admin-status[data-state="error"] { border-left-color: #9c312a; }
.admin-badge { font-size: 12px; letter-spacing: .05em; text-transform: uppercase; color: var(--site-muted); }
.admin-badge[data-state="published"] { color: var(--site-accent); }
.admin-stats { display: flex; flex-wrap: wrap; gap: 1px; background: var(--site-rule); border: 1px solid var(--site-rule); margin: 0 0 28px; max-width: 900px; }
.admin-stat { background: var(--site-ground); padding: 16px 20px; flex: 1 1 160px; }
.admin-stat strong { display: block; font-size: 30px; line-height: 1.1; font-variant-numeric: tabular-nums; }
.admin-stat span { display: block; font-size: 13px; color: var(--site-muted); margin-top: 4px; }
@media (prefers-reduced-motion: reduce) { * { animation: none !important; transition: none !important; } }
`

var siteCSSModTime = time.Now()

func publicStylesheetHandler() http.Handler {
	body := []byte(siteCSS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		http.ServeContent(w, r, "site.css", siteCSSModTime, bytes.NewReader(body))
	})
}
