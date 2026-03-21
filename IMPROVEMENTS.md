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


### Database Migrations
**Library:** `github.com/pressly/goose`
**Why:** Current `internal/server/migrations.go` is clunky with embedded SQL functions. These libraries provide:
- Clean separation of migration logic (file-based SQL)
- Better tracking of migration state
- Easier to reason about schema evolution
- No redundant transaction handling code
**Cost:** Low (mostly file restructuring)
**How:** Move SQL from `migrations.go` functions into separate numbered files (`001_base_schema.sql`, etc.). Use goose/migrate CLI to manage state.

## Medium Priority


### Docker/OCI Build
**Library:** Not a library, but consider adding multi-stage Dockerfile
**Why:** Currently no visibility into how this is deployed. Container image would:
- Ensure build reproducibility
- Simplify deployment across environments
- Allow volume mounting for data persistence
**Cost:** Low (standard pattern)
**How:** Add Dockerfile with Go build stage and minimal runtime stage.

## Lower Priority (Consider Later)


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
