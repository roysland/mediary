const fs = require("node:fs");
const path = require("node:path");
const { test, expect } = require("@playwright/test");

const entriesScript = fs.readFileSync(
  path.resolve(__dirname, "../../web/static/entries.js"),
  "utf8",
);

test("entries context actions open dialogs and remove entry only after confirmed successful delete", async ({ page }) => {
  await page.setContent(`
    <main class="container"></main>
    <section class="entries-page" data-sensitive-filter-root>
      <div
        id="entries-i18n"
        data-delete-confirm="Delete this entry?"
        data-delete-failed="Delete failed"
        data-delete-error="Delete error"
        data-entry-add-title="Add entry"
        data-entry-add-text-title="Add text"
        data-entry-edit-text-title="Edit text"
        hidden
      ></div>

      <section class="entries-list-section">
        <ul class="entries-timeline" data-entries-timeline>
          <li class="entry-card-li" data-id="entry-42" data-entry-private="false">
            <article class="entry-card">
              <div class="entry-card__body">
                <p class="entry-card__note">Existing note</p>
                <textarea id="entry-42-note-source" hidden>Existing note</textarea>
                <div class="entry-card__actions">
                  <button
                    id="entry-42-context-button"
                    type="button"
                    class="entry-card__more-btn"
                    popovertarget="entry-42-context"
                  >
                    More
                  </button>
                  <div class="entry-card__context-menu" popover id="entry-42-context" anchor="entry-42-context-button">
                    <button
                      type="button"
                      class="edit-entry-button"
                      data-entry-id="42"
                      data-entry-date="2026-03-20"
                      data-entry-private="false"
                      data-entry-has-note="true"
                      data-entry-note-source="entry-42-note-source"
                    >
                      Edit text
                    </button>
                    <button type="button" class="edit-trackables-button" data-entry-id="42">Add trackable</button>
                    <button type="button" class="delete-note-button" data-entry-id="42">Delete note</button>
                  </div>
                </div>
              </div>
            </article>
          </li>
        </ul>
        <p class="entries-empty" data-entries-empty-state hidden>No entries</p>
      </section>

      <dialog
        id="entry-note-dialog"
        class="sheet"
        data-entry-note-dialog
        data-default-action="/entry/add"
        data-default-entry-date="2026-03-21"
        data-today-str="2026-03-21"
      >
        <div class="entry-note-dialog">
          <h2 data-entry-note-dialog-title>Add entry</h2>
          <form method="post" action="/entry/add" data-entry-form>
            <input type="hidden" name="entry_id" value="" data-entry-form-entry-id>
            <input type="hidden" name="entry_date" value="" data-entry-form-date>
            <p data-entry-form-date-label></p>
            <p data-entry-form-warning hidden>Past day warning</p>
            <label for="is_private-entry">
              <input type="checkbox" id="is_private-entry" name="is_private_entry">
              Private
            </label>
            <textarea id="entry_input" name="entry_input" data-entry-form-textarea></textarea>
          </form>
        </div>
      </dialog>

      <dialog id="add-quick-trackable-dialog" class="sheet">
        <section class="trackable-picker" data-trackable-picker>
          <form class="autosave-form" method="post" action="/trackables/9/add"></form>
        </section>
      </dialog>
    </section>
  `);

  await page.evaluate(() => {
    window.__confirmQueue = [];
    window.__confirmMessages = [];
    window.__fetchCalls = [];
    window.__pendingFetchResolvers = [];

    window.confirm = (message) => {
      window.__confirmMessages.push(String(message));
      if (window.__confirmQueue.length > 0) {
        return Boolean(window.__confirmQueue.shift());
      }
      return true;
    };

    window.fetch = (url, options = {}) => {
      window.__fetchCalls.push({
        url: String(url),
        method: String(options.method || "GET"),
      });

      return new Promise((resolve) => {
        window.__pendingFetchResolvers.push(resolve);
      });
    };

    window.__resolveNextFetch = (ok) => {
      const resolve = window.__pendingFetchResolvers.shift();
      if (!resolve) {
        return false;
      }

      resolve({ ok: Boolean(ok) });
      return true;
    };
  });

  await page.addScriptTag({ content: entriesScript });

  const contextButton = page.locator("#entry-42-context-button");
  const contextMenu = page.locator("#entry-42-context");
  const editEntryButton = page.locator("#entry-42-context .edit-entry-button");
  const editTrackablesButton = page.locator("#entry-42-context .edit-trackables-button");
  const deleteButton = page.locator("#entry-42-context .delete-note-button");

  await contextButton.click();
  await expect(contextMenu).toBeVisible();

  await editEntryButton.click();
  const noteDialog = page.locator("#entry-note-dialog");
  await expect(noteDialog).toHaveJSProperty("open", true);
  await expect(page.locator("[data-entry-note-dialog-title]")).toHaveText("Edit text");
  await expect(page.locator("[data-entry-form-entry-id]")).toHaveValue("42");
  await expect(page.locator("[data-entry-form-textarea]")).toHaveValue("Existing note");
  await noteDialog.evaluate((dialog) => dialog.close());

  await contextButton.click();
  await editTrackablesButton.click();
  const trackableDialog = page.locator("#add-quick-trackable-dialog");
  await expect(trackableDialog).toHaveJSProperty("open", true);
  await expect(page.locator("#add-quick-trackable-dialog input[name='entry_id']")).toHaveValue("42");

  await trackableDialog.evaluate((dialog) => dialog.close());

  await page.evaluate(() => {
    window.__confirmQueue.push(false);
  });
  await contextButton.click();
  await deleteButton.click();

  await expect(page.locator("[data-id='entry-42']")).toHaveCount(1);
  await expect
    .poll(async () => page.evaluate(() => window.__fetchCalls.length))
    .toBe(0);

  await page.evaluate(() => {
    window.__confirmQueue.push(true);
  });
  await contextButton.click();
  await deleteButton.click();

  await expect
    .poll(async () => page.evaluate(() => window.__fetchCalls.length))
    .toBe(1);

  await expect(page.locator("[data-id='entry-42']")).toHaveCount(1);

  await page.waitForTimeout(150);
  await expect(page.locator("[data-id='entry-42']")).toHaveCount(1);

  await page.evaluate(() => {
    window.__resolveNextFetch(true);
  });

  await expect(page.locator("[data-id='entry-42']")).toHaveCount(0);
  await expect
    .poll(async () => page.evaluate(() => window.__confirmMessages[0]))
    .toBe("Delete this entry?");
});
