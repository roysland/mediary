# Onboarding
We need to create several onboarding screens.
* Simple explanation of how good and secure passkeys are
* Language selection
* Adding trackables, predefined or custom
* Making audio recordings
* Navigation explanation, difference between just going to trackables, and adding trackable to an entry.

# Service worker
Set up a service worker
* There should be a very high threshold for adding notifications. This is not the kind of app that should disturb the user. However, we do need to encourage keeping the diary. 
* Proper cache of javascript and css and images.

# Idle detection 
There is a setting for screen lock, but it currently does nothing.
As a web app, Safari and Firefox still doesn't support Idle detection. If using this browser, the setting should be disabled.

# CSFR token
There is no XSS guards. Are they needed?

# Goose migrations
Current migration method is clunky. We can surely benefit from a third party library here?

# Third party libraries

## High Priority

### WebAuthn Implementation
**Library:** `github.com/go-webauthn/webauthn`
**Why:** The database schema already supports WebAuthn credentials, but the actual authentication flow is missing. Implementing this from scratch requires:
- Complex cryptographic operations (verifying signatures, handling attestation)
- Understanding COSE keys, P-256 curves, and WebAuthn protocols
- Significant security-related code that's easy to get wrong
**Cost:** Medium to High (but significantly less than custom implementation)
**How:** Replace `internal/auth/auth.go` stub with webauthn library integration. Add registration and authentication endpoints. Reference: https://github.com/go-webauthn/webauthn provides good examples. User is anonymous. The only valid way to authenticate is via passkey. Use WebAuthn.
Prefer platform authenticator. 
Credentials should be discoverable.

We need to support multi-device accounts. If a user has a passkey manager like 1password, I guess there is no problem. But many won't, but might still need to access from another device. This can be solved by scanning a qr-code from an already authenticated device. If the user does this, then we probably need to support multiple passkeys per user. 

### Database Migrations
**Library:** `github.com/pressly/goose` or `github.com/golang-migrate/migrate`
**Why:** Current `internal/server/migrations.go` is clunky with embedded SQL functions. These libraries provide:
- Clean separation of migration logic (file-based SQL)
- Better tracking of migration state
- Easier to reason about schema evolution
- No redundant transaction handling code
**Cost:** Low (mostly file restructuring)
**How:** Move SQL from `migrations.go` functions into separate numbered files (`001_base_schema.sql`, etc.). Use goose/migrate CLI to manage state.

## Medium Priority

### CSRF Protection
**Library:** `net/http.CrossOriginProtection`
**Why:** Directly addresses the "XSS/CSRF guards" question in IMPROVEMENTS.md. Since this is a health tracker, preventing cross-site hijacking of symptom logs is critical.
Modern Defense: Unlike older libraries, this uses Fetch Metadata (Sec-Fetch-Site) and Origin checks.
Zero-Friction: Because it doesn't use tokens, you don't have to manually inject hidden fields into every HTMX fragment or manage token-syncing between the PWA and the server.
Privacy-Aligned: No extra CSRF cookies are needed, reducing the app's tracking footprint.
**Cost:** Low (middleware wrapper around handlers)
**How:** 
1. Wrap your main http.Handler with http.NewCrossOriginProtection.
2. Ensure all state-changing actions (Symptom logs, Auth) use POST/PUT/DELETE.
3. Rejects any cross-origin write request with a 403 Forbidden automatically.

### Docker/OCI Build
**Library:** Not a library, but consider adding multi-stage Dockerfile
**Why:** Currently no visibility into how this is deployed. Container image would:
- Ensure build reproducibility
- Simplify deployment across environments
- Allow volume mounting for data persistence
**Cost:** Low (standard pattern)
**How:** Add Dockerfile with Go build stage and minimal runtime stage.

## Lower Priority (Consider Later)

### Route/Handler Framework
**Current:** `http.ServeMux` (Go standard library)
**Alternatives:** `chi`, `echo`, `gin`
**Assessment:** ServeMux in Go 1.22+ is quite good. Only upgrade if you need:
- Automatic middleware chains across groups of routes
- Structured logging throughout request pipeline
- Built-in error handling patterns
**Current state:** Not needed now. ServeMux handles your simple RESTful routing fine.

### Input Validation
**Library:** `github.com/go-playground/validator`
**Why:** Currently validation is scattered across handlers (e.g., `requireNonEmpty`, `checkboxToInt64`). This library would:
- Consolidate validation rule definitions
- Reduce boilerplate in handlers
- Provide consistent error messages
**Cost:** Medium (refactoring handler validation logic)
**Assessment:** Current approach is explicit and works well. Worth considering only if validation becomes more complex.

### Structured Logging
**Library:** `golang.org/x/exp/slog` (Go 1.21+) or `uber-go/zap`
**Why:** Currently using standard `log` package. Structured logging helps with:
- Better error tracking in production
- Easier log parsing/aggregation
- Context propagation  
**Cost:** Low to Medium (replacing log statements)
**Assessment:** Not urgent. Current logging is fine for MVP. Upgrade when observability becomes important.
