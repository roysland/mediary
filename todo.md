Prioritized Roadmap
I prioritized these by user impact first, then implementation risk, then dependency order.

Onboarding screens (Highest, needed now)
Reason: biggest immediate UX gain. You already have strong core features (passkeys, trackables, audio), but users need guidance to discover and trust them.

Update/alert system (High, needed soon)
Reason: low-to-medium effort, high communication value after releases, and helps users notice improvements without relearning the app.

Image upload (High, needed soon)
Reason: clear clinical journaling value (for visible symptoms like rashes), practical and concrete, with manageable engineering risk.

Safe database schema upgrade (Medium, partially already solved)
Reason: not a new system to build from scratch. You already have migration infrastructure; the priority is strengthening verification discipline, not redesigning architecture.

Service worker (Medium-Low, defer until cache strategy is ready)
Reason: useful for performance/offline resilience, but easy to introduce stale-asset bugs. Notifications are likely overkill now unless explicitly opt-in and very conservative.

Vector SQLite search (Low for near-term, strategic long-term)
Reason: potentially valuable, but not urgent before core workflows are fully polished and measurable search pain is confirmed.

Vectorized graphs (Low, overkill for near-term)
Reason: depends on mature vector pipeline and introduces interpretation complexity with uncertain immediate user value.

Suggested symptoms from vectors (Low, overkill for near-term)
Reason: should come only after vector quality is validated; otherwise risk of weak or misleading suggestions.

Implementation Plan (Near-Term + Long-Term Split)

Phase 1: Onboarding foundation (Sprint 1-2)
Add multi-step onboarding: passkey explainer, language selection, trackable setup, audio intro, and navigation explanation.
Persist onboarding completion and wire first-login redirect logic.

Phase 2: Update alerts (Sprint 2-3)
Add versioned home alert model + per-user dismissal tracking.
Render a dismissible banner on the home page and keep dismissal permanent for that user/version.

Phase 3: Image upload (Sprint 3-4)
Implement secure multipart image uploads using the same hardening pattern as audio uploads.
Store image metadata, enforce MIME/size validation, and attach images to entries.

Phase 4: Service worker (Candidate defer, post core UX)
Only proceed after asset versioning/cache invalidation is explicit.
Start with static asset caching only. Keep notifications disabled by default.

Phase 5: Vector search R&D to MVP (Long-term)
Pick embedding path (cloud allowed per your preference), add embedding storage/indexing, evaluate retrieval relevance + latency + cost thresholds.

Phase 6: Advanced intelligence UX (Long-term, only if Phase 5 succeeds)
Add vector graphs and symptom suggestions as optional features after proving search quality and usefulness.

Needed vs Overkill Summary

Needed now: onboarding, update alerts, image upload.
Needed but mostly process hardening: safe schema upgrades.
Useful later: service worker (careful rollout).
Overkill for now: vector graphs and symptom suggestions.
Strategic later: vector semantic search.
Key Files To Reuse/Modify

onboardingUser.html
routers.go
settings.go
home.go
home.html
voice.go
schema.sql
00008_legacy_transition.sql
queries.sql
main.js
Verification Plan

Run unit/integration tests after each phase with go test ./....
Run first-time-user E2E flow: signup/login -> onboarding -> home -> add trackable -> audio/image entry.
Validate migrations on both fresh DB and existing DB snapshots before release.
For service worker, run forced-refresh/version-bump checks to detect stale-cache regressions early.
For vector phases, define and enforce acceptance gates before rollout: relevance, latency, and monthly cost ceilings.