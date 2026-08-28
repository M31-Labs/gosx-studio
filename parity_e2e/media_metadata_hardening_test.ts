import { expect, test, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const runtimeScript = readFileSync(resolve(__dirname, "../mediaruntime/media_runtime.js"), "utf8");

async function installMediaMetadataFixture(page: Page) {
  await page.setContent(`
    <!doctype html>
    <html>
      <head><base href="http://media-contract.test/admin/editor"></head>
      <body>
        <form id="media-form">
          <h1 id="heading">Media picker</h1>
          <datalist id="media-urls">
            <option value="/media/extensionless" label="extensionless" data-media-alt="Extensionless" data-media-content-type="image/png">extensionless</option>
            <option value="/media/wrong.jpg" label="wrong.jpg" data-media-alt="Text file" data-media-content-type="text/plain">wrong.jpg</option>
            <option value="/media/fallback.webp" label="fallback.webp" data-media-alt="Fallback">fallback.webp</option>
            <option value="https://media-contract.test/media/absolute" label="absolute" data-media-content-type="image/jpeg">absolute</option>
            <option value="http://media-contract.test/media/http" label="http-absolute" data-media-content-type="image/png">http-absolute</option>
            <option value="/media/extensionless" label="duplicate" data-media-content-type="image/png">duplicate</option>
            <option value="/media/unknown" label="unknown-image" data-media-content-type="image/x-unknown">unknown-image</option>
            <option value="/media/escaped" label="escaped" data-media-content-type="text/plain&quot; onerror=&quot;window.__metadataEscape = true">escaped</option>
            <option value="javascript:alert(1)" label="javascript">javascript</option>
            <option value="data:image/png;base64,AAAA" label="data">data</option>
            <option value="blob:https://media-contract.test/id" label="blob">blob</option>
            <option value="file:///tmp/cup.jpg" label="file">file</option>
            <option value="mailto:artist@example.test" label="mailto">mailto</option>
            <option value="//cdn.example.test/cup.jpg" label="protocol-relative">protocol-relative</option>
            <option value="///cdn.example.test/cup.jpg" label="triple-slash">triple-slash</option>
            <option value="/media/cup%ZZ.jpg" label="invalid-percent">invalid-percent</option>
            <option value="/media/cup&#10;name.jpg" label="control">control</option>
            <option value="/media/edge-control.jpg&#10;" label="edge-control">edge-control</option>
            <option value="https:///cup.jpg" label="empty-authority">empty-authority</option>
            <option value="https:////host/cup.jpg" label="quad-authority">quad-authority</option>
          </datalist>
          <input id="hero" name="hero" list="media-urls" data-media-alt-target="hero-alt" value="">
          <input id="hero-alt" name="hero-alt" value="">
          <textarea id="images" name="images" data-media-lines-list="media-urls">/media/wrong.jpg | Text entry</textarea>
          <button id="save" type="submit">Save</button>
        </form>
      </body>
    </html>
  `);
  await page.addScriptTag({ content: runtimeScript });
}

test.describe("media metadata and URL hardening", () => {
  test("uses explicit MIME metadata, deduplicates safely, and never previews files as images", async ({ page }) => {
    const imageRequests: string[] = [];
    const pageErrors: string[] = [];
    const tinyPNG = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=", "base64");
    await page.route(/^https?:\/\/media-contract\.test\//, async (route) => {
      await route.fulfill({ status: 200, contentType: "image/png", body: tinyPNG });
    });
    page.on("pageerror", (error) => pageErrors.push(error.message));
    page.on("request", (request) => {
      if (request.resourceType() === "image") imageRequests.push(request.url());
    });
    await installMediaMetadataFixture(page);

    const assets = page.locator("#hero + .media-picker .media-picker__asset");
    await expect(assets).toHaveCount(7);
    const assetByLabel = (label: string) => page.locator(`#hero + .media-picker .media-picker__asset[aria-label="Use ${label}"]`);
    for (const label of ["extensionless", "wrong.jpg", "fallback.webp", "absolute", "http-absolute", "unknown-image", "escaped"]) {
      await expect(assetByLabel(label)).toHaveCount(1);
    }
    for (const label of [
      "duplicate",
      "javascript",
      "data",
      "blob",
      "file",
      "mailto",
      "protocol-relative",
      "triple-slash",
      "invalid-percent",
      "control",
      "edge-control",
      "empty-authority",
      "quad-authority",
    ]) {
      await expect(assetByLabel(label)).toHaveCount(0);
    }

    const extensionless = assetByLabel("extensionless");
    const wrongExtension = assetByLabel("wrong.jpg");
    const fallback = assetByLabel("fallback.webp");
    const absolute = assetByLabel("absolute");
    const httpAbsolute = assetByLabel("http-absolute");
    const unknownImage = assetByLabel("unknown-image");
    const escapedMetadata = assetByLabel("escaped");
    await expect(extensionless.locator("img")).toHaveCount(1);
    await expect(wrongExtension.locator("img")).toHaveCount(0);
    await expect(wrongExtension.locator(".media-picker__file")).toHaveCount(1);
    await expect(fallback.locator("img")).toHaveCount(1);
    await expect(absolute.locator("img")).toHaveCount(1);
    await expect(httpAbsolute.locator("img")).toHaveCount(1);
    await expect(unknownImage.locator("img")).toHaveCount(0);
    await expect(unknownImage.locator(".media-picker__file")).toHaveCount(1);
    await expect(escapedMetadata.locator("img")).toHaveCount(0);
    await expect(escapedMetadata.locator(".media-picker__file")).toHaveCount(1);
    expect(await page.locator("option[onerror]").count()).toBe(0);
    expect(await page.evaluate(() => (window as unknown as { __metadataEscape?: boolean }).__metadataEscape)).toBeUndefined();

    const lineItem = page.locator("#images + .media-list-editor [data-media-item-id]").first();
    await expect(lineItem.locator("img")).toHaveCount(0);
    await expect(lineItem.locator(".media-picker__file")).toHaveCount(1);

    const trigger = page.locator("#hero + .media-picker .media-picker__trigger");
    await trigger.click();
    await wrongExtension.click();
    await expect(page.locator("#hero + .media-picker .media-picker__preview img")).toHaveCount(0);
    await expect(page.locator("#hero + .media-picker .media-picker__preview .media-picker__file")).toHaveCount(1);
    await trigger.click();
    await unknownImage.click();
    await expect(page.locator("#hero + .media-picker .media-picker__preview img")).toHaveCount(0);
    await expect(page.locator("#hero + .media-picker .media-picker__preview .media-picker__file")).toHaveCount(1);
    await page.locator("#hero").evaluate((input) => {
      const control = input as HTMLInputElement;
      control.value = "javascript:alert(1)";
      control.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await expect(page.locator("#hero + .media-picker .media-picker__preview img")).toHaveCount(0);

    await expect.poll(() => Array.from(new Set(imageRequests)).sort()).toEqual([
      "http://media-contract.test/media/extensionless",
      "http://media-contract.test/media/fallback.webp",
      "http://media-contract.test/media/http",
      "https://media-contract.test/media/absolute",
    ]);
    expect(imageRequests.some((url) => url.endsWith("wrong.jpg"))).toBe(false);
    expect(imageRequests.some((url) => url.endsWith("escaped"))).toBe(false);
    expect(pageErrors).toEqual([]);
  });

  test("preserves keyboard picker behavior while rejecting unsupported destinations", async ({ page }) => {
    const tinyPNG = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=", "base64");
    const pageErrors: string[] = [];
    await page.route(/^https?:\/\/media-contract\.test\//, async (route) => {
      await route.fulfill({ status: 200, contentType: "image/png", body: tinyPNG });
    });
    page.on("pageerror", (error) => pageErrors.push(error.message));
    await installMediaMetadataFixture(page);
    await page.locator("#media-form").evaluate((form) => {
      form.addEventListener("submit", (event) => {
        event.preventDefault();
        (window as unknown as { submitCount?: number }).submitCount = ((window as unknown as { submitCount?: number }).submitCount ?? 0) + 1;
      });
    });

    const trigger = page.locator("#hero + .media-picker .media-picker__trigger");
    const search = page.locator("#hero + .media-picker .media-picker__search");
    const extensionless = page.locator("#hero + .media-picker .media-picker__asset[aria-label=\"Use extensionless\"]");
    const heading = page.locator("#heading");
    await heading.click();
    for (let i = 0; i < 8 && !(await trigger.evaluate((node) => document.activeElement === node)); i++) {
      await page.keyboard.press("Tab");
    }
    await expect(trigger).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(search).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(trigger).toBeFocused();

    await page.keyboard.press("Enter");
    await expect(search).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(extensionless).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(page.locator("#hero")).toHaveValue("/media/extensionless");
    await expect(page.locator("#hero")).toBeFocused();
    expect(await page.evaluate(() => (window as unknown as { submitCount?: number }).submitCount ?? 0)).toBe(0);
    expect(pageErrors).toEqual([]);
  });
});
