(function () {
  "use strict";
  // Published/preview-page runtime for the reveal-on-scroll interaction
  // primitive (core.InteractionRevealOnScroll). hover-focus-state needs no
  // JS at all — it is pure CSS keyed on the same data-gosx-interaction*
  // attributes (see hostruntime/assets/studio.css) so that :hover and
  // :focus-visible always share one rule and a mouse-only affordance can
  // never exist.
  //
  // Contract (matches core.Interaction.RenderAttrs()):
  //   data-gosx-interaction="reveal-on-scroll"
  //   data-gosx-interaction-effect="fade|fade-up|fade-left|fade-right"
  //   data-gosx-interaction-duration-ms="150|250|400|600"
  //   data-gosx-interaction-delay-ms="0|100|200|300"
  //   data-gosx-interaction-once="true|false"
  //
  // This runtime only ever toggles one attribute —
  // data-gosx-interaction-revealed — and only on elements matched by that
  // fixed selector. It never evaluates author content, never accepts an
  // arbitrary selector, and never touches display/visibility (only opacity
  // and transform do the animating, in CSS), so a focusable descendant is
  // never removed from the tab order or the accessibility tree by this
  // runtime or by the CSS it drives.
  var SELECTOR = "[data-gosx-interaction='reveal-on-scroll']";

  function reducedMotion() {
    return !!(window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches);
  }

  function reveal(el) {
    el.setAttribute("data-gosx-interaction-revealed", "true");
  }

  function unreveal(el) {
    el.setAttribute("data-gosx-interaction-revealed", "false");
  }

  function isOnce(el) {
    return el.getAttribute("data-gosx-interaction-once") !== "false";
  }

  function bind(root) {
    var elements = (root || document).querySelectorAll(SELECTOR);
    if (!elements.length) return;
    // Reduced motion and "no IntersectionObserver" both fail open to fully
    // visible content — never fail closed to hidden. A user who cannot run
    // this script (or who asked for less motion) must never see less
    // content or lose keyboard access to it.
    if (reducedMotion() || typeof IntersectionObserver !== "function") {
      Array.prototype.forEach.call(elements, reveal);
      return;
    }
    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          var el = entry.target;
          if (entry.isIntersecting) {
            reveal(el);
            if (isOnce(el)) observer.unobserve(el);
          } else if (!isOnce(el)) {
            unreveal(el);
          }
        });
      },
      { root: null, rootMargin: "0px 0px -10% 0px", threshold: 0.15 }
    );
    Array.prototype.forEach.call(elements, function (el) {
      if (el.__gosxInteractionObserved) return;
      el.__gosxInteractionObserved = true;
      observer.observe(el);
    });
  }

  window.GoSXStudioInteractionRuntime = { bind: bind, reveal: reveal };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () { bind(document); }, { once: true });
  } else {
    bind(document);
  }
})();
