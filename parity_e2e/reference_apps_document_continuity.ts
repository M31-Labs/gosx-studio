import { expect, type Page, type Request } from "@playwright/test";

export interface EditorDocumentContinuityOptions {
  sentinelAttribute?: string;
  sentinelValue?: string;
  settleMs?: number;
}

export interface EditorDocumentContinuityProbe {
  readonly sentinelAttribute: string;
  readonly sentinelValue: string;
  dispose(): void;
  mainDocumentRequests(): readonly string[];
  assertStillOnSameDocument(): Promise<void>;
}

export async function installEditorDocumentContinuityProbe(
  page: Page,
  options: EditorDocumentContinuityOptions = {},
): Promise<EditorDocumentContinuityProbe> {
  const sentinelAttribute = options.sentinelAttribute ?? "data-gosx-studio-document-continuity";
  const sentinelValue = options.sentinelValue ?? `continuity-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  const settleMs = options.settleMs ?? 50;
  const mainFrame = page.mainFrame();
  const documentRequests: string[] = [];
  let disposed = false;

  await page.evaluate(
    ({ attribute, value }) => {
      document.documentElement.setAttribute(attribute, value);
    },
    { attribute: sentinelAttribute, value: sentinelValue },
  );

  const onRequest = (request: Request) => {
    if (disposed) return;
    if (request.resourceType() !== "document") return;
    if (request.frame() !== mainFrame) return;
    documentRequests.push(request.url());
  };

  page.on("request", onRequest);

  return {
    sentinelAttribute,
    sentinelValue,
    dispose() {
      if (disposed) return;
      disposed = true;
      page.off("request", onRequest);
    },
    mainDocumentRequests() {
      return [...documentRequests];
    },
    async assertStillOnSameDocument() {
      await page.waitForTimeout(settleMs);
      const actualSentinel = await page.evaluate((attribute) => {
        return document.documentElement.getAttribute(attribute);
      }, sentinelAttribute);
      expect(documentRequests, "editor main-frame document requests").toEqual([]);
      expect(actualSentinel, "editor document continuity sentinel").toBe(sentinelValue);
    },
  };
}

export async function expectEditorDocumentContinuity<T>(
  page: Page,
  action: () => Promise<T>,
  options: EditorDocumentContinuityOptions = {},
): Promise<T> {
  const probe = await installEditorDocumentContinuityProbe(page, options);
  try {
    const result = await action();
    await probe.assertStillOnSameDocument();
    return result;
  } finally {
    probe.dispose();
  }
}
