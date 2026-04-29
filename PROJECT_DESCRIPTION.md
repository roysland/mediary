# Symptom Tracker — Product & Feature Description

> **Audience:** Graphic designer. This document describes every screen, user flow, and UI element in the app. You have full creative latitude on visual style, but the structure and interaction flows described here must be preserved.

---

## What Is This App?

A **private, low-friction diary and symptom tracker** for people with chronic illness (specifically ME/CFS). The core philosophy is:

- **Minimum effort to log** — quick text entry or voice recording, one tap away.
- **Privacy first** — passwordless authentication (no email, no passwords), local data only, sensitive items hidden by default.
- **Mobile-first** — designed to be used on a phone while lying down or exhausted.

The app is a web app, but it behaves like a mobile app (single-page navigation, bottom nav bar, modal sheets).

---

## Global Layout

The app uses a consistent shell:

```
┌────────────────────────────┐
│        Page Content        │
│                            │
│                            │
│                            │
├────────────────────────────┤
│ Home | Entries | Track | ⚙ │  ← Bottom navigation bar (4 tabs)
└────────────────────────────┘
```

- The **bottom navigation bar** is always visible. Four tabs: **Home**, **Entries**, **Trackables**, **Settings**.
- The active tab is visually highlighted.
- Page content scrolls vertically within the shell.
- There is a top navigation bar in the codebase but it is currently **disabled** (not shown).

### Theming

The app supports **light**, **dark**, and **system** (follows device setting) themes. The theme is set on the root `<html>` element and CSS handles the visual switch automatically. Users choose their theme in Settings.

---

## Feature 1: Authentication (Passkeys / WebAuthn)

### What it is
The auth page is the only page accessible without being logged in. Authentication is **fully passwordless** — users sign in with a device passkey (fingerprint, Face ID, Windows Hello, etc.). There are no passwords and no email addresses.

### Screens

**Single screen: `/auth`**

```
┌────────────────────────────┐
│                            │
│   [App logo / name]        │
│                            │
│   Welcome back.            │
│   Sign in with your        │
│   passkey to continue.     │
│                            │
│   ┌──────────────────────┐ │
│   │  Device name (opt.)  │ │
│   └──────────────────────┘ │
│                            │
│   [  Continue  ]           │  ← Login (existing user)
│   [  I'm new   ]           │  ← Register (new user)
│                            │
│   [Status message area]    │  ← Live feedback (errors, loading…)
└────────────────────────────┘
```

- **Device name** is an optional text field. It labels the passkey being created (e.g. "My iPhone"). Only shown/used during registration.
- **Continue** triggers the browser's built-in passkey UI (native system sheet — fingerprint prompt, Face ID, etc.). This is not designed by us; it is the OS/browser UI.
- **I'm new** registers a brand-new account and passkey.
- The status area shows error messages or loading state using accessible live text.

### Login flow
1. User lands on `/auth`.
2. Taps **Continue**.
3. Browser shows its native passkey picker.
4. User authenticates (fingerprint / Face ID / PIN).
5. App redirects to Home.

### Registration flow
1. User lands on `/auth`.
2. Optionally enters a device name.
3. Taps **I'm new**.
4. Browser shows its native passkey creation UI.
5. User authenticates.
6. Account created, app redirects to Home.

### Notes for designer
- The status area must support both error states (red/warning) and neutral/loading states.
- The two action buttons ("Continue" and "I'm new") should be visually distinct but equally accessible.
- The browser's native passkey UI appears as an overlay — we have zero control over its appearance.

---

## Feature 2: Home — Quick Capture

### What it is
The Home page (`/`) is the primary entry point for logging. It gives users two quick ways to record how they're feeling: **typing a note** or **recording audio**. It is designed for minimal effort.

### Screen layout

```
┌────────────────────────────┐
│   Today — March 20         │  ← Current date header
│                            │
│  ┌──────────────────────┐  │
│  │  How are you today?  │  │  ← Text entry card
│  │  ┌────────────────┐  │  │
│  │  │  [textarea]    │  │  │
│  │  └────────────────┘  │  │
│  │  🔒 Private  [toggle]│  │
│  │  [    Save    ✓    ] │  │  ← Animated submit button
│  └──────────────────────┘  │
│                            │
│  ┌──────────────────────┐  │
│  │  🎙️ Voice entry      │  │  ← Voice recorder card
│  │  [  🎙 Record  ]     │  │  (state: idle)
│  └──────────────────────┘  │
│                            │
├────────────────────────────┤
│ Home | Entries | Track | ⚙ │
└────────────────────────────┘
```

### Text Entry flow
1. User taps the textarea and types their note.
2. Optionally toggles the **Private** switch (hides this entry from default view).
3. Taps **Save**.
4. The button animates: → spinner (after ~180 ms) → checkmark (✓) held for ~800 ms → resets.
5. The textarea clears. No navigation. User stays on Home.

### Voice Entry flow

The voice card has **four visual states**:

**State 1 — Idle:** Big microphone button.

```
┌──────────────────────────┐
│  🎙️  Tap to record       │
│  [  ●  Mic Button  ]     │
└──────────────────────────┘
```

**State 2 — Recording:** Animated recording indicator + elapsed timer + stop button. Maximum 60 seconds; auto-stops.

```
┌──────────────────────────┐
│  🔴  Recording…  00:12   │  ← Animated red dot + MM:SS timer
│  [  ■  Stop  ]           │
└──────────────────────────┘
```

**State 3 — Uploading:** Spinner while audio is being uploaded.

```
┌──────────────────────────┐
│  ⏳  Saving…             │
└──────────────────────────┘
```

**State 4 — Result (success):** Server confirms save; shows a link to the entries page.

```
┌──────────────────────────┐
│  ✅  Saved!              │
│  [  View today's entries ]│
└──────────────────────────┘
```

**State 5 — Error:** Shown if something went wrong.

```
┌──────────────────────────┐
│  ⚠️  Something went wrong │
│  Please try again.       │
└──────────────────────────┘
```

### Notes for designer
- The animated submit button (spinner → checkmark) is a key piece of UX — confirm it's highly visible.
- The private toggle should feel like a lightweight secondary control, not a dominant element.
- The voice card states must be distinct — the user needs to immediately know if they're recording or not.
- Recording state should use a color or animation that clearly communicates "mic is live" (red dot / pulsing animation suggested).

---

## Feature 3: Entries — Diary Log

### What it is
The Entries page (`/entries`) is the main log view. It shows all diary entries for a selected day, with a horizontal 7-day navigation strip at the top so users can browse recent days.

### Screen layout

```
┌────────────────────────────┐
│  ← Mon Tue Wed Thu Fri → │  ← 7-day horizontal scroll strip
│       [Today highlighted]  │
│                            │
│  ┌──────────────────────┐  │
│  │ 🔒  3:42 PM          │  │  ← Entry card
│  │ "Felt very tired,    │  │
│  │  pain in shoulders"  │  │
│  │ ⚡ Energy: 3/10      │  │  ← Trackable chips
│  │ 💊 Meds: Yes         │  │
│  │              [⋯]    │  │  ← Context menu button
│  └──────────────────────┘  │
│                            │
│  ┌──────────────────────┐  │
│  │  10:15 AM            │  │
│  │  🎙️ Transcribing...  │  │  ← Draft / pending transcription
│  └──────────────────────┘  │
│                            │
│               [+]          │  ← Floating Action Button
├────────────────────────────┤
│ Home | Entries | Track | ⚙ │
└────────────────────────────┘
```

### Day navigation strip

- 7 days shown (3 before today, today, 3 after — or similar window).
- Each day tile shows: day name abbreviation (Mon, Tue…) + day number.
- Today is highlighted as current.
- Tapping a day loads entries for that day (HTMX partial update, URL updates too).
- The strip is horizontally scrollable on small screens.

### Entry card anatomy

Each entry is a card containing:

| Element | When shown |
|---|---|
| **Timestamp** | Always. Live-updating relative time ("3 minutes ago", "an hour ago"). |
| **"Logged for [date]" badge** | When entry was recorded on a different date than shown (retroactive entries). |
| **🔒 Private badge** | When `is_private` is true. Hidden until sensitive content filter is ON. |
| **Note text** | The typed or transcribed text. |
| **Trackable chips** | Icon + name + value, one per attached trackable. Sensitive trackables are greyed/hidden unless filter is ON. |
| **Draft badge** | If voice transcription is pending: "🎙️ Transcribing…" or "🎙️ Transcription failed". |
| **Audio player** | `<audio>` control if an audio file was attached. |
| **⋯ context menu** | Always (opens a popover with 3 actions). |

### Swipe-to-delete
- Swipe right on an entry card to reveal a deletion action.
- A background appears behind the card (red / delete icon area — `slot="background"`).
- Swiping past a threshold triggers a confirmation prompt, then removes the entry.

### Context menu (`⋯` popover)
Opens inline, contains three actions:
1. **Edit text** — opens the note editor dialog (pre-filled).
2. **Add trackable** — opens the trackable picker dialog for this entry.
3. **Delete** — triggers confirmation prompt, then deletes.

### Add Entry dialog (FAB / modal sheet)
- Tapping `+` FAB opens a **bottom sheet dialog** from the bottom edge of the screen.
- Inside: same textarea + private toggle + animated save button as on Home.
- On save: dialog resets, entry list refreshes in the background.

```
┌────────────────────────────┐
│ [drag handle]              │
│ Add entry                  │
│ ┌──────────────────────┐  │
│ │  [textarea]          │  │
│ └──────────────────────┘  │
│ 🔒 Private  [toggle]      │
│ [    Save    ]             │
└────────────────────────────┘ ← bottom of screen
```

### Edit entry sheet
Same as add, but pre-filled with existing note text. Save updates the entry.

### Sensitive content filter
A toggle visible on the entries page:
- **OFF (default):** Private entries and sensitive trackable values are hidden or visually suppressed.
- **ON:** Everything is shown. Private entries show their full content; sensitive trackables show their real name and value.
- State is persisted in the browser (localStorage), not the server — it resets per browser.

### Notes for designer
- The 7-day strip + entry list is the densest screen in the app. Entry cards should be scannable at a glance.
- Entry timestamps are the primary label — they should be prominent.
- Trackable chips should be compact and readable inline within the card.
- Draft entries ("Transcribing…") should look visually different / pending — perhaps muted colors or an animation.
- The `⋯` button must be easy to tap on mobile (min 44×44 pt touch target).
- The swipe-to-delete gesture needs a background layer that's revealed on swipe — this is a custom web component.

---

## Feature 4: Trackables — Custom Metrics

### What it is
"Trackables" are user-defined metrics that get attached to diary entries. Examples: "Pain level" (0–10 slider), "Took medication" (yes/no), "Mood notes" (free text). The app ships with a library of preset templates the user can adopt.

### Three types of trackable

| Value type | Control | Example |
|---|---|---|
| **Integer** | Horizontal range slider with live numeric readout | Energy level: 4/10 |
| **Boolean** | Tap-to-toggle button | "Took meds: Yes / No" |
| **Text** | Short text input | "Mood notes: Frustrated" |

---

### Sub-feature 4a: Trackable Picker (`/trackables`)

The full-page picker is also embedded as a dialog (sheet) when adding a trackable to a specific entry.

#### Layout

```
┌────────────────────────────┐
│  Track something           │
│                            │
│  ┌──────────────────────┐  │
│  │  ⚡ Energy level     │  │  ← Trackable item
│  │  ████████░░  7 /10   │  │  ← Integer slider
│  │  [  Saved ✓  ]       │  │
│  └──────────────────────┘  │
│                            │
│  ┌──────────────────────┐  │
│  │  💊 Took medication  │  │
│  │  [  Yes  ]  [  No  ] │  │  ← Boolean toggle
│  └──────────────────────┘  │
│                            │
│  ▶ Dismissed for today (2) │  ← Collapsed section
│                            │
│  [+ Add new trackable]     │
├────────────────────────────┤
│ Home | Entries | Track | ⚙ │
└────────────────────────────┘
```

#### Swipe-to-dismiss
- Swipe a trackable row to the right → it disappears from the main list and moves to a collapsed "Dismissed for today" section.
- Purpose: hide trackables not relevant for today without deleting them.
- A "Restore" button brings it back.

#### Save feedback
- Integer slider: saving happens on release (`pointerup`) or on slider change. An inline `aria-live` region shows "✓ Saved" for 2 seconds.
- Boolean: saves immediately on tap.
- Text: saves on blur or after 1 s of inactivity.

---

### Sub-feature 4b: Add Trackable (`/trackables/add`)

#### Screen layout

```
┌────────────────────────────┐
│  Add Trackable             │
│                            │
│  Name: [______________]    │
│  Sensitive: [□]            │
│                            │
│  ── Or pick a preset ──    │
│  [⚡ Energy] [💊 Meds]    │  ← Scrollable preset buttons
│  [😴 Sleep ] [🤕 Pain ]   │
│  (filtered as user types)  │
│                            │
│  Value type: [integer ▼]   │
│  Icon: [⚡]                │
│  Category: [symptom ▼]     │
│  Min value: [0]            │
│  Max value: [10]           │  ← Only shown for integer type
│                            │
│  ▶ Advanced options        │  ← Collapsible
│     Unit: [____________]   │
│                            │
│  [  Save  ]                │
└────────────────────────────┘
```

#### Presets behavior
- A scrollable row of preset buttons (system-provided templates: Energy level, Pain, Sleep quality, Medication, etc.).
- Clicking a preset auto-fills: name, icon, value type, category, min/max.
- As the user types in the Name field, the preset list filters to show only matching presets.
- If the user manually edits a field that was populated by a preset, the preset link is cleared.

#### Sensitive trackable
- Checking "Sensitive" reveals an additional **Private label** field.
- Private label: an alternate display name shown when the sensitive filter is OFF (e.g. trackable is named "Alcohol intake" but shows as "Beverage" in default view).

#### Field visibility rules
| Field | Visible when |
|---|---|
| Min / Max value | Value type = Integer |
| Unit | Always (in Advanced options) |
| Private label | Sensitive is checked |

#### Categories
Five options: Default, Symptom, Activity, Measurement, State.

---

## Feature 5: Settings (`/settings`)

### Screen layout

```
┌────────────────────────────┐
│  Settings                  │
│                            │
│  Language  [English ▼]     │
│  Theme     [System ▼]      │
│  Screen lock [None ▼]      │
│  Share timer [5 min ▼]     │
│                            │
│  ── Security ──            │
│  [  Add passkey  ]         │
│  [  Add another device ]   │
│  [  Log out  ]             │
│                            │
│  ── Data ──                │
│  [  Export all data  ]     │
│  [  Clear all data  ]      │
├────────────────────────────┤
│ Home | Entries | Track | ⚙ │
└────────────────────────────┘
```

### Preferences

| Setting | Options |
|---|---|
| **Language** | English, Norwegian |
| **Theme** | Light, Dark, System |
| **Screen lock** | None, 1 min, 5 min, 10 min _(setting stored, not yet functional)_ |
| **Share timer** | 5 min, 10 min, 30 min |

Each preference is a `<select>` dropdown. The form has a single **Save** button with the animated spinner→checkmark behavior. Saving does a full-page POST + redirect back to settings.

### Security section

**Add passkey** — triggers the browser's passkey creation UI (same device). Adds an additional credential for this account (useful if you want to add a fingerprint after initially registering with a PIN, etc.).

**Add another device** — opens a panel with a QR code. See "Cross-device linking" below.

**Log out** — ends the session, redirects to `/auth`.

### Cross-device passkey linking

```
┌────────────────────────────┐
│  Add another device        │
│                            │
│  [  Generate QR code  ]    │
│                            │
│  ┌──────────────────────┐  │
│  │  [QR CODE IMAGE]     │  │  ← Appears after button click
│  └──────────────────────┘  │
│  Or copy this link:        │
│  https://app.example/…     │
│                            │
│  Expires in 5 minutes.     │
└────────────────────────────┘
```

Flow:
1. User clicks "Add another device" → QR code appears (expires in 5 minutes).
2. User scans QR with second device's camera.
3. Second device opens a dedicated `/link` page.
4. Second device taps "Register passkey on this device".
5. Browser passkey creation UI runs on the second device.
6. Second device is now enrolled.

**`/link` page (second device)**:

```
┌────────────────────────────┐
│                            │
│  Add this device           │
│                            │
│  You're about to add a     │
│  passkey to your account   │
│  on this device.           │
│                            │
│  [  Register passkey  ]    │
│                            │
└────────────────────────────┘
```

### Data section

**Export all data** — immediately downloads a `.json` file with all of the user's data:
- All diary entries
- All trackable values
- All trackable definitions
- Dismissal history
- Settings
- WebAuthn credential identifiers

**Clear all data** — a destructive action. Flow:
1. User clicks "Clear all data".
2. A **confirmation popover** appears inline on the page.
3. User confirms in the popover.
4. A second `window.confirm()` dialog appears as an extra safeguard.
5. On double confirmation: all data for the user is permanently deleted. Redirect back to settings.

### Notes for designer
- The "Clear all data" action must feel dangerous — use warning colors, a clear "this cannot be undone" message, and the double-confirmation friction is intentional.
- The QR code area should appear smoothly (not jarring) when the button is clicked.
- Preference selects should feel native / lightweight, not heavy form inputs.

---

## Feature 6: Animated Submit Button (universal component)

Used on every form in the app. A single button has three visual states triggered automatically by form submission:

| State | Visual | Duration |
|---|---|---|
| **Idle** | Button label (e.g. "Save") | Until submit |
| **Loading** | Spinner animation | Until response (min ~180 ms delay before appearing) |
| **Success** | Checkmark ✓ | ~800 ms, then resets to Idle |

The form resets (clears all fields) after a successful save unless configured otherwise.

### Notes for designer
- All three states live inside the same button element — only one is visible at a time.
- The success checkmark should feel satisfying and clear.
- The spinner delay (180 ms) prevents a flickering spinner on fast connections — do not remove this.

---

## Feature 7: Relative Timestamps

All entry timestamps use a custom `<time>` element that displays time relative to now:
- "Just now"
- "3 minutes ago"
- "An hour ago"
- "Yesterday at 3:42 PM"

These update automatically every 60 seconds without any page reload.

---

## Feature 8: Transcription (Voice → Text)

When a voice entry is recorded and uploaded:
1. The server creates a "draft" entry immediately.
2. A background worker processes the audio file using Whisper (speech-to-text AI).
3. When complete, the entry is updated with the transcribed text and its draft status is cleared.

### Draft entry states (visible in entries list)

| State | Display |
|---|---|
| `pending` | "🎙️ Transcribing…" badge instead of note text |
| `failed` | "🎙️ Transcription failed" badge |
| `completed` | Normal note text (transcription result) |
| `none` | Normal note text (manually typed, no transcription) |

The user must manually refresh to see transcription results — there is no live updating yet.

---

## Feature 9: International Language Support

The app is available in **English** and **Norwegian**. Language is set in Settings. All user-facing text in templates uses a translation key system — adding a new locale requires only a new translation file.

---

## Feature 10: Onboarding Flow

### What it is
New users are guided through a 5-step onboarding wizard on first login. The flow is skippable at every step. Once completed (or fully skipped), it never appears again.

A "How to get started" button on the auth page links to a public preview of step 1 — no login required.

### Steps

| Step | Content |
|---|---|
| 1 — Passkey explainer | Explains what passkeys are, why they're more secure than passwords, and that the app assumes one device (with a helper for adding more). |
| 2 — Language selection | Pick a language; selection is saved immediately to user settings. |
| 3 — Trackable setup | Browse and add preset trackables inline. |
| 4 — Audio introduction | Explains voice recording and transcription. |
| 5 — Navigation | Explains the difference between the Trackables list and adding a trackable to an entry. |

### Screen layout (shared shell)

```
┌────────────────────────────┐
│  ● ● ○ ○ ○   Step 2 of 5  │  ← Progress indicator
│                            │
│  [Step illustration]       │
│                            │
│  [Step content / form]     │
│                            │
│  [  Skip  ]  [  Next →  ] │
└────────────────────────────┘
```

### Notes for designer
- Each step has a dedicated illustration (see image descriptions in design doc).
- The progress indicator should be subtle — not a dominant element.
- Skip and Next should both be clearly tappable but Next should be the primary action.

---

## Feature 11: Home Page Alert Banner

### What it is
A dismissible informational banner shown on the home page after an app update. Each release can define a new versioned alert message. Once dismissed, it never reappears for that user (even for that version).

```
┌────────────────────────────┐
│  ℹ️  What's new: [message] │
│                    [✕ Dismiss] │
└────────────────────────────┘
```

- The banner is removed from the page immediately on dismiss (no page reload — HTMX swap).
- If no active alert is configured, nothing is shown.
- Dismissing one version has no effect on future alert versions.

### Notes for designer
- Should feel informational, not alarming — use a neutral or soft accent color.
- The dismiss button must be easy to tap on mobile.

---

## Feature 12: Image Attachments on Entries

### What it is
Users can attach images to diary entries to document visible symptoms (e.g. rashes, swelling). This is not a gallery — images are attached per entry and displayed inline.

### Constraints
- Max 2 MB per image (enforced server-side; client-side resize attempted first via Canvas API).
- Accepted formats: JPEG, PNG, WebP, GIF.
- Images are stored locally on the server, never using the user-supplied filename.
- Storage tier is tracked per image to support future migration to object storage.

### Entry card (with image)

```
┌──────────────────────────┐
│  3:42 PM                 │
│  "Rash on left arm"      │
│  ┌────────────────────┐  │
│  │  [image thumbnail] │  │  ← Inline image
│  └────────────────────┘  │
│  ⚡ Energy: 3/10         │
│                   [⋯]   │
└──────────────────────────┘
```

### Add entry form (with image upload)

```
┌────────────────────────────┐
│  [textarea]                │
│  🔒 Private  [toggle]      │
│  📎 [Attach image]         │  ← File input (image types only)
│  [thumbnail preview + ✕]  │  ← Shown after selection
│  [    Save    ]            │
└────────────────────────────┘
```

### Notes for designer
- Image thumbnails in entry cards should be compact — not full-width.
- The attach button should feel secondary to the text area.
- Upload errors (too large, wrong type) should appear inline near the file input.

---

## Feature 13: Secure Share Links

### What it is
Users can generate a one-time share link and short password to give a doctor or carer read-only access to a scoped health report — no account or passkey required on the recipient's end.

### Flow (user side)

1. User opens Settings → Share → "Generate share link".
2. Selects a date range and whether to include private entries.
3. App generates a unique URL and a short 7-character password (e.g. `K7MR2XP`).
4. A QR code is shown alongside the URL and password — **displayed exactly once**.
5. User hands the URL + password to their doctor (verbally, printed, or via QR scan).

### Flow (recipient side)

1. Doctor opens the share URL in any browser.
2. Enters the password.
3. A read-only, print-friendly health report is rendered.
4. The link is immediately invalidated — it cannot be used again.

### Confirmation screen (user, shown once)

```
┌────────────────────────────┐
│  Share link created        │
│                            │
│  ┌──────────────────────┐  │
│  │  [QR CODE]           │  │
│  └──────────────────────┘  │
│  URL: https://app/share/…  │
│  Password: K7MR2XP         │
│                            │
│  ⚠️ Save these now.        │
│  This screen won't repeat. │
│                            │
│  Expires in 30 minutes.    │
└────────────────────────────┘
```

### Password entry form (recipient)

```
┌────────────────────────────┐
│  Health report             │
│                            │
│  Enter the password you    │
│  received to view this     │
│  report.                   │
│                            │
│  [  Password  ]            │
│  [  View report  ]         │
└────────────────────────────┘
```

### Active tokens list (Settings → Share)

```
┌────────────────────────────┐
│  Active share links        │
│                            │
│  Mar 20 – Mar 27           │
│  Expires in 12 min         │
│  [  Revoke  ]              │
└────────────────────────────┘
```

### Security properties
- Token has at least 128 bits of entropy; password is 7 chars from an unambiguous alphanumeric set.
- Only hashes are stored in the database — plaintext token and password are never persisted.
- Links expire after 30 minutes and are single-use.
- Expired or used tokens return 404 — no information leakage.
- All share pages include `X-Robots-Tag: noindex` and `Referrer-Policy: no-referrer` headers.
- The QR code encodes only the URL — the password is never in the QR.

### Notes for designer
- The confirmation screen must feel urgent — "save this now" — without being alarming.
- The password should be displayed in a large, easy-to-read monospace font.
- The report page needs a clean print stylesheet — doctors may print or save as PDF.
- The password form should give no hint about whether a token exists or has expired.

---

## Screen Map

```
/auth                → Auth (login / register)
/onboarding/{step}   → Onboarding wizard (steps 1–5, new users only)
/onboarding/preview  → Public onboarding preview (no login required)
/                    → Home (quick capture)
/entries             → Entries list (day browsing)
/trackables          → Trackable picker (full page)
/trackables/add      → Add new trackable form
/settings            → Settings
/settings/shares     → Active share tokens list
/share/create        → Generate share link form
/share/{token}       → Share report (password-protected, public)
/link?t=<token>      → Cross-device passkey enrollment
```

---

## Key UX Principles (for designer)

1. **Mobile-first, touch-friendly.** Tap targets ≥ 44×44 pt. Swipe gestures on cards.
2. **Minimum friction.** The home page exists specifically to make logging as fast as possible.
3. **Low cognitive load.** The target user may be fatigued or in pain. Avoid cluttered layouts, prefer clear negative space.
4. **Privacy signals.** The private/sensitive toggle and lock icons are reassuring — make them visible but not alarming.
5. **Clear save feedback.** Users with chronic illness need to know their data was saved without doubt. The animated checkmark is essential, not decorative.
6. **No modals for destructive actions without friction.** The double-confirmation for "clear data" is intentional.
7. **Dark mode is a first-class concern.** Many ME/CFS patients are light-sensitive.
