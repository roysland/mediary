# Auth flow
User is anonymous. The only valid way to authenticate is via passkey. Use WebAuthn.
Prefer platform authenticator. 
Credentials should be discoverable.

We need to support multi-device accounts. If a user has a passkey manager like 1password, I guess there is no problem. But many won't, but might still need to access from another device. This can be solved by scanning a qr-code from an already authenticated device. If the user does this, then we probably need to support multiple passkeys per user. 

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