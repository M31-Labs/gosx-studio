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
.site-article .button { margin: 4px 0 20px; }
.site-article > p:first-of-type { font-size: 18px; color: var(--site-muted); line-height: 1.55; }
.site-button, .admin-button, .site-article .button {
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

/* ---------- setup wizard ---------- */
.wz { min-height: 100vh; display: grid; place-items: center; padding: clamp(20px, 5vw, 56px) max(16px, 4vw); background: var(--site-surface); }
.wz-card { width: 100%; max-width: 620px; background: var(--site-ground); border: 1px solid var(--site-rule); padding: clamp(26px, 5vw, 44px); }
.wz-card--quiet { text-align: center; max-width: 460px; }
.wz-progress { display: flex; flex-wrap: wrap; gap: 6px 18px; list-style: none; margin: 0 0 30px; padding: 0; font-size: 13px; color: var(--site-muted); counter-reset: wz; }
.wz-step { counter-increment: wz; display: flex; align-items: center; gap: 7px; }
.wz-step::before {
  content: counter(wz); display: grid; place-items: center;
  width: 21px; height: 21px; border-radius: 50%; font-size: 11.5px;
  border: 1px solid var(--site-rule); color: var(--site-muted);
}
.wz-step[data-state="current"] { color: var(--site-ink); font-weight: 500; }
.wz-step[data-state="current"]::before { background: var(--site-accent); border-color: var(--site-accent); color: var(--site-ground); }
.wz-step[data-state="done"]::before { content: "✓"; border-color: var(--site-accent); color: var(--site-accent); }
.wz h1 { font-size: clamp(26px, 4.5vw, 34px); line-height: 1.12; margin: 0 0 10px; text-wrap: balance; }
.wz-lede { color: var(--site-muted); margin: 0 0 26px; max-width: 54ch; }
.wz-hint { font-size: 13.5px; color: var(--site-muted); margin: -6px 0 22px; }
.wz-problem { border-left: 3px solid #9c312a; background: rgba(156,49,42,.08); padding: 11px 15px; margin: 0 0 22px; font-size: 14.5px; }
.wz-field { display: block; margin: 0 0 18px; }
.wz-field span { display: block; font-size: 14.5px; margin: 0 0 6px; }
.wz-field input {
  width: 100%; padding: 11px 13px; font: inherit; font-size: 16px;
  border: 1px solid var(--site-rule); border-radius: 2px;
  background: var(--site-ground); color: var(--site-ink);
}
.wz-field input:focus-visible { outline: 2px solid var(--site-accent); outline-offset: 1px; }
.wz-choices { display: grid; grid-template-columns: repeat(auto-fit, minmax(230px, 1fr)); gap: 10px; margin: 0 0 28px; }
.wz-choice {
  position: relative; display: flex; flex-direction: column; gap: 3px;
  border: 1px solid var(--site-rule); padding: 15px 16px; cursor: pointer; border-radius: 2px;
}
.wz-choice:hover { border-color: var(--site-accent); }
.wz-choice input { position: absolute; opacity: 0; pointer-events: none; }
.wz-choice:has(input:checked) { border-color: var(--site-accent); box-shadow: inset 0 0 0 1px var(--site-accent); }
.wz-choice:has(input:focus-visible) { outline: 2px solid var(--site-accent); outline-offset: 2px; }
.wz-choice__label { font-weight: 600; font-size: 15.5px; }
.wz-choice__blurb { font-size: 14px; color: var(--site-muted); line-height: 1.45; }
.wz-choice__eg { font-size: 12.5px; color: var(--site-muted); opacity: .8; margin-top: 3px; }
.wz-actions { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
.wz-btn {
  padding: 11px 22px; font: inherit; font-size: 15px; font-weight: 500;
  border: 1px solid var(--site-accent); background: var(--site-accent); color: var(--site-ground);
  border-radius: 2px; cursor: pointer; text-decoration: none; display: inline-block;
}
.wz-btn--ghost { background: transparent; color: var(--site-ink); border-color: var(--site-rule); }

/* ---------- editor ---------- */
.gosx-site--admin:has(.ed) { overflow: hidden; }
.ed { display: flex; flex-direction: column; height: 100vh; }
.ed-bar {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 9px 16px; border-bottom: 1px solid var(--site-rule);
  background: var(--site-ground); flex: 0 0 auto; flex-wrap: wrap;
}
.ed-bar__left, .ed-bar__right { display: flex; align-items: center; gap: 10px; min-width: 0; }
.ed-back { text-decoration: none; font-size: 14px; color: var(--site-muted); white-space: nowrap; }
.ed-page-name { font-weight: 600; font-size: 15px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ed-chip {
  font-size: 11px; letter-spacing: .05em; text-transform: uppercase;
  border: 1px solid var(--site-rule); padding: 2px 8px; border-radius: 999px; color: var(--site-muted); white-space: nowrap;
}
.ed-chip[data-live="true"] { color: var(--site-accent); border-color: var(--site-accent); }
.ed-save { font-size: 13px; color: var(--site-muted); white-space: nowrap; }
.ed-save[data-save-status="dirty"] { color: var(--site-ink); }
.ed-save[data-save-status="error"] { color: #9c312a; }
.ed-btn {
  padding: 8px 15px; font: inherit; font-size: 14px; border-radius: 2px; cursor: pointer;
  border: 1px solid var(--site-rule); background: var(--site-ground); color: var(--site-ink);
  text-decoration: none; display: inline-block; white-space: nowrap;
}
.ed-btn--primary { background: var(--site-accent); border-color: var(--site-accent); color: var(--site-ground); font-weight: 500; }
.ed-btn[disabled] { opacity: .5; cursor: default; }
.ed-body { flex: 1 1 auto; display: flex; min-height: 0; }
.ed-side {
  flex: 0 0 272px; border-right: 1px solid var(--site-rule); background: var(--site-surface);
  overflow-y: auto; padding: 18px 16px 40px;
}
.ed-side__block { margin: 0 0 26px; }
.ed-side h2 { font-size: 13px; letter-spacing: .05em; text-transform: uppercase; color: var(--site-muted); margin: 0 0 10px; }
.ed-hint { font-size: 13px; color: var(--site-muted); margin: 0 0 12px; line-height: 1.45; }
.ed-add-grid { display: grid; gap: 6px; }
.ed-add {
  display: flex; flex-direction: column; gap: 1px; text-align: left; cursor: pointer;
  border: 1px solid var(--site-rule); background: var(--site-ground); color: var(--site-ink);
  padding: 9px 12px; font: inherit; border-radius: 2px;
}
.ed-add:hover { border-color: var(--site-accent); }
.ed-add__label { font-size: 14px; font-weight: 500; }
.ed-add__hint { font-size: 12px; color: var(--site-muted); }
.ed-field { display: block; margin: 0 0 14px; }
.ed-field span { display: block; font-size: 12.5px; color: var(--site-muted); margin: 0 0 4px; }
.ed-field input {
  width: 100%; padding: 7px 9px; font: inherit; font-size: 14px;
  border: 1px solid var(--site-rule); background: var(--site-ground); color: var(--site-ink); border-radius: 2px;
}
.ed-field small { display: block; font-size: 11.5px; color: var(--site-muted); margin-top: 3px; }
.ed-stage { flex: 1 1 auto; overflow-y: auto; padding: 26px max(16px, 3vw) 90px; background: var(--site-surface); }
.ed-frame {
  max-width: 940px; margin: 0 auto; background: var(--site-ground);
  border: 1px solid var(--site-rule); box-shadow: 0 1px 3px rgba(0,0,0,.05);
}
.ed-canvas .site-header { pointer-events: none; opacity: .75; }
.ed-canvas .site-main { padding-bottom: 40px; }
.ed-block { position: relative; padding: 2px 0; margin: 0 0 2px; border-radius: 2px; }
.ed-block:hover { box-shadow: inset 0 0 0 1px var(--site-rule); }
.ed-block:focus-within { box-shadow: inset 0 0 0 1px var(--site-accent); }
.ed-block [data-text] { outline: none; }
.ed-block [data-text]:focus-visible { outline: none; }
.ed-block__tools {
  position: absolute; top: -13px; right: 4px; display: none; gap: 2px; z-index: 3;
  background: var(--site-ground); border: 1px solid var(--site-rule); border-radius: 2px; padding: 2px;
}
.ed-block:hover .ed-block__tools, .ed-block:focus-within .ed-block__tools { display: flex; }
.ed-tool {
  width: 25px; height: 25px; display: grid; place-items: center; cursor: pointer;
  border: 0; background: transparent; color: var(--site-muted); font-size: 13px; border-radius: 2px;
}
.ed-tool:hover { background: var(--site-surface); color: var(--site-ink); }
.ed-levels {
  position: absolute; top: -13px; left: 4px; display: none; gap: 2px; z-index: 3;
  background: var(--site-ground); border: 1px solid var(--site-rule); border-radius: 2px; padding: 2px;
}
.ed-block:hover .ed-levels, .ed-block:focus-within .ed-levels { display: flex; }
.ed-level {
  padding: 2px 7px; cursor: pointer; border: 0; background: transparent;
  color: var(--site-muted); font: inherit; font-size: 11.5px; border-radius: 2px;
}
.ed-level[aria-pressed="true"] { background: var(--site-accent); color: var(--site-ground); }
.ed-insert {
  position: absolute; bottom: -11px; left: 50%; transform: translateX(-50%);
  width: 21px; height: 21px; display: none; place-items: center; z-index: 2;
  border: 1px solid var(--site-rule); background: var(--site-ground);
  color: var(--site-muted); border-radius: 50%; cursor: pointer; font-size: 13px; line-height: 1;
}
.ed-block:hover .ed-insert, .ed-block:focus-within .ed-insert { display: grid; }
.ed-insert:hover { border-color: var(--site-accent); color: var(--site-accent); }
.ed-button-row { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
.ed-inline-input {
  padding: 5px 8px; font: inherit; font-size: 12.5px; border-radius: 2px;
  border: 1px dashed var(--site-rule); background: transparent; color: var(--site-muted); min-width: 180px;
}
.ed-inline-input:focus-visible { outline: 2px solid var(--site-accent); outline-offset: 1px; color: var(--site-ink); }
.ed-figure { display: flex; flex-direction: column; gap: 7px; margin: 20px 0; }
.ed-image-empty {
  display: grid; place-items: center; min-height: 130px; border: 1px dashed var(--site-rule);
  color: var(--site-muted); font-size: 13.5px; text-align: center; padding: 16px;
}
.ed-menu {
  position: absolute; z-index: 20; display: flex; flex-direction: column; min-width: 168px;
  background: var(--site-ground); border: 1px solid var(--site-rule); border-radius: 2px;
  box-shadow: 0 4px 14px rgba(0,0,0,.12); padding: 4px;
}
.ed-menu__item {
  text-align: left; padding: 8px 11px; cursor: pointer; border: 0;
  background: transparent; color: var(--site-ink); font: inherit; font-size: 14px; border-radius: 2px;
}
.ed-menu__item:hover { background: var(--site-surface); }
/* display:flex above would otherwise beat the UA rule for [hidden]. */
.ed-menu[hidden] { display: none; }
@media (max-width: 820px) {
  .ed-body { flex-direction: column; }
  .ed-side { flex: 0 0 auto; border-right: 0; border-bottom: 1px solid var(--site-rule); max-height: 42vh; }
}

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
