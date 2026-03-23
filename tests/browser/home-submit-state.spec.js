const fs = require("node:fs");
const path = require("node:path");
const { test, expect } = require("@playwright/test");

const submitStateScript = fs.readFileSync(
  path.resolve(__dirname, "../../web/static/submit-state.js"),
  "utf8",
);

test("quick-capture submit state transitions and resets form inputs on success", async ({ page }) => {
  await page.setContent(`
    <main>
      <section class="card">
        <form
          method="post"
          action="/entry/add"
          data-submit-state
          onsubmit="event.preventDefault()"
        >
          <textarea id="entry_input" name="entry_input" required data-entry-form-textarea></textarea>
          <label for="is_private-entry">
            <input type="checkbox" id="is_private-entry" name="is_private_entry" role="switch">
            Private
          </label>
          <button
            id="entry_submit_button"
            type="submit"
            data-submit-state-button
            data-submit-visual="idle"
          >
            <span class="submit-layer submit-label">Save</span>
            <span class="submit-layer submit-spinner" aria-hidden="true"></span>
            <span class="submit-layer submit-check" aria-hidden="true"></span>
          </button>
        </form>
      </section>
    </main>
  `);

  await page.addScriptTag({ content: submitStateScript });

  const form = page.locator("form[data-submit-state]");
  const textarea = form.locator("[data-entry-form-textarea]");
  const privateToggle = form.locator("#is_private-entry");
  const submitButton = form.locator("#entry_submit_button");

  await textarea.fill("captured from home quick-capture");
  await privateToggle.check();
  await submitButton.click();

  await expect(submitButton).toHaveAttribute("data-submit-visual", "idle");
  await expect(submitButton).toHaveAttribute("aria-busy", "true");
  await expect(submitButton).toBeDisabled();

  await page.waitForTimeout(220);
  await expect(submitButton).toHaveAttribute("data-submit-visual", "loading");

  await page.evaluate(() => {
    const managedForm = document.querySelector("form[data-submit-state]");
    document.body.dispatchEvent(
      new CustomEvent("htmx:afterRequest", {
        detail: {
          elt: managedForm,
          successful: true,
        },
      }),
    );
  });

  await expect(submitButton).toHaveAttribute("data-submit-visual", "success");
  await expect(submitButton).toBeDisabled();
  await expect(submitButton).not.toHaveAttribute("aria-busy", "true");

  await expect(textarea).toHaveValue("");
  await expect(privateToggle).not.toBeChecked();

  await page.waitForTimeout(900);
  await expect(submitButton).toHaveAttribute("data-submit-visual", "idle");
  await expect(submitButton).toBeEnabled();
});
