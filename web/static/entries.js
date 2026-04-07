let sensitiveFilterPreference = null;

function getEntriesMessages() {
  const i18n = document.getElementById("entries-i18n");
  if (!(i18n instanceof HTMLElement)) {
    return {
      deleteConfirm: "Are you sure you want to delete this entry? This action cannot be undone.",
      deleteFailed: "Failed to delete entry. Please try again.",
      deleteError: "An error occurred while deleting the entry. Please try again.",
      entryAddTitle: "Add entry",
      entryAddTextTitle: "Add text",
      entryEditTextTitle: "Edit text",
    };
  }

  return {
    deleteConfirm:
      i18n.dataset.deleteConfirm || "Are you sure you want to delete this entry? This action cannot be undone.",
    deleteFailed: i18n.dataset.deleteFailed || "Failed to delete entry. Please try again.",
    deleteError: i18n.dataset.deleteError || "An error occurred while deleting the entry. Please try again.",
    entryAddTitle: i18n.dataset.entryAddTitle || "Add entry",
    entryAddTextTitle: i18n.dataset.entryAddTextTitle || "Add text",
    entryEditTextTitle: i18n.dataset.entryEditTextTitle || "Edit text",
  };
}

function readSensitiveFilterPreference() {
  if (typeof sensitiveFilterPreference === "boolean") {
    return sensitiveFilterPreference;
  }

  const toggle = document.querySelector("[data-sensitive-filter-toggle]");
  if (toggle instanceof HTMLInputElement) {
    sensitiveFilterPreference = toggle.checked;
    return sensitiveFilterPreference;
  }

  sensitiveFilterPreference = false;
  return sensitiveFilterPreference;
}

function writeSensitiveFilterPreference(showSensitive) {
  sensitiveFilterPreference = showSensitive;
}

function persistSensitiveFilterPreference(showSensitive) {
  return fetch("/settings/sensitive-content", {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      "HX-Request": "true",
    },
    body: `show_sensitive_content=${showSensitive ? "true" : "false"}`,
  }).then((response) => {
    if (!response.ok) {
      throw new Error(`Failed to persist sensitive filter preference (${response.status})`);
    }
  });
}

function setSensitiveContentState(showSensitive) {
  document.documentElement.dataset.showSensitiveContent = showSensitive ? "true" : "false";
}

function updateEntriesEmptyState(root) {
  if (!(root instanceof HTMLElement)) {
    return;
  }

  const timeline = root.querySelector("[data-entries-timeline]");
  const emptyState = root.querySelector("[data-entries-empty-state]");
  if (!(timeline instanceof HTMLElement) || !(emptyState instanceof HTMLElement)) {
    return;
  }

  const hasVisibleEntries = Array.from(timeline.children).some(
    (child) => child instanceof HTMLElement && !child.hidden,
  );

  timeline.hidden = !hasVisibleEntries;
  emptyState.hidden = hasVisibleEntries;
}

function updateDismissedTrackablePanels(root) {
  console.log(root)
  if (!(root instanceof Element) && !(root instanceof Document)) {
    return;
  }

  root.querySelectorAll("[data-trackable-picker]").forEach((picker) => {
    if (!(picker instanceof HTMLElement)) {
      return;
    }

    const dismissedPanel = picker.querySelector("[data-dismissed-trackables-panel]");
    const dismissedTrackablesList = picker.querySelector("[data-dismissed-trackables-list]");
    if (!(dismissedPanel instanceof HTMLElement) || !(dismissedTrackablesList instanceof HTMLElement)) {
      return;
    }

    const hasVisibleDismissedTrackables = Array.from(dismissedTrackablesList.children).some(
      (child) => child instanceof HTMLElement && !child.hidden,
    );
    dismissedPanel.hidden = !hasVisibleDismissedTrackables;
  });
}

function applySensitiveFilter() {
  const showSensitive = readSensitiveFilterPreference();
  setSensitiveContentState(showSensitive);

  document.querySelectorAll("[data-sensitive-filter-toggle]").forEach((toggle) => {
    if (toggle instanceof HTMLInputElement) {
      toggle.checked = showSensitive;
    }
  });

  document.querySelectorAll("[data-sensitive-filter-root]").forEach((root) => {
    if (!(root instanceof HTMLElement)) {
      return;
    }

    root.querySelectorAll("[data-entry-private='true']").forEach((entry) => {
      if (entry instanceof HTMLElement) {
        entry.hidden = !showSensitive;
      }
    });

    root.querySelectorAll("[data-sensitive-trackable='true']").forEach((trackable) => {
      if (trackable instanceof HTMLElement) {
        trackable.hidden = !showSensitive;
      }
    });

    updateEntriesEmptyState(root);
    updateDismissedTrackablePanels(root);
  });
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

      try {
        const entryElement = document.querySelector(`[data-id='entry-${entryId}']`);
        if (entryElement) {
          entryElement.remove();
          applySensitiveFilter();
        }
      } catch (error) {
        console.error("Entry deleted, but UI refresh failed:", error);
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

function refreshEntriesPage() {
  const page = document.querySelector(".entries-page");
  const target = document.querySelector("main.container");
  const url = window.location.pathname + window.location.search;
  if (!(page instanceof HTMLElement) || !(target instanceof HTMLElement) || typeof htmx === "undefined") {
    return;
  }

  htmx.ajax("GET", url, {
    target,
    select: "main.container",
    swap: "outerHTML",
  });
}

function configureEntryNoteDialog(options = {}) {
  const dialog = document.getElementById("entry-note-dialog");
  if (!(dialog instanceof HTMLDialogElement)) {
    return null;
  }

  const form = dialog.querySelector("[data-entry-form]");
  const title = dialog.querySelector("[data-entry-note-dialog-title]");
  const textarea = dialog.querySelector("[data-entry-form-textarea]");
  const privacy = dialog.querySelector("#is_private-entry");
  const entryIdInput = dialog.querySelector("[data-entry-form-entry-id]");
  const dateInput = dialog.querySelector("[data-entry-form-date]");
  const dateLabel = dialog.querySelector("[data-entry-form-date-label]");
  const warning = dialog.querySelector("[data-entry-form-warning]");
  if (!(form instanceof HTMLFormElement) || !(textarea instanceof HTMLTextAreaElement)) {
    return null;
  }

  const messages = getEntriesMessages();
  const entryDate = options.entryDate || dialog.dataset.defaultEntryDate || "";
  const mode = options.mode || "add";
  const action = dialog.dataset.defaultAction || "/entry/add";

  form.reset();
  form.action = action;
  if (form.hasAttribute("hx-post")) {
    form.setAttribute("hx-post", action);
  }
  if (entryIdInput instanceof HTMLInputElement) {
    entryIdInput.value = mode === "edit" && Number.isFinite(options.entryId) && options.entryId > 0
      ? String(options.entryId)
      : "";
  }

  textarea.value = options.noteText || "";
  if (privacy instanceof HTMLInputElement) {
    privacy.checked = options.isPrivate === true;
  }
  if (dateInput instanceof HTMLInputElement) {
    dateInput.value = entryDate;
  }
  if (dateLabel instanceof HTMLElement) {
    dateLabel.textContent = entryDate;
  }
  if (warning instanceof HTMLElement) {
    warning.hidden = !(entryDate && dialog.dataset.todayStr && entryDate < dialog.dataset.todayStr);
  }
  if (title instanceof HTMLElement) {
    if (mode === "edit") {
      title.textContent = options.hasNote ? messages.entryEditTextTitle : messages.entryAddTextTitle;
    } else {
      title.textContent = messages.entryAddTitle;
    }
  }

  if (!dialog.open) {
    dialog.showModal();
  }

  window.requestAnimationFrame(() => {
    textarea.focus();
    textarea.setSelectionRange(textarea.value.length, textarea.value.length);
  });

  return dialog;
}

function initEntriesInteractions() {
  scrollDayControl();

  // Swipe-to-delete: mecfs-swipe-dismiss fires "swipedismiss" when threshold met.
  document.body.addEventListener("swipedismiss", (event) => {
    const wrapper = event.target.closest(".entry-card-wrapper");
    if (!wrapper) return;

    const entryId = Number(wrapper.dataset.entryId);
    if (!Number.isFinite(entryId) || entryId <= 0) return;

    const messages = getEntriesMessages();
    const confirmed = window.confirm(messages.deleteConfirm);

    if (!confirmed) {
      // Reset the swipe animation so card slides back into place.
      const foreground = wrapper.querySelector(":not([slot=background])");
      if (foreground instanceof HTMLElement) {
        foreground.style.transition = "transform 180ms ease";
        foreground.style.transform = "translateX(0)";
      }
      return;
    }

    fetch(`/entry/${entryId}/delete`, {
      method: "POST",
      headers: { "HX-Request": "true" },
    })
      .then((response) => {
        if (!response.ok) {
          window.alert(messages.deleteFailed);
          const foreground = wrapper.querySelector(":not([slot=background])");
          if (foreground instanceof HTMLElement) {
            foreground.style.transition = "transform 180ms ease";
            foreground.style.transform = "translateX(0)";
          }
          return;
        }

        try {
          const li = wrapper.closest("[data-id]");
          if (li) {
            li.remove();
            applySensitiveFilter();
          }
        } catch (error) {
          console.error("Entry deleted by swipe, but UI refresh failed:", error);
        }
      })
      .catch((error) => {
        console.error("Error deleting entry:", error);
        window.alert(messages.deleteError);
        const foreground = wrapper.querySelector(":not([slot=background])");
        if (foreground instanceof HTMLElement) {
          foreground.style.transition = "transform 180ms ease";
          foreground.style.transform = "translateX(0)";
        }
      });
  });

  document.addEventListener("click", (event) => {
    const target = event.target instanceof Element
      ? event.target
      : event.target instanceof Node
        ? event.target.parentElement
        : null;

    if (!target) {
      return;
    }

    if (target instanceof HTMLDialogElement) {
      if (target.id === "add-quick-trackable-dialog" || target.id === "entry-note-dialog") {
        target.close();
      }
      return;
    }

    const openEntryDialog = target.closest("[data-open-entry-dialog]");
    if (openEntryDialog) {
      event.preventDefault();
      configureEntryNoteDialog({ mode: "add" });
      return;
    }

    const editEntryButton = target.closest(".edit-entry-button");
    if (editEntryButton) {
      const popover = editEntryButton.closest("[popover]");
      if (popover && typeof popover.hidePopover === "function") {
        popover.hidePopover();
      }

      const entryId = Number(editEntryButton.dataset.entryId);
      const noteSourceId = editEntryButton.dataset.entryNoteSource || "";
      const noteSource = noteSourceId ? document.getElementById(noteSourceId) : null;
      const noteText = noteSource instanceof HTMLTextAreaElement ? noteSource.value : "";

      configureEntryNoteDialog({
        mode: "edit",
        entryId,
        entryDate: editEntryButton.dataset.entryDate || "",
        isPrivate: editEntryButton.dataset.entryPrivate === "true",
        hasNote: editEntryButton.dataset.entryHasNote === "true",
        noteText,
      });
      return;
    }

    const deleteButton = target.closest(".delete-note-button");
    if (deleteButton) {
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

    const trigger = target.closest(".edit-trackables-button");
    if (trigger) {
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
      return;
    }

    const uploadImageButton = target.closest(".upload-image-button");
    if (uploadImageButton) {
      const popover = uploadImageButton.closest("[popover]");
      if (popover && typeof popover.hidePopover === "function") {
        popover.hidePopover();
      }

      const entryId = Number(uploadImageButton.dataset.entryId);
      const noteSourceId = uploadImageButton.dataset.entryNoteSource || "";
      const noteSource = noteSourceId ? document.getElementById(noteSourceId) : null;
      const noteText = noteSource instanceof HTMLTextAreaElement ? noteSource.value : "";

      const dialog = configureEntryNoteDialog({
        mode: "edit",
        entryId,
        entryDate: uploadImageButton.dataset.entryDate || "",
        isPrivate: uploadImageButton.dataset.entryPrivate === "true",
        hasNote: uploadImageButton.dataset.entryHasNote === "true",
        noteText,
      });
      if (!(dialog instanceof HTMLDialogElement)) {
        return;
      }

      const fileInput = dialog.querySelector("[data-image-upload-input]");
      if (fileInput instanceof HTMLInputElement) {
        window.requestAnimationFrame(() => fileInput.click());
      }
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

  document.body.addEventListener("change", (event) => {
    if (!(event.target instanceof HTMLInputElement) || !event.target.matches("[data-sensitive-filter-toggle]")) {
      return;
    }

    writeSensitiveFilterPreference(event.target.checked);
    applySensitiveFilter();

    persistSensitiveFilterPreference(event.target.checked).catch((error) => {
      console.error("Failed to persist sensitive filter preference:", error);
    });
  });

  document.body.addEventListener("htmx:afterSwap", () => {
    applySensitiveFilter();
  });

  document.body.addEventListener("htmx:afterRequest", (event) => {
    const form = getRequestForm(event.detail);
    if (!(form instanceof HTMLFormElement) || form.dataset.entryDialogForm !== "true" || !event.detail.successful) {
      return;
    }

    const dialog = form.closest("dialog");
    if (dialog instanceof HTMLDialogElement && dialog.open) {
      dialog.close();
    }

    refreshEntriesPage();
  });

  applySensitiveFilter();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initEntriesInteractions, { once: true });
} else {
  initEntriesInteractions();
}
