function ensureDismissedPanelVisibility(root) {
  const dismissedPanel = root.querySelector("[data-dismissed-trackables-panel]");
  const dismissedTrackablesList = root.querySelector("[data-dismissed-trackables-list]");
  if (!dismissedPanel || !dismissedTrackablesList) {
    return;
  }

  dismissedPanel.hidden = dismissedTrackablesList.children.length === 0;
}

function createDismissedItem(trackableElement, restoreLabel) {
  const item = document.createElement("li");
  item.className = "dismissed-trackable";
  item.dataset.trackableId = trackableElement.dataset.trackableId || "";
  item.dataset.sensitiveTrackable = trackableElement.dataset.sensitiveTrackable || "false";

  const meta = document.createElement("div");
  meta.className = "dismissed-trackable__meta";

  const icon = trackableElement.querySelector(".icon");
  if (icon) {
    const iconSpan = document.createElement("span");
    iconSpan.className = "icon";
    iconSpan.textContent = icon.textContent;
    meta.appendChild(iconSpan);
  }

  const nameSpan = document.createElement("span");
  nameSpan.textContent = trackableElement.dataset.trackableName || "";
  meta.appendChild(nameSpan);

  const form = document.createElement("form");
  form.method = "POST";
  form.action = `/trackables/${trackableElement.dataset.trackableId}/dismissal`;
  form.style.display = "inline";

  const hiddenInput = document.createElement("input");
  hiddenInput.type = "hidden";
  hiddenInput.name = "dismissed";
  hiddenInput.value = "false";
  form.appendChild(hiddenInput);

  const button = document.createElement("button");
  button.type = "submit";
  button.className = "restore-trackable";
  button.textContent = restoreLabel;
  button.setAttribute("hx-post", `/trackables/${trackableElement.dataset.trackableId}/dismissal`);
  button.setAttribute("hx-trigger", "click");
  button.setAttribute("hx-swap", "none");
  form.appendChild(button);

  item.appendChild(meta);
  item.appendChild(form);

  return item;
}

function initRangeOutputBindings(root) {
  const sliders = root.querySelectorAll("input[type='range'][data-range-output]");
  sliders.forEach((slider) => {
    if (!(slider instanceof HTMLInputElement) || slider.dataset.rangeReady === "true") {
      return;
    }

    const outputId = slider.dataset.rangeOutput;
    let output = null;
    if (outputId) {
      try {
        output = root.querySelector(`#${CSS.escape(outputId)}`);
      } catch {
        output = root.querySelector(`#${outputId}`);
      }
    }
    if (!(output instanceof HTMLOutputElement)) {
      output = slider.closest("form")?.querySelector("output");
    }
    if (!(output instanceof HTMLOutputElement)) {
      return;
    }

    slider.dataset.rangeReady = "true";
    const sync = () => {
      output.value = slider.value;
      output.textContent = slider.value;
    };

    slider.addEventListener("input", sync);
    slider.addEventListener("change", sync);
    sync();
  });
}

function syncRangeOutputFromSlider(slider) {
  if (!(slider instanceof HTMLInputElement)) {
    return;
  }

  const outputId = slider.dataset.rangeOutput;
  if (!outputId) {
    return;
  }

  const pickerRoot = slider.closest("[data-trackable-picker]");
  let output = null;
  if (pickerRoot instanceof Element) {
    try {
      output = pickerRoot.querySelector(`#${CSS.escape(outputId)}`);
    } catch {
      output = pickerRoot.querySelector(`#${outputId}`);
    }
  }

  if (!(output instanceof HTMLElement)) {
    output = slider.closest("form")?.querySelector("output");
  }
  if (!(output instanceof HTMLElement)) {
    return;
  }

  output.textContent = slider.value;
  if ("value" in output) {
    output.value = slider.value;
  }
}

function initTrackablePickers(scope = document) {
  const roots = scope instanceof Element && scope.matches("[data-trackable-picker]")
    ? [scope]
    : Array.from(scope.querySelectorAll?.("[data-trackable-picker]") ?? []);

  roots.forEach((root) => {
    if (!(root instanceof HTMLElement) || root.dataset.trackablePickerReady === "true") {
      return;
    }

    root.dataset.trackablePickerReady = "true";
    ensureDismissedPanelVisibility(root);
    initRangeOutputBindings(root);
  });
}

window.initTrackablePickers = initTrackablePickers;

function initTrackablePickerEvents() {
  initTrackablePickers(document);

  const delegatedRangeSync = (event) => {
    if (!(event.target instanceof HTMLInputElement)) {
      return;
    }
    if (!event.target.matches("input[type='range'][data-range-output]")) {
      return;
    }
    syncRangeOutputFromSlider(event.target);
  };

  document.body.addEventListener("input", delegatedRangeSync);
  document.body.addEventListener("change", delegatedRangeSync);

  // Intercept HTMX requests on elements with data-confirm-label and prompt the user.
  document.body.addEventListener("htmx:confirm", (event) => {
    const label = event.detail.elt?.dataset?.confirmLabel;
    if (!label) return;
    event.preventDefault();
    if (window.confirm(label)) {
      event.detail.issueRequest(true);
    }
  });

  document.addEventListener("swipedismiss", async (event) => {
    const trackableElement = event.target.closest(".trackable-item");
    const root = trackableElement?.closest("[data-trackable-picker]");
    const activeTrackables = root?.querySelector("[data-active-trackables]");
    const dismissedTrackablesList = root?.querySelector("[data-dismissed-trackables-list]");

    if (!trackableElement || !root || !activeTrackables || !dismissedTrackablesList) {
      return;
    }

    const trackableId = trackableElement.dataset.trackableId;
    const restoreLabel = root.dataset.restoreLabel || "Restore";
    const dismissedItem = createDismissedItem(trackableElement, restoreLabel);
    const swiped = event.target;

    swiped.remove();
    dismissedTrackablesList.appendChild(dismissedItem);
    ensureDismissedPanelVisibility(root);

    try {
      const response = await fetch(`/trackables/${trackableId}/dismissal`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ dismissed: true }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
    } catch (error) {
      console.error("Failed to save dismissal:", error);
      dismissedItem.remove();
      activeTrackables.appendChild(trackableElement);
      ensureDismissedPanelVisibility(root);
    }
  });

  document.body.addEventListener("htmx:afterRequest", (event) => {
    const form = event.detail.elt.closest?.("form");
    const root = event.detail.elt.closest?.("[data-trackable-picker]");
    if (!root) {
      return;
    }

    if (form?.classList?.contains?.("autosave-form")) {
      const statusEl = form.querySelector(".save-status");
      const isError = event.detail.xhr?.status >= 400;

      if (isError) {
        if (statusEl) {
          statusEl.textContent = "✗ Error";
        }
      } else if (statusEl) {
        statusEl.textContent = "✓ Saved";
        window.setTimeout(() => {
          if (statusEl.textContent === "✓ Saved") {
            statusEl.textContent = "";
          }
        }, 2000);
      }

      if (!isError) {
        let responseData = null;
        try {
          responseData = JSON.parse(event.detail.xhr?.responseText ?? "null");
        } catch {
          responseData = null;
        }

        if (responseData?.entry_id) {
          document.body.dispatchEvent(
            new CustomEvent("entrytrackable:saved", {
              detail: responseData,
            }),
          );
        }
      }
    }

    if (event.detail.elt?.classList?.contains?.("restore-trackable")) {
      const isError = event.detail.xhr?.status >= 400;
      if (!isError) {
        window.location.reload();
      }
    }

    if (event.detail.elt?.classList?.contains?.("delete-trackable")) {
      const isError = event.detail.xhr?.status >= 400;
      if (!isError) {
        const li = event.detail.elt.closest(".dismissed-trackable");
        if (li) {
          li.remove();
          const root = event.detail.elt.closest("[data-trackable-picker]");
          if (root) ensureDismissedPanelVisibility(root);
        }
      }
    }
  });

  document.body.addEventListener("htmx:afterSwap", (event) => {
    if (event.detail.target instanceof Element) {
      initTrackablePickers(event.detail.target);
    }
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initTrackablePickerEvents, { once: true });
} else {
  initTrackablePickerEvents();
}
