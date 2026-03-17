function initTrackableFormOptions(scope = document) {
  const forms = scope instanceof Element && scope.matches("form[action='/trackables/add']")
    ? [scope]
    : Array.from(scope.querySelectorAll?.("form[action='/trackables/add']") ?? []);

  forms.forEach((form) => {
    if (!(form instanceof HTMLFormElement) || form.dataset.trackableFormReady === "true") {
      return;
    }

    const sensitiveCheckbox = form.querySelector("#trackable_is_sensitive, #trackable_is-sensitive");
    const privateLabelInput = form.querySelector("#trackable_private_label, #trackable-private-label");
    const valueTypeSelect = form.querySelector("#trackable_value_type, #trackable_value-type");
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
    initTrackableFormOptions(document);
  }, { once: true });
} else {
  initTrackableFormOptions(document);
}

document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.detail.target instanceof Element) {
    initTrackableFormOptions(event.detail.target);
  }
});
