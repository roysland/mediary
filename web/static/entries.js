function deleteEntry(entryId) {
  const confirmed = window.confirm("Are you sure you want to delete this entry? This action cannot be undone.");
  if (!confirmed) {
    return;
  }

  fetch(`/entry/${entryId}/delete`, {
    method: "POST",
    headers: {
      "HX-Request": "true",
    },
  })
    .then((response) => {
      if (!response.ok) {
        window.alert("Failed to delete entry. Please try again.");
        return;
      }

      const entryElement = document.querySelector(`[data-id='entry-${entryId}']`);
      if (entryElement) {
        entryElement.remove();
      }
    })
    .catch((error) => {
      console.error("Error deleting entry:", error);
      window.alert("An error occurred while deleting the entry. Please try again.");
    });
}

function scrollDayControl() {
  const el = document.querySelector(".day-control");
  if (el) {
    el.scrollTo({ left: el.scrollWidth, behavior: "instant" });
  }
}

function initEntriesInteractions() {
  scrollDayControl();

  document.body.addEventListener("click", (event) => {
    const trigger = event.target.closest(".edit-note-button");
    if (!trigger) {
      const deleteButton = event.target.closest(".delete-note-button");
      if (!deleteButton) {
        return;
      }

      const entryId = Number(deleteButton.dataset.entryId);
      if (Number.isFinite(entryId) && entryId > 0) {
        deleteEntry(entryId);
      }
      return;
    }

    const popover = trigger.closest("[popover]");
    if (popover && typeof popover.hidePopover === "function") {
      popover.hidePopover();
    }
  });

  document.body.addEventListener("htmx:afterSwap", (event) => {
    scrollDayControl();

    const quickTrackableDialog = document.getElementById("add-quick-trackable-dialog");
    if (!quickTrackableDialog || event.detail.target !== quickTrackableDialog) {
      return;
    }

    if (!quickTrackableDialog.open) {
      quickTrackableDialog.showModal();
    }
  });

  document.body.addEventListener("entrytrackable:saved", (event) => {
    const entryId = Number(event.detail?.entry_id);
    if (!Number.isFinite(entryId) || entryId <= 0 || typeof htmx === "undefined") {
      return;
    }

    const entryElement = document.querySelector(`[data-id='entry-${entryId}']`);
    if (!entryElement) {
      return;
    }

    htmx.ajax("GET", `/entry/${entryId}`, {
      target: entryElement,
      swap: "outerHTML",
    });
  });

  const quickTrackableDialog = document.getElementById("add-quick-trackable-dialog");
  if (quickTrackableDialog) {
    quickTrackableDialog.addEventListener("click", (event) => {
      if (event.target === quickTrackableDialog) {
        quickTrackableDialog.close();
      }
    });
  }
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initEntriesInteractions, { once: true });
} else {
  initEntriesInteractions();
}
