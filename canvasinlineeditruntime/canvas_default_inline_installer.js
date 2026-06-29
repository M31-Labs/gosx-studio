;(function () {
  "use strict";

  if (typeof window === "undefined") return;
  if (typeof document === "undefined") return;

  var MARKER_SEL = "[data-studio-site-map-canvas-default=\"true\"]";
  var HTML_SEL = "[data-gosx-canvas-html]";
  var INSTALLED_FLAG = "__gosxStudioCanvasDefaultInlineEditInstalled";
  var TIMER_LIMIT = 120;

  function installHost(host, api) {
    if (!host || !api || typeof api.install !== "function") return;
    var layers = [];
    if (host.__gosxBoardHTMLLayer) layers.push(host.__gosxBoardHTMLLayer);
    if (typeof host.querySelectorAll === "function") {
      var mounts = host.querySelectorAll(HTML_SEL);
      for (var i = 0; i < mounts.length; i++) {
        var layer = mounts[i] && mounts[i].parentElement;
        if (layer) layers.push(layer);
      }
    }
    for (var j = 0; j < layers.length; j++) {
      var layer = layers[j];
      if (!layer || layer[INSTALLED_FLAG]) continue;
      var installed = api.install(layer);
      if (installed === false) continue;
      layer[INSTALLED_FLAG] = true;
    }
  }

  function install() {
    var api = window.GoSXStudioCanvasInlineEditRuntime;
    if (!api || typeof api.install !== "function") return;
    if (typeof document === "undefined" || typeof document.querySelectorAll !== "function") return;
    var markers = document.querySelectorAll(MARKER_SEL);
    for (var i = 0; i < markers.length; i++) {
      var marker = markers[i];
      var host = marker && marker.parentElement;
      installHost(host, api);
    }
  }

  function start() {
    install();
    var tries = 0;
    var timer = setInterval(function () {
      install();
      tries += 1;
      if (tries > TIMER_LIMIT) clearInterval(timer);
    }, 250);
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", start);
  else start();

  var api = { install: install, installHost: installHost };
  window.GoSXStudioCanvasDefaultInlineInstallerRuntime = api;
})();
