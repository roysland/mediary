# Symptom Tracker Roadmap: ME/PEM Optimized

This document outlines the feature implementation plan for the symptom tracker. The core goal is supporting extremely low cognitive load, privacy, and reliability within a Go + SQLite + HTMX architecture.

---

## Phase 1: Core Usability & Capture

### 1. Voice Logging (Speech-to-Text)
**Priority:** High  
**Goal:** Allow users to log symptoms quickly without typing.

* **UX Flow:** Large mic button -> "Listening..." state -> Auto-upload on stop.
* **Key Design Rule:** Do not force immediate transcription correction. Allow "Delayed Refinement."
* **Implementation:** MediaRecorder API + `hx-post`. Server-side Whisper (whisper.cpp) processing.
* **Footgun Alert:** Avoid long loading spinners. Transition to a "Saved as Draft" state immediately after upload while processing occurs in the background.

### 2. Transactional Draft System
**Priority:** High  
**Goal:** Prevent data loss from crashes, session timeouts, or battery failure.

* **Implementation:** HTMX autosave using `hx-trigger="keyup changed delay:2s"`.
* **Recovery:** Prompt the user to restore unsaved drafts upon next login.
* **Footgun Alert:** Implement "Draft UUIDs" to prevent an idle tab on another device from overwriting fresh data via autosave.

### 3. Secure Share Links
**Priority:** High  
**Goal:** Allow medical/caregiver data sharing without bypassing Passkey security.

* **Implementation:** 128-bit entropy tokens with specific scopes (e.g., date ranges, privacy filters).
* **Pre-authorization:** Allow users to generate "Doctor Links" valid for 30 days during "good" periods.
* **Footgun Alert:** Set `X-Robots-Tag: noindex` to prevent search engines from crawling sensitive health links.

Disable referrer leakage. If a shared report links to external resources, the browser could expose the token URL through the Referer header. Adding Referrer-Policy: no-referrer on the share page prevents this. Another small improvement is to ensure tokens are stored hashed in the database rather than plaintext. This follows the same principle as password storage and prevents accidental disclosure through database access.

---

## Phase 2: Stability & Context

### 4. PWA Offline-First Save
**Priority:** High  
**Goal:** Ensure logging works in clinics or hospitals with poor connectivity.

* **Implementation:** Service Worker + IndexedDB queue.
* **Footgun Alert:** Use client-side `created_at` timestamps. If entries arrive out of order once the user is back online, the timeline remains chronologically accurate.

### 5. Privacy Blur (Screen Lock)
**Priority:** Medium  
**Goal:** A visual shield for sensitive data in public spaces.

* **UX:** Blur the UI after inactivity; unblur with a single tap.
* **Constraint:** Pause the inactivity timer if a `<textarea>` is focused.
* **Footgun Alert:** Ensure the "Unblur" button is a large, easy-to-hit target. Do not require a full biometric re-auth for a simple privacy shield.

---

## Phase 3: Optional Refinements

### 7. Low-Energy UI Mode
**Priority:** Medium  
**Goal:** Reduce sensory overload.

* **UX:** Low contrast (soft grays/pastels), no animations, larger line spacing.
* **Footgun Alert:** Serve the preference via cookie to prevent a "flash of high-contrast content" during initial server render.

### 8. Photo Attachments ("Quick Snap")
**Priority:** Low  
**Goal:** Evidence capture (BP monitors, smartwatches) without OCR friction.

* **Implementation:** Simple image upload tied to entry IDs.
* **Footgun Alert:** Resize and compress images client-side before upload to save bandwidth and storage.

---

## Core Principle
**Capture must always be easier than organization.** Priority is given to minimal thinking, fast logging, and reliability under fatigue.