function nullableString(value, fallback = "") {
  if (value == null) {
    return fallback;
  }
  if (typeof value === "string") {
    return value || fallback;
  }
  if (typeof value.String === "string") {
    return value.String || fallback;
  }
  return fallback;
}

function nullableInt(value, fallback = 0) {
  if (value == null) {
    return fallback;
  }
  if (typeof value === "number") {
    return value;
  }
  if (typeof value.Int64 === "number") {
    return value.Int64;
  }
  return fallback;
}

function parsePresetList(root) {
  const raw = root.dataset.trackablePresets;
  if (!raw) {
    return [];
  }

  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    console.error("Unable to parse trackable presets", error);
    return [];
  }
}

function updateFieldsFromPreset(form, preset) {
  if (!preset) {
    return;
  }

  const nameInput = form.querySelector("#trackable_name");
  const hiddenPresetIdInput = form.querySelector("#trackable_template_id, #presetId");
  const iconInput = form.querySelector("#trackable_icon");
  const valueTypeInput = form.querySelector("#trackable_value_type, #trackable_value-type");
  const categoryInput = form.querySelector("#trackable_category");
  const minValueInput = form.querySelector("#trackable_min_value");
  const maxValueInput = form.querySelector("#trackable_max_value");

  if (nameInput instanceof HTMLInputElement) {
    nameInput.value = preset.name ?? "";
  }
  if (hiddenPresetIdInput instanceof HTMLInputElement) {
    hiddenPresetIdInput.value = preset.id ?? "";
  }
  if (iconInput instanceof HTMLInputElement) {
    iconInput.value = nullableString(preset.icon, "");
  }
  if (valueTypeInput instanceof HTMLSelectElement) {
    valueTypeInput.value = preset.value_type ?? "integer";
    valueTypeInput.dispatchEvent(new Event("change", { bubbles: true }));
  }
  if (categoryInput instanceof HTMLSelectElement) {
    categoryInput.value = nullableString(preset.category, "default");
  }
  if (minValueInput instanceof HTMLInputElement) {
    minValueInput.value = String(nullableInt(preset.min_value, 0));
  }
  if (maxValueInput instanceof HTMLInputElement) {
    maxValueInput.value = String(nullableInt(preset.max_value, 10));
  }
}

function findPreset(presets, button) {
  const templateIDRaw = button.dataset.templateId ?? "";
  if (templateIDRaw !== "") {
    const templateID = Number(templateIDRaw);
    if (!Number.isNaN(templateID)) {
      return presets.find((item) => Number(item?.id) === templateID) ?? null;
    }
  }

  const presetName = button.dataset.name || "";
  return presets.find((item) => item?.name === presetName) ?? null;
}

function initTrackablePresets(scope = document) {
  const lists = scope instanceof Element && scope.matches(".trackablePresetList")
    ? [scope]
    : Array.from(scope.querySelectorAll?.(".trackablePresetList") ?? []);

  lists.forEach((list) => {
    if (!(list instanceof HTMLElement) || list.dataset.presetReady === "true") {
      return;
    }

    const form = list.closest("form");
    if (!(form instanceof HTMLFormElement)) {
      return;
    }

    const presets = parsePresetList(list);
    const hiddenPresetIdInput = form.querySelector("#trackable_template_id, #presetId");
    const trackableNameInput = form.querySelector("#trackable_name");

    list.dataset.presetReady = "true";

    list.addEventListener("click", (event) => {
      const button = event.target.closest("button[data-name]");
      if (!(button instanceof HTMLButtonElement)) {
        return;
      }

      const preset = findPreset(presets, button);
      updateFieldsFromPreset(form, preset);

      if (trackableNameInput instanceof HTMLInputElement) {
        trackableNameInput.dispatchEvent(new Event("input", { bubbles: true }));
      }
    });

    if (trackableNameInput instanceof HTMLInputElement) {
      trackableNameInput.addEventListener("input", () => {
        if (hiddenPresetIdInput instanceof HTMLInputElement) {
          hiddenPresetIdInput.value = "";
        }

        const inputValue = trackableNameInput.value.toLowerCase();
        const listItems = list.querySelectorAll("li[data-name]");
        listItems.forEach((listItem) => {
          const name = (listItem.getAttribute("data-name") || "").toLowerCase();
          listItem.style.display = name.includes(inputValue) ? "" : "none";
        });
      });
    }
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => initTrackablePresets(document), { once: true });
} else {
  initTrackablePresets(document);
}

document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.detail.target instanceof Element) {
    initTrackablePresets(event.detail.target);
  }
});
