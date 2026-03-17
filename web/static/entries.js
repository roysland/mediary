function getEntriesMessages() {
  const i18n = document.getElementById("entries-i18n");
  if (!(i18n instanceof HTMLElement)) {
    return {
      deleteConfirm: "Are you sure you want to delete this entry? This action cannot be undone.",
      deleteFailed: "Failed to delete entry. Please try again.",
      deleteError: "An error occurred while deleting the entry. Please try again.",
    };
  }

  return {
    deleteConfirm:
      i18n.dataset.deleteConfirm || "Are you sure you want to delete this entry? This action cannot be undone.",
    deleteFailed: i18n.dataset.deleteFailed || "Failed to delete entry. Please try again.",
    deleteError: i18n.dataset.deleteError || "An error occurred while deleting the entry. Please try again.",
  };
}

function deleteEntry(entryId) {
  const messages = getEntriesMessages();
  const confirmed = window.confirm(messages.deleteConfirm);
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
        window.alert(messages.deleteFailed);
        return;
      }

      const entryElement = document.querySelector(`[data-id='entry-${entryId}']`);
      if (entryElement) {
        entryElement.remove();
      }
    })
    .catch((error) => {
      console.error("Error deleting entry:", error);
      window.alert(messages.deleteError);
    });
}

function scrollDayControl() {
  const el = document.querySelector(".day-control");
  if (el) {
    // Keep this compatible across browsers; "instant" is not universally supported.
    el.scrollLeft = el.scrollWidth;
  }
}

function setDialogEntryId(dialog, entryId) {
  const forms = dialog.querySelectorAll(".trackable-picker .autosave-form");
  forms.forEach((form) => {
    const existingInput = form.querySelector("input[name='entry_id']");
    if (existingInput) {
      existingInput.remove();
    }

    const entryInput = document.createElement("input");
    entryInput.type = "hidden";
    entryInput.name = "entry_id";
    entryInput.value = String(entryId);
    form.append(entryInput);
  });
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

      const popover = deleteButton.closest("[popover]");
      if (popover && typeof popover.hidePopover === "function") {
        popover.hidePopover();
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

    const entryId = Number(trigger.dataset.entryId);
    const quickTrackableDialog = document.getElementById("add-quick-trackable-dialog");
    if (!quickTrackableDialog || !Number.isFinite(entryId) || entryId <= 0) {
      return;
    }

    setDialogEntryId(quickTrackableDialog, entryId);
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
