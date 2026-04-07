const MAX_UPLOAD_BYTES = 2 * 1024 * 1024;
const ALLOWED_IMAGE_TYPES = new Set(["image/jpeg", "image/png", "image/webp", "image/gif"]);

function readEntryID(root) {
  const container = root.parentElement;
  if (!(container instanceof HTMLElement)) {
    return 0;
  }

  const entryIDInput = container.querySelector("[data-entry-form-entry-id]");
  if (!(entryIDInput instanceof HTMLInputElement)) {
    return 0;
  }

  const entryID = Number(entryIDInput.value);
  if (!Number.isFinite(entryID) || entryID <= 0) {
    return 0;
  }

  return entryID;
}

function setStatus(root, message) {
  const status = root.querySelector("[data-image-upload-status]");
  if (status instanceof HTMLElement) {
    status.textContent = message;
  }
}

function parseMessages(root) {
  return {
    missingEntry: root.dataset.missingEntry || "Save the entry first, then upload images.",
    uploading: root.dataset.uploading || "Uploading image...",
    failed: root.dataset.uploadFailed || "Image upload failed. Please try again.",
    deleted: root.dataset.deleteFailed || "Could not delete image.",
  };
}

function fileFromBlob(blob, originalName) {
  const extensionMap = {
    "image/jpeg": ".jpg",
    "image/png": ".png",
    "image/webp": ".webp",
    "image/gif": ".gif",
  };

  const ext = extensionMap[blob.type] || "";
  const safeName = originalName && originalName.includes(".")
    ? originalName.replace(/\.[^.]+$/, ext)
    : `image${ext}`;

  return new File([blob], safeName, { type: blob.type, lastModified: Date.now() });
}

async function blobFromCanvas(canvas, mimeType, quality) {
  return new Promise((resolve) => {
    canvas.toBlob((blob) => resolve(blob), mimeType, quality);
  });
}

async function loadBitmap(file) {
  if (typeof createImageBitmap === "function") {
    return createImageBitmap(file);
  }

  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = reject;
    img.src = URL.createObjectURL(file);
  });
}

function drawScaled(bitmap, width, height) {
  const canvas = document.createElement("canvas");
  canvas.width = Math.max(1, Math.round(width));
  canvas.height = Math.max(1, Math.round(height));
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    return null;
  }

  ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
  return canvas;
}

async function resizeToFitUpload(file) {
  if (file.size <= MAX_UPLOAD_BYTES || !ALLOWED_IMAGE_TYPES.has(file.type)) {
    return file;
  }

  try {
    const bitmap = await loadBitmap(file);
    const initialScale = Math.sqrt(MAX_UPLOAD_BYTES / file.size);
    let width = Math.max(1, Math.floor(bitmap.width * initialScale));
    let height = Math.max(1, Math.floor(bitmap.height * initialScale));
    const mimeType = file.type === "image/gif" ? "image/png" : file.type;
    let quality = mimeType === "image/png" ? undefined : 0.88;

    for (let attempt = 0; attempt < 6; attempt += 1) {
      const canvas = drawScaled(bitmap, width, height);
      if (!canvas) {
        return file;
      }

      const blob = await blobFromCanvas(canvas, mimeType, quality);
      if (!blob) {
        return file;
      }

      if (blob.size <= MAX_UPLOAD_BYTES) {
        return fileFromBlob(blob, file.name);
      }

      width = Math.max(1, Math.floor(width * 0.85));
      height = Math.max(1, Math.floor(height * 0.85));
      if (typeof quality === "number") {
        quality = Math.max(0.5, quality - 0.08);
      }
    }
  } catch (_) {
    return file;
  }

  return file;
}

function appendThumbnail(root, file, imageID, entryID) {
  const list = root.querySelector("[data-image-upload-list]");
  if (!(list instanceof HTMLElement)) {
    return;
  }

  const item = document.createElement("li");
  item.dataset.imageId = String(imageID);
  item.className = "entry-image-upload__item";

  const preview = document.createElement("img");
  preview.src = URL.createObjectURL(file);
  preview.alt = file.name || "Uploaded image";
  preview.loading = "lazy";
  preview.className = "entry-image-upload__thumb";

  const remove = document.createElement("button");
  remove.type = "button";
  remove.className = "button button--ghost";
  remove.textContent = "Remove";
  remove.dataset.entryId = String(entryID);
  remove.dataset.imageId = String(imageID);

  item.appendChild(preview);
  item.appendChild(remove);
  list.appendChild(item);
}

function extractImageID(fragmentHTML) {
  if (!fragmentHTML) {
    return 0;
  }

  const container = document.createElement("div");
  container.innerHTML = fragmentHTML;
  const marker = container.querySelector("[data-image-id]");
  if (!(marker instanceof HTMLElement)) {
    return 0;
  }

  const parsed = Number(marker.dataset.imageId);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 0;
  }

  return parsed;
}

async function handleUpload(root, form, fileInput) {
  const messages = parseMessages(root);
  const entryID = readEntryID(root);
  if (entryID <= 0) {
    setStatus(root, messages.missingEntry);
    return;
  }

  const file = fileInput.files?.[0];
  if (!(file instanceof File)) {
    return;
  }

  let uploadFile = await resizeToFitUpload(file);
  if (uploadFile.size > MAX_UPLOAD_BYTES) {
    setStatus(root, messages.failed);
    return;
  }

  const dataTransfer = new DataTransfer();
  dataTransfer.items.add(uploadFile);
  fileInput.files = dataTransfer.files;

  const endpoint = `/entry/${entryID}/images`;
  form.action = endpoint;
  form.setAttribute("hx-post", endpoint);

  const formData = new FormData();
  formData.set("image", uploadFile);

  setStatus(root, messages.uploading);
  const response = await fetch(endpoint, {
    method: "POST",
    headers: {
      "HX-Request": "true",
    },
    body: formData,
  });

  if (!response.ok) {
    setStatus(root, messages.failed);
    return;
  }

  const fragment = await response.text();
  const imageID = extractImageID(fragment);
  if (imageID > 0) {
    appendThumbnail(root, uploadFile, imageID, entryID);
  }
  fileInput.value = "";
  setStatus(root, "");
}

async function handleDelete(root, button) {
  const messages = parseMessages(root);
  const entryID = Number(button.dataset.entryId);
  const imageID = Number(button.dataset.imageId);
  if (!Number.isFinite(entryID) || entryID <= 0 || !Number.isFinite(imageID) || imageID <= 0) {
    return;
  }

  const response = await fetch(`/entry/${entryID}/images/${imageID}`, {
    method: "DELETE",
    headers: {
      "HX-Request": "true",
    },
  });

  if (!response.ok) {
    setStatus(root, messages.deleted);
    return;
  }

  const item = button.closest("[data-image-id]");
  if (item instanceof HTMLElement) {
    item.remove();
  }
}

function initImageUpload(rootScope = document) {
  const roots = rootScope instanceof Element && rootScope.matches("[data-image-upload-root]")
    ? [rootScope]
    : Array.from(rootScope.querySelectorAll?.("[data-image-upload-root]") || []);

  roots.forEach((root) => {
    if (!(root instanceof HTMLElement) || root.dataset.imageUploadReady === "true") {
      return;
    }

    const form = root.querySelector("[data-image-upload-form]");
    const fileInput = root.querySelector("[data-image-upload-input]");
    if (!(form instanceof HTMLFormElement) || !(fileInput instanceof HTMLInputElement)) {
      return;
    }

    root.dataset.imageUploadReady = "true";

    form.addEventListener("submit", (event) => {
      event.preventDefault();
      handleUpload(root, form, fileInput).catch(() => {
        const messages = parseMessages(root);
        setStatus(root, messages.failed);
      });
    });

    root.addEventListener("click", (event) => {
      const target = event.target instanceof Element ? event.target.closest("button[data-image-id][data-entry-id]") : null;
      if (!(target instanceof HTMLButtonElement)) {
        return;
      }

      handleDelete(root, target).catch(() => {
        const messages = parseMessages(root);
        setStatus(root, messages.deleted);
      });
    });
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => initImageUpload(document), { once: true });
} else {
  initImageUpload(document);
}

document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.detail.target instanceof Element) {
    initImageUpload(event.detail.target);
  }
});
