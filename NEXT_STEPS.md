## Remaining Follow-ups

- Coordinate git-history cleanup for prior runtime artifacts.
	Why: Artifacts are no longer tracked in current commits, but old history still contains paths such as `server`, `tmp/app`, and `data/app.db`.
	How: Align with collaborators, then run `git filter-repo --path server --path tmp/app --path data/app.db --invert-paths` and force-push with clear migration instructions.

- Add browser-level test for home quick-capture submit-state transitions.
	Why: Server-side tests cover hooks, but they do not verify client-side loading/success/idle state transitions.
	How: Add a Playwright test that submits the home quick-capture form and asserts submit-state attribute transitions and input reset behavior.

- Add browser-level regression test for entries context actions.
	Why: Entries context menu, add-trackable dialog opening, and delete confirmation are browser-driven and can regress without server test failures.
	How: Add a Playwright test that opens the entry context menu, verifies dialog behavior, and confirms deletion updates the DOM only after successful confirmation.

