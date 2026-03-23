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

# Third party libraries


## Lower Priority (Consider Later)


### Input Validation
**Library:** `github.com/go-playground/validator`
**Why:** Currently validation is scattered across handlers (e.g., `requireNonEmpty`, `checkboxToInt64`). This library would:
- Consolidate validation rule definitions
- Reduce boilerplate in handlers
- Provide consistent error messages
**Cost:** Medium (refactoring handler validation logic)
**Assessment:** Current approach is explicit and works well. Worth considering only if validation becomes more complex.
