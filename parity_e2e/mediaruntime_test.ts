import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

declare global {
  interface Window {
    GoSXStudioMediaRuntime: {
      init(root?: Document | Element): void;
      mediaDragType: string;
    };
    revokedURLs?: string[];
  }
}

const runtimeScript = readFileSync(resolve(__dirname, "../mediaruntime/media_runtime.js"), "utf8");

async function appendManagedRuntimeScript(page: import("@playwright/test").Page, id: string) {
  await page.evaluate(({ id, source }) => {
    const script = document.createElement("script");
    script.id = id;
    script.setAttribute("data-gosx-script", "managed");
    script.setAttribute("data-gosx-script-load", "dom");
    script.setAttribute("data-gosx-script-loaded", "pending");
    script.textContent = source;
    document.head.appendChild(script);
  }, { id, source: runtimeScript });
}

async function installMediaRuntime(page: import("@playwright/test").Page, managed = false) {
  await page.setContent(`
    <!doctype html>
    <html>
      <body>
        <form id="media-form">
          <datalist id="product-media-urls">
            <option value="/media/cup.jpg" label="cup.jpg" data-media-alt="Cup alt">cup.jpg</option>
            <option value="/media/vase.webp" label="vase.webp" data-media-alt="Vase alt">vase.webp</option>
            <option value="javascript:alert(1)" label="bad.jpg" data-media-alt="Bad">bad.jpg</option>
          </datalist>
          <input id="hero" name="hero" list="product-media-urls" data-media-alt-target="heroAlt" value="">
          <input id="heroAlt" name="heroAlt" value="">
          <textarea id="imagesText" name="imagesText" data-media-lines-list="product-media-urls"></textarea>
          <div data-media-upload>
            <button type="button" data-media-upload-drop-zone aria-describedby="mediaUploadHelp mediaUploadStatus">
              <strong>Drop one file here</strong>
            </button>
            <input id="file" name="file" type="file" accept="image/png,image/jpeg,.txt" data-media-upload-input>
            <p id="mediaUploadHelp">Dropping only selects.</p>
            <output id="mediaUploadStatus" data-media-upload-status data-media-picker-filename aria-live="polite">No file selected.</output>
            <p data-media-upload-error hidden></p>
            <img data-media-upload-preview hidden>
            <button type="button" data-media-upload-clear data-media-picker-clear hidden>Clear file</button>
          </div>
          <button
            type="button"
            id="focal"
            data-media-focal-preview
            style="width: 200px; height: 100px; position: relative; display: block;"
          >
            <span data-media-focal-marker></span>
          </button>
          <input id="focalX" data-media-focal-x value="0.25">
          <input id="focalY" data-media-focal-y value="0.75">
        </form>
      </body>
    </html>
  `);
  if (managed) await appendManagedRuntimeScript(page, "media-runtime-first");
  else await page.addScriptTag({ content: runtimeScript });
}

async function dropFiles(page: import("@playwright/test").Page, files: Array<{ name: string; type: string; body: string }>) {
  await page.locator("[data-media-upload-drop-zone]").dispatchEvent("drop", {
    dataTransfer: await page.evaluateHandle((payload) => {
      const transfer = new DataTransfer();
      for (const item of payload) {
        transfer.items.add(new File([item.body], item.name, { type: item.type }));
      }
      return transfer;
    }, files),
  });
}

test.describe("GoSXStudioMediaRuntime", () => {
  test("file drop selects one valid file, requires explicit submit, previews, and clears", async ({ page }) => {
    await installMediaRuntime(page);
    let submitted = 0;
    await page.locator("#media-form").evaluate((form) => {
      form.addEventListener("submit", (event) => {
        event.preventDefault();
        (window as unknown as { submitted: number }).submitted = ((window as unknown as { submitted?: number }).submitted ?? 0) + 1;
      });
    });

    await dropFiles(page, [{ name: "cup.jpg", type: "image/jpeg", body: "jpeg-body" }]);

    await expect(page.locator("#mediaUploadStatus")).toContainText("cup.jpg selected");
    await expect(page.locator("[data-media-upload-preview]")).toHaveJSProperty("hidden", false);
    await expect(page.locator("[data-media-upload-preview]")).toHaveAttribute("src", /^blob:/);
    expect(await page.locator("#file").evaluate((input: HTMLInputElement) => input.files?.[0]?.name)).toBe("cup.jpg");
    submitted = await page.evaluate(() => (window as unknown as { submitted?: number }).submitted ?? 0);
    expect(submitted).toBe(0);

    await page.locator("[data-media-upload-clear]").click();
    await expect(page.locator("#mediaUploadStatus")).toHaveText("No file selected.");
    await expect(page.locator("[data-media-upload-preview]")).toBeHidden();
    expect(await page.locator("#file").evaluate((input: HTMLInputElement) => input.files?.length ?? 0)).toBe(0);
  });

  test("invalid and multiple drops are recoverable and do not overwrite the current file", async ({ page }) => {
    await installMediaRuntime(page);
    await dropFiles(page, [{ name: "cup.jpg", type: "image/jpeg", body: "jpeg-body" }]);
    expect(await page.locator("#file").evaluate((input: HTMLInputElement) => input.files?.[0]?.name)).toBe("cup.jpg");

    await dropFiles(page, [{ name: "script.js", type: "text/javascript", body: "alert(1)" }]);
    await expect(page.locator("[data-media-upload-error]")).toContainText("not accepted");
    expect(await page.locator("#file").evaluate((input: HTMLInputElement) => input.files?.[0]?.name)).toBe("cup.jpg");

    await dropFiles(page, [
      { name: "a.jpg", type: "image/jpeg", body: "a" },
      { name: "b.jpg", type: "image/jpeg", body: "b" },
    ]);
    await expect(page.locator("[data-media-upload-error]")).toContainText("one file");
    expect(await page.locator("#file").evaluate((input: HTMLInputElement) => input.files?.[0]?.name)).toBe("cup.jpg");
  });

  test("native file input fallback preserves browser input behavior", async ({ page }) => {
    await installMediaRuntime(page);

    const chooserPromise = page.waitForEvent("filechooser");
    await page.locator("[data-media-upload-drop-zone]").click();
    const chooser = await chooserPromise;
    await chooser.setFiles({
      name: "notes.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("hello"),
    });

    await expect(page.locator("#mediaUploadStatus")).toContainText("notes.txt selected");
    expect(await page.locator("#file").evaluate((input: HTMLInputElement) => ({
      name: input.files?.[0]?.name,
      listAttr: input.getAttribute("accept"),
      stillInForm: input.form?.id,
    }))).toEqual({
      name: "notes.txt",
      listAttr: "image/png,image/jpeg,.txt",
      stillInForm: "media-form",
    });

    const keyboardChooserPromise = page.waitForEvent("filechooser");
    await page.locator("[data-media-upload-drop-zone]").focus();
    await page.keyboard.press("Enter");
    const keyboardChooser = await keyboardChooserPromise;
    await keyboardChooser.setFiles({
      name: "keyboard.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("keyboard"),
    });
    await expect(page.locator("#mediaUploadStatus")).toContainText("keyboard.txt selected");
  });

  test("invalid native picker selection clears stale preview and clear affordance", async ({ page }) => {
    await installMediaRuntime(page);

    await dropFiles(page, [{ name: "cup.jpg", type: "image/jpeg", body: "jpeg-body" }]);
    await expect(page.locator("[data-media-upload-preview]")).toHaveJSProperty("hidden", false);
    await expect(page.locator("[data-media-upload-clear]")).not.toBeHidden();

    await page.locator("#file").setInputFiles({
      name: "script.js",
      mimeType: "text/javascript",
      buffer: Buffer.from("alert(1)"),
    });

    await expect(page.locator("[data-media-upload-error]")).toContainText("not accepted");
    await expect(page.locator("[data-media-upload-preview]")).toBeHidden();
    await expect(page.locator("[data-media-upload-clear]")).toBeHidden();
    await expect(page.locator("#mediaUploadStatus")).toContainText("not accepted");
    expect(await page.locator("#file").evaluate((input: HTMLInputElement) => input.files?.length ?? 0)).toBe(0);
  });

  test("object URLs are revoked when upload roots detach or navigation cleans current previews", async ({ page }) => {
    await installMediaRuntime(page);
    await page.evaluate(() => {
      window.revokedURLs = [];
      const originalRevoke = URL.revokeObjectURL.bind(URL);
      URL.revokeObjectURL = (url: string) => {
        window.revokedURLs?.push(url);
        originalRevoke(url);
      };
    });

    await dropFiles(page, [{ name: "cup.jpg", type: "image/jpeg", body: "jpeg-body" }]);
    const firstURL = await page.locator("[data-media-upload-preview]").getAttribute("src");
    expect(firstURL).toMatch(/^blob:/);

    await page.dispatchEvent("body", "gosx:navigate");
    await expect(page.locator("[data-media-upload-preview]")).toBeHidden();
    await expect.poll(() => page.evaluate(() => window.revokedURLs ?? [])).toContain(firstURL);

    await dropFiles(page, [{ name: "second.jpg", type: "image/jpeg", body: "jpeg-body" }]);
    const secondURL = await page.locator("[data-media-upload-preview]").getAttribute("src");
    expect(secondURL).toMatch(/^blob:/);
    await page.locator("[data-media-upload]").evaluate((root) => root.remove());
    await expect.poll(() => page.evaluate(() => window.revokedURLs ?? [])).toContain(secondURL);
  });

  test("focal pointercancel and Escape restore the drag start values", async ({ page }) => {
    await installMediaRuntime(page);
    await page.locator("#focal").evaluate((node) => {
      (node as HTMLElement).setPointerCapture = () => {};
    });

    await page.locator("#focal").dispatchEvent("pointerdown", { button: 0, pointerId: 1, clientX: 100, clientY: 50 });
    await page.locator("#focal").dispatchEvent("pointermove", { pointerId: 1, clientX: 190, clientY: 10 });
    await expect(page.locator("#focalX")).not.toHaveValue("0.25");
    await expect(page.locator("#focalY")).not.toHaveValue("0.75");
    await page.locator("#focal").dispatchEvent("pointercancel", { pointerId: 1 });
    await expect(page.locator("#focalX")).toHaveValue("0.25");
    await expect(page.locator("#focalY")).toHaveValue("0.75");

    await page.locator("#focal").dispatchEvent("pointerdown", { button: 0, pointerId: 2, clientX: 100, clientY: 50 });
    await page.locator("#focal").dispatchEvent("pointermove", { pointerId: 2, clientX: 180, clientY: 20 });
    await expect(page.locator("#focalX")).not.toHaveValue("0.25");
    await expect(page.locator("#focalY")).not.toHaveValue("0.75");
    await page.locator("#focal").focus();
    await page.keyboard.press("Escape");
    await expect(page.locator("#focalX")).toHaveValue("0.25");
    await expect(page.locator("#focalY")).toHaveValue("0.75");
  });

  test("picker chooses URL, fills alt text, ignores unsafe options, and duplicate init is idempotent", async ({ page }) => {
    await installMediaRuntime(page);

    await page.locator(".media-picker__trigger").first().click();
    const singlePickerAssets = page.locator("#hero + .media-picker .media-picker__asset");
    await expect(singlePickerAssets).toHaveCount(2);
    await singlePickerAssets.filter({ hasText: "cup.jpg" }).click();

    await expect(page.locator("#hero")).toHaveValue("/media/cup.jpg");
    await expect(page.locator("#heroAlt")).toHaveValue("Cup alt");
    await page.evaluate(() => window.GoSXStudioMediaRuntime.init(document));
    await page.dispatchEvent("body", "gosxstudio:content-editor-render");
    await expect(page.locator(".media-picker")).toHaveCount(1);
    await expect(page.locator("#hero + .media-picker")).toHaveCount(1);
    await expect(page.locator("#hero")).toHaveAttribute("list", "product-media-urls");

    await page.evaluate(() => {
      const input = document.createElement("input");
      input.id = "dynamicHero";
      input.setAttribute("list", "product-media-urls");
      document.querySelector("#media-form")?.append(input);
      window.GoSXStudioMediaRuntime.init(input);
    });
    await expect(page.locator("#dynamicHero + .media-picker")).toHaveCount(1);
  });

  test("marks managed scripts loaded and keeps one media runtime across cleanup and new roots", async ({ page }) => {
    await installMediaRuntime(page, true);
    await expect(page.locator("#media-runtime-first")).toHaveAttribute("data-gosx-script-loaded", "true");
    await expect(page.locator("#hero + .media-picker")).toHaveCount(1);
    await page.evaluate(() => {
      const win = window as unknown as { firstMediaRuntime?: unknown; GoSXStudioMediaRuntime?: unknown };
      win.firstMediaRuntime = win.GoSXStudioMediaRuntime;
    });
    await page.evaluate(() => {
      const stats = { documentListeners: 0, windowListeners: 0, observers: 0 };
      const documentAdd = Document.prototype.addEventListener;
      Document.prototype.addEventListener = function (
        type: string,
        listener: EventListenerOrEventListenerObject,
        options?: boolean | AddEventListenerOptions,
      ) {
        stats.documentListeners += 1;
        return documentAdd.call(this, type, listener, options);
      };
      const windowAdd = Window.prototype.addEventListener;
      Window.prototype.addEventListener = function (
        type: string,
        listener: EventListenerOrEventListenerObject,
        options?: boolean | AddEventListenerOptions,
      ) {
        stats.windowListeners += 1;
        return windowAdd.call(this, type, listener, options);
      };
      const OriginalObserver = window.MutationObserver;
      const observed = class extends OriginalObserver {
        constructor(callback: MutationCallback) {
          stats.observers += 1;
          super(callback);
        }
      };
      window.MutationObserver = observed as unknown as typeof MutationObserver;
      (window as unknown as { mediaInstallListenerStats?: typeof stats }).mediaInstallListenerStats = stats;
    });

    await appendManagedRuntimeScript(page, "media-runtime-second");
    await expect(page.locator("#media-runtime-second")).toHaveAttribute("data-gosx-script-loaded", "true");
    expect(await page.evaluate(() => {
      const win = window as unknown as { firstMediaRuntime?: unknown; GoSXStudioMediaRuntime?: unknown };
      return win.firstMediaRuntime === win.GoSXStudioMediaRuntime;
    })).toBe(true);
    expect(await page.evaluate(() => (window as unknown as {
      mediaInstallListenerStats?: { documentListeners: number; windowListeners: number; observers: number };
    }).mediaInstallListenerStats)).toEqual({ documentListeners: 0, windowListeners: 0, observers: 0 });
    await expect(page.locator("#hero + .media-picker")).toHaveCount(1);

    await page.evaluate(() => {
      window.revokedURLs = [];
      const originalRevoke = URL.revokeObjectURL.bind(URL);
      URL.revokeObjectURL = (url: string) => {
        window.revokedURLs?.push(url);
        originalRevoke(url);
      };
    });
    await dropFiles(page, [{ name: "managed.jpg", type: "image/jpeg", body: "jpeg-body" }]);
    const uploadedURL = await page.locator("[data-media-upload-preview]").getAttribute("src");
    expect(uploadedURL).toMatch(/^blob:/);
    const revokedBeforeNavigation = await page.evaluate(() => window.revokedURLs ?? []);
    expect(revokedBeforeNavigation).toHaveLength(1);
    await page.dispatchEvent("body", "gosx:navigate");
    await expect.poll(() => page.evaluate(() => window.revokedURLs ?? [])).toEqual([...revokedBeforeNavigation, uploadedURL]);

    await page.evaluate(() => {
      const root = document.createElement("div");
      root.id = "new-media-root";
      const list = document.createElement("datalist");
      list.id = "new-media-urls";
      const option = document.createElement("option");
      option.value = "/media/new.jpg";
      option.label = "new.jpg";
      list.append(option);
      const input = document.createElement("input");
      input.id = "newMedia";
      input.setAttribute("list", list.id);
      root.append(list, input);
      document.body.append(root);
    });
    await expect(page.locator("#newMedia + .media-picker")).toHaveCount(1);
    const pickerIDs = await page.locator(".media-picker__panel").evaluateAll((panels) => panels.map((panel) => panel.id));
    expect(pickerIDs).toEqual(["media-picker-1", "media-lines-2", "media-picker-3"]);
  });

  test("asset grid exposes draggable Studio media payload", async ({ page }) => {
    await installMediaRuntime(page);
    await page.locator(".media-picker__trigger").first().click();

    const payload = await page.locator(".media-picker__asset", { hasText: "vase.webp" }).first().evaluate((button) => {
      const event = new DragEvent("dragstart", { bubbles: true, cancelable: true, dataTransfer: new DataTransfer() });
      button.dispatchEvent(event);
      return event.dataTransfer?.getData("application/x-gosx-studio-media") ?? "";
    });

    expect(JSON.parse(payload)).toEqual({
      url: "/media/vase.webp",
      alt: "Vase alt",
      label: "vase.webp",
      kind: "webp",
    });
  });
});
