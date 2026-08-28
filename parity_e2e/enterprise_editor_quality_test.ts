import { expect, test, type Page, type TestInfo } from "@playwright/test";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

// This is an isolated, fixture-only quality gate. It deliberately loads the
// same embedded CSS and browser runtimes that Studio serves, but never boots a
// host application or points at customer/staging data. The markup below is
// the Page CMS detail renderer contract from backoffice/backend_page_detail.go
// (with the host's editor-page class), not a synthetic workbench shell.
const studioCSS = readFileSync(
  path.resolve(__dirname, "../hostruntime/assets/studio.css"),
  "utf8",
);
const contentRuntimeJS = readFileSync(
  path.resolve(__dirname, "../contenteditorruntime/content_editor.js"),
  "utf8",
);
const mediaRuntimeJS = readFileSync(
  path.resolve(__dirname, "../mediaruntime/media_runtime.js"),
  "utf8",
);
// Keep quality04 artifacts separate from other polish owners and put them in
// the ignored shared handoff tree requested by the orchestrator. The default
// root preserves the canonical quality04 paths; the opt-in cross-browser
// config supplies a root whose project subdirectory is selected at test time.
const defaultQualityArtifactRoot = path.resolve(
  __dirname,
  "../.tiller/scratch/codex/enterprise-polish-20260827/qa/quality04",
);
const configuredQualityArtifactRoot = process.env.QUALITY_ARTIFACT_ROOT
  ? path.resolve(process.env.QUALITY_ARTIFACT_ROOT)
  : undefined;

type QualityViewport = {
  name: "1600" | "1280" | "390";
  width: number;
  height: number;
};

const viewports: QualityViewport[] = [
  { name: "1600", width: 1600, height: 900 },
  { name: "1280", width: 1280, height: 800 },
  { name: "390", width: 390, height: 844 },
];

type QualityBlock = {
  id: string;
  type: "heading" | "paragraph" | "quote" | "button" | "image" | "gallery" | "product" | "flow";
  text?: string;
  level?: string;
  label?: string;
  href?: string;
  url?: string;
  alt?: string;
  images?: Array<{ url: string; alt: string }>;
  productRef?: string;
  flowKey?: string;
};

type PageDetailFixtureOptions = {
  action?: string;
  expectedRevision?: string;
};

function escapeHTML(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function fixtureBlocks(count = 6): QualityBlock[] {
  const imageA = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='640' height='360'%3E%3Crect width='640' height='360' fill='%23cbb39a'/%3E%3C/svg%3E";
  const imageB = "/media/enterprise/asset-01-long-editorial-filename.jpg";
  const imageC = "/media/enterprise/asset-02-long-editorial-filename-x.jpg";
  const longHeading = "Enterprise publishing workspace — a heading that remains readable when the selected block carries a very long editorial title and a narrow viewport";
  const blocks: QualityBlock[] = [
    { id: "hero-heading", type: "heading", text: longHeading, level: "2" },
    { id: "intro", type: "paragraph", text: "A representative authoring surface keeps the draft local while the editor remains usable with keyboard, pointer, and touch input." },
    { id: "quote", type: "quote", text: "Every edit should be visible, reversible, and safe to continue." },
    { id: "cta", type: "button", label: "Read the complete editorial workflow", href: "/editorial/workflows/enterprise/long-route" },
    { id: "image", type: "image", url: imageA, alt: "Warm paper editorial preview" },
    { id: "gallery", type: "gallery", images: [
      { url: imageB, alt: "Sunlit paper" },
      { url: imageC, alt: "Moss ink" },
    ] },
  ];
  while (blocks.length < count) {
    blocks.push({
      id: `paragraph-${blocks.length}`,
      type: "paragraph",
      text: `Representative content block ${blocks.length + 1}: a stable draft item for interaction timing and keyboard traversal.`,
    });
  }
  return blocks;
}

function mediaOptions(count = 24): string {
  return Array.from({ length: count }, (_, index) => {
    const slug = `asset-${String(index + 1).padStart(2, "0")}-long-editorial-filename-${"x".repeat(index % 4)}`;
    const url = `/media/enterprise/${slug}.jpg`;
    return `<option value="${url}" label="${slug}.jpg" data-media-alt="Editorial asset ${index + 1}">${slug}.jpg</option>`;
  }).join("");
}

function pageDetailFixtureHTML(blocks: QualityBlock[], options: PageDetailFixtureOptions = {}): string {
  const source = escapeHTML(JSON.stringify({
    version: 1,
    blocks,
  }, null, 2));
  const selectedMedia = "/media/enterprise/asset-01-long-editorial-filename.jpg";
  const title = "Enterprise editorial page with a deliberately long title for wrapping checks";
  const slug = "enterprise-editorial-page-with-a-deliberately-long-slug-for-narrow-layout-checks";
  const action = escapeHTML(options.action || "/admin/pages/page_1/__actions/save");
  const revision = options.expectedRevision == null
    ? ""
    : `<input type="hidden" name="expectedRevision" value="${escapeHTML(options.expectedRevision)}" data-content-editor-revision="true">`;
  return `
    <!doctype html>
    <html>
      <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <style>${studioCSS}</style>
        <style>
          /* Only normalize the standalone document; component layout comes
             from the shared stylesheet and the real Page CMS markup. */
          html, body { margin: 0; min-height: 100%; }
          body { background: var(--color-canvas); }
        </style>
      </head>
      <body>
        <div class="admin-page editor-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-page-detail-renderer="gosx-studio">
          <section class="admin-heading">
            <p class="kicker">CMS</p>
            <h1>Edit page</h1>
            <p>${escapeHTML(`Editing ${title}; the route /admin/pages/${slug}/edit must remain readable when labels and URLs wrap.`)}</p>
          </section>
          <datalist id="page-media-urls">${mediaOptions()}</datalist>
          <form class="panel admin-form" method="post" action="${action}" data-content-editor-enhanced-submit="true">
            <input type="hidden" name="csrf_token" value="quality-fixture-csrf">
            <input type="hidden" name="id" value="page_1">
            ${revision}
            <div class="button-row admin-form__primary-actions" data-content-editor-primary-actions="true">
              <button class="button button--primary" type="submit">Save page</button>
              <a class="button button--secondary" href="/enterprise-editorial?preview=1" data-gosx-link="true">Preview</a>
            </div>
            <div class="edit-preview edit-preview--text">
              <div><strong>Draft preview</strong><span>Preview is provided by the host route.</span></div>
            </div>
            <div class="field-row">
              <label for="title">Title</label>
              <input id="title" name="title" value="${escapeHTML(title)}">
            </div>
            <div class="field-row">
              <label for="slug">Slug</label>
              <input id="slug" name="slug" value="${escapeHTML(slug)}">
            </div>
            <div class="field-row field-row--wide">
              <label for="bodyFormat">Body format</label>
              <select id="bodyFormat" name="bodyFormat" data-content-editor-format="true">
                <option value="blocks" selected>GoSX blocks</option>
                <option value="mdpp">MDPP</option>
              </select>
            </div>
            <div class="field-row field-row--wide content-editor" data-content-editor="true" data-content-editor-media-list="page-media-urls">
              <label for="body">Body</label>
              <textarea class="content-editor__source" id="body" name="body" rows="10" data-content-editor-source="true">${source}</textarea>
              <div class="content-editor__toolbar">
                <button type="button" class="home-section-move" data-content-add="heading">Heading</button>
                <button type="button" class="home-section-move" data-content-add="paragraph">Paragraph</button>
                <button type="button" class="home-section-move" data-content-add="image">Image</button>
                <button type="button" class="home-section-move" data-content-add="gallery">Gallery</button>
                <button type="button" class="home-section-move" data-content-add="button">Button</button>
                <button type="button" class="home-section-move" data-content-add="product">Product</button>
                <button type="button" class="home-section-move" data-content-add="quote">Quote</button>
              </div>
              <div class="content-block-list" data-content-editor-list="true"></div>
            </div>
            <fieldset class="form-section">
              <legend>SEO</legend>
              <div class="field-row">
                <label for="metaTitle">Meta title</label>
                <input id="metaTitle" name="metaTitle" value="${escapeHTML(title)}">
              </div>
              <div class="field-row">
                <label for="metaImageUrl">Meta image URL</label>
                <input id="metaImageUrl" name="metaImageUrl" list="page-media-urls" value="${selectedMedia}" data-media-alt-target="metaImageAlt">
              </div>
              <div class="field-row field-row--wide">
                <label for="metaDescription">Meta description</label>
                <textarea id="metaDescription" name="metaDescription" rows="3">${escapeHTML("A long-form editorial page used to verify the Page CMS authoring surface remains legible at narrow widths.")}</textarea>
              </div>
              <div class="field-row field-row--wide">
                <label for="metaImageAlt">Meta image alt text</label>
                <input id="metaImageAlt" name="metaImageAlt" value="Editorial hero image">
              </div>
            </fieldset>
            <div class="check-row"><label><input type="checkbox" name="published"> Published</label></div>
            <div class="button-row admin-form__secondary-actions" data-content-editor-secondary-actions="true">
              <a class="button button--secondary" href="/admin/pages" data-gosx-link="true">Back to pages</a>
            </div>
          </form>
          <section class="panel" data-studio-revision-history="true">
            <div class="panel__header"><h2>Revision history</h2></div>
            <p class="field-note">Local fixture history placeholder; persistence is outside this isolated CSS/runtime pass.</p>
          </section>
        </div>
      </body>
    </html>
  `;
}

async function mountFixture(page: Page, blocks = fixtureBlocks(), options: PageDetailFixtureOptions = {}): Promise<void> {
  await page.setContent(pageDetailFixtureHTML(blocks, options));
  await page.addScriptTag({ content: contentRuntimeJS });
  await page.addScriptTag({ content: mediaRuntimeJS });
  await expect(page.locator("[data-content-editor]"))
    .toHaveAttribute("data-content-editor-source-state", "ready");
  await expect(page.locator("[data-content-block-id]"))
    .toHaveCount(blocks.length);
}

async function mountFixtureAtURL(page: Page, blocks = fixtureBlocks(), options: PageDetailFixtureOptions = {}): Promise<void> {
  const editorURL = "http://127.0.0.1:4173/editor";
  await page.unroute(editorURL);
  await page.route(editorURL, async (route) => {
    await route.fulfill({ contentType: "text/html", body: pageDetailFixtureHTML(blocks, options) });
  });
  await page.goto(editorURL, { waitUntil: "domcontentloaded" });
  await page.addScriptTag({ content: contentRuntimeJS });
  await page.addScriptTag({ content: mediaRuntimeJS });
  await expect(page.locator("[data-content-editor]"))
    .toHaveAttribute("data-content-editor-source-state", "ready");
  await expect(page.locator("[data-content-block-id]"))
    .toHaveCount(blocks.length);
}

function artifactPath(name: string): string {
  const root = configuredQualityArtifactRoot
    ? path.join(configuredQualityArtifactRoot, test.info().project.name)
    : defaultQualityArtifactRoot;
  mkdirSync(root, { recursive: true });
  return path.join(root, name);
}

async function capture(page: Page, name: string): Promise<string> {
  const output = artifactPath(name);
  await page.screenshot({ path: output, fullPage: false });
  return output;
}

async function assertNoPageOverflow(page: Page, label: string): Promise<void> {
  const bounds = await page.evaluate(() => ({
    innerWidth: window.innerWidth,
    documentWidth: document.documentElement.scrollWidth,
    bodyWidth: document.body.scrollWidth,
  }));
  expect(bounds.documentWidth, `${label} document should not horizontally overflow`).toBeLessThanOrEqual(bounds.innerWidth + 1);
  expect(bounds.bodyWidth, `${label} body should not horizontally overflow`).toBeLessThanOrEqual(bounds.innerWidth + 1);
}

async function assertVisiblePrimaryControls(page: Page, label: string): Promise<void> {
  const controls = page.locator("[data-content-editor-primary-actions] .button");
  await expect(controls).toHaveCount(2);
  for (let index = 0; index < await controls.count(); index += 1) {
    const control = controls.nth(index);
    await expect(control, `${label} primary control ${index}`).toBeVisible();
    const box = await control.boundingBox();
    expect(box, `${label} primary control ${index} should have geometry`).toBeTruthy();
    expect(box!.x, `${label} primary control ${index} left bound`).toBeGreaterThanOrEqual(0);
    expect(box!.x + box!.width, `${label} primary control ${index} right bound`).toBeLessThanOrEqual((await page.evaluate(() => window.innerWidth)) + 1);
  }
}

async function assertPageDetailViewport(page: Page, viewport: QualityViewport, state: string): Promise<void> {
  await page.setViewportSize({ width: viewport.width, height: viewport.height });
  await page.evaluate(() => window.scrollTo(0, 0));
  await expect(page.locator("[data-gosx-studio-backend-page-detail-renderer='gosx-studio']")).toBeVisible();
  await assertVisiblePrimaryControls(page, `${viewport.name} ${state}`);
  await assertNoPageOverflow(page, `${viewport.name} ${state}`);
}

async function assertSaveStatusLayout(page: Page, label: string): Promise<Record<string, number>> {
  const metrics = await page.evaluate(() => {
    const status = document.querySelector<HTMLElement>("[data-content-editor-save-status]");
    const detail = document.querySelector<HTMLElement>("[data-content-editor-save-detail-text]");
    if (!status || !detail) return null;
    const statusBox = status.getBoundingClientRect();
    const detailBox = detail.getBoundingClientRect();
    return {
      statusWidth: statusBox.width,
      statusHeight: statusBox.height,
      detailWidth: detailBox.width,
      detailHeight: detailBox.height,
      viewportWidth: window.innerWidth,
      viewportHeight: window.innerHeight,
      documentWidth: document.documentElement.scrollWidth,
      bodyWidth: document.body.scrollWidth,
    };
  });
  expect(metrics, `${label} save status should have geometry`).toBeTruthy();
  expect(metrics!.statusWidth, `${label} status should use the available editor width`).toBeGreaterThan(metrics!.viewportWidth * 0.7);
  expect(metrics!.detailWidth, `${label} detail must retain a readable line width`).toBeGreaterThan(100);
  expect(metrics!.statusHeight, `${label} status must not become a page-height column`).toBeLessThan(320);
  expect(metrics!.documentWidth, `${label} document should not horizontally overflow`).toBeLessThanOrEqual(metrics!.viewportWidth + 1);
  expect(metrics!.bodyWidth, `${label} body should not horizontally overflow`).toBeLessThanOrEqual(metrics!.viewportWidth + 1);
  return metrics!;
}

async function writeEvidence(name: string, payload: Record<string, unknown>): Promise<void> {
  writeFileSync(artifactPath(name), JSON.stringify(payload, null, 2) + "\n", "utf8");
}

test.describe("@quality enterprise Page CMS editor polish", () => {
  test.describe.configure({ mode: "serial", timeout: 120_000 });

  test("Page CMS detail markup keeps primary controls and long content reachable", async ({ page }, testInfo: TestInfo) => {
    const evidence: Record<string, unknown> = { test: testInfo.title, viewports: {} };
    for (const viewport of viewports) {
      await mountFixture(page);
      await assertPageDetailViewport(page, viewport, "initial");
      const initialScreenshot = await capture(page, `page-detail-initial-${viewport.name}.png`);

      const heading = page.locator("[data-content-block-id='hero-heading'] input").first();
      await heading.focus();
      await page.keyboard.press("End");
      await page.keyboard.type(" — edited");
      await expect(heading).toBeFocused();
      await expect(page.locator("#body")).toHaveValue(/edited/);
      await page.locator("[data-content-block-id='hero-heading'] [data-content-editor-action='move-down']").click();
      await expect(page.locator("[data-content-block-id='hero-heading'] [data-content-drag-handle]")).toBeFocused();
      await assertNoPageOverflow(page, `${viewport.name} interaction`);
      const interactionScreenshot = await capture(page, `page-detail-interaction-${viewport.name}.png`);
      (evidence.viewports as Record<string, unknown>)[viewport.name] = {
        initialScreenshot,
        interactionScreenshot,
        width: viewport.width,
        height: viewport.height,
      };
    }
    await writeEvidence("responsive-evidence.json", evidence);
  });

  test("forced colors, reduced motion, and keyboard focus preserve component states", async ({ page }, testInfo: TestInfo) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.emulateMedia({ forcedColors: "active", reducedMotion: "reduce" });
    await mountFixture(page);

    const primary = page.locator("[data-content-editor-primary-actions] .button").first();
    await primary.focus();
    const forcedColorCapabilities = await page.evaluate(() => {
      const forcedColorsMedia = window.matchMedia("(forced-colors: active)");
      return {
        forcedColorAdjustSupported: CSS.supports("forced-color-adjust", "auto"),
        forcedColorsMediaQuery: forcedColorsMedia.media,
        forcedColorsMediaMatches: forcedColorsMedia.matches,
      };
    });
    const forcedColorState = await page.evaluate(() => {
      const primary = document.querySelector("[data-content-editor-primary-actions] .button");
      const selected = document.querySelector("[data-content-block-id='hero-heading']");
      const focus = document.activeElement;
      if (!(primary instanceof HTMLElement) || !(selected instanceof HTMLElement) || !(focus instanceof HTMLElement)) return null;
      const primaryStyle = getComputedStyle(primary);
      const selectedStyle = getComputedStyle(selected);
      const focusStyle = getComputedStyle(focus);
      return {
        primaryColor: primaryStyle.color,
        primaryBackground: primaryStyle.backgroundColor,
        primaryForcedColorAdjust: primaryStyle.forcedColorAdjust,
        selectedForcedColorAdjust: selectedStyle.forcedColorAdjust,
        focusOutlineStyle: focusStyle.outlineStyle,
        focusOutlineWidth: focusStyle.outlineWidth,
      };
    });
    const engine = testInfo.project.name;
    const unsupportedForcedColorFeatures = [
      ...(!forcedColorCapabilities.forcedColorAdjustSupported
        ? ["forced-color-adjust CSS property is unsupported"]
        : []),
      ...(!forcedColorCapabilities.forcedColorsMediaMatches
        ? ["forced-colors active media emulation was not observed after request"]
        : []),
    ];
    await writeEvidence(`forced-colors-capability-${engine}.json`, {
      test: testInfo.title,
      engine,
      requested: { forcedColors: "active", reducedMotion: "reduce" },
      forcedColorCapabilities,
      unsupportedForcedColorFeatures,
      note: "A missing WebKit capability is recorded explicitly; it does not skip the rest of this quality test.",
    });
    expect(forcedColorState).toBeTruthy();
    if (engine !== "webkit") {
      expect(forcedColorCapabilities.forcedColorAdjustSupported, `${engine} should support forced-color-adjust`).toBe(true);
      expect(forcedColorCapabilities.forcedColorsMediaMatches, `${engine} should activate forced-colors emulation`).toBe(true);
    }
    if (forcedColorCapabilities.forcedColorAdjustSupported) {
      expect(forcedColorState!.primaryForcedColorAdjust).toBe("auto");
      expect(forcedColorState!.selectedForcedColorAdjust).toBe("auto");
    }
    expect(forcedColorState!.focusOutlineStyle).not.toBe("none");
    expect(parseFloat(forcedColorState!.focusOutlineWidth)).toBeGreaterThanOrEqual(2);

    const transitionState = await page.evaluate(() => [
      ".button",
      ".content-block",
      ".media-picker__asset",
      ".media-list-editor__item",
    ].map((selector) => {
      const element = document.querySelector<HTMLElement>(selector);
      if (!element) return { selector, found: false, duration: "" };
      return { selector, found: true, duration: getComputedStyle(element).transitionDuration };
    }));
    expect(transitionState.every((entry) => entry.found)).toBe(true);
    for (const entry of transitionState) {
      expect(entry.duration, `${entry.selector} reduced-motion transition`).toMatch(/^(0s)(, 0s)*$/);
    }

    const screenshot = await capture(page, "forced-colors-reduced-motion-page-detail-1280.png");
    await writeEvidence("accessibility-evidence.json", {
      test: testInfo.title,
      screenshot,
      engine,
      forcedColorCapabilities,
      unsupportedForcedColorFeatures,
      forcedColorState,
      transitionState,
    });
  });

  test("dense media picker, gallery list, long URLs, and empty state remain contained", async ({ page }, testInfo: TestInfo) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await mountFixture(page, []);

    const empty = page.locator("[data-content-editor-empty]");
    await expect(empty).toBeVisible();
    await expect(empty).toHaveText("No blocks yet.");
    const heroPicker = page.locator("#metaImageUrl + .media-picker");
    await expect(heroPicker).toBeVisible();
    await heroPicker.getByRole("button", { name: "Choose media" }).click();
    const heroPanel = heroPicker.locator(".media-picker__panel");
    await expect(heroPanel).toBeVisible();
    await expect(heroPanel.locator(".media-picker__asset")).toHaveCount(24);
    await assertNoPageOverflow(page, "dense media 1280");
    const denseScreenshot = await capture(page, "media-picker-dense-empty-1280.png");

    await page.setViewportSize({ width: 390, height: 844 });
    await page.evaluate(() => window.scrollTo(0, 0));
    await assertNoPageOverflow(page, "dense media 390");
    await expect(heroPanel).toBeVisible();
    const narrowBoxes = await page.locator(".media-picker__asset").evaluateAll((assets) => assets.slice(0, 6).map((asset) => {
      const rect = asset.getBoundingClientRect();
      return { left: rect.left, right: rect.right, width: rect.width };
    }));
    expect(narrowBoxes.length).toBeGreaterThan(0);
    for (const box of narrowBoxes) {
      expect(box.left).toBeGreaterThanOrEqual(-1);
      expect(box.right).toBeLessThanOrEqual(391);
      expect(box.width).toBeGreaterThan(0);
    }
    const narrowScreenshot = await capture(page, "media-picker-dense-empty-390.png");
    await page.evaluate(() => {
      document.documentElement.style.zoom = "1.25";
    });
    await assertNoPageOverflow(page, "dense media 390 zoom");
    const zoomScreenshot = await capture(page, "media-picker-dense-empty-390-zoom.png");
    const emptyText = await empty.textContent();
    const assetCount = await heroPanel.locator(".media-picker__asset").count();

    await mountFixture(page);
    const gallery = page.locator("[data-content-block-id='gallery'] .media-list-editor");
    await expect(gallery).toBeVisible();
    await expect(gallery.locator(".media-list-editor__item")).toHaveCount(2);
    const initialGalleryItemCount = await gallery.locator(".media-list-editor__item").count();
    const galleryItem = gallery.locator(".media-list-editor__item").first();
    await expect(galleryItem.locator("small")).toContainText("/media/enterprise/");
    await assertNoPageOverflow(page, "gallery list 390");
    await gallery.scrollIntoViewIfNeeded();
    const galleryScreenshot = await capture(page, "media-list-gallery-390.png");

    const addTrigger = gallery.locator(".media-list-editor__actions .media-picker__trigger");
    await addTrigger.click();
    const galleryPanel = gallery.locator(".media-picker__panel");
    await expect(galleryPanel).toBeVisible();
    await galleryPanel.locator(".media-picker__asset").first().click();
    await expect(gallery.locator(".media-list-editor__item")).toHaveCount(3);
    await addTrigger.click();
    const selectedGalleryAsset = galleryPanel.locator(".media-picker__asset[data-media-asset-selected='true']");
    await expect(selectedGalleryAsset).toHaveCount(2);
    await expect(selectedGalleryAsset.first()).toHaveCSS("transform", /matrix/);
    const selectedGalleryScreenshot = await capture(page, "media-list-gallery-selected-390.png");
    await galleryPanel.press("Escape");
    await expect(galleryPanel).toBeHidden();

    while (await gallery.locator(".media-list-editor__item").count() > 0) {
      await gallery.locator(".media-list-editor__item").last().locator("button[data-media-list-action='remove']").click();
    }
    await expect(gallery.locator(".media-list-editor__items .empty")).toHaveText("No images selected.");
    await expect(gallery.locator(".media-list-editor__item")).toHaveCount(0);
    await expect(addTrigger).toBeFocused();
    await addTrigger.click();
    await expect(galleryPanel.locator(".media-picker__asset[data-media-asset-selected='true']")).toHaveCount(0);
    await galleryPanel.press("Escape");
    await expect(galleryPanel).toBeHidden();
    await gallery.scrollIntoViewIfNeeded();
    const galleryEmptyScreenshot = await capture(page, "media-list-gallery-empty-390.png");
    await writeEvidence("media-empty-evidence.json", {
      test: testInfo.title,
      denseScreenshot,
      narrowScreenshot,
      zoomScreenshot,
      galleryScreenshot,
      selectedGalleryScreenshot,
      galleryEmptyScreenshot,
      emptyText,
      assetCount,
      initialGalleryItemCount,
      galleryItemCountAfterRemoval: await gallery.locator(".media-list-editor__item").count(),
      narrowBoxes,
    });
  });

  test("structured conflict diagnostics remain readable and recoverable at desktop and phone widths", async ({ page }, testInfo: TestInfo) => {
    const conflictMessage = "This page changed in another editor while your draft was being prepared. Your local changes remain available; compare the current version before choosing which content to keep, then retry the save when the revision is current.";
    const revisionMessage = "The marked revision is no longer current. Open the current version in a new tab to compare the server copy with this local draft before saving again.";
    const statusMetrics: Record<string, Record<string, number>> = {};
    let requestCount = 0;
    const requestTokens: string[][] = [];
    await page.route("http://127.0.0.1:4173/admin/pages/page_1/__actions/save", async (route) => {
      requestCount += 1;
      const body = route.request().postData() || "";
      requestTokens.push((body.match(/name="expectedRevision"[^]*?\r?\n\r?\n([^\r\n]*)/) || [])[1] ? ["present"] : []);
      await route.fulfill({
        status: 409,
        contentType: "application/json",
        body: JSON.stringify({
          ok: false,
          conflict: true,
          message: conflictMessage,
          fieldErrors: { expectedRevision: revisionMessage },
        }),
      });
    });

    for (const viewport of [viewports[1], viewports[2]]) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      const revision = `marked-revision-${viewport.name}`;
      await mountFixtureAtURL(page, fixtureBlocks(), { expectedRevision: revision });
      const root = page.locator("[data-content-editor]");
      const token = page.locator("[data-content-editor-revision='true']");
      await expect(token).toHaveValue(revision);
      await page.getByRole("button", { name: "Save page" }).click();

      await expect(root).toHaveAttribute("data-content-editor-save-state", "conflict");
      const status = root.locator("[data-content-editor-save-status]");
      const errors = root.locator("[data-content-editor-save-errors]");
      const currentVersion = root.locator("[data-content-editor-save-conflict-link]");
      await expect(status).toHaveRole("alert");
      await expect(root.locator("[data-content-editor-save-detail-text]")).toContainText(conflictMessage);
      await expect(errors).toContainText(revisionMessage);
      await expect(errors).toBeFocused();
      await expect(token).toHaveValue(revision);
      await expect(currentVersion).toBeVisible();
      await expect(currentVersion).toHaveAttribute("href", "http://127.0.0.1:4173/editor");
      await expect(currentVersion).toHaveAttribute("target", "_blank");
      await expect(currentVersion).toHaveAttribute("rel", "noopener");
      await expect(currentVersion).toHaveCSS("text-decoration-line", "none");
      await page.keyboard.press("Tab");
      await expect(currentVersion).toBeFocused();
      const focusStyle = await currentVersion.evaluate((element) => {
        const style = getComputedStyle(element);
        return { outlineStyle: style.outlineStyle, outlineWidth: style.outlineWidth };
      });
      expect(focusStyle.outlineStyle, `${viewport.name} conflict recovery link focus`).not.toBe("none");
      expect(parseFloat(focusStyle.outlineWidth), `${viewport.name} conflict recovery link focus width`).toBeGreaterThanOrEqual(2);
      const linkBox = await currentVersion.boundingBox();
      expect(linkBox, `${viewport.name} conflict recovery link should have geometry`).toBeTruthy();
      expect(linkBox!.width, `${viewport.name} conflict recovery link should have a tap target`).toBeGreaterThan(0);
      statusMetrics[viewport.name] = await assertSaveStatusLayout(page, `${viewport.name} structured conflict`);
      await assertNoPageOverflow(page, `${viewport.name} structured conflict`);
      await capture(page, `save-conflict-${viewport.name}.png`);
    }

    await writeEvidence("save-conflict-evidence.json", {
      test: testInfo.title,
      requestCount,
      requestTokens,
      statusMetrics,
      conflictMessage,
      revisionMessage,
      note: "Structured 409 fixture only; no host or customer endpoint was contacted.",
    });
  });

  test("validation errors and create completion keep full messages and the saved-item recovery link", async ({ page }, testInfo: TestInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });
    const validationMessage = "The page could not be saved because one or more fields need attention. Your local edits remain available while you correct the marked values and retry.";
    const titleMessage = "Add a descriptive title so readers and assistive technology can identify this page.";
    const bodyMessage = "Add at least one body block before saving this new page; the draft remains local until the form is valid.";
    const createMessage = "The new page was created while a newer local edit remained in this browser.";
    let requestCount = 0;
    let releaseCreate: (() => void) | undefined;
    await page.route("http://127.0.0.1:4173/admin/pages/create", async (route) => {
      requestCount += 1;
      if (requestCount === 1) {
        await route.fulfill({
          status: 422,
          contentType: "application/json",
          body: JSON.stringify({
            ok: false,
            message: validationMessage,
            fieldErrors: { title: titleMessage, body: bodyMessage },
          }),
        });
        return;
      }
      await new Promise<void>((resolve) => {
        releaseCreate = resolve;
      });
      await route.fulfill({
        status: 201,
        contentType: "application/json",
        body: JSON.stringify({
          ok: true,
          message: createMessage,
          redirect: "/admin/pages/page_1/edit",
        }),
      });
    });

    await mountFixtureAtURL(page, fixtureBlocks(), { action: "/admin/pages/create" });
    const root = page.locator("[data-content-editor]");
    const title = page.locator("#title");
    const status = root.locator("[data-content-editor-save-status]");
    const errors = root.locator("[data-content-editor-save-errors]");
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "failed");
    await expect(status).toHaveRole("alert");
    await expect(root.locator("[data-content-editor-save-detail-text]")).toContainText(validationMessage);
    await expect(errors).toContainText(titleMessage);
    await expect(errors).toContainText(bodyMessage);
    await expect(title).toHaveAttribute("aria-invalid", "true");
    await expect(title).toBeFocused();
    await expect(root.locator("[data-content-editor-save-retry]")).toBeVisible();
    await expect(root.locator("[data-content-editor-save-conflict-link]")).toBeHidden();
    const validationMetrics = await assertSaveStatusLayout(page, "390 validation");
    await assertNoPageOverflow(page, "390 validation");
    const validationScreenshot = await capture(page, "save-validation-390.png");

    await title.fill("A valid title for the new page");
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "saving");
    await page.locator("#title").evaluate((control) => {
      if (!(control instanceof HTMLInputElement)) throw new Error("title control missing");
      control.value = "A newer local title kept after create";
      control.dispatchEvent(new Event("input", { bubbles: true }));
    });
    releaseCreate?.();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "dirty");
    await expect(title).toHaveValue("A newer local title kept after create");
    const createdLink = root.locator("[data-content-editor-save-created-link]");
    await expect(createdLink).toBeVisible();
    await expect(createdLink).toHaveAttribute("href", "http://127.0.0.1:4173/admin/pages/page_1/edit");
    await expect(createdLink).toHaveAttribute("data-content-editor-discard", "true");
    await expect(root.locator("[data-content-editor-save-retry]")).toBeHidden();
    await expect(root.locator("[data-content-editor-save-conflict-link]")).toBeHidden();
    const createdMetrics = await assertSaveStatusLayout(page, "390 create completion");
    await assertNoPageOverflow(page, "390 create completion");
    const createdScreenshot = await capture(page, "save-created-link-390.png");

    await writeEvidence("save-validation-created-evidence.json", {
      test: testInfo.title,
      requestCount,
      validationMetrics,
      createdMetrics,
      validationScreenshot,
      createdScreenshot,
      note: "Structured 422/201 fixtures only; newer title edit was dispatched through the real form while the create response was pending.",
    });
  });

  test("100-block representative sequence records interaction timing without a performance claim", async ({ page }, testInfo: TestInfo) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    const blocks = fixtureBlocks(100);
    const started = performance.now();
    await mountFixture(page, blocks);
    const mounted = performance.now();
    const editor = page.locator("[data-content-editor]");
    await editor.locator("[data-content-block-id='paragraph-50']").click();
    const selected = performance.now();
    await editor.locator("[data-content-block-id='paragraph-50'] [data-content-editor-action='move-up']").click();
    const moved = performance.now();
    await editor.locator("[data-content-editor-search]").fill("representative content block 51");
    const filtered = performance.now();
    await expect(editor.locator("[data-content-block-id]")).toHaveCount(1);
    await editor.locator("[data-content-editor-search-clear]").click();
    const restored = performance.now();
    await expect(editor.locator("[data-content-block-id]")).toHaveCount(100);
    await assertNoPageOverflow(page, "100-block sequence");
    const screenshot = await capture(page, "100-block-interaction.png");
    const timing = {
      test: testInfo.title,
      blockCount: blocks.length,
      mountMs: Number((mounted - started).toFixed(2)),
      selectMs: Number((selected - mounted).toFixed(2)),
      moveMs: Number((moved - selected).toFixed(2)),
      filterMs: Number((filtered - moved).toFixed(2)),
      restoreMs: Number((restored - filtered).toFixed(2)),
      totalMs: Number((restored - started).toFixed(2)),
      screenshot,
      note: "Fixture interaction timings only; not an enterprise performance certification or budget claim.",
    };
    await writeEvidence("100-block-timing.json", timing);
    expect(timing.blockCount).toBe(100);
    expect(timing.totalMs).toBeGreaterThanOrEqual(0);
  });
});
