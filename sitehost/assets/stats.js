/* GoSX site visitor counter.
 *
 * One request per page view, carrying the path, the referring site, and the
 * screen width. No cookie, no identifier, nothing stored in the browser. The
 * server counts a visitor once per day from a hash it throws away at
 * midnight, so nobody can be followed from one day to the next.
 */
(function () {
  if (navigator.webdriver) return;
  var payload = JSON.stringify({
    p: location.pathname,
    r: document.referrer || "",
    w: (window.screen && window.screen.width) || 0
  });
  try {
    if (navigator.sendBeacon && navigator.sendBeacon("/stats/hit", new Blob([payload], { type: "text/plain" }))) return;
  } catch (e) {}
  try {
    fetch("/stats/hit", { method: "POST", body: payload, keepalive: true, credentials: "omit", headers: { "Content-Type": "text/plain" } });
  } catch (e) {}
})();
