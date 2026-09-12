/* Cookie consent for third-party code.
 *
 * The owner's pasted code sits in a <template> and is only moved into the
 * document once the visitor accepts. Declining stores that choice too, so
 * the banner does not reappear on every page. Nothing here sets a cookie
 * itself; the choice lives in localStorage.
 */
(function () {
  "use strict";
  var KEY = "gosx-site-consent";
  var template = document.getElementById("site-head-code");
  var banner = document.querySelector("[data-consent]");
  if (!template) return;

  function choice() {
    try { return localStorage.getItem(KEY) || ""; } catch (e) { return ""; }
  }
  function remember(value) {
    try { localStorage.setItem(KEY, value); } catch (e) {}
  }

  /* innerHTML never runs scripts; each one is rebuilt so the browser does. */
  function inject() {
    var nodes = template.content.cloneNode(true).childNodes;
    Array.prototype.slice.call(nodes).forEach(function (node) {
      if (node.nodeType !== 1) return;
      if (node.tagName === "SCRIPT") {
        var script = document.createElement("script");
        for (var i = 0; i < node.attributes.length; i++) {
          script.setAttribute(node.attributes[i].name, node.attributes[i].value);
        }
        script.text = node.textContent;
        document.head.appendChild(script);
      } else {
        document.head.appendChild(node.cloneNode(true));
      }
    });
  }

  var current = choice();
  if (current === "yes") { inject(); return; }
  if (current === "no" || !banner) return;

  banner.hidden = false;
  var accept = banner.querySelector("[data-consent-accept]");
  var decline = banner.querySelector("[data-consent-decline]");
  if (accept) accept.addEventListener("click", function () { remember("yes"); banner.hidden = true; inject(); });
  if (decline) decline.addEventListener("click", function () { remember("no"); banner.hidden = true; });
})();
