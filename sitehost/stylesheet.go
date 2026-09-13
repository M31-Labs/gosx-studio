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
  --site-font-display: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
  --site-font-body: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
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
  font: 16px/1.6 var(--site-font-body);
  -webkit-font-smoothing: antialiased;
}
.site-brand, .site-title, .site-article h2, .site-article h3, .site-article h4 { font-family: var(--site-font-display); }
img { max-width: 100%; height: auto; }
a { color: var(--site-accent); }
.site-shell { display: flex; flex-direction: column; min-height: 100vh; }
.site-skip { position: absolute; left: 12px; top: -60px; z-index: 100; padding: 8px 14px; border-radius: 6px; background: var(--site-accent); color: var(--site-ground); text-decoration: none; font-weight: 600; transition: top .15s; }
.site-skip:focus, .site-skip:focus-visible { top: 12px; outline: 2px solid var(--site-ink); }
.gosx-site--public :focus-visible { outline: 2px solid var(--site-accent); outline-offset: 3px; border-radius: 2px; }
.gosx-site--public [id] { scroll-margin-top: 96px; }
html { scroll-behavior: smooth; }
.gosx-site--public p { text-wrap: pretty; }
@media (prefers-reduced-motion: reduce) { html { scroll-behavior: auto; } }
.site-space.site-space--tight > * { margin-block: 4px !important; }
.site-space.site-space--roomy > * { margin-block: 48px !important; }
.site-space.site-space--extra > * { margin-block: 96px !important; }
.ed-block.site-space--tight { padding-block: 0; }
.ed-block.site-space--roomy { padding-block: 22px; }
.ed-block.site-space--extra { padding-block: 46px; }
@media print { .site-header, .site-social, .site-consent, .site-announce, .site-skip, .site-lightbox, .site-nav__cta { display: none !important; } .gosx-site--public { color: #000; background: #fff; } .site-section--dark, .site-section--accent, .site-section--image { background: none !important; color: #000 !important; } }
.site-header {
  display: flex; flex-wrap: wrap; align-items: baseline; gap: 12px 28px;
  padding: 20px max(16px, 5vw);
  border-bottom: 1px solid var(--site-rule);
}
.site-brand { font-weight: 600; font-size: 18px; text-decoration: none; color: var(--site-ink); display: inline-flex; align-items: center; }
.site-logo { display: block; max-height: 44px; width: auto; max-width: 220px; }
.site-header--centered { flex-direction: column; align-items: center; text-align: center; gap: 12px; }
.site-header--centered .site-nav { justify-content: center; }
.site-nav { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 20px; }
.site-nav a { text-decoration: none; }
.site-nav a[aria-current="page"] { text-decoration: underline; text-underline-offset: 4px; }
.site-header--sticky { position: sticky; top: 0; z-index: 30; background: var(--site-ground); }
.site-announce { text-align: center; padding: 8px max(16px, 5vw); background: var(--site-accent); color: var(--site-ground); font-size: 14px; }
.site-announce__link { color: inherit; text-decoration: underline; text-underline-offset: 3px; }
.site-nav__group { position: relative; }
.site-nav__parent::after { content: " ▾"; font-size: .8em; opacity: .7; }
.site-nav__menu { position: absolute; left: -12px; top: calc(100% + 6px); min-width: 180px; display: none; flex-direction: column; padding: 8px 0; background: var(--site-ground); border: 1px solid var(--site-rule); border-radius: calc(var(--site-radius, 2px) * 2); box-shadow: 0 10px 30px rgba(0,0,0,.12); z-index: 40; }
.site-nav__menu a { padding: 7px 14px; color: var(--site-ink); }
.site-nav__menu a:hover, .site-nav__menu a:focus-visible { background: var(--site-surface); }
.site-nav__group:hover .site-nav__menu, .site-nav__group:focus-within .site-nav__menu { display: flex; }
.site-nav__cta.button { margin: 0; padding: 8px 14px; font-size: 14px; }
.site-header--centered .site-nav__menu { left: 50%; transform: translateX(-50%); }
.site-footer__grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 24px 32px; margin-bottom: 20px; }
.site-footer__col--about { grid-column: span 1; }
.site-footer__head { margin: 0 0 8px; font-size: 12px; letter-spacing: .08em; text-transform: uppercase; color: var(--site-muted); font-family: var(--site-font-body); }
.site-footer__links { list-style: none; padding: 0; margin: 0; display: grid; gap: 6px; }
.site-footer__links a { color: var(--site-ink); text-decoration: none; }
.site-footer__links a:hover { text-decoration: underline; }
.site-footer__sub { padding-left: 12px; font-size: 14px; }
.site-social__link { display: inline-flex; align-items: center; gap: 8px; }
.site-social__name { font-size: 14px; }
@container site (max-width: 640px) {
  .site-nav__menu { position: static; display: flex; box-shadow: none; border: 0; padding: 0 0 0 12px; background: transparent; }
  .site-nav__group { display: contents; }
  .site-nav__parent::after { content: ""; }
}
.site-main { flex: 1 1 auto; padding: calc(clamp(28px, 6vw, 64px) * var(--site-space, 1)) max(16px, 5vw); }
.site-article, .site-main > * { max-width: var(--site-measure); }
.site-title { font-size: calc(clamp(30px, 5vw, 44px) * var(--site-heading-scale, 1)); line-height: 1.1; margin: 0 0 20px; text-wrap: balance; }
.site-title--quiet { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); white-space: nowrap; margin: 0; }
.ed-title--quiet { font-size: 13px !important; font-family: var(--site-font-body); font-weight: 500; color: var(--site-muted); margin: 0 0 14px !important; padding: 4px 8px; border: 1px dashed var(--site-rule); border-radius: 4px; display: inline-block; }
.ed-title--quiet::before { content: "Page name (menu and tab): "; opacity: .7; }
.site-lede { color: var(--site-muted); }
.site-article h2 { font-size: calc(clamp(21px, 3vw, 27px) * var(--site-heading-scale, 1)); line-height: 1.2; margin: calc(32px * var(--site-space, 1)) 0 12px; text-wrap: balance; }
.site-figure { margin: 24px 0; }
.site-figure img { display: block; width: 100%; height: auto; border-radius: calc(var(--site-radius, 2px) * 2); }
.site-figure--wide { width: min(calc(100cqw - 2 * max(16px, 5vw)), 1120px); }
.site-figure--medium { max-width: 62%; }
.site-figure--small { max-width: 40%; }
.site-figure--crop-wide img { aspect-ratio: 16 / 9; object-fit: cover; }
.site-figure--crop-square img { aspect-ratio: 1; object-fit: cover; }
.site-figure--crop-round img { aspect-ratio: 1; object-fit: cover; border-radius: 50%; }
.site-figure figcaption { margin-top: 8px; font-size: 14px; color: var(--site-muted); }
.site-figure a { display: block; }
.site-article h3 { font-size: 19px; margin: 26px 0 10px; }
.site-article p { margin: 0 0 calc(16px * var(--site-space, 1)); }
.site-article blockquote {
  margin: 22px 0; padding: 4px 0 4px 18px;
  border-left: 3px solid var(--site-accent); color: var(--site-muted);
}
.site-article figure { margin: 24px 0; }
.site-section { padding: 0; }
.site-section__inner { max-width: var(--site-measure); }
.site-section--tinted, .site-section--accent, .site-section--dark, .site-section--image { width: 100cqw; margin-left: calc(-1 * max(16px, 5vw)); max-width: none; box-sizing: border-box; }
.site-section--tinted .site-section__inner, .site-section--accent .site-section__inner, .site-section--dark .site-section__inner, .site-section--image .site-section__inner { margin-left: 0; }
.site-section--tinted { background: var(--site-surface); margin: calc(28px * var(--site-space, 1)) calc(-1 * max(16px, 5vw)); padding: calc(28px * var(--site-space, 1)) max(16px, 5vw); }
.site-section--accent { background: var(--site-accent); color: var(--site-ground); margin: calc(28px * var(--site-space, 1)) calc(-1 * max(16px, 5vw)); padding: calc(28px * var(--site-space, 1)) max(16px, 5vw); }
.site-section--accent a, .site-section--accent h2, .site-section--accent h3, .site-section--accent blockquote { color: inherit; }
.gosx-site .site-section--accent .button { background: var(--site-ground); color: var(--site-accent); }
.site-section--accent blockquote { border-left-color: currentColor; }
.site-section--dark { --site-ground: #16201d; --site-surface: rgba(255,255,255,.07); --site-ink: #f4f1ea; --site-muted: rgba(244,241,234,.72); --site-rule: rgba(255,255,255,.18); background: #16201d; color: #f4f1ea; margin: calc(28px * var(--site-space, 1)) calc(-1 * max(16px, 5vw)); padding: calc(28px * var(--site-space, 1)) max(16px, 5vw); }
.site-section--accent { --site-surface: rgba(255,255,255,.12); --site-muted: color-mix(in srgb, var(--site-ground) 78%, transparent); --site-rule: color-mix(in srgb, var(--site-ground) 30%, transparent); }
.site-section--accent .site-features__card, .site-section--accent .site-pricing__card, .site-section--dark .site-features__card, .site-section--dark .site-pricing__card { background: var(--site-surface); border-color: var(--site-rule); }
.site-section--accent .site-features__icon, .site-section--accent .site-stats__value, .site-section--accent .site-hero__eyebrow, .site-section--accent .site-team__role { color: inherit; }
.site-section--dark a, .site-section--dark h2, .site-section--dark h3, .site-section--dark blockquote, .site-section--dark .site-hero__headline { color: inherit; }
.gosx-site .site-section--dark .button--primary { background: #f4f1ea; color: #16201d; }
.site-section--image { position: relative; isolation: isolate; color: #fff; margin: calc(28px * var(--site-space, 1)) calc(-1 * max(16px, 5vw)); padding: calc(48px * var(--site-space, 1)) max(16px, 5vw); background: #222 var(--section-image) center / cover no-repeat; }
.site-section--image { --site-ground: #1b1b1b; --site-surface: rgba(255,255,255,.1); --site-ink: #fff; --site-muted: rgba(255,255,255,.78); --site-rule: rgba(255,255,255,.25); }
.site-section--image::before { content: ""; position: absolute; inset: 0; background: rgba(0,0,0,.45); z-index: -1; }
.site-section--image a, .site-section--image h2, .site-section--image h3, .site-section--image blockquote, .site-section--image p { color: inherit; }
.gosx-site .site-section--image .button--primary { background: #fff; color: #16201d; }
.site-section--align-center { text-align: center; }
.site-section--align-center .site-section__inner { margin-inline: auto; }
.site-section--align-center .site-hero__actions, .site-section--align-center .site-cta { justify-content: center; }
.site-section--width-narrow .site-section__inner { max-width: 560px; }
.site-section--width-wide .site-section__inner { max-width: 1120px; }
.site-section--width-full .site-section__inner { max-width: none; }
.site-section--width-wide, .site-section--width-full { width: 100cqw; margin-left: calc(-1 * max(16px, 5vw)); padding-inline: max(16px, 5vw); box-sizing: border-box; }
.site-section--align-center .site-section__inner { margin-inline: auto; }
.site-section--space-compact { padding-block: calc(12px * var(--site-space, 1)) !important; }
.site-section--space-compact .site-section__inner > * { margin-block: 6px; }
.site-section--space-roomy { padding-block: calc(72px * var(--site-space, 1)) !important; }
.site-section--width-wide, .site-section--width-full { max-width: none; }

/* Ready-made sections. Most of them are wider by nature than a column of
   text, so they take the wide measure while the article's text keeps its
   own; the container query keeps them inside the frame on a phone. */
.site-hero, .site-features, .site-testimonials, .site-pricing, .site-cta, .site-stats, .site-team, .site-imagetext, .site-map, .site-faq { width: min(calc(100cqw - 2 * max(16px, 5vw)), 1120px); }
.site-section--width-narrow .site-section__inner > * { width: auto; }

.site-hero { display: grid; gap: 28px; align-items: center; margin: 0 0 32px; max-width: none; }
.site-hero--split { grid-template-columns: 1fr 1fr; }
.site-hero--center { text-align: center; }
.site-hero--center .site-hero__actions { justify-content: center; }
.site-hero--center .site-hero__copy { margin-inline: auto; max-width: 720px; }
.site-hero--cover { position: relative; isolation: isolate; color: #fff; padding: clamp(48px, 10vw, 120px) clamp(20px, 5vw, 56px); border-radius: calc(var(--site-radius, 2px) * 2); background: #222 var(--hero-image, none) center / cover no-repeat; text-align: center; }
.site-hero--cover::before { content: ""; position: absolute; inset: 0; background: rgba(0,0,0,.42); border-radius: inherit; z-index: -1; }
.site-hero--cover .site-hero__headline, .site-hero--cover .site-hero__text, .site-hero--cover .site-hero__eyebrow { color: inherit; }
.site-hero--cover .site-hero__actions { justify-content: center; }
.gosx-site .site-hero--cover .button--primary { background: #fff; color: #16201d; }
.site-hero__eyebrow { margin: 0 0 8px; font-size: 13px; letter-spacing: .12em; text-transform: uppercase; color: var(--site-accent); font-weight: 600; }
.site-hero__headline { font-size: clamp(32px, 5.5vw, 56px); line-height: 1.05; margin: 0 0 16px !important; text-wrap: balance; }
.site-hero__text { font-size: 18px; color: var(--site-muted); line-height: 1.55; max-width: 60ch; }
.site-hero--center .site-hero__text { margin-inline: auto; }
.site-hero__actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 8px; }
.site-hero__picture { margin: 0; }
.site-hero__picture img { width: 100%; height: auto; border-radius: calc(var(--site-radius, 2px) * 2); display: block; }
.button--ghost { background: transparent !important; color: var(--site-accent) !important; border: 1.5px solid currentColor; }
.site-features { max-width: none; margin: 0 0 32px; }
.site-features__intro { color: var(--site-muted); max-width: 60ch; }
.site-features__grid { list-style: none; padding: 0; margin: 16px 0 0; display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 18px; }
.site-features__card { padding: 22px; border: 1px solid var(--site-rule); border-radius: calc(var(--site-radius, 2px) * 2); background: var(--site-ground); }
.site-features--plain .site-features__card { padding: 0; border: 0; background: transparent; }
.site-features__icon { display: inline-grid; place-items: center; min-width: 40px; height: 40px; padding: 0 10px; border-radius: 999px; background: color-mix(in srgb, var(--site-accent) 14%, transparent); color: var(--site-accent); font-weight: 700; margin-bottom: 12px; }
.site-features--numbered .site-features__icon { background: var(--site-accent); color: var(--site-ground); }
.site-features__title { margin: 0 0 6px !important; font-size: 18px; }
.site-features__text { margin: 0; color: var(--site-muted); }
.site-testimonials { max-width: none; margin: 0 0 32px; }
.site-testimonials__grid { list-style: none; padding: 0; margin: 16px 0 0; display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 18px; }
.site-testimonials--single .site-testimonials__grid { grid-template-columns: 1fr; max-width: 720px; }
.site-testimonials__card { padding: 22px; border-radius: calc(var(--site-radius, 2px) * 2); background: var(--site-surface); }
.site-testimonials__quote { margin: 0 0 14px; border: 0; padding: 0; font-size: 17px; line-height: 1.5; font-style: normal; }
.site-testimonials--single .site-testimonials__quote { font-size: clamp(20px, 3vw, 26px); }
.site-testimonials__quote::before { content: "“"; color: var(--site-accent); font-size: 1.4em; line-height: 0; margin-right: 2px; }
.site-testimonials__who { display: flex; align-items: center; gap: 12px; }
.site-testimonials__photo { margin: 0; width: 44px; height: 44px; flex: none; }
.site-testimonials__photo img { width: 44px; height: 44px; border-radius: 50%; object-fit: cover; display: block; }
.site-testimonials__name { display: block; font-weight: 600; }
.site-testimonials__role { display: block; font-size: 13px; color: var(--site-muted); }
.site-pricing { max-width: none; margin: 0 0 32px; }
.site-pricing__intro { color: var(--site-muted); max-width: 60ch; }
.site-pricing__grid { list-style: none; padding: 0; margin: 16px 0 0; display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 18px; align-items: start; }
.site-pricing--simple .site-pricing__grid { grid-template-columns: 1fr; max-width: 640px; }
.site-pricing__card { position: relative; padding: 24px; border: 1px solid var(--site-rule); border-radius: calc(var(--site-radius, 2px) * 2); background: var(--site-ground); }
.site-pricing__card--highlight { border-color: var(--site-accent); box-shadow: 0 0 0 1px var(--site-accent); }
.site-pricing__name { margin: 0 0 8px !important; font-size: 18px; }
.site-pricing__price { margin: 0 0 8px; }
.site-pricing__amount { font-size: 32px; font-weight: 700; letter-spacing: -.02em; }
.site-pricing__period { color: var(--site-muted); }
.site-pricing__blurb { color: var(--site-muted); margin: 0 0 12px; }
ul.site-pricing__features { list-style: none; padding: 0; margin: 0 0 16px; }
ul.site-pricing__features li { padding: 6px 0; border-top: 1px solid var(--site-rule); }
ul.site-pricing__features li::before { content: "✓ "; color: var(--site-accent); }
.site-faq { max-width: none; margin: 0 0 32px; }
.site-faq__list { margin: 12px 0 0; border-top: 1px solid var(--site-rule); }
.site-faq__item { border-bottom: 1px solid var(--site-rule); }
.site-faq__question { cursor: pointer; padding: 14px 0; font-weight: 600; font-size: 17px; list-style: none; display: flex; justify-content: space-between; gap: 12px; }
.site-faq__question::-webkit-details-marker { display: none; }
.site-faq__question::after { content: "+"; color: var(--site-accent); font-weight: 400; }
details[open] > .site-faq__question::after { content: "–"; }
.site-faq__answer { padding: 0 0 16px; color: var(--site-muted); }
.site-cta { display: flex; flex-wrap: wrap; align-items: center; gap: 12px 28px; margin: 0 0 32px; max-width: none; }
.site-cta--band { padding: 28px 30px; border-radius: calc(var(--site-radius, 2px) * 2); background: var(--site-accent); color: var(--site-ground); }
.site-cta--band .site-cta__headline, .site-cta--band .site-cta__text { color: inherit; }
.gosx-site .site-cta--band .button--primary { background: var(--site-ground); color: var(--site-accent); margin: 0; }
.site-cta__headline { margin: 0 !important; font-size: clamp(20px, 3vw, 26px); flex: 1 1 260px; }
.site-cta__text { margin: 0; flex: 2 1 320px; opacity: .9; }
.site-stats { max-width: none; margin: 0 0 32px; }
.site-stats__row { list-style: none; padding: 0; margin: 12px 0 0; display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 18px; }
.site-stats--cards .site-stats__item { padding: 18px; border-radius: calc(var(--site-radius, 2px) * 2); background: var(--site-surface); }
.site-stats__value { display: block; font-size: clamp(30px, 5vw, 44px); font-weight: 700; letter-spacing: -.02em; color: var(--site-accent); line-height: 1.1; }
.site-stats__label { color: var(--site-muted); }
.site-team { max-width: none; margin: 0 0 32px; }
.site-team__intro { color: var(--site-muted); max-width: 60ch; }
.site-team__grid { list-style: none; padding: 0; margin: 16px 0 0; display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 22px; }
.site-team--list .site-team__grid { grid-template-columns: 1fr; }
.site-team--list .site-team__person { display: grid; grid-template-columns: 96px 1fr; gap: 6px 18px; align-items: start; }
.site-team--list .site-team__photo { grid-row: span 3; }
.site-team__photo { margin: 0 0 10px; }
.site-team__photo img { width: 100%; aspect-ratio: 1; object-fit: cover; border-radius: calc(var(--site-radius, 2px) * 2); display: block; }
.site-team__name { margin: 0 !important; font-size: 17px; }
.site-team__role { margin: 0 0 6px; font-size: 13px; color: var(--site-accent); font-weight: 600; }
.site-team__bio { margin: 0; color: var(--site-muted); }
.site-hours { margin: 0 0 32px; }
.site-hours__table { display: grid; grid-template-columns: max-content 1fr; gap: 8px 24px; margin: 12px 0; }
.site-hours__row { display: contents; }
.site-hours--inline .site-hours__table { display: flex; flex-wrap: wrap; gap: 8px 22px; }
.site-hours--inline .site-hours__row { display: inline-flex; gap: 6px; }
.site-hours__day { font-weight: 600; }
.site-hours__note { color: var(--site-muted); font-size: 14px; }
.site-imagetext { display: grid; grid-template-columns: 1fr 1fr; gap: 28px; align-items: center; margin: 0 0 32px; max-width: none; }
.site-imagetext--left .site-imagetext__picture { order: -1; }
.site-imagetext__picture { margin: 0; }
.site-imagetext__picture img { width: 100%; height: auto; display: block; border-radius: calc(var(--site-radius, 2px) * 2); }
.site-imagetext__heading { margin-top: 0 !important; }
.site-map { margin: 0 0 32px; max-width: none; }
.site-map__frame { position: relative; aspect-ratio: 16 / 9; border-radius: calc(var(--site-radius, 2px) * 2); overflow: hidden; background: var(--site-surface); margin: 10px 0; }
.site-map--compact .site-map__frame { aspect-ratio: 3 / 2; max-width: 520px; }
.site-map__frame iframe { position: absolute; inset: 0; width: 100%; height: 100%; border: 0; }
.site-map__address { font-weight: 600; margin: 0; }
.site-map__note { color: var(--site-muted); font-size: 14px; }
.site-posts__heading, .site-products__heading { margin-top: 0 !important; }
.site-posts, .site-products { max-width: none; }
.ed-live { pointer-events: none; opacity: .92; }
.ed-live-hint { padding: 18px; border: 1px dashed var(--site-rule); border-radius: 8px; color: var(--site-muted); font-size: 14px; text-align: center; }
.site-spacer { height: 32px; }
.site-spacer--small { height: 16px; }
.site-spacer--large { height: 72px; }
@container site (max-width: 640px) {
  .site-hero--split, .site-imagetext { grid-template-columns: 1fr; }
  .site-team--list .site-team__person { grid-template-columns: 72px 1fr; }
}
.site-gallery { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 10px; margin: 20px 0 24px; max-width: none; }
.site-gallery__item { margin: 0; }
.site-gallery__open { display: block; }
.site-gallery--strip { display: flex; overflow-x: auto; gap: 10px; scroll-snap-type: x mandatory; padding-bottom: 6px; }
.site-gallery--strip .site-gallery__item { flex: 0 0 min(260px, 70%); scroll-snap-align: start; }
.site-gallery--masonry { display: block; columns: 3 180px; column-gap: 10px; }
.site-gallery--masonry .site-gallery__item { break-inside: avoid; margin-bottom: 10px; }
.site-gallery--masonry .site-gallery__item img { aspect-ratio: auto; height: auto; }
.site-gallery--big { display: grid; grid-template-columns: 1fr; gap: 14px; }
.site-gallery--big .site-gallery__item img { aspect-ratio: 16 / 10; }
.site-lightbox { display: none; position: fixed; inset: 0; z-index: 90; background: rgba(0,0,0,.88); align-items: center; justify-content: center; padding: 24px; }
.site-lightbox:target { display: flex; animation: site-fade .18s ease-out; }
@keyframes site-fade { from { opacity: 0; } to { opacity: 1; } }
.site-lightbox img { max-width: 100%; max-height: 92vh; width: auto; height: auto; object-fit: contain; border-radius: 4px; aspect-ratio: auto; }
.site-lightbox__close { position: absolute; top: 14px; right: 18px; color: #fff; font-size: 26px; text-decoration: none; line-height: 1; }
.ed-block .site-lightbox { display: none !important; }
.site-gallery__item img { width: 100%; height: 100%; aspect-ratio: 4 / 3; object-fit: cover; display: block; border-radius: min(var(--site-radius, 2px), 12px); }
.site-video { position: relative; aspect-ratio: 16 / 9; margin: 20px 0 24px; background: var(--site-surface); border-radius: min(var(--site-radius, 2px), 12px); overflow: hidden; }
.site-video iframe { position: absolute; inset: 0; width: 100%; height: 100%; border: 0; }
.site-columns { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; margin: 0 0 16px; max-width: none; }
.site-columns--3 { grid-template-columns: 1fr 1fr 1fr; }
.site-align-center { text-align: center; }
.site-align-right { text-align: right; }
.site-button-row { margin: 4px 0 20px; }
.site-button-row .button { margin: 0; }
.button--link { background: transparent !important; color: var(--site-accent) !important; padding-left: 0 !important; padding-right: 0 !important; text-decoration: underline !important; text-underline-offset: 3px; }
.ed-columns:not([data-columns="3"]) .site-columns__col--third { display: none; }
.ed-columns { position: relative; padding-top: 30px; }
.ed-columns__count { position: absolute; top: 0; left: 0; margin: 0; }
.ed-button-row.site-align-center { justify-content: center; }
.ed-button-row.site-align-right { justify-content: flex-end; }
.ed-bubble__sep { width: 1px; height: 18px; background: rgba(255,255,255,.3); margin: 0 4px; }
.site-columns__col { min-width: 0; }
/* The site is its own container, so phone rules follow the width of the
   site — the real viewport for visitors, the frame inside the editor. */
.gosx-site--public { container-type: inline-size; container-name: site; }
.site-no-phone { display: contents; }
@container site (max-width: 640px) {
  .site-columns { grid-template-columns: 1fr; }
  .site-no-phone { display: none; }
  .site-product { grid-template-columns: 1fr; gap: 20px; }
  .site-products { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
}
.site-post-meta { color: var(--site-muted); font-size: 14px; margin: -8px 0 24px; }
.site-post-meta a { color: inherit; }
.site-posts { list-style: none; margin: 0; padding: 0; }
.site-post-card { padding: 22px 0; border-top: 1px solid var(--site-rule); }
.site-post-card:first-child { border-top: 0; padding-top: 0; }
.site-post-card__title { margin: 0 0 6px; font-size: clamp(20px, 2.6vw, 25px); line-height: 1.2; }
.site-post-card__title a { text-decoration: none; }
.site-post-card__title a:hover { text-decoration: underline; }
.site-post-card .site-post-meta { margin: 0 0 8px; }
.site-post-card__excerpt { margin: 0; color: var(--site-muted); }
.site-categories { display: flex; flex-wrap: wrap; gap: 8px; margin: 0 0 22px; }
.site-categories a { font-size: 13.5px; padding: 5px 11px; border: 1px solid var(--site-rule); border-radius: 999px; text-decoration: none; color: var(--site-muted); }
.site-categories a[aria-current="page"] { background: var(--site-accent); border-color: var(--site-accent); color: var(--site-on-accent, #fff); }
.site-pager { display: flex; justify-content: space-between; gap: 16px; margin: 28px 0 0; padding-top: 18px; border-top: 1px solid var(--site-rule); }
.site-pager a:only-child { margin-left: auto; }
.site-post-nav { margin: 36px 0 0; padding-top: 18px; border-top: 1px solid var(--site-rule); font-size: 14.5px; }
.site-products { list-style: none; margin: 0 0 24px; padding: 0; display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 18px; max-width: none; }
.site-products--single { grid-template-columns: minmax(180px, 260px); }
.site-product-card__link { display: block; text-decoration: none; color: inherit; }
.site-product-card__picture { display: block; aspect-ratio: 1; background: var(--site-surface); border-radius: min(var(--site-radius, 2px), 12px); overflow: hidden; margin-bottom: 10px; }
.site-product-card__picture img { width: 100%; height: 100%; object-fit: cover; display: block; }
.site-product-card__blank { display: block; width: 100%; height: 100%; }
.site-product-card__name { display: block; font-weight: 600; }
.site-product-card__price { display: block; color: var(--site-muted); font-size: 14.5px; margin-top: 2px; }
.site-price--was { color: var(--site-muted); opacity: .8; }
.site-price--big { font-size: 22px; font-weight: 600; }
.site-badge { display: inline-block; font-size: 11px; letter-spacing: .04em; text-transform: uppercase; padding: 2px 7px; background: var(--site-ink); color: var(--site-ground); border-radius: 2px; vertical-align: middle; }
.site-product { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 36px; max-width: 1040px; }
.site-product__pictures { display: grid; gap: 10px; }
.site-product__picture { margin: 0; }
.site-product__picture img { width: 100%; height: auto; display: block; border-radius: min(var(--site-radius, 2px), 12px); }
.site-product__pictures .site-product__picture + .site-product__picture img { aspect-ratio: 4 / 3; object-fit: cover; }
.site-product__price { margin: -8px 0 18px; }
.site-product__description { margin: 22px 0; }
.site-buy { max-width: 420px; margin: 0 0 8px; }
.site-buy__row { display: flex; gap: 12px; align-items: flex-end; }
.site-buy__qty { max-width: 110px; }
.site-product__stock { display: block; font-size: 14px; color: var(--site-muted); margin-top: 6px; }
.site-cart__save { display: flex; flex-wrap: wrap; gap: 10px; align-items: flex-end; margin-top: 22px; padding-top: 16px; border-top: 1px solid var(--site-rule); max-width: 520px; }
.site-cart__save .site-form__field { flex: 1 1 220px; }
.site-inline-form { display: inline; }
.site-slots { display: flex; flex-wrap: wrap; gap: 8px; margin: 0 0 14px; }
.site-slot { display: inline-flex; align-items: center; gap: 6px; padding: 8px 12px; border: 1px solid var(--site-rule); border-radius: var(--site-radius, 2px); cursor: pointer; }
.site-slot:has(input:checked) { border-color: var(--site-accent); box-shadow: inset 0 0 0 1px var(--site-accent); }
.site-book__day { margin-bottom: 14px; }
.site-downloads { margin: 24px 0; padding: 16px 18px; background: var(--site-surface); border-radius: min(var(--site-radius, 2px), 8px); }
.site-downloads h3 { margin: 0 0 8px; }
.admin-day th { text-align: left; padding-top: 18px; font-size: 13px; text-transform: uppercase; letter-spacing: .04em; color: var(--site-muted); }
.site-notice { padding: 10px 14px; background: var(--site-surface); border-radius: min(var(--site-radius, 2px), 8px); font-size: 14.5px; }
.site-cart-page .site-article { max-width: none; }
.site-cart__table { width: 100%; border-collapse: collapse; margin: 0 0 18px; }
.site-cart__table th { text-align: left; font-size: 12.5px; text-transform: uppercase; letter-spacing: .04em; color: var(--site-muted); padding: 0 8px 8px 0; border-bottom: 1px solid var(--site-rule); }
.site-cart__table td { padding: 12px 8px 12px 0; border-bottom: 1px solid var(--site-rule); vertical-align: middle; }
.site-cart__item { display: flex; gap: 12px; align-items: center; }
.site-cart__thumb { width: 56px; height: 56px; object-fit: cover; border-radius: min(var(--site-radius, 2px), 8px); }
.site-cart__qty { width: 72px; padding: 8px; font: inherit; border: 1px solid var(--site-rule); border-radius: min(var(--site-radius, 2px), 8px); background: var(--site-ground); color: inherit; }
.site-cart__total { text-align: right; font-variant-numeric: tabular-nums; }
.site-cart__remove { font: inherit; background: none; border: 1px solid var(--site-rule); color: var(--site-muted); padding: 4px 8px; cursor: pointer; border-radius: 2px; }
.site-cart__update { font: inherit; background: none; border: 1px solid var(--site-rule); color: inherit; padding: 10px 16px; cursor: pointer; border-radius: var(--site-radius, 2px); }
.site-cart__actions { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
.site-cart__note { color: var(--site-muted); font-size: 14px; }
.ed-product { display: flex; flex-direction: column; gap: 8px; margin: 18px 0 28px; }
.ed-product .site-products { margin: 0; }
.ed-product .site-product-card__link { pointer-events: none; }
.admin-media__thumb--small { width: 56px; height: 56px; object-fit: cover; }
.site-list { margin: 0 0 16px; padding-left: 22px; }
.site-list li { margin: 0 0 6px; }
.site-divider { border: 0; border-top: 1px solid var(--site-rule); margin: 28px 0; }
.site-section--accent .site-divider { border-top-color: currentColor; opacity: .5; }
.site-article strong { font-weight: 600; }
.site-article a { text-decoration: underline; text-underline-offset: 3px; }
.site-article .button { margin: 4px 0 20px; }
.site-article > p:first-of-type { font-size: 18px; color: var(--site-muted); line-height: 1.55; }
.site-button, .admin-button, .site-article .button {
  display: inline-block; padding: 10px 18px; border-radius: var(--site-radius, 2px);
  background: var(--site-accent); color: var(--site-ground);
  text-decoration: none; font-weight: 500; border: 0; cursor: pointer;
  font-size: 15px;
  transition: filter .15s, transform .12s, box-shadow .15s;
}
.site-button:hover, .site-article .button:hover { filter: brightness(1.06); box-shadow: 0 4px 14px rgba(0,0,0,.12); }
.site-button:active, .site-article .button:active { transform: translateY(1px); }
.site-form { display: flex; flex-direction: column; gap: 14px; max-width: 520px; margin: 18px 0 28px; }
.site-form__check { display: flex; gap: 10px; align-items: flex-start; font-size: 15px; }
.site-form__check input { margin-top: 4px; }
.site-form select { width: 100%; padding: 10px 12px; font: inherit; font-size: 16px; border: 1px solid var(--site-rule); border-radius: min(var(--site-radius, 2px), 8px); background: var(--site-ground); color: inherit; }
.ed-form { display: flex; flex-direction: column; gap: 8px; margin: 18px 0 28px; }
.ed-form .site-form { margin: 0; }
.ed-form__bar { display: flex; flex-wrap: wrap; gap: 8px 16px; align-items: center; font-size: 12.5px; color: var(--site-muted); }
.ed-form__bar a { color: var(--site-accent); }
.ed-inline-select { font: inherit; font-size: 12.5px; padding: 3px 6px; border: 1px solid var(--site-rule); background: var(--site-ground); color: var(--site-ink); }
.admin-fields input[type="text"], .admin-fields input[type="number"], .admin-fields select { width: 100%; min-width: 90px; font: inherit; font-size: 13.5px; padding: 6px 8px; border: 1px solid var(--site-rule); background: var(--site-ground); color: inherit; }
.admin-field-row__required { text-align: center; }
.admin-message__fields { margin: 0; display: grid; grid-template-columns: max-content 1fr; gap: 4px 14px; font-size: 14px; }
.admin-message__field { display: contents; }
.admin-message__fields dt { color: var(--site-muted); }
.admin-message__fields dd { margin: 0; white-space: pre-wrap; }
.site-form__field { display: flex; flex-direction: column; gap: 5px; }
.site-form__field span { font-size: 14px; color: var(--site-muted); }
.site-form__field input, .site-form__field textarea {
  width: 100%; padding: 10px 12px; font: inherit; font-size: 16px; border-radius: min(var(--site-radius, 2px), 8px);
  border: 1px solid var(--site-rule); background: var(--site-ground); color: var(--site-ink);
}
.site-form__field input:focus-visible, .site-form__field textarea:focus-visible { outline: 2px solid var(--site-accent); outline-offset: 1px; }
.site-form button.site-button { align-self: flex-start; }
.site-form__error { margin: 0; padding: 10px 14px; border-left: 3px solid #9c312a; background: rgba(156,49,42,.08); font-size: 15px; }
.site-form__hp { position: absolute; left: -10000px; width: 1px; height: 1px; overflow: hidden; }
.site-form--sent { border: 1px solid var(--site-rule); background: var(--site-surface); padding: 20px 22px; }
.site-form--sent h3 { margin: 0 0 6px; }
.site-form--sent p { margin: 0; color: var(--site-muted); }
.ed-form-preview { pointer-events: none; opacity: .92; }
.ed-form-preview .site-button { align-self: flex-start; }
.ed-form-preview__note { margin: 0; font-size: 12.5px; color: var(--site-muted); }
.admin-messages { display: flex; flex-direction: column; gap: 12px; max-width: 900px; }
.admin-message { border: 1px solid var(--site-rule); background: var(--site-ground); padding: 16px 18px; border-left-width: 3px; }
.admin-message[data-state="unread"] { border-left-color: var(--site-accent); }
.admin-message[data-state="read"] { opacity: .8; }
.admin-message__head { display: flex; flex-wrap: wrap; gap: 6px 16px; align-items: baseline; font-size: 14px; }
.admin-message__head time { color: var(--site-muted); margin-left: auto; }
.admin-message__body { margin: 10px 0 12px; white-space: pre-wrap; overflow-wrap: anywhere; max-width: 68ch; }
.admin-message__actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.site-consent {
  position: fixed; left: 16px; right: 16px; bottom: 16px; z-index: 50; max-width: 640px; margin: 0 auto;
  display: flex; flex-wrap: wrap; gap: 12px 20px; align-items: center; justify-content: space-between;
  padding: 16px 18px; border: 1px solid var(--site-rule); background: var(--site-ground); color: var(--site-ink);
  box-shadow: 0 8px 28px rgba(0,0,0,.14); border-radius: 4px; font-size: 14.5px;
}
.site-consent[hidden] { display: none; }
.site-consent p { margin: 0; flex: 1 1 320px; }
.site-consent__actions { display: flex; gap: 8px; }
.site-consent__decline { font: inherit; font-size: 14px; background: none; border: 1px solid var(--site-rule); color: var(--site-ink); padding: 9px 14px; border-radius: 2px; cursor: pointer; }
.site-footer {
  display: flex; flex-direction: column; gap: 10px;
  padding: 28px max(16px, 5vw) 32px;
  border-top: 1px solid var(--site-rule);
  color: var(--site-muted); font-size: 14px;
}
.site-footer p { margin: 0; }
.site-footer__text { color: var(--site-ink); font-size: 15px; max-width: 60ch; }
.site-social { display: flex; flex-wrap: wrap; gap: 8px; }
.site-social__link {
  display: inline-block; padding: 5px 12px; border-radius: 999px; font-size: 13px; text-decoration: none;
  border: 1px solid var(--site-rule); color: var(--site-ink);
}
.site-social__link:hover { border-color: var(--site-accent); color: var(--site-accent); }
.site-footer__contact { display: flex; flex-wrap: wrap; gap: 4px 18px; }
.site-footer__meta { font-size: 13px; opacity: .85; }
.admin-subhead { font-size: 17px; margin: 26px 0 12px; padding-top: 18px; border-top: 1px solid var(--site-rule); }
.admin-field--image { max-width: 640px; }
.admin-image-preview { display: block; max-height: 64px; width: auto; margin: 0 0 8px; padding: 6px; border: 1px solid var(--site-rule); background: var(--site-ground); }
.admin-check { display: block; font-size: 14px; margin: 0 0 8px; }
.admin-choices { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 8px; max-width: 640px; }
.admin-choice { position: relative; display: flex; flex-direction: column; gap: 2px; padding: 10px 12px; border: 1px solid var(--site-rule); border-radius: 2px; cursor: pointer; }
.admin-choice input { position: absolute; opacity: 0; pointer-events: none; }
.admin-choice:has(input:checked) { border-color: var(--site-accent); box-shadow: inset 0 0 0 1px var(--site-accent); }
.admin-choice span { font-size: 14.5px; font-weight: 500; }
.admin-choice small { font-size: 12.5px; color: var(--site-muted); }

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
.admin-badge[data-state="offline"], .admin-badge[data-state="archived"] { color: #9c312a; }
.admin-row-actions { display: flex; flex-wrap: wrap; gap: 4px; justify-content: flex-end; }
.admin-inline-form { display: inline; }
.admin-row-btn {
  font: inherit; font-size: 12.5px; padding: 4px 9px; cursor: pointer; border-radius: 2px;
  border: 1px solid var(--site-rule); background: var(--site-ground); color: var(--site-ink);
}
.admin-row-btn:hover { border-color: var(--site-accent); color: var(--site-accent); }
.admin-row-btn[data-action="archive"]:hover { border-color: #9c312a; color: #9c312a; }
.admin-table--pages td:last-child { white-space: nowrap; }
.admin-stats { display: flex; flex-wrap: wrap; gap: 1px; background: var(--site-rule); border: 1px solid var(--site-rule); margin: 0 0 28px; max-width: 900px; }
.admin-stat { background: var(--site-ground); padding: 16px 20px; flex: 1 1 160px; }
.admin-stat strong { display: block; font-size: 30px; line-height: 1.1; font-variant-numeric: tabular-nums; }
.admin-stat span { display: block; font-size: 13px; color: var(--site-muted); margin-top: 4px; }
.wz-card--narrow { max-width: 460px; }
.wz-actions--alt { margin-top: -6px; }
.wz-actions--alt .wz-btn { width: 100%; text-align: center; }
.admin-radios { display: flex; flex-wrap: wrap; gap: 6px 18px; align-items: center; }
.admin-radios__label { color: var(--site-muted); font-size: 13px; }
.admin-radio { display: inline-flex; align-items: center; gap: 6px; cursor: pointer; }
.admin-nav__account { margin-left: auto; }
.admin-nav__signout { font: inherit; font-size: 14px; background: none; border: 0; color: var(--site-muted); cursor: pointer; padding: 0; text-decoration: underline; text-underline-offset: 3px; }
.admin-code { background: var(--site-surface); border: 1px solid var(--site-rule); padding: 12px 14px; overflow-x: auto; font-size: 13px; line-height: 1.5; white-space: pre-wrap; overflow-wrap: anywhere; }
.admin-table--records code { font-size: 13px; }
.admin-columns { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 20px; max-width: 900px; }
.admin-columns .admin-panel { margin: 0; }
.stats-chart { display: block; width: 100%; max-width: 900px; height: auto; margin-top: 8px; }
.stats-bar { fill: var(--site-accent); }
.stats-bar:hover { fill: var(--site-ink); }
.stats-axis { stroke: var(--site-rule); stroke-width: 1; }
.stats-label { fill: var(--site-muted); font-size: 11px; font-family: var(--site-font-body, inherit); }
.stats-table td { vertical-align: middle; }
.stats-row__label { display: block; overflow-wrap: anywhere; }
.stats-row__bar { display: block; width: 100%; height: 4px; margin-top: 5px; }
.stats-row__bar rect { fill: var(--site-accent); }
.stats-count { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }

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

/* ---------- Look (site-wide theme) ---------- */
.ed-look-group { margin: 0 0 14px; }
.ed-look-group > span { display: block; font-size: 12.5px; color: var(--site-muted); margin: 0 0 6px; }
.ed-swatches { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; }
.ed-swatch {
  position: relative; display: flex; flex-direction: column; gap: 4px; cursor: pointer;
  border: 1px solid var(--site-rule); border-radius: 2px; padding: 6px; background: var(--site-ground);
}
.ed-swatch input { position: absolute; opacity: 0; pointer-events: none; }
.ed-swatch:has(input:checked) { border-color: var(--site-accent); box-shadow: inset 0 0 0 1px var(--site-accent); }
.ed-swatch:has(input:focus-visible) { outline: 2px solid var(--site-accent); outline-offset: 2px; }
.ed-swatch__chip { height: 26px; border-radius: 2px; display: flex; align-items: flex-end; padding: 3px; }
.ed-swatch__chip i { display: block; width: 14px; height: 6px; border-radius: 1px; }
.ed-swatch__name { font-size: 11.5px; color: var(--site-ink); }
.ed-shapes { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; }
.ed-shape { position: relative; display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 8px 4px; border: 1px solid var(--site-rule); background: var(--site-ground); cursor: pointer; font-size: 11.5px; }
.ed-shape input { position: absolute; opacity: 0; pointer-events: none; }
.ed-shape:has(input:checked) { border-color: var(--site-accent); box-shadow: inset 0 0 0 1px var(--site-accent); }
.ed-shape:has(input:focus-visible) { outline: 2px solid var(--site-accent); outline-offset: 2px; }
.ed-shape__sample { display: block; width: 34px; height: 16px; background: var(--site-accent); }
.ed-shape__sample--space { width: 34px; height: 18px; background: none; border-top: 2px solid var(--site-accent); border-bottom: 2px solid var(--site-accent); box-sizing: border-box; }
.ed-shape__sample--space[data-scale="compact"] { height: 10px; }
.ed-shape__sample--space[data-scale="airy"] { height: 26px; }
.ed-fonts { display: grid; gap: 6px; }
.ed-font {
  position: relative; display: flex; align-items: baseline; justify-content: space-between; gap: 8px; cursor: pointer;
  border: 1px solid var(--site-rule); border-radius: 2px; padding: 7px 10px; background: var(--site-ground);
}
.ed-font input { position: absolute; opacity: 0; pointer-events: none; }
.ed-font:has(input:checked) { border-color: var(--site-accent); box-shadow: inset 0 0 0 1px var(--site-accent); }
.ed-font:has(input:focus-visible) { outline: 2px solid var(--site-accent); outline-offset: 2px; }
.ed-font__sample { font-size: 17px; line-height: 1.1; color: var(--site-ink); }
.ed-font__name { font-size: 11.5px; color: var(--site-muted); white-space: nowrap; }
.ed-accent { display: flex; align-items: center; gap: 10px; }
.ed-accent input[type="color"] {
  width: 42px; height: 30px; padding: 2px; border: 1px solid var(--site-rule); border-radius: 2px;
  background: var(--site-ground); cursor: pointer;
}
.ed-accent small { font-size: 12px; color: var(--site-muted); }
.ed-accent button { font: inherit; font-size: 12px; background: none; border: 0; color: var(--site-accent); cursor: pointer; padding: 0; text-decoration: underline; }

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
.ed-people { display: inline-flex; align-items: center; gap: 2px; margin-right: 6px; }
.ed-person { width: 26px; height: 26px; border-radius: 50%; display: inline-grid; place-items: center; font-size: 11px; font-weight: 700; letter-spacing: .02em; color: #fff; background: hsl(var(--person, 200) 55% 42%); border: 2px solid #fff; box-shadow: 0 0 0 1px rgba(0,0,0,.12); cursor: default; }
.ed-person + .ed-person { margin-left: -8px; }
.ed-toast { position: fixed; left: 50%; bottom: 24px; transform: translateX(-50%); background: #1f2933; color: #fff; padding: 10px 16px; border-radius: 8px; font-size: 14px; line-height: 1.4; z-index: 60; max-width: min(90vw, 560px); box-shadow: 0 6px 24px rgba(0,0,0,.25); }
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
  transition: max-width .2s ease;
}
.ed-frame--phone { max-width: 412px; border: 10px solid #1f2933; border-radius: 26px; box-shadow: 0 8px 30px rgba(0,0,0,.18); overflow: hidden; }
.ed-device { display: inline-flex; border: 1px solid var(--site-rule); border-radius: 2px; overflow: hidden; }
.ed-device__btn { font: inherit; font-size: 13px; padding: 7px 11px; background: var(--site-ground); color: var(--site-muted); border: 0; cursor: pointer; }
.ed-device__btn + .ed-device__btn { border-left: 1px solid var(--site-rule); }
.ed-device__btn[aria-pressed="true"] { background: var(--site-ink); color: var(--site-ground); }
.ed-block__badge { display: none; position: absolute; left: 0; top: -9px; font-size: 10.5px; letter-spacing: .04em; text-transform: uppercase; padding: 1px 6px; background: var(--site-ink); color: var(--site-ground); border-radius: 2px; pointer-events: none; }
.ed-block[data-phone="hide"] .ed-block__badge { display: inline-block; }
.ed-frame--phone .ed-block[data-phone="hide"] { opacity: .35; outline: 1px dashed var(--site-muted); }
.ed-tool[aria-pressed="true"] { background: var(--site-ink); color: var(--site-ground); }
.ed-block__lock { display: none; position: absolute; right: 0; top: -9px; font-size: 10.5px; letter-spacing: .04em; text-transform: uppercase; padding: 1px 6px; background: #8a5a00; color: #fff; border-radius: 2px; pointer-events: none; }
.ed-block[data-locked="true"] .ed-block__lock { display: inline-block; }
.ed-block--frozen { opacity: .8; }
.ed-block--frozen .ed-block__tools { display: none; }
.ed[data-can-lock="false"] [data-tool="lock"] { display: none; }
.admin-prune { display: inline-flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.admin-prune input { width: 70px; font: inherit; padding: 5px 8px; border: 1px solid var(--site-rule); }
/* The canvas paints its own ground and ink from the site's theme, so a dark
   palette previews as dark instead of borrowing the editor frame's white. */
.ed-canvas { background: var(--site-ground); color: var(--site-ink); font-family: var(--site-font-body); }
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
.ed-tool[data-tool="grab"] { cursor: grab; touch-action: none; }
.ed-canvas .site-article { position: relative; }
.ed-block.is-dragging { opacity: .4; }
.ed-drop-line {
  position: absolute; left: 0; right: 0; height: 3px; border-radius: 2px;
  background: var(--site-accent); pointer-events: none; z-index: 5;
}
body.ed-is-dragging { user-select: none; cursor: grabbing; }
body.ed-is-dragging .ed-block__tools, body.ed-is-dragging .ed-insert, body.ed-is-dragging .ed-levels { display: none; }
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
.admin-choices { border: 0; padding: 0; margin: 0 0 14px; display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 8px; }
.admin-choices legend { font-size: 13px; font-weight: 600; margin-bottom: 8px; }
.admin-choice { display: grid; grid-template-columns: auto 1fr; gap: 2px 8px; align-items: baseline; padding: 10px 12px; border: 1px solid var(--admin-rule, #e3e8ee); border-radius: 8px; cursor: pointer; background: #fff; }
.admin-choice:has(input:checked) { border-color: var(--admin-accent, #0b6e4f); background: #f2faf6; }
.admin-choice__name { font-weight: 600; } .admin-choice__blurb { grid-column: 2; font-size: 12.5px; color: #5f6b7a; }
.ed-figure__options { display: flex; flex-wrap: wrap; gap: 8px; }
.ed-figure__options .ed-variant { margin: 0; }
.ed-caption { font-style: italic; }
.ed-custom-colours { display: none; flex-wrap: wrap; gap: 8px 14px; align-items: center; margin-top: 8px; font-size: 12.5px; color: var(--site-muted); }
.ed-custom-colours label { display: inline-flex; align-items: center; gap: 6px; }
.ed-custom-colours input[type=color] { width: 32px; height: 24px; padding: 0; border: 1px solid var(--site-rule); border-radius: 4px; background: none; }
.ed-custom-colours small { flex-basis: 100%; }
[data-look][data-custom] .ed-custom-colours { display: flex; }
.ed-custom-fonts { display: none; flex-direction: column; gap: 6px; margin-top: 8px; font-size: 12.5px; color: var(--site-muted); }
.ed-custom-fonts label { display: grid; grid-template-columns: 64px 1fr; align-items: center; gap: 6px; }
.ed-custom-fonts input { font: inherit; font-size: 12.5px; padding: 4px 6px; border: 1px solid var(--site-rule); border-radius: 4px; background: var(--site-ground); color: var(--site-ink); }
[data-look][data-custom-fonts] .ed-custom-fonts { display: flex; }
.ed-shape--text span { font-size: 12.5px; }
.ed-section-bar {
  display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px 16px;
  margin: 14px 0 6px; padding: 8px 12px; border: 1px dashed var(--site-rule); border-radius: 2px;
  font-size: 12.5px; color: var(--site-muted); background: var(--site-surface);
}
.ed-section-bar[data-section="tinted"] { border-color: var(--site-muted); }
.ed-section-bar[data-section="accent"] { border-color: var(--site-accent); color: var(--site-accent); }
.ed-section-bar__label { letter-spacing: .05em; text-transform: uppercase; font-size: 11px; }
.ed-section-bar__style { display: inline-flex; align-items: center; gap: 6px; }
.ed-section-bar[data-section="dark"] { border-color: #16201d; color: #16201d; }
.ed-section-bar[data-section="image"] { border-color: var(--site-accent); }
.ed-section-bar__image { display: none; flex-basis: 100%; }
.ed-section-bar[data-section="image"] .ed-section-bar__image { display: block; }
.ed-figure--small [data-img] { max-height: 90px; width: auto; }
.ed-variant { display: inline-flex; align-items: center; gap: 6px; font-size: 12.5px; color: var(--site-muted); margin: 0 0 10px; padding: 3px 8px; border: 1px dashed var(--site-rule); border-radius: 6px; background: var(--site-ground); }
.ed-variant select { font: inherit; font-size: 12.5px; padding: 2px 6px; border: 1px solid var(--site-rule); border-radius: 4px; background: var(--site-ground); color: var(--site-ink); }
.ed-item { position: relative; }
.ed-item__tools { position: absolute; top: 6px; right: 6px; z-index: 2; display: inline-flex; gap: 2px; padding: 2px; border-radius: 999px; border: 1px solid var(--site-rule); background: var(--site-ground); box-shadow: 0 2px 8px rgba(0,0,0,.12); opacity: 0; transition: opacity .15s; }
.ed-item:hover .ed-item__tools, .ed-item:focus-within .ed-item__tools { opacity: 1; }
.ed-item__tool { width: 22px; height: 22px; border-radius: 50%; border: 0; background: transparent; color: var(--site-muted); font-size: 12px; line-height: 1; cursor: pointer; }
.ed-item__tool:hover, .ed-item__tool:focus-visible { background: var(--site-surface); color: var(--site-ink); outline: none; }
.ed-item__tool[data-item-grab] { cursor: grab; touch-action: none; }
.ed-item--dragging { opacity: .55; outline: 2px dashed var(--site-accent); outline-offset: 2px; }
.ed-gallery__item .ed-item__tools { top: 4px; right: 4px; }
.ed-tool--select { width: auto; min-width: 104px; font: inherit; font-size: 11.5px; padding: 2px 4px; border: 0; background: transparent; color: inherit; cursor: pointer; display: inline-block; }
.ed-item.site-hours__row { display: grid; grid-column: 1 / -1; grid-template-columns: subgrid; position: relative; padding-right: 116px; }
.site-hours--inline .ed-item.site-hours__row { display: inline-flex; grid-column: auto; }
.ed-item.site-faq__item { padding-right: 116px; }
.ed-preset { display: grid; grid-template-columns: 1fr auto; gap: 4px; align-items: stretch; }
.ed-preset__add { text-align: left; }
.ed-preset__add strong { display: block; } .ed-preset__add span { font-size: 12px; color: var(--site-muted); }
.ed-preset__delete { border: 1px solid var(--site-rule); background: var(--site-ground); color: var(--site-muted); border-radius: 6px; width: 30px; cursor: pointer; }
.ed-preset__delete:hover { color: var(--site-accent); border-color: var(--site-accent); }
.ed-block:focus-visible { outline: 2px solid var(--site-accent); outline-offset: 4px; border-radius: 4px; }
.ed-block::before { content: attr(data-label); position: absolute; left: 0; top: -13px; z-index: 3; font-size: 10.5px; letter-spacing: .04em; text-transform: uppercase; padding: 2px 7px; background: var(--site-ground); border: 1px solid var(--site-rule); color: var(--site-muted); border-radius: 2px; pointer-events: none; display: none; }
.ed-block:hover::before, .ed-block:focus-within::before { display: block; }
.ed-palette-btn { display: inline-flex; align-items: center; gap: 10px; font: inherit; font-size: 13px; color: var(--site-muted); background: var(--site-surface); border: 1px solid var(--site-rule); border-radius: 6px; padding: 6px 10px; cursor: text; min-width: 220px; }
.ed-palette-btn:hover { border-color: var(--site-accent); color: var(--site-ink); }
.ed-palette-btn kbd, .ed-keys kbd, .ed-coach kbd { font: inherit; font-size: 11px; padding: 1px 6px; border: 1px solid var(--site-rule); border-bottom-width: 2px; border-radius: 4px; background: var(--site-ground); color: var(--site-muted); margin-left: auto; }
.ed-btn--icon { width: 34px; padding: 8px 0; text-align: center; font-weight: 700; }
.ed-palette, .ed-keys { position: fixed; inset: 0; z-index: 80; background: rgba(15, 23, 32, .45); display: flex; align-items: flex-start; justify-content: center; padding: 10vh 16px 16px; }
.ed-palette[hidden], .ed-keys[hidden] { display: none; }
.ed-palette__box, .ed-keys__box { width: min(640px, 100%); background: var(--site-ground); color: var(--site-ink); border-radius: 10px; box-shadow: 0 20px 60px rgba(0,0,0,.35); overflow: hidden; }
.ed-palette__input { width: 100%; box-sizing: border-box; font: inherit; font-size: 17px; padding: 16px 18px; border: 0; border-bottom: 1px solid var(--site-rule); background: transparent; color: inherit; outline: none; }
.ed-palette__list { list-style: none; margin: 0; padding: 6px; max-height: 52vh; overflow-y: auto; }
.ed-palette__item { display: grid; grid-template-columns: 84px 1fr auto; gap: 12px; align-items: baseline; padding: 9px 12px; border-radius: 6px; cursor: pointer; font-size: 14.5px; }
.ed-palette__item.is-active, .ed-palette__item:hover { background: var(--site-surface); }
.ed-palette__item.is-active { box-shadow: inset 3px 0 0 var(--site-accent); }
.ed-palette__group { font-size: 11px; letter-spacing: .05em; text-transform: uppercase; color: var(--site-muted); }
.ed-palette__label { font-weight: 500; }
.ed-palette__hint { font-size: 12.5px; color: var(--site-muted); text-align: right; }
.ed-palette__none { padding: 14px 12px; color: var(--site-muted); font-size: 14px; }
.ed-palette__hint:empty { display: none; }
p.ed-palette__hint { margin: 0; padding: 8px 18px 12px; font-size: 12px; color: var(--site-muted); border-top: 1px solid var(--site-rule); text-align: left; }
.ed-keys__box { padding: 20px 22px; }
.ed-keys__box h2 { margin: 0 0 14px; font-size: 17px; }
.ed-keys__row { display: grid; grid-template-columns: 190px 1fr; gap: 12px; padding: 7px 0; border-top: 1px solid var(--site-rule); font-size: 14px; align-items: center; }
.ed-keys__keys kbd { margin: 0 2px 0 0; }
.ed-keys__close { margin-top: 16px; font: inherit; padding: 8px 16px; border-radius: 6px; border: 1px solid var(--site-rule); background: var(--site-surface); color: inherit; cursor: pointer; }
.ed-coach { display: flex; flex-wrap: wrap; align-items: center; gap: 10px 22px; margin: 0 auto 16px; max-width: 1120px; padding: 10px 16px; border: 1px solid var(--site-rule); border-radius: 8px; background: var(--site-ground); font-size: 13.5px; color: var(--site-muted); }
.ed-coach[hidden] { display: none; }
.ed-coach__tip b { color: var(--site-accent); font-weight: 700; margin-right: 2px; }
.ed-coach__dismiss { margin-left: auto; font: inherit; font-size: 13px; padding: 5px 12px; border-radius: 6px; border: 1px solid var(--site-rule); background: var(--site-surface); color: var(--site-ink); cursor: pointer; }
.ed-empty { text-align: center; padding: 48px 20px; border: 2px dashed var(--site-rule); border-radius: 10px; margin: 24px 0; color: var(--site-muted); }
.ed-empty[hidden] { display: none; }
.ed-empty p { margin: 0 0 14px; font-size: 18px; color: var(--site-ink); }
.ed-empty .ed-hint { display: block; margin-top: 10px; }
.ed-find { width: 100%; box-sizing: border-box; font: inherit; font-size: 14px; padding: 8px 10px; margin: 0 0 10px; border: 1px solid var(--site-rule); border-radius: 6px; background: var(--site-ground); color: inherit; }
.ed-find:focus-visible { outline: 2px solid var(--site-accent); outline-offset: 1px; }
.ed-find__none { margin: -4px 0 10px; }
.ed-side [data-add][hidden] { display: none; }
.ed-side__close { display: none; }
.ed-fab { display: none; }
.ed-toast { display: flex; align-items: center; gap: 14px; }
.ed-toast[hidden] { display: none; }
.ed-toast__action { font: inherit; font-weight: 700; color: #fff; background: transparent; border: 1px solid rgba(255,255,255,.5); border-radius: 6px; padding: 4px 10px; cursor: pointer; }
.ed-toast__action:hover { background: rgba(255,255,255,.12); }
.admin-panel--key { border: 2px solid var(--site-accent); }
.admin-secret--wide { width: 100%; box-sizing: border-box; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 14px; padding: 8px 10px; }
.admin-list { padding-left: 20px; font-size: 15px; line-height: 1.5; }
.admin-list li { margin: 0 0 6px; }
.admin-label { display: block; font-weight: 600; margin: 0 0 6px; }
.admin-muted { color: var(--site-muted); font-size: 13px; }
.ed-item-add { display: inline-block; margin: 10px 0 0; padding: 6px 12px; border: 1px dashed var(--site-accent); border-radius: 999px; background: transparent; color: var(--site-accent); font: inherit; font-size: 13px; cursor: pointer; }
.ed-item-add:hover { background: color-mix(in srgb, var(--site-accent) 10%, transparent); }
.ed-link-field { display: inline-flex; flex-direction: column; gap: 4px; vertical-align: top; margin-right: 8px; }
.ed-link-field .ed-inline-input { font-size: 11.5px; min-width: 160px; }
.ed-image-field { display: flex; flex-direction: column; gap: 6px; margin: 0; }
.ed-image-field [data-img] { max-height: 320px; object-fit: cover; }
.ed-image-field__controls { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.ed-image-field__controls .ed-inline-input { flex: 1 1 140px; font-size: 11.5px; }
.ed-flag { display: inline-flex; align-items: center; gap: 4px; font-size: 12.5px; color: var(--site-muted); margin-top: 8px; }
.ed-block [data-field]:empty::before { content: attr(data-placeholder); color: var(--site-muted); opacity: .6; }
.ed-spacer__label { display: block; text-align: center; font-size: 11px; letter-spacing: .1em; text-transform: uppercase; color: var(--site-muted); border: 1px dashed var(--site-rule); border-radius: 4px; padding: 4px; }
.ed-block .site-spacer { height: auto; }
.ed-side__sub { margin: 18px 0 8px; font-size: 11px; letter-spacing: .08em; text-transform: uppercase; color: var(--site-muted); }
.ed-section-bar__style select { font: inherit; font-size: 12.5px; padding: 3px 6px; border: 1px solid var(--site-rule); border-radius: 2px; background: var(--site-ground); color: var(--site-ink); }
.ed-bubble {
  position: absolute; z-index: 30; display: flex; align-items: center; gap: 2px;
  padding: 3px; border: 1px solid var(--site-rule); border-radius: 3px; background: var(--site-ground);
  box-shadow: 0 4px 14px rgba(0,0,0,.14);
}
.ed-bubble[hidden] { display: none; }
.ed-bubble button { font: inherit; font-size: 13px; padding: 4px 9px; border: 0; background: transparent; color: var(--site-ink); border-radius: 2px; cursor: pointer; }
.ed-bubble button:hover { background: var(--site-surface); }
.ed-bubble__link { display: inline-flex; gap: 4px; align-items: center; padding-left: 4px; border-left: 1px solid var(--site-rule); }
.ed-bubble__link[hidden] { display: none; }
.ed-bubble__link input { font: inherit; font-size: 12.5px; width: 200px; padding: 4px 7px; border: 1px solid var(--site-rule); border-radius: 2px; background: var(--site-ground); color: var(--site-ink); }
.ed-inline-input {
  padding: 5px 8px; font: inherit; font-size: 12.5px; border-radius: 2px;
  border: 1px dashed var(--site-rule); background: transparent; color: var(--site-muted); min-width: 180px;
}
.ed-inline-input:focus-visible { outline: 2px solid var(--site-accent); outline-offset: 1px; color: var(--site-ink); }
.ed-figure { display: flex; flex-direction: column; gap: 7px; margin: 20px 0; }
.ed-post-meta { cursor: default; }
.ed-checks { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.ed-check { font-size: 12.5px; line-height: 1.4; padding: 8px 10px 8px 26px; position: relative; background: #fff7e6; border: 1px solid #f0d9a8; border-radius: 3px; color: #5a4200; }
.ed-check::before { content: "!"; position: absolute; left: 9px; top: 7px; font-weight: 700; }
.ed-check--ok { background: #eef7f0; border-color: #cfe6d4; color: #1f5a30; }
.ed-check--ok::before { content: "✓"; }
.site-preview-banner { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; gap: 10px 20px; padding: 10px 20px; background: #1f2933; color: #fff; font-size: 14px; }
.site-preview-banner a { color: #fff; }
.site-preview-banner__actions { display: flex; gap: 14px; align-items: center; }
.site-preview-banner .admin-row-btn { background: #fff; color: #1f2933; border-color: #fff; }
.site-preview-banner__sendback { display: inline-flex; gap: 6px; }
.site-preview-banner__sendback input { font: inherit; font-size: 13px; padding: 4px 8px; border: 1px solid #fff; border-radius: 2px; min-width: 180px; }
.ed-share__result { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; margin-top: 8px; }
.ed-share__result .ed-inline-input { flex: 1 1 160px; min-width: 0; }
.ed-share__result small { flex-basis: 100%; color: var(--site-muted); font-size: 11.5px; }
.ed-post-meta a { pointer-events: none; }
.ed-field input[type="datetime-local"] { width: 100%; }
.ed-video { display: flex; flex-direction: column; gap: 7px; margin: 20px 0; }
.ed-video .site-video { margin: 0; }
.ed-video .site-video iframe { pointer-events: none; }
.ed-columns .site-columns__col { min-height: 2.5em; padding: 6px; border: 1px dashed transparent; border-radius: 2px; }
.ed-columns .site-columns__col:hover, .ed-columns .site-columns__col:focus { border-color: var(--site-rule); }
.ed-gallery { display: flex; flex-direction: column; gap: 8px; margin: 20px 0; }
.ed-gallery__grid { margin: 0; min-height: 40px; }
.ed-gallery__item { position: relative; display: flex; flex-direction: column; gap: 4px; }
.ed-gallery__item img { aspect-ratio: 4 / 3; }
.ed-gallery__remove { position: absolute; top: 6px; right: 6px; background: var(--site-ground); border: 1px solid var(--site-rule); }
.ed-gallery__item .ed-inline-input { min-width: 0; font-size: 11.5px; }
.ed-gallery__controls { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.ed-upload {
  display: inline-flex; align-items: center; gap: 8px; align-self: flex-start; cursor: pointer;
  padding: 6px 12px; border: 1px solid var(--site-accent); color: var(--site-accent);
  border-radius: 2px; font-size: 13px; background: transparent;
}
.ed-upload input { position: absolute; width: 1px; height: 1px; opacity: 0; pointer-events: none; }
.ed-upload:has(input:focus-visible) { outline: 2px solid var(--site-accent); outline-offset: 2px; }
.ed-upload[data-busy="true"] { opacity: .6; cursor: progress; }
.ed-library-btn {
  align-self: flex-start; cursor: pointer; padding: 6px 12px; border-radius: 2px; font: inherit; font-size: 13px;
  border: 1px solid var(--site-rule); background: transparent; color: var(--site-ink);
}
.ed-library-btn:hover { border-color: var(--site-accent); color: var(--site-accent); }
.ed-picker { display: grid; grid-template-columns: repeat(4, 84px); gap: 6px; padding: 8px; max-height: 320px; overflow-y: auto; }
.ed-picker__hint { grid-column: 1 / -1; margin: 6px; font-size: 13px; color: var(--site-muted); }
.ed-picker__item { padding: 0; border: 1px solid var(--site-rule); background: var(--site-ground); cursor: pointer; border-radius: 2px; overflow: hidden; aspect-ratio: 1; }
.ed-picker__item img { width: 100%; height: 100%; object-fit: cover; display: block; }
.ed-picker__item:hover { border-color: var(--site-accent); }
.admin-media-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 14px; max-width: 1000px; }
.admin-media { border: 1px solid var(--site-rule); background: var(--site-ground); padding: 10px; display: flex; flex-direction: column; gap: 8px; }
.admin-media__thumb { width: 100%; aspect-ratio: 4 / 3; object-fit: cover; display: block; background: var(--site-surface); }
.admin-media__meta { display: flex; flex-direction: column; gap: 3px; font-size: 12.5px; color: var(--site-muted); }
.admin-media__usage { color: var(--site-ink); }
.admin-media__link { font: inherit; font-size: 12px; padding: 5px 7px; border: 1px solid var(--site-rule); background: var(--site-surface); color: var(--site-ink); border-radius: 2px; }
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
  .ed-bar { flex-wrap: nowrap; overflow-x: auto; gap: 8px; padding: 8px 12px; scrollbar-width: none; }
  .ed-bar::-webkit-scrollbar { display: none; }
  .ed-bar__left, .ed-bar__right { flex: 0 0 auto; gap: 8px; }
  .ed-palette-btn { min-width: 0; padding: 6px 9px; }
  .ed-palette-btn kbd { display: none; }
  .ed-page-name { max-width: 110px; }
  .ed-save { display: none; }
  .ed-body { flex-direction: column; }
  .ed-side { position: fixed; left: 0; right: 0; bottom: 0; z-index: 70; max-height: 74vh; border-right: 0; border-top: 1px solid var(--site-rule); border-radius: 14px 14px 0 0; box-shadow: 0 -8px 30px rgba(0,0,0,.2); transform: translateY(105%); transition: transform .22s ease-out; padding-top: 40px; }
  .ed-side.is-open { transform: none; }
  .ed-side__close { display: grid; place-items: center; position: absolute; top: 8px; right: 10px; width: 32px; height: 32px; border-radius: 50%; border: 1px solid var(--site-rule); background: var(--site-ground); font-size: 14px; cursor: pointer; }
  .ed-fab { display: inline-flex; align-items: center; position: fixed; right: 16px; bottom: 16px; z-index: 69; font: inherit; font-size: 15px; font-weight: 600; padding: 12px 18px; border-radius: 999px; border: 0; background: var(--site-accent); color: var(--site-ground); box-shadow: 0 6px 20px rgba(0,0,0,.25); cursor: pointer; }
  .ed-stage { padding: 14px 12px 110px; }
  .ed-coach { font-size: 12.5px; gap: 6px 14px; }
  .ed-keys__row { grid-template-columns: 1fr; gap: 2px; }
  .ed-palette__item { grid-template-columns: 1fr; gap: 2px; }
  .ed-palette__hint { text-align: left; }
  .ed-palette, .ed-keys { padding: 4vh 8px 8px; }
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
