# Requirements Document

## Introduction

This document covers the prioritized feature roadmap for the Go-based diary and symptoms tracker web app. The roadmap is split into near-term features (onboarding, update alerts, image upload, safe schema upgrades, service worker) and long-term strategic features (vector semantic search, vector graphs, suggested symptoms). Requirements are ordered by priority as established in the roadmap.

## Glossary

- **App**: The Go-based diary and symptoms tracker web application.
- **User**: An authenticated person using the App via passkey/WebAuthn.
- **Onboarding_Flow**: The multi-step first-login wizard that introduces the App to a new User.
- **Onboarding_State**: The persisted record of whether a User has completed the Onboarding_Flow.
- **Alert**: A versioned, dismissible informational banner shown on the home page after an App update.
- **Alert_Version**: A string identifier (e.g. `"2025-07-01"`) that uniquely identifies a specific Alert.
- **Alert_Dismissal**: A per-user, per-Alert_Version record indicating the User has permanently dismissed that Alert.
- **Image_Upload**: A multipart form submission containing an image file attached to a diary entry.
- **Image_Metadata**: The database record storing file path, MIME type, size, and entry association for an uploaded image.
- **Migration**: A numbered SQL file in `db/migrations/` applied by the goose migration runner at startup.
- **Migration_Verification**: The process of confirming a Migration applies cleanly to both a fresh database and an existing production snapshot.
- **Service_Worker**: A browser-side script registered at the root scope that intercepts network requests for caching.
- **Asset_Cache**: The Service_Worker cache storing versioned static assets (JS, CSS, images).
- **Embedding**: A fixed-length float vector representing the semantic content of a diary entry.
- **Vector_Store**: The SQLite table storing Embeddings alongside their source entry IDs.
- **UMAP**: Uniform Manifold Approximation and Projection — a dimensionality reduction algorithm for visualizing high-dimensional vectors.
- **t-SNE**: t-distributed Stochastic Neighbor Embedding — an alternative dimensionality reduction algorithm.
- **Trackable**: A user-defined or preset measurable dimension (symptom, activity, measurement, or state) that can be attached to entries.
- **Share_Token**: A cryptographically random, single-access token that grants time-limited, read-only access to a generated health report without requiring passkey authentication.
- **Share_Password**: A short (6–8 alphanumeric characters), human-typeable one-time password generated alongside a Share_Token, shown once to the User and never stored in plaintext.
- **Share_Report**: A read-only, printable HTML page rendered from a scoped subset of the User's diary data, accessible only by presenting a valid Share_Token URL and the correct Share_Password.
- **Scope**: The set of constraints attached to a Share_Token, including a permitted date range and privacy filter settings that limit which entry data is visible in the Share_Report.
- **Share_QR**: A QR code encoding the full Share_Token URL (without the password), displayed to the User so a recipient can scan it to open the Share_Report page.

---

## Requirements

### Requirement 1: Multi-Step Onboarding Flow

**User Story:** As a new User, I want a guided onboarding experience after first login, so that I understand passkeys, can select my language, set up trackables, learn about audio recording, and understand navigation before using the App. Our target audience is totally unaware of passkeys and how to use them, and they don't understand that it's safer than passwords. They don't know that passkeys are device specific, and even though we support multiple passkeys, this might be to "high tech" for them to understand. We kindly just say we assume the user only uses one device, but there is a helper for connecting another device.

#### Acceptance Criteria
0. At the login screen, a button should be able to start a "how to get started" onboarding. 
1. WHEN a User logs in for the first time and the Onboarding_State is not complete, THE App SHALL redirect the User to the first step of the Onboarding_Flow before showing the home page.
2. THE Onboarding_Flow SHALL consist of exactly five ordered steps: passkey explainer, language selection, trackable setup, audio introduction, and navigation explanation.
3. WHEN the User is on the passkey explainer step, THE Onboarding_Flow SHALL display an explanation of how passkeys work and why they are more secure than passwords.
4. WHEN the User is on the language selection step, THE Onboarding_Flow SHALL present all supported languages and allow the User to select one, persisting the selection to the User's settings.
5. WHEN the User is on the trackable setup step, THE Onboarding_Flow SHALL display the list of preset Trackables and allow the User to add one or more Trackables to their profile.
6. WHEN the User is on the audio introduction step, THE Onboarding_Flow SHALL explain how to make audio recordings and what happens after a recording is submitted.
7. WHEN the User is on the navigation explanation step, THE Onboarding_Flow SHALL explain the difference between navigating to the Trackables list and adding a Trackable to a diary entry.
8. WHEN the User completes the final step of the Onboarding_Flow, THE App SHALL persist the Onboarding_State as complete for that User.
9. WHEN a User logs in and the Onboarding_State is already complete, THE App SHALL not redirect the User to the Onboarding_Flow.
10. WHEN the User navigates directly to an onboarding URL and the Onboarding_State is already complete, THE App SHALL redirect the User to the home page.
11. THE Onboarding_Flow SHALL allow the User to skip any individual step and proceed to the next step.
12. Create a detailed image description for use for each onboarding screen. 

---

### Requirement 2: Persist Onboarding Completion

**User Story:** As a User, I want my onboarding completion to be remembered, so that I am never shown the onboarding flow again after finishing it.

#### Acceptance Criteria

1. WHEN the User completes or skips all steps of the Onboarding_Flow, THE App SHALL write an Onboarding_State record to the `settings` table using the key `onboarding_complete` with value `1` for that User.
2. THE App SHALL read the Onboarding_State from the `settings` table on every authenticated request to the home page route.
3. IF the `settings` table write for Onboarding_State fails, THEN THE App SHALL log the error and still allow the User to proceed to the home page.

---

### Requirement 3: Versioned Home Page Alert Banner

**User Story:** As a User, I want to see a dismissible alert banner on the home page after an App update, so that I am informed about new or changed features.

#### Acceptance Criteria

1. WHEN the App has an active Alert and the User has not dismissed that Alert_Version, THE App SHALL render a dismissible banner on the home page containing the Alert message.
2. WHEN the User dismisses the banner, THE App SHALL record an Alert_Dismissal for that User and Alert_Version in the database.
3. WHEN the User dismisses the banner, THE App SHALL remove the banner from the page without a full page reload.
4. WHEN the User returns to the home page after dismissing an Alert_Version, THE App SHALL not render the banner for that Alert_Version.
5. THE App SHALL support multiple Alert_Versions over time; dismissing one version SHALL not affect the display of a future Alert_Version.
6. IF no active Alert is configured, THEN THE App SHALL not render any alert banner on the home page.
7. THE Alert message SHALL be configurable without requiring a code change to handler logic (e.g. via a config value or embedded constant per release).

---

### Requirement 4: Secure Image Upload

**User Story:** As a User, I want to attach an image to a diary entry, so that I can document visible symptoms such as rashes. This is not an image gallery, so we don't need insane resolution. We should resize them if they exceed 2mb.
We are on a limited storage VPS. We need a "tier" for the user, where basic tier allows 50 images or 100mb. 
We might need a service that warns when storage space is about to run out. 
We also need to consider block vs object storage for expanding storage.

#### Acceptance Criteria

1. WHEN the User submits an image via the entry form, THE App SHALL accept the image as a multipart form upload.
1.1 Javascript resizes the image to a reasonable size.
2. THE App SHALL enforce a maximum image upload size of 2 MB per file.
3. THE App SHALL only accept images with MIME types `image/jpeg`, `image/png`, `image/webp`, or `image/gif`.
4. IF the uploaded file exceeds 2MB, THEN THE App SHALL return a 400 Bad Request response with a descriptive error message.
5. IF the uploaded file has a disallowed MIME type, THEN THE App SHALL return a 400 Bad Request response with a descriptive error message.
6. THE App SHALL store uploaded images in a dedicated directory separate from audio files, using a filename derived from the User ID and a timestamp, never using the user-supplied filename.
7. THE App SHALL store Image_Metadata (file path, MIME type, original size in bytes, entry ID, user ID, created timestamp) in a dedicated `entry_images` database table.
8. WHEN an image is successfully uploaded and metadata is stored, THE App SHALL associate the image with the diary entry and confirm success to the User.
9. IF the database write for Image_Metadata fails after the file is saved, THEN THE App SHALL delete the saved file and return a 500 error.
10. THE App SHALL use `os.OpenRoot` for all image file writes, following the same directory-confinement pattern used for audio uploads.
11. WHEN an entry with attached images is deleted, THE App SHALL delete the associated image files from disk and remove the Image_Metadata records.



---

### Requirement 5: Safe Database Schema Upgrade

**User Story:** As a developer, I want a reliable and verified migration process, so that schema changes apply cleanly to both fresh databases and existing production databases without data loss.

#### Acceptance Criteria

1. THE App SHALL apply all pending Migrations automatically at startup using the goose migration runner before accepting HTTP traffic.
2. WHEN a new Migration is added, THE Migration SHALL be verified to apply cleanly against a fresh database before being committed.
3. WHEN a new Migration is added, THE Migration SHALL be verified to apply cleanly against a snapshot of the most recent production database schema before being committed.
4. IF a Migration fails at startup, THEN THE App SHALL log the error with the Migration filename and exit with a non-zero status code.
5. THE App SHALL not run Migrations and serve HTTP traffic concurrently; migration MUST complete before the HTTP listener starts.
6. WHERE a Migration involves destructive changes (column removal, table drop), THE Migration SHALL include a comment documenting the reason and confirming data has been backed up or is intentionally discarded.

---

### Requirement 6: Service Worker with Static Asset Caching

**User Story:** As a User, I want the App's static assets to load quickly on repeat visits, so that the interface feels responsive even on slow connections.

#### Acceptance Criteria

1. THE App SHALL register a Service_Worker scoped to the root path `/` on supported browsers.
2. WHEN the Service_Worker is installed, THE Service_Worker SHALL pre-cache all versioned static assets (JS, CSS, and images served from `/static/`).
3. WHEN a cached static asset is requested, THE Service_Worker SHALL serve it from the Asset_Cache without a network request.
4. WHEN a new version of a static asset is deployed, THE Service_Worker SHALL invalidate the previous Asset_Cache entry and fetch the updated asset.
5. THE Service_Worker SHALL not intercept or cache API requests, HTML page responses, audio files, or image uploads.
6. THE Service_Worker SHALL not register or request permission for push notifications by default.
7. WHERE push notifications are added in a future release, THE Service_Worker SHALL only request notification permission after explicit User opt-in.
8. IF the Service_Worker fails to install or activate, THEN THE App SHALL continue to function normally without caching.

---

### Requirement 7: Vector Semantic Search for Entries (Strategic)

**User Story:** As a User, I want to search my diary entries by meaning rather than exact keywords, so that I can find entries describing similar symptoms even when I used different words.

#### Acceptance Criteria

1. THE App SHALL store an Embedding for each diary entry in the Vector_Store after the entry is created or updated.
2. WHEN the User submits a search query, THE App SHALL compute an Embedding for the query and return the entries whose Embeddings are most similar, ranked by cosine similarity.
3. THE App SHALL support a configurable embedding provider (local model or external API) selectable without code changes to the search handler.
4. WHEN the embedding provider is an external API, THE App SHALL not send personally identifiable entry content to the provider without explicit User consent.
5. THE App SHALL enforce a maximum embedding generation latency of 2000ms per entry before logging a warning.
6. IF the embedding provider is unavailable, THEN THE App SHALL fall back to full-text search and indicate to the User that semantic search is temporarily unavailable.
7. THE App SHALL record the embedding model name and version alongside each stored Embedding so that re-indexing can be triggered when the model changes.

---

### Requirement 8: Vector Cluster Visualization (Strategic)

**User Story:** As a User, I want to see a visual cluster map of my diary entries, so that I can identify patterns and groupings in my symptom history.

#### Acceptance Criteria

1. WHERE the Vector_Store contains at least 20 Embeddings for a User, THE App SHALL offer a cluster map visualization on the entries or analytics page.
2. THE App SHALL reduce Embedding dimensions to 2D using either UMAP or t-SNE before rendering the cluster map.
3. WHEN the cluster map is rendered, THE App SHALL display each entry as a point, with points colored or grouped by the dominant Trackable category of that entry.
4. WHEN the User clicks a point on the cluster map, THE App SHALL navigate to or display a preview of that entry.
5. IF fewer than 20 Embeddings exist for a User, THEN THE App SHALL display a message explaining that more entries are needed before the cluster map is available.

---

### Requirement 9: Suggested Trackables from Vectors (Strategic)

**User Story:** As a User, I want the App to suggest Trackables I might find useful based on the semantic content of my entries, so that I can discover relevant symptoms or activities I haven't tracked yet.

#### Acceptance Criteria

1. WHERE the Vector_Store contains at least 10 Embeddings for a User, THE App SHALL generate Trackable suggestions based on the semantic similarity between the User's entry Embeddings and the descriptions of available Trackable templates.
2. WHEN suggestions are generated, THE App SHALL present at most 5 suggested Trackables that the User has not already added.
3. WHEN the User accepts a suggestion, THE App SHALL add the suggested Trackable to the User's profile using the same flow as manual Trackable addition.
4. WHEN the User dismisses a suggestion, THE App SHALL not show that suggestion again for at least 30 days.
5. IF the Vector_Store quality score (mean cosine similarity within clusters) falls below a configurable threshold, THEN THE App SHALL suppress suggestions and log a quality warning rather than surfacing low-confidence recommendations.

---

### Requirement 10: Secure Share Links

**User Story:** As a User, I want to generate a one-time share link and short password for a scoped health report, so that I can hand a URL and password to my doctor who can open the report in their browser, print it, and import it into their journaling system — without requiring the doctor to have an account or use a passkey.

#### Acceptance Criteria

1. WHEN a User requests a share link, THE App SHALL generate a Share_Token with at least 128 bits of cryptographic entropy using a cryptographically secure random source, and a Share_Password of 6–8 alphanumeric characters using a cryptographically secure random source.
2. WHEN a Share_Token is created, THE App SHALL store only a cryptographic hash of the Share_Token and only a cryptographic hash of the Share_Password in the database, never the plaintext values.
3. WHEN a Share_Token is created, THE App SHALL associate it with a Scope defining the permitted date range and privacy filter settings for that token.
4. WHEN a User generates a share link, THE App SHALL allow the User to specify the Scope (date range and privacy filters) before the token is issued.
5. WHEN a share link is generated, THE App SHALL display the Share_Token URL, the Share_QR code, and the Share_Password to the User exactly once; these values SHALL NOT be retrievable again after the User leaves or dismisses the confirmation screen.
6. THE Share_Token URL SHALL follow the path pattern `/share/{token}` on the App domain.
7. THE Share_QR SHALL encode only the full Share_Token URL; the Share_Password SHALL NOT be encoded in the QR code.
8. THE App SHALL set a Share_Token validity duration of 30 minutes from the time of creation.
9. WHEN a recipient opens a Share_Token URL in a browser, THE App SHALL present a password entry form and SHALL NOT render any report content before the correct Share_Password is submitted.
10. WHEN a recipient submits the correct Share_Password for a valid, unexpired Share_Token, THE App SHALL render the Share_Report as a read-only HTML page containing only the entry data permitted by the token's Scope, and SHALL immediately invalidate the Share_Token so it cannot be used again.
11. WHEN a Share_Report is rendered, THE App SHALL include a print-friendly stylesheet so the recipient can print or save the page as a PDF.
12. WHEN a share link is accessed with an expired Share_Token, THE App SHALL return a 404 response and SHALL NOT reveal whether the token ever existed.
13. IF a Share_Token cannot be found after hashing the presented value, THEN THE App SHALL return a 404 response and SHALL NOT reveal whether the token ever existed.
14. IF an incorrect Share_Password is submitted, THEN THE App SHALL return a 401 response and SHALL NOT reveal whether the token exists or is expired.
15. THE App SHALL set the `X-Robots-Tag: noindex` response header on all share link pages to prevent search engine indexing of sensitive health data.
16. THE App SHALL set the `Referrer-Policy: no-referrer` response header on all share link pages to prevent the token URL from leaking via the `Referer` header.
17. WHEN a User views their settings, THE App SHALL display all active (not yet accessed and not yet expired) Share_Tokens for that User, showing the Scope and expiry time of each token.
18. WHEN a User revokes a Share_Token, THE App SHALL immediately invalidate the token so that subsequent requests using that token return a 404 response.
19. THE App SHALL automatically delete Share_Token records from the database after the token has been invalidated (accessed or expired), with a cleanup interval not exceeding 5 minutes.
