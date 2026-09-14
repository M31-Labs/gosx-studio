/* Backdrops: generative canvas effects behind a section or a hero, drawn
   from the site's own palette and a seed, so every site's default is its
   own and every owner can shuffle, calm, or switch it off.

   Rules the engine keeps:
   - No dependencies, one canvas per host, one animation loop for all.
   - Colours come from the CSS custom properties around the host, so a
     change of Look changes every backdrop.
   - A seed (data-fx-seed) makes the picture reproducible; "shuffle" is a
     new seed. Everything random goes through the seeded generator.
   - Motion is layered: the owner's site-wide choice (data-site-motion:
     full, calm, off), the section's own (still, slow, normal), and the
     visitor's prefers-reduced-motion. Off, still, or reduced draws one
     frame and stops. Calm halves the speed and the frame rate.
   - Hosts off screen or in a hidden tab do not draw. Canvas resolution is
     capped so a big monitor never costs more than it should. */
(function () {
  "use strict";
  var MAX_DPR = 1.5;
  var MAX_WIDTH = 1800;
  var reduced = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  var finePointer = window.matchMedia && window.matchMedia("(hover: hover) and (pointer: fine)").matches;

  /* ---------- seeded randomness ---------- */

  function hash(str) {
    var h = 2166136261 >>> 0;
    for (var i = 0; i < str.length; i++) { h ^= str.charCodeAt(i); h = Math.imul(h, 16777619) >>> 0; }
    return h >>> 0;
  }
  function rng(seed) {
    var a = seed >>> 0;
    return function () {
      a = (a + 0x6D2B79F5) >>> 0;
      var t = a;
      t = Math.imul(t ^ (t >>> 15), t | 1);
      t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }

  /* ---------- colour ---------- */

  var probe = document.createElement("canvas").getContext("2d");
  function parseColor(value, fallback) {
    value = (value || "").trim();
    if (!value) return fallback;
    try {
      probe.fillStyle = "#000";
      probe.fillStyle = value;
      var out = probe.fillStyle;
      var m;
      if ((m = /^#([0-9a-f]{6})$/i.exec(out))) return [parseInt(m[1].slice(0, 2), 16), parseInt(m[1].slice(2, 4), 16), parseInt(m[1].slice(4, 6), 16)];
      if ((m = /^rgba?\(([^)]+)\)/.exec(out))) { var p = m[1].split(",").map(parseFloat); return [p[0], p[1], p[2]]; }
    } catch (e) {}
    return fallback;
  }
  function rgbToHsl(c) {
    var r = c[0] / 255, g = c[1] / 255, b = c[2] / 255;
    var max = Math.max(r, g, b), min = Math.min(r, g, b), h = 0, s = 0, l = (max + min) / 2;
    if (max !== min) {
      var d = max - min;
      s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
      if (max === r) h = (g - b) / d + (g < b ? 6 : 0); else if (max === g) h = (b - r) / d + 2; else h = (r - g) / d + 4;
      h /= 6;
    }
    return [h * 360, s, l];
  }
  function hsla(h, s, l, a) { return "hsla(" + (((h % 360) + 360) % 360).toFixed(1) + "," + (s * 100).toFixed(1) + "%," + (l * 100).toFixed(1) + "%," + a + ")"; }
  function luminance(c) { return (0.2126 * c[0] + 0.7152 * c[1] + 0.0722 * c[2]) / 255; }

  function paletteFor(host) {
    var cs = getComputedStyle(host);
    var accent = parseColor(cs.getPropertyValue("--site-accent"), [46, 111, 92]);
    var ground = parseColor(cs.getPropertyValue("--site-ground"), [255, 250, 243]);
    var ink = parseColor(cs.getPropertyValue("--site-ink"), [28, 30, 33]);
    var a = rgbToHsl(accent), g = rgbToHsl(ground), k = rgbToHsl(ink);
    var dark = luminance(ground) < 0.45;
    return { accent: a, ground: g, ink: k, dark: dark, accentRGB: accent, groundRGB: ground, inkRGB: ink };
  }

  /* ---------- geometry ---------- */

  function project(x, y, z, s) {
    // A camera at the origin looking down +z; f is the focal length.
    var f = s.w * 0.9;
    var k = f / (z * s.w * 0.5 + f);
    return { x: s.w / 2 + x * s.w * 0.5 * k, y: s.h / 2 + y * s.h * 0.5 * k, k: k };
  }

  /* value noise on a grid, cheap and smooth enough for contours */
  function noise2(seed) {
    var r = rng(seed);
    var size = 64, grid = new Float32Array(size * size);
    for (var i = 0; i < grid.length; i++) grid[i] = r();
    function at(ix, iy) { return grid[((iy % size + size) % size) * size + ((ix % size + size) % size)]; }
    function fade(t) { return t * t * (3 - 2 * t); }
    return function (x, y) {
      var x0 = Math.floor(x), y0 = Math.floor(y), fx = fade(x - x0), fy = fade(y - y0);
      var a = at(x0, y0), b = at(x0 + 1, y0), c = at(x0, y0 + 1), d = at(x0 + 1, y0 + 1);
      return (a + (b - a) * fx) * (1 - fy) + (c + (d - c) * fx) * fy;
    };
  }

  /* ---------- the effects ---------- */

  var effects = {};

  effects.aurora = {
    init: function (s) {
      var n = Math.round(5 * s.amount);
      s.blobs = [];
      for (var i = 0; i < n; i++) {
        s.blobs.push({ ax: 0.25 + s.r() * 0.5, ay: 0.25 + s.r() * 0.5, rx: 0.18 + s.r() * 0.3, ry: 0.18 + s.r() * 0.3, f1: 0.05 + s.r() * 0.08, f2: 0.04 + s.r() * 0.08, p1: s.r() * 6.28, p2: s.r() * 6.28, hue: (s.r() - 0.5) * 90 + (i % 2 ? 30 : -30), size: 0.45 + s.r() * 0.35 });
      }
    },
    frame: function (ctx, s, t) {
      var p = s.palette;
      ctx.clearRect(0, 0, s.w, s.h);
      ctx.globalCompositeOperation = p.dark ? "lighter" : "source-over";
      var R = Math.max(s.w, s.h);
      for (var i = 0; i < s.blobs.length; i++) {
        var b = s.blobs[i];
        var x = s.w * (b.ax + b.rx * Math.sin(t * b.f1 + b.p1));
        var y = s.h * (b.ay + b.ry * Math.cos(t * b.f2 + b.p2));
        var r = R * b.size * s.scale;
        var hue = p.accent[0] + b.hue;
        var sat = Math.min(1, p.accent[1] * (p.dark ? 1.1 : 0.9) + 0.15);
        var light = p.dark ? 0.32 + 0.1 * (i % 3) : 0.78 - 0.06 * (i % 3);
        var g = ctx.createRadialGradient(x, y, 0, x, y, r);
        g.addColorStop(0, hsla(hue, sat, light, (p.dark ? 0.55 : 0.5) * s.alpha));
        g.addColorStop(0.55, hsla(hue, sat, light, (p.dark ? 0.16 : 0.18) * s.alpha));
        g.addColorStop(1, hsla(hue, sat, light, 0));
        ctx.fillStyle = g;
        ctx.fillRect(x - r, y - r, r * 2, r * 2);
      }
      ctx.globalCompositeOperation = "source-over";
    },
  };

  effects.particles = {
    init: function (s) {
      var n = Math.round(Math.max(40, Math.min(170, (s.w * s.h) / 5200)) * s.amount);
      s.dots = [];
      for (var i = 0; i < n; i++) {
        var z = 0.35 + s.r() * 0.65;
        s.dots.push({ x: s.r() * s.w, y: s.r() * s.h, z: z, vx: (s.r() - 0.5) * 12 * z, vy: (s.r() - 0.5) * 12 * z });
      }
    },
    frame: function (ctx, s, t, dt) {
      var p = s.palette;
      ctx.clearRect(0, 0, s.w, s.h);
      var link = 140 * s.scale;
      var ink = p.dark ? [p.ink[0], p.ink[1], 0.85] : [p.accent[0], p.accent[1], Math.min(0.45, p.accent[2])];
      for (var i = 0; i < s.dots.length; i++) {
        var d = s.dots[i];
        d.x += d.vx * dt; d.y += d.vy * dt;
        if (d.x < -20) d.x = s.w + 20; if (d.x > s.w + 20) d.x = -20;
        if (d.y < -20) d.y = s.h + 20; if (d.y > s.h + 20) d.y = -20;
      }
      ctx.lineWidth = 1;
      for (i = 0; i < s.dots.length; i++) {
        var a = s.dots[i];
        for (var j = i + 1; j < s.dots.length; j++) {
          var b = s.dots[j];
          var dx = a.x - b.x, dy = a.y - b.y, dist = Math.sqrt(dx * dx + dy * dy);
          var reach = link * Math.min(a.z, b.z) + link * 0.4;
          if (dist < reach) {
            ctx.strokeStyle = hsla(ink[0], ink[1], ink[2], (1 - dist / reach) * 0.5 * s.alpha * Math.min(a.z, b.z));
            ctx.beginPath(); ctx.moveTo(a.x, a.y); ctx.lineTo(b.x, b.y); ctx.stroke();
          }
        }
      }
      for (i = 0; i < s.dots.length; i++) {
        var q = s.dots[i];
        ctx.fillStyle = hsla(ink[0], ink[1], ink[2], (0.5 + 0.5 * q.z) * s.alpha);
        ctx.beginPath(); ctx.arc(q.x, q.y, (1.2 + 2.4 * q.z) * s.scale, 0, 6.2832); ctx.fill();
      }
    },
  };

  effects.waves = {
    init: function (s) {
      var n = Math.round(4 * s.amount);
      s.layers = [];
      for (var i = 0; i < n; i++) s.layers.push({ amp: 0.06 + s.r() * 0.08, freq: 0.8 + s.r() * 1.4, speed: 0.15 + s.r() * 0.25, phase: s.r() * 6.28, base: 0.55 + (i / n) * 0.35, hue: (s.r() - 0.5) * 40 });
    },
    frame: function (ctx, s, t) {
      var p = s.palette;
      ctx.clearRect(0, 0, s.w, s.h);
      for (var i = 0; i < s.layers.length; i++) {
        var L = s.layers[i];
        var depth = i / Math.max(1, s.layers.length - 1);
        ctx.beginPath();
        ctx.moveTo(0, s.h);
        for (var x = 0; x <= s.w; x += 8) {
          var u = x / s.w;
          var y = s.h * (L.base + L.amp * s.scale * Math.sin(u * L.freq * 6.2832 + t * L.speed + L.phase) + L.amp * 0.4 * Math.sin(u * L.freq * 2.1 * 6.2832 - t * L.speed * 0.6));
          ctx.lineTo(x, y);
        }
        ctx.lineTo(s.w, s.h);
        ctx.closePath();
        var light = p.dark ? 0.28 + depth * 0.18 : 0.62 + depth * 0.2;
        ctx.fillStyle = hsla(p.accent[0] + L.hue, Math.min(1, p.accent[1] + 0.1), light, (0.28 - depth * 0.12) * s.alpha);
        ctx.fill();
      }
    },
  };

  effects.orbs = {
    init: function (s) {
      var n = Math.round(11 * s.amount);
      s.orbs = [];
      for (var i = 0; i < n; i++) s.orbs.push({ r0: 0.4 + s.r() * 0.9, y: (s.r() - 0.5) * 1.6, ang: s.r() * 6.28, speed: (0.08 + s.r() * 0.12) * (s.r() < 0.5 ? 1 : -1), size: 0.05 + s.r() * 0.11, hue: (s.r() - 0.5) * 50, bob: s.r() * 6.28 });
    },
    frame: function (ctx, s, t) {
      var p = s.palette;
      ctx.clearRect(0, 0, s.w, s.h);
      var list = [];
      for (var i = 0; i < s.orbs.length; i++) {
        var o = s.orbs[i];
        var a = o.ang + t * o.speed;
        var x = Math.cos(a) * o.r0, z = 1.6 + Math.sin(a) * o.r0, y = o.y + 0.08 * Math.sin(t * 0.5 + o.bob);
        var q = project(x, y, z, s);
        list.push({ x: q.x, y: q.y, r: o.size * s.w * q.k * s.scale, k: q.k, hue: o.hue });
      }
      list.sort(function (a, b) { return a.k - b.k; });
      for (i = 0; i < list.length; i++) {
        var b = list[i];
        var near = Math.min(1, Math.max(0, (b.k - 0.35) / 0.65));
        var hue = p.accent[0] + b.hue, sat = Math.min(1, p.accent[1] + 0.1);
        var g = ctx.createRadialGradient(b.x - b.r * 0.35, b.y - b.r * 0.35, b.r * 0.1, b.x, b.y, b.r);
        g.addColorStop(0, hsla(hue, sat, p.dark ? 0.72 : 0.9, (0.9) * s.alpha * (0.45 + 0.55 * near)));
        g.addColorStop(0.55, hsla(hue, sat, p.dark ? 0.42 : 0.62, (0.85) * s.alpha * (0.4 + 0.6 * near)));
        g.addColorStop(1, hsla(hue, sat, p.dark ? 0.2 : 0.45, 0.55 * s.alpha * (0.3 + 0.7 * near)));
        ctx.fillStyle = g;
        ctx.beginPath(); ctx.arc(b.x, b.y, b.r, 0, 6.2832); ctx.fill();
        // A soft shadow under the near ones lifts them off the page.
        if (near > 0.5) {
          var sh = ctx.createRadialGradient(b.x, b.y + b.r * 1.15, 0, b.x, b.y + b.r * 1.15, b.r * 1.1);
          sh.addColorStop(0, hsla(p.ink[0], 0.2, 0.1, 0.18 * (near - 0.5) * s.alpha));
          sh.addColorStop(1, hsla(p.ink[0], 0.2, 0.1, 0));
          ctx.fillStyle = sh;
          ctx.beginPath(); ctx.ellipse(b.x, b.y + b.r * 1.15, b.r * 1.1, b.r * 0.35, 0, 0, 6.2832); ctx.fill();
        }
      }
    },
  };

  effects.grid = {
    init: function (s) { s.lines = Math.round(14 * s.amount); },
    frame: function (ctx, s, t) {
      var p = s.palette;
      ctx.clearRect(0, 0, s.w, s.h);
      var horizon = s.h * 0.48, vx = s.w / 2;
      var glow = ctx.createLinearGradient(0, horizon - s.h * 0.25, 0, horizon + s.h * 0.15);
      glow.addColorStop(0, hsla(p.accent[0], p.accent[1], p.dark ? 0.5 : 0.7, 0));
      glow.addColorStop(0.75, hsla(p.accent[0], p.accent[1], p.dark ? 0.5 : 0.7, 0.22 * s.alpha));
      glow.addColorStop(1, hsla(p.accent[0], p.accent[1], p.dark ? 0.5 : 0.7, 0));
      ctx.fillStyle = glow;
      ctx.fillRect(0, horizon - s.h * 0.25, s.w, s.h * 0.4);
      var col = function (a) { return hsla(p.accent[0], Math.min(1, p.accent[1] + 0.1), p.dark ? 0.6 : 0.42, a * s.alpha); };
      ctx.lineWidth = 1;
      // Verticals converge on the vanishing point.
      for (var i = -s.lines; i <= s.lines; i++) {
        var xb = vx + (i / s.lines) * s.w * 1.6 * s.scale;
        ctx.strokeStyle = col(0.35 - Math.abs(i / s.lines) * 0.2);
        ctx.beginPath(); ctx.moveTo(vx, horizon); ctx.lineTo(xb, s.h + 2); ctx.stroke();
      }
      // Horizontals roll toward the viewer.
      var rows = 18, phase = (t * 0.35) % 1;
      for (var r = 0; r < rows; r++) {
        var u = (r + phase) / rows;
        var y = horizon + (s.h - horizon) * (u * u);
        ctx.strokeStyle = col(0.08 + u * 0.35);
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(s.w, y); ctx.stroke();
      }
      // The horizon line itself.
      ctx.strokeStyle = col(0.55);
      ctx.beginPath(); ctx.moveTo(0, horizon); ctx.lineTo(s.w, horizon); ctx.stroke();
    },
  };

  effects.stars = {
    init: function (s) {
      var n = Math.round(Math.min(420, (s.w * s.h) / 3200) * s.amount);
      s.stars = [];
      for (var i = 0; i < n; i++) s.stars.push({ x: s.r(), y: s.r(), z: 0.2 + s.r() * 0.8, tw: s.r() * 6.28, tf: 0.5 + s.r() * 1.5 });
      s.nebulae = [];
      for (i = 0; i < 3; i++) s.nebulae.push({ x: s.r(), y: s.r(), r: 0.3 + s.r() * 0.35, hue: (s.r() - 0.5) * 80 });
    },
    frame: function (ctx, s, t) {
      var p = s.palette;
      ctx.clearRect(0, 0, s.w, s.h);
      for (var i = 0; i < s.nebulae.length; i++) {
        var nb = s.nebulae[i];
        var g = ctx.createRadialGradient(nb.x * s.w, nb.y * s.h, 0, nb.x * s.w, nb.y * s.h, nb.r * Math.max(s.w, s.h) * s.scale);
        g.addColorStop(0, hsla(p.accent[0] + nb.hue, p.accent[1], p.dark ? 0.4 : 0.75, 0.22 * s.alpha));
        g.addColorStop(1, hsla(p.accent[0] + nb.hue, p.accent[1], p.dark ? 0.4 : 0.75, 0));
        ctx.fillStyle = g;
        ctx.fillRect(0, 0, s.w, s.h);
      }
      var drift = t * 0.008;
      var star = p.dark ? [p.ink[0], 0.1, 0.96] : [p.accent[0], p.accent[1], 0.35];
      for (i = 0; i < s.stars.length; i++) {
        var st = s.stars[i];
        var x = ((st.x + drift * st.z) % 1) * s.w, y = ((st.y + drift * 0.3 * st.z) % 1) * s.h;
        var a = (0.5 + 0.5 * (0.5 + 0.5 * Math.sin(t * st.tf + st.tw))) * (0.4 + 0.6 * st.z) * s.alpha;
        var r = (0.7 + 1.8 * st.z) * s.scale;
        if (st.z > 0.85) {
          var halo = ctx.createRadialGradient(x, y, 0, x, y, r * 4);
          halo.addColorStop(0, hsla(star[0], star[1], star[2], a * 0.5));
          halo.addColorStop(1, hsla(star[0], star[1], star[2], 0));
          ctx.fillStyle = halo;
          ctx.beginPath(); ctx.arc(x, y, r * 4, 0, 6.2832); ctx.fill();
        }
        ctx.fillStyle = hsla(star[0], star[1], star[2], a);
        ctx.beginPath(); ctx.arc(x, y, r, 0, 6.2832); ctx.fill();
      }
    },
  };

  effects.topo = {
    init: function (s) { s.n1 = noise2(s.seed); s.n2 = noise2(s.seed + 977); s.levels = Math.round(9 * s.amount); s.cell = 22 / s.scale; },
    frame: function (ctx, s, t) {
      var p = s.palette;
      ctx.clearRect(0, 0, s.w, s.h);
      var cols = Math.ceil(s.w / s.cell) + 1, rows = Math.ceil(s.h / s.cell) + 1;
      var blend = 0.5 + 0.5 * Math.sin(t * 0.12);
      var field = new Float32Array(cols * rows);
      var sc = 0.11;
      for (var j = 0; j < rows; j++) for (var i = 0; i < cols; i++) {
        var x = i * sc + t * 0.02, y = j * sc;
        field[j * cols + i] = s.n1(x, y) * (1 - blend) + s.n2(x + 3.7, y + 1.3) * blend;
      }
      ctx.lineWidth = 1;
      var hue = p.accent[0], sat = p.accent[1], light = p.dark ? 0.62 : 0.38;
      for (var l = 1; l <= s.levels; l++) {
        var level = l / (s.levels + 1);
        ctx.strokeStyle = hsla(hue, sat, light, (0.12 + 0.3 * (l % 3 === 0 ? 1 : 0.5)) * s.alpha);
        ctx.beginPath();
        for (j = 0; j < rows - 1; j++) for (i = 0; i < cols - 1; i++) {
          var a = field[j * cols + i], b = field[j * cols + i + 1], c = field[(j + 1) * cols + i + 1], d = field[(j + 1) * cols + i];
          var idx = (a > level ? 8 : 0) | (b > level ? 4 : 0) | (c > level ? 2 : 0) | (d > level ? 1 : 0);
          if (idx === 0 || idx === 15) continue;
          var x0 = i * s.cell, y0 = j * s.cell;
          var top = { x: x0 + s.cell * ((level - a) / (b - a || 1e-6)), y: y0 };
          var right = { x: x0 + s.cell, y: y0 + s.cell * ((level - b) / (c - b || 1e-6)) };
          var bottom = { x: x0 + s.cell * ((level - d) / (c - d || 1e-6)), y: y0 + s.cell };
          var left = { x: x0, y: y0 + s.cell * ((level - a) / (d - a || 1e-6)) };
          var seg = function (m, n) { ctx.moveTo(m.x, m.y); ctx.lineTo(n.x, n.y); };
          switch (idx) {
            case 1: case 14: seg(left, bottom); break;
            case 2: case 13: seg(bottom, right); break;
            case 3: case 12: seg(left, right); break;
            case 4: case 11: seg(top, right); break;
            case 5: seg(top, left); seg(bottom, right); break;
            case 6: case 9: seg(top, bottom); break;
            case 7: case 8: seg(top, left); break;
            case 10: seg(top, right); seg(left, bottom); break;
          }
        }
        ctx.stroke();
      }
    },
  };

  effects.ribbons = {
    init: function (s) {
      var n = Math.round(3 * s.amount);
      s.ribbons = [];
      for (var i = 0; i < n; i++) s.ribbons.push({ y: (s.r() - 0.5) * 0.7, amp: 0.18 + s.r() * 0.22, f1: 0.7 + s.r() * 0.7, f2: 0.5 + s.r() * 0.7, speed: 0.18 + s.r() * 0.22, phase: s.r() * 6.28, hue: (s.r() - 0.5) * 60, width: 0.035 + s.r() * 0.035 });
    },
    frame: function (ctx, s, t) {
      var p = s.palette;
      ctx.clearRect(0, 0, s.w, s.h);
      var steps = 110;
      for (var i = 0; i < s.ribbons.length; i++) {
        var R = s.ribbons[i];
        var pts = [];
        for (var k = 0; k <= steps; k++) {
          var u = k / steps;
          var x = (u - 0.5) * 2.6, y = R.y + R.amp * s.scale * Math.sin(u * R.f1 * 6.2832 + t * R.speed + R.phase);
          var z = 1.5 + 0.6 * Math.cos(u * R.f2 * 6.2832 + t * R.speed * 0.8 + R.phase);
          var q = project(x, y, z, s);
          pts.push({ x: q.x, y: q.y, k: q.k });
        }
        // Offset each point along its normal to get the two edges of the band.
        var top = [], bottom = [];
        for (k = 0; k <= steps; k++) {
          var prev = pts[Math.max(0, k - 1)], next = pts[Math.min(steps, k + 1)];
          var dx = next.x - prev.x, dy = next.y - prev.y, len = Math.sqrt(dx * dx + dy * dy) || 1;
          var nx = -dy / len, ny = dx / len;
          var half = Math.max(2, R.width * s.h * pts[k].k * s.scale);
          top.push({ x: pts[k].x + nx * half, y: pts[k].y + ny * half });
          bottom.push({ x: pts[k].x - nx * half, y: pts[k].y - ny * half });
        }
        var hue = p.accent[0] + R.hue, sat = Math.min(1, p.accent[1] + 0.1);
        var g = ctx.createLinearGradient(0, 0, s.w, 0);
        g.addColorStop(0, hsla(hue - 20, sat, p.dark ? 0.45 : 0.58, 0.55 * s.alpha));
        g.addColorStop(0.5, hsla(hue + 15, sat, p.dark ? 0.6 : 0.7, 0.62 * s.alpha));
        g.addColorStop(1, hsla(hue + 40, sat, p.dark ? 0.42 : 0.55, 0.5 * s.alpha));
        ctx.beginPath();
        ctx.moveTo(top[0].x, top[0].y);
        for (k = 1; k <= steps; k++) ctx.lineTo(top[k].x, top[k].y);
        for (k = steps; k >= 0; k--) ctx.lineTo(bottom[k].x, bottom[k].y);
        ctx.closePath();
        ctx.fillStyle = g;
        ctx.fill();
        // A thin light edge along the top reads as a sheen.
        ctx.beginPath();
        ctx.moveTo(top[0].x, top[0].y);
        for (k = 1; k <= steps; k++) ctx.lineTo(top[k].x, top[k].y);
        ctx.strokeStyle = hsla(hue + 10, sat, 0.88, 0.35 * s.alpha);
        ctx.lineWidth = 1;
        ctx.stroke();
      }
    },
  };

  /* ---------- hosts and the loop ---------- */

  var hosts = [];
  var running = false;
  var last = 0;

  function motionFor(host) {
    var site = (host.closest("[data-site-motion]") || document.body).getAttribute("data-site-motion") || "full";
    var own = host.getAttribute("data-fx-motion") || "normal";
    if (reduced || site === "off" || own === "still") return { still: true, speed: 0, fps: 0 };
    var speed = own === "slow" ? 0.45 : 1;
    var fps = 60;
    if (site === "calm") { speed *= 0.5; fps = 30; }
    return { still: false, speed: speed, fps: fps };
  }

  function setup(host) {
    var name = host.getAttribute("data-fx") || "";
    var effect = effects[name];
    if (!effect) return null;
    var canvas = host.querySelector(":scope > canvas.site-fx");
    if (!canvas) {
      canvas = document.createElement("canvas");
      canvas.className = "site-fx";
      canvas.setAttribute("aria-hidden", "true");
      host.insertBefore(canvas, host.firstChild);
    }
    var rect = host.getBoundingClientRect();
    var cssW = Math.max(1, rect.width), cssH = Math.max(1, rect.height);
    var dpr = Math.min(MAX_DPR, window.devicePixelRatio || 1);
    if (cssW * dpr > MAX_WIDTH) dpr = MAX_WIDTH / cssW;
    canvas.width = Math.round(cssW * dpr);
    canvas.height = Math.round(cssH * dpr);
    var ctx = canvas.getContext("2d");
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    var intensity = host.getAttribute("data-fx-intensity") || "normal";
    var seedText = host.getAttribute("data-fx-seed") || (document.title + name);
    var seed = hash(seedText);
    var s = {
      host: host, canvas: canvas, ctx: ctx, w: cssW, h: cssH, dpr: dpr, effect: effect, name: name, seed: seed, r: rng(seed),
      amount: intensity === "subtle" ? 0.6 : intensity === "bold" ? 1.45 : 1,
      alpha: intensity === "subtle" ? 0.65 : intensity === "bold" ? 1.2 : 1,
      scale: intensity === "subtle" ? 0.85 : intensity === "bold" ? 1.2 : 1,
      palette: paletteFor(host), motion: motionFor(host), t: seed % 1000 / 10, visible: true, acc: 0,
    };
    effect.init(s);
    effect.frame(ctx, s, s.t, 0);
    return s;
  }

  function draw(s, dt) {
    if (s.motion.still) return;
    s.t += dt * s.motion.speed;
    s.effect.frame(s.ctx, s, s.t, dt * s.motion.speed);
  }

  function loop(now) {
    running = false;
    var dt = Math.min(0.1, (now - last) / 1000 || 0.016);
    last = now;
    var any = false;
    for (var i = 0; i < hosts.length; i++) {
      var s = hosts[i];
      if (s.motion.still || !s.visible) continue;
      any = true;
      s.acc += dt;
      var step = 1 / s.motion.fps;
      if (s.acc >= step) { draw(s, s.acc); s.acc = 0; }
    }
    if (any && !document.hidden) { running = true; requestAnimationFrame(loop); }
  }

  function wake() {
    if (running || document.hidden) return;
    running = true;
    last = performance.now();
    requestAnimationFrame(loop);
  }

  var observer = window.IntersectionObserver ? new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      for (var i = 0; i < hosts.length; i++) if (hosts[i].host === entry.target) hosts[i].visible = entry.isIntersecting;
    });
    wake();
  }, { rootMargin: "80px" }) : null;

  var resizer = window.ResizeObserver ? new ResizeObserver(function (entries) {
    entries.forEach(function (entry) {
      for (var i = 0; i < hosts.length; i++) {
        var s = hosts[i];
        if (s.host !== entry.target) continue;
        var rect = s.host.getBoundingClientRect();
        if (Math.abs(rect.width - s.w) > 2 || Math.abs(rect.height - s.h) > 2) refresh(s.host);
      }
    });
  }) : null;

  function forget(host) {
    for (var i = hosts.length - 1; i >= 0; i--) if (hosts[i].host === host) hosts.splice(i, 1);
    if (observer) observer.unobserve(host);
    if (resizer) resizer.unobserve(host);
  }

  function refresh(host) {
    forget(host);
    var s = setup(host);
    if (!s) {
      var stale = host.querySelector(":scope > canvas.site-fx");
      if (stale) stale.parentNode.removeChild(stale);
      return;
    }
    hosts.push(s);
    if (observer) observer.observe(host);
    if (resizer) resizer.observe(host);
    wake();
  }

  function scan(rootNode) {
    var nodes = (rootNode || document).querySelectorAll("[data-fx]");
    Array.prototype.forEach.call(nodes, function (host) {
      var known = false;
      for (var i = 0; i < hosts.length; i++) if (hosts[i].host === host) known = true;
      if (!known) refresh(host);
    });
    for (var i = hosts.length - 1; i >= 0; i--) if (!document.contains(hosts[i].host)) forget(hosts[i].host);
  }

  document.addEventListener("visibilitychange", function () { if (!document.hidden) wake(); });

  /* ---------- HTML in 3D: cards that tilt toward the pointer ---------- */

  var TILT_TARGETS = ".site-features__card, .site-pricing__card, .site-testimonials__card, .site-team__person, .site-hero__picture, .site-imagetext__picture, .site-gallery__item";
  function tiltSetup(host) {
    if (!finePointer || reduced || host.getAttribute("data-tilt-ready") === "true") return;
    var site = (host.closest("[data-site-motion]") || document.body).getAttribute("data-site-motion");
    if (site === "off") return;
    host.setAttribute("data-tilt-ready", "true");
    var cards = host.querySelectorAll(TILT_TARGETS);
    Array.prototype.forEach.call(cards, function (card) {
      card.classList.add("site-tilt__card");
      card.addEventListener("pointermove", function (event) {
        var rect = card.getBoundingClientRect();
        var px = (event.clientX - rect.left) / rect.width - 0.5;
        var py = (event.clientY - rect.top) / rect.height - 0.5;
        card.style.transform = "perspective(900px) rotateX(" + (-py * 8).toFixed(2) + "deg) rotateY(" + (px * 8).toFixed(2) + "deg) translateZ(6px)";
        card.style.setProperty("--tilt-x", (px * 100 + 50).toFixed(1) + "%");
        card.style.setProperty("--tilt-y", (py * 100 + 50).toFixed(1) + "%");
      });
      card.addEventListener("pointerleave", function () { card.style.transform = ""; });
    });
  }
  function scanTilt(rootNode) {
    Array.prototype.forEach.call((rootNode || document).querySelectorAll("[data-tilt]"), tiltSetup);
  }

  function rescan(rootNode) { scan(rootNode); scanTilt(rootNode); }

  window.gosxFX = { refresh: refresh, rescan: rescan, effects: Object.keys(effects), hosts: function () { return hosts.length; } };

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { rescan(); });
  else rescan();
})();
