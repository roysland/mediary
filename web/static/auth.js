function base64UrlToBytes(value) {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
  const padding = normalized.length % 4 === 0 ? "" : "=".repeat(4 - (normalized.length % 4));
  const decoded = atob(normalized + padding);
  return Uint8Array.from(decoded, (char) => char.charCodeAt(0));
}

function bytesToBase64Url(bytes) {
  const uint8 = bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  let binary = "";
  for (let i = 0; i < uint8.length; i++) {
    binary += String.fromCharCode(uint8[i]);
  }
  const value = btoa(binary);
  return value.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function normalizeCreationOptions(options) {
  const normalized = structuredClone(options);
  normalized.publicKey.challenge = base64UrlToBytes(normalized.publicKey.challenge);
  normalized.publicKey.user.id = base64UrlToBytes(normalized.publicKey.user.id);

  if (Array.isArray(normalized.publicKey.excludeCredentials)) {
    normalized.publicKey.excludeCredentials = normalized.publicKey.excludeCredentials.map((credential) => ({
      ...credential,
      id: base64UrlToBytes(credential.id),
    }));
  }

  return normalized;
}

function normalizeRequestOptions(options) {
  const normalized = structuredClone(options);
  normalized.publicKey.challenge = base64UrlToBytes(normalized.publicKey.challenge);

  if (Array.isArray(normalized.publicKey.allowCredentials)) {
    normalized.publicKey.allowCredentials = normalized.publicKey.allowCredentials.map((credential) => ({
      ...credential,
      id: base64UrlToBytes(credential.id),
    }));
  }

  return normalized;
}

function serializeCredential(credential) {
  const response = credential.response;
  const payload = {
    id: credential.id,
    rawId: bytesToBase64Url(credential.rawId),
    type: credential.type,
    response: {},
  };

  if (response.clientDataJSON) {
    payload.response.clientDataJSON = bytesToBase64Url(response.clientDataJSON);
  }
  if (response.attestationObject) {
    payload.response.attestationObject = bytesToBase64Url(response.attestationObject);
  }
  if (response.authenticatorData) {
    payload.response.authenticatorData = bytesToBase64Url(response.authenticatorData);
  }
  if (response.signature) {
    payload.response.signature = bytesToBase64Url(response.signature);
  }
  if (response.userHandle) {
    payload.response.userHandle = bytesToBase64Url(response.userHandle);
  }

  if (typeof credential.getClientExtensionResults === "function") {
    payload.clientExtensionResults = credential.getClientExtensionResults();
  }

  if (credential.authenticatorAttachment) {
    payload.authenticatorAttachment = credential.authenticatorAttachment;
  }

  return payload;
}

async function fetchJSON(url, init = {}) {
  const response = await fetch(url, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers || {}),
    },
  });

  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    const message = payload.error || payload.message || "Request failed";
    throw new Error(message);
  }

  return payload;
}

function setStatus(target, message, isError = false) {
  if (!target) {
    return;
  }

  target.textContent = message;
  target.dataset.error = isError ? "true" : "false";
}

let pendingConditionalLoginController = null;

function abortPendingConditionalLogin() {
  if (!pendingConditionalLoginController) {
    return;
  }

  pendingConditionalLoginController.abort();
  pendingConditionalLoginController = null;
}

function isAbortError(error) {
  if (!error) {
    return false;
  }

  return error.name === "AbortError";
}

async function performRegistration(beginURL, finishURL, displayName = "") {
  const payload = {
    device_name: displayName,
    display_name: displayName,
  };
  const options = await fetchJSON(beginURL, { method: "POST", body: JSON.stringify(payload) });
  const credential = await navigator.credentials.create(normalizeCreationOptions(options));
  if (!credential) {
    throw new Error("Passkey creation was cancelled");
  }

  const result = await fetchJSON(finishURL, {
    method: "POST",
    body: JSON.stringify(serializeCredential(credential)),
  });

  return result;
}

async function performLogin({ conditional = false, signal } = {}) {
  const options = await fetchJSON(`/webauthn/login/options${conditional ? "?conditional=1" : ""}`, { method: "POST", body: "{}" });
  const normalized = normalizeRequestOptions(options);
  const getOptions = { publicKey: normalized.publicKey };
  if (conditional) {
    getOptions.mediation = "conditional";
  }
  if (signal) {
    getOptions.signal = signal;
  }

  const assertion = await navigator.credentials.get(getOptions);
  if (!assertion) {
    throw new Error("Passkey sign-in was cancelled");
  }

  const result = await fetchJSON("/webauthn/login/verify", {
    method: "POST",
    body: JSON.stringify(serializeCredential(assertion)),
  });

  return result;
}

function initAuthPage() {
  const root = document.querySelector("[data-auth-passkey]");
  if (!root) {
    return;
  }

  const registerBtn = root.querySelector("[data-auth-register]");
  const loginBtn = root.querySelector("[data-auth-login]");
  const statusEl = root.querySelector("[data-auth-status]");
  const deviceNameInput = root.querySelector("[data-auth-device-name]");

  if (!window.PublicKeyCredential || !navigator.credentials) {
    setStatus(statusEl, "Passkeys are not supported in this browser", true);
    if (registerBtn) registerBtn.disabled = true;
    if (loginBtn) loginBtn.disabled = true;
    return;
  }

  registerBtn?.addEventListener("click", async () => {
    try {
      abortPendingConditionalLogin();
      if (registerBtn) registerBtn.disabled = true;
      if (loginBtn) loginBtn.disabled = true;
      setStatus(statusEl, "Creating passkey...");
      const deviceName = deviceNameInput?.value?.trim() || "";

      const result = await performRegistration("/webauthn/register/options", "/webauthn/register/verify", deviceName);
      setStatus(statusEl, "Passkey created. Redirecting...");
      window.location.assign(result.redirect || "/");
    } catch (error) {
      setStatus(statusEl, error.message || "Failed to create passkey", true);
    } finally {
      if (registerBtn) registerBtn.disabled = false;
      if (loginBtn) loginBtn.disabled = false;
    }
  });

  loginBtn?.addEventListener("click", async () => {
    try {
      abortPendingConditionalLogin();
      if (registerBtn) registerBtn.disabled = true;
      if (loginBtn) loginBtn.disabled = true;
      setStatus(statusEl, "Waiting for your passkey...");

      const result = await performLogin();
      setStatus(statusEl, "Signed in. Redirecting...");
      window.location.assign(result.redirect || "/");
    } catch (error) {
      setStatus(statusEl, error.message || "Failed to sign in", true);
    } finally {
      if (registerBtn) registerBtn.disabled = false;
      if (loginBtn) loginBtn.disabled = false;
    }
  });

  maybeStartConditionalLogin(statusEl, registerBtn, loginBtn);
}

async function maybeStartConditionalLogin(statusEl, registerBtn, loginBtn) {
  if (typeof window.PublicKeyCredential?.isConditionalMediationAvailable !== "function") {
    return;
  }

  try {
    const supported = await window.PublicKeyCredential.isConditionalMediationAvailable();
    if (!supported) {
      return;
    }

    const controller = new AbortController();
    pendingConditionalLoginController = controller;

    setStatus(statusEl, "Looking for saved passkeys...");
    const result = await performLogin({ conditional: true, signal: controller.signal });
    setStatus(statusEl, "Signed in. Redirecting...");
    window.location.assign(result.redirect || "/");
  } catch (error) {
    if (isAbortError(error)) {
      return;
    }

    // Conditional mediation can fail silently; keep manual buttons available.
    if (registerBtn) registerBtn.disabled = false;
    if (loginBtn) loginBtn.disabled = false;
  } finally {
    pendingConditionalLoginController = null;
  }
}

function initAddPasskey() {
  const button = document.querySelector("[data-auth-add-passkey]");
  const statusEl = document.querySelector("[data-auth-passkey-status]");
  if (!button) {
    return;
  }

  if (!window.PublicKeyCredential || !navigator.credentials) {
    button.disabled = true;
    setStatus(statusEl, "Passkeys are not supported in this browser", true);
    return;
  }

  button.addEventListener("click", async () => {
    try {
      button.disabled = true;
      setStatus(statusEl, "Registering another passkey...");
      await performRegistration("/webauthn/passkeys/options", "/webauthn/passkeys/verify");
      setStatus(statusEl, "Additional passkey registered.");
    } catch (error) {
      setStatus(statusEl, error.message || "Failed to register passkey", true);
    } finally {
      button.disabled = false;
    }
  });
}

function initDeviceLinkCreator() {
  const button = document.querySelector("[data-auth-device-link-create]");
  const panel = document.querySelector("[data-auth-device-link-panel]");
  const qrImage = document.querySelector("[data-auth-device-link-qr]");
  const linkEl = document.querySelector("[data-auth-device-link-url]");
  const statusEl = document.querySelector("[data-auth-device-link-status]");
  if (!button || !panel || !qrImage || !linkEl) {
    return;
  }

  button.addEventListener("click", async () => {
    try {
      button.disabled = true;
      setStatus(statusEl, button.dataset.messageGenerating || "Generating QR code...");

      const result = await fetchJSON("/auth/device-link/create", { method: "POST", body: "{}" });
      qrImage.src = result.qr_data_url || "";
      linkEl.href = result.link_url || "#";
      linkEl.textContent = result.link_url || "";
      panel.hidden = false;
      setStatus(statusEl, button.dataset.messageReady || "QR code ready.");
    } catch (error) {
      setStatus(statusEl, error.message || "Failed to generate device link", true);
    } finally {
      button.disabled = false;
    }
  });
}

function initLinkedDeviceRegistration() {
  const root = document.querySelector("[data-auth-link-passkey]");
  const button = root?.querySelector("[data-auth-link-start]");
  const statusEl = root?.querySelector("[data-auth-link-status]");
  if (!root || !button) {
    return;
  }

  if (!window.PublicKeyCredential || !navigator.credentials) {
    button.disabled = true;
    setStatus(statusEl, "Passkeys are not supported in this browser", true);
    return;
  }

  let inProgress = false;
  const start = async () => {
    if (inProgress) {
      return;
    }

    try {
      inProgress = true;
      button.disabled = true;
      setStatus(statusEl, button.dataset.messageStarting || "Starting passkey registration...");
      const result = await performRegistration("/auth/passkeys/register/options", "/auth/passkeys/register/verify");
      setStatus(statusEl, button.dataset.messageCreated || "Passkey created. Redirecting...");
      window.location.assign(result.redirect || "/");
    } catch (error) {
      setStatus(statusEl, error.message || "Failed to add this device", true);
    } finally {
      inProgress = false;
      button.disabled = false;
    }
  };

  button.addEventListener("click", start);
  if (root.dataset.autostart === "true") {
    void start();
  }
}

initAuthPage();
initAddPasskey();
initDeviceLinkCreator();
initLinkedDeviceRegistration();
