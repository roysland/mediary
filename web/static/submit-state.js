const SPINNER_DELAY_MS = 180;
const SUCCESS_HOLD_MS = 800;
const submitState = new WeakMap();

function getMeta(button) {
  let meta = submitState.get(button);
  if (!meta) {
    meta = {
      mode: "idle",
      spinnerTimer: null,
      resetTimer: null,
      widthLocked: false,
    };
    submitState.set(button, meta);
  }
  return meta;
}

function clearTimers(meta) {
  if (meta.spinnerTimer) {
    window.clearTimeout(meta.spinnerTimer);
    meta.spinnerTimer = null;
  }
  if (meta.resetTimer) {
    window.clearTimeout(meta.resetTimer);
    meta.resetTimer = null;
  }
}

function lockButtonWidth(button, meta) {
  if (meta.widthLocked) {
    return;
  }

  const width = Math.ceil(button.getBoundingClientRect().width);
  button.style.setProperty("--submit-fixed-width", `${width}px`);
  button.setAttribute("data-fixed-width", "");
  meta.widthLocked = true;
}

function unlockButtonWidth(button, meta) {
  if (!meta.widthLocked) {
    return;
  }

  button.removeAttribute("data-fixed-width");
  button.style.removeProperty("--submit-fixed-width");
  meta.widthLocked = false;
}

function setLoading(button) {
  if (!(button instanceof HTMLButtonElement)) {
    return;
  }

  const meta = getMeta(button);
  if (meta.mode === "loading") {
    return;
  }

  clearTimers(meta);
  lockButtonWidth(button, meta);

  meta.mode = "loading";
  button.disabled = true;
  button.setAttribute("aria-busy", "true");
  button.dataset.submitVisual = "idle";

  meta.spinnerTimer = window.setTimeout(() => {
    if (meta.mode === "loading") {
      button.dataset.submitVisual = "loading";
    }
  }, SPINNER_DELAY_MS);
}

function setSuccess(button) {
  if (!(button instanceof HTMLButtonElement)) {
    return;
  }

  const meta = getMeta(button);
  if (meta.mode === "success") {
    return;
  }

  clearTimers(meta);
  if (!meta.widthLocked) {
    lockButtonWidth(button, meta);
  }

  meta.mode = "success";
  button.disabled = true;
  button.removeAttribute("aria-busy");
  button.dataset.submitVisual = "success";

  meta.resetTimer = window.setTimeout(() => {
    reset(button);
  }, SUCCESS_HOLD_MS);
}

function reset(button) {
  if (!(button instanceof HTMLButtonElement)) {
    return;
  }

  const meta = getMeta(button);

  clearTimers(meta);
  meta.mode = "idle";

  button.disabled = false;
  button.removeAttribute("aria-busy");
  button.dataset.submitVisual = "idle";
  unlockButtonWidth(button, meta);
}

function hasManagedSubmitState(form) {
  return form instanceof HTMLFormElement && form.matches("[data-submit-state]");
}

function findSubmitButton(form, submitter) {
  if (submitter instanceof HTMLButtonElement) {
    return submitter;
  }

  return form.querySelector("button[type='submit'][data-submit-state-button], button[type='submit']");
}

function getRequestForm(eventDetail) {
  const elt = eventDetail?.elt;
  if (elt instanceof HTMLFormElement) {
    return elt;
  }

  if (elt instanceof Element) {
    return elt.closest("form");
  }

  return null;
}

function shouldResetOnSuccess(form) {
  return form.dataset.submitStateReset !== "false";
}

function initSubmitState() {
  document.addEventListener("submit", (event) => {
    const form = event.target;
    if (!hasManagedSubmitState(form)) {
      return;
    }

    const button = findSubmitButton(form, event.submitter);
    if (button) {
      setLoading(button);
    }
  });

  document.body.addEventListener("htmx:afterRequest", (event) => {
    const form = getRequestForm(event.detail);
    if (!hasManagedSubmitState(form)) {
      return;
    }

    const button = findSubmitButton(form, null);
    if (!button) {
      return;
    }

    if (event.detail.successful) {
      setSuccess(button);
      if (shouldResetOnSuccess(form)) {
        form.reset();
      }
      return;
    }

    reset(button);
  });

  window.SubmitState = {
    setLoading,
    setSuccess,
    reset,
  };
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initSubmitState, { once: true });
} else {
  initSubmitState();
}
