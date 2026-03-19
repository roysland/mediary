function base64UrlToBytes(value) {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
  const padding = normalized.length % 4 === 0 ? "" : "=".repeat(4 - (normalized.length % 4));
  const decoded = atob(normalized + padding);
  return Uint8Array.from(decoded, (char) => char.charCodeAt(0));
}

function bytesToBase64Url(bytes) {
  const value = btoa(String.fromCharCode(...new Uint8Array(bytes)));
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

async function performLogin({ conditional = false } = {}) {
  const options = await fetchJSON(`/webauthn/login/options${conditional ? "?conditional=1" : ""}`, { method: "POST", body: "{}" });
  const normalized = normalizeRequestOptions(options);
  const getOptions = { publicKey: normalized.publicKey };
  if (conditional) {
    getOptions.mediation = "conditional";
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
      registerBtn.disabled = true;
      loginBtn.disabled = true;
      setStatus(statusEl, "Creating passkey...");
      const deviceName = deviceNameInput?.value?.trim() || "";

      const result = await performRegistration("/webauthn/register/options", "/webauthn/register/verify", deviceName);
      setStatus(statusEl, "Passkey created. Redirecting...");
      window.location.assign(result.redirect || "/");
    } catch (error) {
      setStatus(statusEl, error.message || "Failed to create passkey", true);
    } finally {
      registerBtn.disabled = false;
      loginBtn.disabled = false;
    }
  });

  loginBtn?.addEventListener("click", async () => {
    try {
      registerBtn.disabled = true;
      loginBtn.disabled = true;
      setStatus(statusEl, "Waiting for your passkey...");

      const result = await performLogin();
      setStatus(statusEl, "Signed in. Redirecting...");
      window.location.assign(result.redirect || "/");
    } catch (error) {
      setStatus(statusEl, error.message || "Failed to sign in", true);
    } finally {
      registerBtn.disabled = false;
      loginBtn.disabled = false;
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

    setStatus(statusEl, "Looking for saved passkeys...");
    const result = await performLogin({ conditional: true });
    setStatus(statusEl, "Signed in. Redirecting...");
    window.location.assign(result.redirect || "/");
  } catch (error) {
    // Conditional mediation can fail silently; keep manual buttons available.
    registerBtn.disabled = false;
    loginBtn.disabled = false;
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

initAuthPage();
initAddPasskey();
