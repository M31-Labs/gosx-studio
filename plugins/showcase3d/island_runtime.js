(function () {
  // showcase3d-runtime — the plugin's client island, contributed through the
  // gosx-studio Plugin v2 island channel and bundled into the engine script
  // exactly like the built-in runtimes. This is intentionally minimal: it proves
  // a plugin can ship browser behavior. The pop-out viewer wiring lands with the
  // plugin render channel.
  if (window.__gosxShowcase3DRuntimeInstalled) {
    return;
  }
  window.__gosxShowcase3DRuntimeInstalled = true;
  window.GoSXShowcase3DRuntime = window.GoSXShowcase3DRuntime || {
    install: function install() {
      // Placeholder install hook. Real placement/pop-out wiring is added with
      // the plugin render channel (Plugin v2, render milestone).
    },
  };
})();
