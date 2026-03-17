function initSettingsDangerZone() {
  const clearDataForm = document.getElementById("clear_data_form");
  const deleteButton = document.getElementById("delete_everything");
  if (!(clearDataForm instanceof HTMLFormElement) || !(deleteButton instanceof HTMLButtonElement)) {
    return;
  }

  if (clearDataForm.dataset.confirmReady === "true") {
    return;
  }

  clearDataForm.dataset.confirmReady = "true";
  clearDataForm.addEventListener("submit", (event) => {
    event.preventDefault();

    const message = deleteButton.dataset.confirmMessage || deleteButton.dataset.confirmFallback || "Are you sure?";
    if (window.confirm(message)) {
      clearDataForm.submit();
    }
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initSettingsDangerZone, { once: true });
} else {
  initSettingsDangerZone();
}

document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.detail.target instanceof Element) {
    initSettingsDangerZone();
  }
});
