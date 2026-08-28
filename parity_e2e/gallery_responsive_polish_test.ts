import { readFileSync } from "node:fs";
import { join } from "node:path";
import { expect, test, type Page } from "@playwright/test";

const studioCSS = readFileSync(join(__dirname, "..", "hostruntime", "assets", "studio.css"), "utf8");

// This is the same header-driven metadata contract emitted by
// backoffice.renderBackendResourceTable: every data cell gets its matching
// header text, and the intentionally blank trailing action header gets the
// generic "Actions" label for the stacked presentation.
const galleryHeaders = ["Work", "Year", "Visibility", "Updated", ""];
const galleryColumnLabel = (index: number) => {
  const header = galleryHeaders[index]?.trim() ?? "";
  return header || (index === galleryHeaders.length - 1 ? "Actions" : "");
};
const galleryHeaderMarkup = galleryHeaders
  .map((header) => `<th scope="col">${header}</th>`)
  .join("");
const galleryCellMarkup = (index: number, content: string) =>
  `<td data-label="${galleryColumnLabel(index)}">${content}</td>`;

const validThumbnail =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 8 8'%3E%3Crect width='8' height='8' fill='%23cbb39a'/%3E%3Cpath d='M0 6 3 3l2 2 1-1 2 2v2H0z' fill='%232f5143'/%3E%3C/svg%3E";

const galleryMarkup = (title: string, materials: string, panelWidth?: number) => `<!doctype html>
<html>
  <head>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
      html, body { margin: 0; padding: 0; min-width: 0; }
      *, *::before, *::after { box-sizing: border-box; }
      img { display: block; }
      ${studioCSS}
    </style>
  </head>
  <body>
    <div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-gallery-index-renderer="gosx-studio">
      <section class="admin-heading">
        <p class="kicker">CMS</p>
        <h1>Gallery works</h1>
        <p>Archive pieces, studies, and non-sale surfaces for the public gallery.</p>
      </section>
      <section class="panel"${panelWidth ? ` style="width:${panelWidth}px"` : ""}>
        <table class="data-table">
          <thead><tr>${galleryHeaderMarkup}</tr></thead>
          <tbody>
            <tr>
                  ${galleryCellMarkup(0, `<div class="table-product"><img src="${validThumbnail}" alt="${title}" /><div><strong>${title}</strong><span>${materials}</span></div></div>`)}
              ${galleryCellMarkup(1, "2026")}
              ${galleryCellMarkup(2, "Published")}
              ${galleryCellMarkup(3, `<time datetime="2026-08-22T16:24:00Z" data-viewer-time="datetime">Aug 22, 2026 4:24 PM PDT</time>`)}
              ${galleryCellMarkup(4, `<a href="/admin/gallery/work-1" data-gosx-link="true" tabindex="0">Edit</a>`)}
            </tr>
          </tbody>
        </table>
      </section>
    </div>
  </body>
</html>`;

type Box = { left: number; right: number; top: number; bottom: number; width: number; height: number };

type Geometry = {
  viewportWidth: number;
  scrollWidth: number;
  clientWidth: number;
  tableWidth: number;
  panel: Box;
  row: Box;
  workCell: Box;
  yearCell: Box;
  workInfo: Box;
  titleLines: Box[];
  materialLines: Box[];
  titleOverlapsYear: boolean;
  materialsOverlapsYear: boolean;
  titleFitsWorkCell: boolean;
  materialsFitWorkCell: boolean;
  titleReadableWidth: boolean;
  materialsReadableWidth: boolean;
  nativeTable: boolean;
  tableDisplay: string;
  rowDisplay: string;
  labels: string[];
  labelContent: string[];
  headers: string[];
  headerScopes: string[];
  thumbnailLoaded: boolean;
  thumbnailWidth: number;
  editVisible: boolean;
  editBox: Box;
  pageLevelOverflow: boolean;
};

const measureGalleryRow = async (page: Page): Promise<Geometry> =>
  page.evaluate(() => {
    const table = document.querySelector<HTMLTableElement>("table.data-table");
    const panel = table?.closest<HTMLElement>(".panel");
    const row = table?.tBodies[0]?.rows[0];
    const workCell = row?.cells[0];
    const yearCell = row?.cells[1];
    const product = workCell?.querySelector<HTMLElement>(".table-product");
    const info = product?.querySelector<HTMLElement>(":scope > div");
    const title = info?.querySelector<HTMLElement>("strong");
    const materials = info?.querySelector<HTMLElement>("span");
    const thumbnail = product?.querySelector<HTMLImageElement>(":scope > img");
    const edit = row?.querySelector<HTMLAnchorElement>("a[href='/admin/gallery/work-1']");

    if (!table || !panel || !row || !workCell || !yearCell || !product || !info || !title || !materials || !thumbnail || !edit) {
      throw new Error("actual Gallery table-product markup is incomplete");
    }

    const box = (element: Element): Box => {
      const rect = element.getBoundingClientRect();
      return {
        left: rect.left,
        right: rect.right,
        top: rect.top,
        bottom: rect.bottom,
        width: rect.width,
        height: rect.height,
      };
    };
    const lineBoxes = (element: Element) => {
      const range = document.createRange();
      range.selectNodeContents(element);
      return Array.from(range.getClientRects()).map((rect) => ({
        left: rect.left,
        right: rect.right,
        top: rect.top,
        bottom: rect.bottom,
        width: rect.width,
        height: rect.height,
      }));
    };
    const overlaps = (a: Box, b: Box) =>
      a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top;
    const work = box(workCell);
    const year = box(yearCell);
    const titleLines = lineBoxes(title);
    const materialLines = lineBoxes(materials);
    const titleRect = box(title);
    const materialRect = box(materials);
    const labels = Array.from(row.cells, (cell) => cell.getAttribute("data-label") ?? "");
    const labelContent = Array.from(row.cells, (cell) => getComputedStyle(cell, "::before").content);

    return {
      viewportWidth: window.innerWidth,
      scrollWidth: document.documentElement.scrollWidth,
      clientWidth: document.documentElement.clientWidth,
      tableWidth: box(table).width,
      panel: box(panel),
      row: box(row),
      workCell: work,
      yearCell: year,
      workInfo: box(info),
      titleLines,
      materialLines,
      titleOverlapsYear: titleLines.some((line) => overlaps(line, year)),
      materialsOverlapsYear: materialLines.some((line) => overlaps(line, year)),
      titleFitsWorkCell: titleLines.every((line) => line.left >= work.left - 0.5 && line.right <= work.right + 0.5),
      materialsFitWorkCell: materialLines.every((line) => line.left >= work.left - 0.5 && line.right <= work.right + 0.5),
      titleReadableWidth: titleRect.width >= 8 * parseFloat(getComputedStyle(document.documentElement).fontSize),
      materialsReadableWidth: materialRect.width >= 8 * parseFloat(getComputedStyle(document.documentElement).fontSize),
      nativeTable: table instanceof HTMLTableElement,
      tableDisplay: getComputedStyle(table).display,
      rowDisplay: getComputedStyle(row).display,
      labels,
      labelContent,
      headers: Array.from(table.tHead?.rows[0]?.cells ?? [], (cell) => cell.textContent?.trim() ?? ""),
      headerScopes: Array.from(table.tHead?.rows[0]?.cells ?? [], (cell) => cell.getAttribute("scope") ?? ""),
      thumbnailLoaded: thumbnail.complete && thumbnail.naturalWidth > 0,
      thumbnailWidth: box(thumbnail).width,
      editVisible: !!(edit.offsetWidth || edit.offsetHeight || edit.getClientRects().length),
      editBox: box(edit),
      pageLevelOverflow: document.documentElement.scrollWidth > window.innerWidth || document.body.scrollWidth > window.innerWidth,
    };
  });

const expectNoTextOverflow = (geometry: Geometry, width: number) => {
  expect(geometry.pageLevelOverflow, `page overflow at ${width}px: ${JSON.stringify(geometry)}`).toBe(false);
  expect(geometry.scrollWidth, `document overflow at ${width}px`).toBeLessThanOrEqual(width);
  expect(geometry.clientWidth).toBeLessThanOrEqual(width);
  expect(geometry.row.left).toBeGreaterThanOrEqual(geometry.panel.left - 0.5);
  expect(geometry.row.right).toBeLessThanOrEqual(geometry.panel.right + 0.5);
  expect(geometry.titleOverlapsYear, `title/Year overlap at ${width}px: ${JSON.stringify(geometry)}`).toBe(false);
  expect(geometry.materialsOverlapsYear, `materials/Year overlap at ${width}px: ${JSON.stringify(geometry)}`).toBe(false);
  expect(geometry.titleFitsWorkCell).toBe(true);
  expect(geometry.materialsFitWorkCell).toBe(true);
  expect(geometry.titleReadableWidth, `title collapsed below readable width at ${width}px`).toBe(true);
  expect(geometry.materialsReadableWidth, `materials collapsed below readable width at ${width}px`).toBe(true);
  expect(geometry.thumbnailLoaded).toBe(true);
  expect(geometry.thumbnailWidth).toBeGreaterThanOrEqual(64);
  expect(geometry.editVisible).toBe(true);
  expect(geometry.editBox.right).toBeLessThanOrEqual(width + 0.5);
};

test.describe("Gallery index responsive Work cell", () => {
  test("stacks long Work content at phone widths without losing labels or table headers", async ({ page }, testInfo) => {
    for (const scenario of [
      { width: 390, title: "Pressed wall tile study", materials: "Stoneware, iron wash, clear glaze" },
      { width: 320, title: "A very long gallery work title that must wrap without crossing the Year column", materials: "StonewareMaterialNameThatHasNoWordBreaksAndMustWrapSafely" },
    ]) {
      await page.setViewportSize({ width: scenario.width, height: 844 });
      await page.setContent(galleryMarkup(scenario.title, scenario.materials));
      await expect(page.locator(".table-product > img")).toHaveJSProperty("complete", true);
      const geometry = await measureGalleryRow(page);
      await page.screenshot({ path: testInfo.outputPath(`gallery-${scenario.width}x844.png`), fullPage: true });

      expectNoTextOverflow(geometry, scenario.width);
      expect(geometry.nativeTable).toBe(true);
      expect(geometry.tableDisplay).toBe(scenario.width <= 820 ? "block" : "table");
      expect(geometry.rowDisplay).toBe(scenario.width <= 820 ? "grid" : "table-row");
      expect(geometry.labels).toEqual(["Work", "Year", "Visibility", "Updated", "Actions"]);
      expect(geometry.labelContent.every((content) => content !== "none" && content !== "normal")).toBe(true);
      expect(geometry.headers).toEqual(galleryHeaders);
      expect(geometry.headerScopes).toEqual(["col", "col", "col", "col", "col"]);
      const ariaSnapshot = await page.locator("table.data-table").ariaSnapshot();
      expect(ariaSnapshot).toContain("table");
      expect(ariaSnapshot).toContain("row");
      expect(ariaSnapshot).toContain("columnheader \"Work\"");
      expect(ariaSnapshot).toContain("columnheader \"Year\"");
    }
  });

  test("keeps the Edit action in the natural Tab order and Enter-activatable on mobile", async ({ page }) => {
    await page.setViewportSize({ width: 320, height: 844 });
    await page.setContent(galleryMarkup("Pressed wall tile study", "Stoneware, iron wash, clear glaze"));
    const edit = page.locator("a[href='/admin/gallery/work-1']");
    await expect(edit).toBeVisible();
    await expect(edit).toHaveAttribute("data-gosx-link", "true");
    await expect(edit).toHaveAttribute("tabindex", "0");

    expect(await edit.evaluate((element) => element.tabIndex), "Edit link must remain in the natural tab order").toBe(0);

    // Exercise actual user navigation from the document, rather than treating
    // programmatic focus as a substitute for Tab reachability. The bounded
    // loop allows the fixture to grow another ordinary focusable control while
    // keeping this assertion deterministic.
    let tabbedToEdit = false;
    for (let step = 0; step < 8; step += 1) {
      await page.keyboard.press("Tab");
      if (await edit.evaluate((element) => document.activeElement === element)) {
        tabbedToEdit = true;
        break;
      }
    }
    expect.soft(tabbedToEdit, "Edit link must be reachable by keyboard Tab").toBe(true);

    await edit.evaluate((element) => {
      element.addEventListener("click", (event) => {
        event.preventDefault();
        element.setAttribute("data-keyboard-activated", "true");
      }, { once: true });
    });
    await edit.focus();
    await expect(edit).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(edit).toHaveAttribute("data-keyboard-activated", "true");
    const editBox = await edit.boundingBox();
    expect(editBox).not.toBeNull();
    // Keep a half-CSS-pixel margin for browser subpixel rounding (Firefox can
    // report 39.999992 for a computed 40px target), not for a smaller target.
    expect(editBox?.width ?? 0).toBeGreaterThanOrEqual(39.5);
    expect(editBox?.height ?? 0).toBeGreaterThanOrEqual(32);
  });

  test("keeps Work copy readable in the constrained host-width phone panel", async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 320, height: 844 });
    // The real host candidate measured a 272px panel and a 238px table at
    // this viewport. Keep that inner constraint in the fixture so the Work
    // thumbnail/copy layout cannot pass only at a wider synthetic width.
    await page.setContent(galleryMarkup(
      "A very long gallery work title that must wrap without crossing the Year column",
      "StonewareMaterialNameThatHasNoWordBreaksAndMustWrapSafely",
      272,
    ));
    await expect(page.locator(".table-product > img")).toHaveJSProperty("complete", true);
    const geometry = await measureGalleryRow(page);
    await page.screenshot({ path: testInfo.outputPath("gallery-constrained-320x844.png"), fullPage: true });

    expectNoTextOverflow(geometry, 320);
    expect(geometry.panel.width).toBeCloseTo(272, 1);
    expect(geometry.tableWidth).toBeCloseTo(238, 1);
    expect(geometry.workInfo.width).toBeGreaterThanOrEqual(128);
    expect(geometry.workInfo.width).toBeGreaterThanOrEqual(geometry.thumbnailWidth);
    expect(geometry.workInfo.left).toBeCloseTo(geometry.workCell.left, 1);
    expect(geometry.rowDisplay).toBe("grid");
  });

  for (const width of [1280, 1600]) {
    test(`preserves the desktop Gallery table at ${width}px`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width, height: 844 });
      await page.setContent(galleryMarkup("Pressed wall tile study", "Stoneware, iron wash, clear glaze"));
      const geometry = await measureGalleryRow(page);
      await page.screenshot({ path: testInfo.outputPath(`gallery-${width}x844.png`), fullPage: true });

      expectNoTextOverflow(geometry, width);
      expect(geometry.nativeTable).toBe(true);
      expect(geometry.tableDisplay).toBe("table");
      expect(geometry.rowDisplay).toBe("table-row");
      expect(geometry.labels).toEqual(["Work", "Year", "Visibility", "Updated", "Actions"]);
      expect(geometry.headers).toEqual(galleryHeaders);
    });
  }
});
