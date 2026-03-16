function initSettingsDangerZone() {
  const deleteButton = document.getElementById("delete_everything");
  if (!(deleteButton instanceof HTMLButtonElement)) {
    return;
  }

  if (deleteButton.dataset.confirmReady === "true") {
    return;
  }

  deleteButton.dataset.confirmReady = "true";
  deleteButton.addEventListener("click", (event) => {
    event.preventDefault();

    const message = deleteButton.dataset.confirmMessage || "Are you sure?";
    if (window.confirm(message)) {
      // Clear-data flow is not implemented yet.
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
