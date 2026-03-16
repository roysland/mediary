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

function findSubmitButton(form, submitter) {
  if (submitter instanceof HTMLButtonElement && submitter.matches("[data-submit-state]")) {
    return submitter;
  }

  return form.querySelector("button[type='submit'][data-submit-state]");
}

function initSubmitState() {
  document.addEventListener("submit", (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) {
      return;
    }

    const button = findSubmitButton(form, event.submitter);
    if (button) {
      setLoading(button);
    }
  });

  document.body.addEventListener("htmx:afterRequest", (event) => {
    const form = event.detail.elt;
    if (!(form instanceof HTMLFormElement)) {
      return;
    }

    const button = findSubmitButton(form, null);
    if (!button) {
      return;
    }

    if (event.detail.successful) {
      setSuccess(button);
      form.reset();
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

function initTrackableFormOptions(scope = document) {
  const forms = scope instanceof Element && scope.matches("form[action='/trackables/add']")
    ? [scope]
    : Array.from(scope.querySelectorAll?.("form[action='/trackables/add']") ?? []);

  forms.forEach((form) => {
    if (!(form instanceof HTMLFormElement) || form.dataset.trackableFormReady === "true") {
      return;
    }

    const sensitiveCheckbox = form.querySelector("#trackable_is-sensitive");
    const privateLabelInput = form.querySelector("#trackable-private-label");
    const valueTypeSelect = form.querySelector("#trackable_value-type");
    const minMaxValueContainer = form.querySelector(".value_treshold");
    const unitInput = form.querySelector("#trackable_unit");

    if (
      !(sensitiveCheckbox instanceof HTMLInputElement) ||
      !(privateLabelInput instanceof HTMLInputElement) ||
      !(valueTypeSelect instanceof HTMLSelectElement) ||
      !(minMaxValueContainer instanceof HTMLElement) ||
      !(unitInput instanceof HTMLInputElement) ||
      !(unitInput.parentElement instanceof HTMLElement)
    ) {
      return;
    }

    form.dataset.trackableFormReady = "true";

    const updateUnitVisibility = () => {
      if (valueTypeSelect.value === "integer") {
        unitInput.parentElement.style.display = "block";
        minMaxValueContainer.style.display = "block";
        return;
      }

      unitInput.parentElement.style.display = "none";
      unitInput.value = "";
      minMaxValueContainer.style.display = "none";
    };

    const updatePrivateLabelVisibility = () => {
      if (sensitiveCheckbox.checked) {
        privateLabelInput.parentElement.style.display = "block";
        return;
      }

      privateLabelInput.parentElement.style.display = "none";
      privateLabelInput.value = "";
    };

    sensitiveCheckbox.addEventListener("change", updatePrivateLabelVisibility);
    valueTypeSelect.addEventListener("change", updateUnitVisibility);

    updatePrivateLabelVisibility();
    updateUnitVisibility();
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => {
    initSubmitState();
    initTrackableFormOptions(document);
  }, { once: true });
} else {
  initSubmitState();
  initTrackableFormOptions(document);
}

document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.detail.target instanceof Element) {
    initTrackableFormOptions(event.detail.target);
  }
});
