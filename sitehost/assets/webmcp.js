/* WebMCP: the editor and the admin announce their tools to the browser, so
   a browser with a built-in assistant can work the site for its owner. The
   tools call the site's own agent API as the signed-in person (cookie plus
   the CSRF token), so they can do exactly what the person can do and every
   change is a draft in History. Browsers without navigator.modelContext
   simply see nothing. */
(function () {
  "use strict";
  var csrfMeta = document.querySelector('meta[name="csrf-token"]');
  var CSRF = csrfMeta ? csrfMeta.getAttribute("content") : "";
  var editor = document.querySelector("[data-editor]");
  var kind = editor ? (editor.getAttribute("data-kind") || "page") : "";
  var docId = editor ? editor.getAttribute("data-page-id") : "";
  var collection = kind === "post" ? "posts" : "pages";

  function api(method, path, body) {
    var init = { method: method, credentials: "same-origin", headers: { "X-CSRF-Token": CSRF, Accept: "application/json, text/markdown" } };
    /* Name this tab, so the editor does not announce its own change as someone else's. */
    if (window.gosxEditor && window.gosxEditor.clientId) init.headers["X-Editor-Client"] = window.gosxEditor.clientId;
    if (body !== undefined) {
      init.headers["Content-Type"] = "application/json";
      init.body = JSON.stringify(body);
    }
    return fetch(path, init).then(function (r) {
      return r.text().then(function (text) {
        var out;
        try { out = JSON.parse(text); } catch (e) { out = { text: text }; }
        if (!r.ok) {
          var message = (out && out.error && out.error.message) || text || ("The site answered " + r.status);
          throw new Error(message);
        }
        return out;
      });
    });
  }

  /* Settle the editor's own pending save before writing from a tool, so
     the two never race. */
  function beforeWrite() {
    if (window.gosxEditor && typeof window.gosxEditor.flush === "function") return window.gosxEditor.flush();
    return Promise.resolve();
  }

  function refreshed(result) {
    if (window.gosxEditor && typeof window.gosxEditor.refresh === "function") window.gosxEditor.refresh();
    return result;
  }

  function str(description) { return { type: "string", description: description }; }
  function bool(description) { return { type: "boolean", description: description }; }
  function obj(required, props) {
    var schema = { type: "object", properties: props };
    if (required && required.length) schema.required = required;
    return schema;
  }
  var block = { type: "object", description: "One block: {kind, ...}. Simple blocks use text/url/alt; ready-made sections use fields, items, variant. See get_schema.", required: ["kind"], properties: { kind: str("Block kind, e.g. paragraph, heading, hero, pricing, faq") }, additionalProperties: true };
  var blocks = { type: "array", items: block };

  var tools = [];
  if (editor && docId) {
    var base = "/agent/v1/" + collection + "/" + encodeURIComponent(docId);
    tools.push(
      { name: "describe_this_page", description: "The " + kind + " being edited: its title, address, status, and every block in editable form.", inputSchema: obj([], {}),
        execute: function () { return api("GET", base); } },
      { name: "add_section", description: "Add a block or ready-made section to the end of this " + kind + ". Give kind plus fields/items (sections) or text/url (simple blocks). Call get_schema for field names.", inputSchema: block,
        execute: function (args) { return beforeWrite().then(function () { return api("PATCH", base, { append: [args] }); }).then(refreshed); } },
      { name: "edit_this_page", description: "Change this " + kind + ": title, description, or blocks by index (replace, remove, insert, append, move). Indexes refer to the page before the call; describe_this_page shows them.",
        inputSchema: obj([], { title: str("New title"), description: str("One line for search results"), replace: { type: "array", items: obj(["index", "block"], { index: { type: "integer" }, block: block }) }, remove: { type: "array", items: { type: "integer" } }, insert: { type: "array", items: obj(["index", "block"], { index: { type: "integer", description: "Insert before this index" }, block: block }) }, append: blocks, move: obj(["from", "to"], { from: { type: "integer" }, to: { type: "integer" } }) }),
        execute: function (args) { return beforeWrite().then(function () { return api("PATCH", base, args); }).then(refreshed); } },
      { name: "replace_this_page", description: "Replace every block on this " + kind + ". Prefer edit_this_page for small changes.", inputSchema: obj(["blocks"], { blocks: blocks, title: str("") }),
        execute: function (args) { return beforeWrite().then(function () { return api("PUT", base, args); }).then(refreshed); } },
      { name: "publish_this_page", description: "Make the saved draft of this " + kind + " live for visitors.", inputSchema: obj([], {}),
        execute: function () { return beforeWrite().then(function () { return api("POST", base + "/publish"); }).then(refreshed); } },
      { name: "undo_last_change", description: "Undo the last change made in this editor session.", inputSchema: obj([], {}),
        execute: function () { if (window.gosxEditor && window.gosxEditor.undo) window.gosxEditor.undo(); return Promise.resolve({ ok: true }); } }
    );
    if (kind === "page") {
      tools.push({ name: "read_this_page_as_text", description: "This page as Markdown, the way a visitor reads it.", inputSchema: obj([], {}),
        execute: function () { return api("GET", base + "/markdown"); } });
    }
  }
  tools.push(
    { name: "get_site", description: "The whole site at a glance: name, tagline, contact, header, footer, Look, every page with its status, and the menu.", inputSchema: obj([], {}),
      execute: function () { return api("GET", "/agent/v1/site"); } },
    { name: "get_schema", description: "Every block kind with its fields, items, variants, and choices, plus page templates and Look options. Read it before composing blocks.", inputSchema: obj([], {}),
      execute: function () { return api("GET", "/agent/v1/schema"); } },
    { name: "list_pages", description: "Every page with id, address, and status.", inputSchema: obj([], {}),
      execute: function () { return api("GET", "/agent/v1/pages"); } },
    { name: "create_page", description: "Create a page from a template key or from blocks. A draft unless publish is true.", inputSchema: obj(["title"], { title: str(""), slug: str("Address like about-us"), description: str(""), template: str("Page template key from get_schema"), blocks: blocks, publish: bool("Publish at once") }),
      execute: function (args) { return api("POST", "/agent/v1/pages", args); } },
    { name: "edit_page", description: "Change another page by id or address: title, description, or blocks by index.", inputSchema: obj(["page"], { page: str("Page id or address"), title: str(""), description: str(""), replace: { type: "array", items: obj(["index", "block"], { index: { type: "integer" }, block: block }) }, remove: { type: "array", items: { type: "integer" } }, insert: { type: "array", items: obj(["index", "block"], { index: { type: "integer" }, block: block }) }, append: blocks, publish: bool("") }),
      execute: function (args) { var page = args.page; var body = {}; Object.keys(args).forEach(function (k) { if (k !== "page") body[k] = args[k]; }); return api("PATCH", "/agent/v1/pages/" + encodeURIComponent(page), body); } },
    { name: "publish_page", description: "Publish a page by id or address.", inputSchema: obj(["page"], { page: str("Page id or address") }),
      execute: function (args) { return api("POST", "/agent/v1/pages/" + encodeURIComponent(args.page) + "/publish"); } },
    { name: "open_page_in_editor", description: "Open a page in the editor by id, address, or title.", inputSchema: obj(["page"], { page: str("Page id, address, or title") }),
      execute: function (args) {
        return api("GET", "/agent/v1/pages").then(function (out) {
          var want = String(args.page || "").toLowerCase().replace(/^\//, "");
          var hit = (out.pages || []).filter(function (p) { return p.id === args.page || p.slug === want || String(p.title || "").toLowerCase() === want; })[0];
          if (!hit) throw new Error("No page matches " + args.page);
          window.location.href = "/admin/edit/" + encodeURIComponent(hit.id);
          return { ok: true, opened: hit.title };
        });
      } },
    { name: "update_site", description: "Change the site's title, tagline, description, contact details, header (announcement, sticky, menu button), footer (menu, links), or social links. Send only what changes.",
      inputSchema: obj([], { title: str(""), tagline: str(""), description: str(""), contact: obj([], { email: str(""), phone: str(""), location: str("") }), header: obj([], { announceText: str(""), announceLink: str(""), announceOn: bool(""), sticky: bool(""), menuButton: str(""), menuButtonTo: str("") }), footer: obj([], { menu: bool(""), links: { type: "array", items: { type: "array", items: { type: "string" } } } }), social: { type: "object", additionalProperties: { type: "string" } } }),
      execute: function (args) { return api("PATCH", "/agent/v1/site", args).then(refreshed); } },
    { name: "set_look", description: "Change the Look: palette, fonts, accent (hex), buttons, spacing, headings, width; custom palette with ground and ink; custom fonts with fontHead and fontBody.",
      inputSchema: obj([], { palette: str(""), fonts: str(""), accent: str("Hex colour"), buttons: str(""), spacing: str(""), headings: str(""), width: str(""), ground: str(""), ink: str(""), fontHead: str(""), fontBody: str("") }),
      execute: function (args) { return api("PUT", "/agent/v1/look", args).then(function (out) { if (editor) window.location.reload(); return out; }); } },
    { name: "list_messages", description: "Messages visitors sent through the site's forms, newest first.", inputSchema: obj([], { limit: { type: "integer" } }),
      execute: function (args) { return api("GET", "/agent/v1/messages" + (args && args.limit ? "?limit=" + encodeURIComponent(args.limit) : "")); } },
    { name: "get_stats", description: "Visitor counts by day, page, source, and device.", inputSchema: obj([], { days: { type: "integer" } }),
      execute: function (args) { return api("GET", "/agent/v1/stats" + (args && args.days ? "?days=" + encodeURIComponent(args.days) : "")); } }
  );

  function wrap(tool) {
    return {
      name: tool.name,
      description: tool.description,
      inputSchema: tool.inputSchema,
      execute: function (args) {
        return Promise.resolve(tool.execute(args || {})).then(function (result) {
          return { content: [{ type: "text", text: typeof result === "string" ? result : JSON.stringify(result) }] };
        }, function (err) {
          return { content: [{ type: "text", text: String(err && err.message || err) }], isError: true };
        });
      },
    };
  }

  var registered = tools.map(wrap);
  window.gosxWebMCP = {
    tools: registered.map(function (t) { return t.name; }),
    describe: registered.map(function (t) { return { name: t.name, description: t.description, inputSchema: t.inputSchema }; }),
    call: function (name, args) {
      var tool = registered.filter(function (t) { return t.name === name; })[0];
      if (!tool) return Promise.reject(new Error("unknown tool " + name));
      return tool.execute(args || {});
    },
  };

  var mc = navigator.modelContext;
  if (mc) {
    try {
      if (typeof mc.registerTool === "function") registered.forEach(function (t) { mc.registerTool(t); });
      else if (typeof mc.provideContext === "function") mc.provideContext({ tools: registered });
      window.gosxWebMCP.announced = true;
    } catch (e) { window.gosxWebMCP.announced = false; }
  }

  var list = document.querySelector("[data-webmcp-tools]");
  if (list) {
    list.innerHTML = "";
    registered.forEach(function (t) {
      var li = document.createElement("li");
      var code = document.createElement("code");
      code.textContent = t.name;
      li.appendChild(code);
      li.appendChild(document.createTextNode(" — " + t.description));
      list.appendChild(li);
    });
    var note = document.createElement("li");
    note.className = "admin-muted";
    note.textContent = mc ? "Your browser supports WebMCP: these tools are announced on every admin page and in the editor." : "This browser has no built-in assistant (no navigator.modelContext), so the tools are listed here only.";
    list.appendChild(note);
  }
})();
