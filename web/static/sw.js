const STATIC_ASSET_MANIFEST = Object.freeze([
  "/static/auth.js",
  "/static/dist/mecfs-base.css",
  "/static/dist/mecfs.css",
  "/static/dist/mecfs.js",
  "/static/entries.js",
  "/static/forms.js",
  "/static/image-upload.js",
  "/static/images/onboard_language.png",
  "/static/images/onboard_mic.png",
  "/static/images/onboard_navigation.png",
  "/static/images/onboard_passkey.png",
  "/static/images/onboard_symptoms.png",
  "/static/main.js",
  "/static/security.svg",
  "/static/settings.js",
  "/static/share-print.css",
  "/static/style.css",
  "/static/submit-state.js",
  "/static/trackable-picker.js",
  "/static/trackable-presets.js",
  "/static/voice-recorder.js",
  "/static/sw.js"
]);

const DEFAULT_BUILD_VERSION = "dev";
const versionFromURL = new URL(self.location.href).searchParams.get("v") || "";
const BUILD_VERSION = /^[A-Za-z0-9._-]+$/.test(versionFromURL) ? versionFromURL : DEFAULT_BUILD_VERSION;
const CACHE_NAME = `static-v${BUILD_VERSION}`;

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(STATIC_ASSET_MANIFEST))
  );
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) =>
      Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME) {
            return caches.delete(cacheName);
          }
          return Promise.resolve(false);
        })
      )
    )
  );
  self.clients.claim();
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") {
    return;
  }

  const requestURL = new URL(request.url);
  if (!requestURL.pathname.startsWith("/static/")) {
    return;
  }

  event.respondWith(
    caches.open(CACHE_NAME).then((cache) =>
      cache.match(request).then((cached) => {
        if (cached) {
          return cached;
        }

        return fetch(request).then((response) => {
          if (response && response.ok) {
            cache.put(request, response.clone());
          }
          return response;
        });
      })
    )
  );
});
