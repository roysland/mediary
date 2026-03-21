const { test, expect } = require("@playwright/test");

const e2eAuthToken = process.env.PLAYWRIGHT_E2E_AUTH_TOKEN || "playwright-e2e-token";

test.describe("authenticated live server flow", () => {
  test.describe.configure({ mode: "serial" });

  async function login(page, redirectPath = "/") {
    const url = `/auth/e2e/login?token=${encodeURIComponent(e2eAuthToken)}&redirect=${encodeURIComponent(redirectPath)}`;
    await page.goto(url);
    await expect(page).toHaveURL(new RegExp(`${redirectPath.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`));
  }

  async function createEntryFromHome(page, note) {
    await page.goto("/");

    const form = page.locator("form[data-submit-state][action='/entry/add']").first();
    const textarea = form.locator("[data-entry-form-textarea]");
    const submitButton = form.locator("[data-submit-state-button]");

    await textarea.fill(note);
    await submitButton.click();

    await expect(submitButton).toHaveAttribute("data-submit-visual", "loading", { timeout: 5000 });
    await expect(submitButton).toHaveAttribute("data-submit-visual", "success", { timeout: 5000 });
    await expect(textarea).toHaveValue("");
    await expect(submitButton).toHaveAttribute("data-submit-visual", "idle", { timeout: 5000 });
  }

  test("home quick-capture submit-state transitions on live page", async ({ page }) => {
    await login(page, "/");

    const note = `live quick capture ${Date.now()}`;
    await createEntryFromHome(page, note);

    await page.goto("/entries");
    await expect(page.locator(".entries-timeline")).toContainText(note);
  });

  test("entries context actions and deletion timing on live page", async ({ page }) => {
    await login(page, "/");

    const note = `live context menu note ${Date.now()}`;
    await createEntryFromHome(page, note);

    await page.goto("/entries");

    const entry = page.locator(".entry-card-li", { hasText: note }).first();
    await expect(entry).toHaveCount(1);

    const dataId = await entry.getAttribute("data-id");
    expect(dataId).toMatch(/^entry-\d+$/);
    const entryId = dataId.replace("entry-", "");

    await entry.locator(".entry-card__more-btn").click();
    await entry.locator(".edit-entry-button").click();

    const noteDialog = page.locator("#entry-note-dialog");
    await expect(noteDialog).toHaveJSProperty("open", true);
    await expect(noteDialog.locator("[data-entry-form-entry-id]")).toHaveValue(entryId);
    await expect(noteDialog.locator("[data-entry-form-textarea]")).toHaveValue(note);
    await noteDialog.evaluate((dialog) => dialog.close());

    await entry.locator(".entry-card__more-btn").click();
    await entry.locator(".edit-trackables-button").click();

    const trackableDialog = page.locator("#add-quick-trackable-dialog");
    await expect(trackableDialog).toHaveJSProperty("open", true);
    await expect(trackableDialog.locator("[data-trackable-picker]")).toHaveCount(1);
    await trackableDialog.evaluate((dialog) => dialog.close());

    let deleteRequestCount = 0;
    let releaseDelete;
    await page.route(`**/entry/${entryId}/delete`, async (route, request) => {
      deleteRequestCount += 1;
      expect(request.method()).toBe("POST");
      await new Promise((resolve) => {
        releaseDelete = resolve;
      });
      await route.fulfill({ status: 200, body: "" });
    });

    page.once("dialog", (dialog) => dialog.dismiss());
    await entry.locator(".entry-card__more-btn").click();
    await entry.locator(".delete-note-button").click();

    await expect(entry).toHaveCount(1);
    expect(deleteRequestCount).toBe(0);

    page.once("dialog", (dialog) => dialog.accept());
    await entry.locator(".entry-card__more-btn").click();
    await entry.locator(".delete-note-button").click();

    await expect.poll(() => deleteRequestCount).toBe(1);
    await expect(entry).toHaveCount(1);

    releaseDelete();
    await expect(entry).toHaveCount(0);
  });
});
