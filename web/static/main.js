import "./dist/mecfs.js";
import "./submit-state.js";
import "./forms.js";
import "./entries.js";
import "./trackable-picker.js";
import "./trackable-presets.js";
import "./settings.js";
import "./voice-recorder.js";
import "./auth.js";
import "./image-upload.js";

/* function syncPopoverDataOpen(popover) {
  if (!(popover instanceof HTMLElement) || !popover.hasAttribute("popover")) {
    return;
  }

  const isOpen =
    popover.matches(":popover-open") ||
    popover.matches("[open]") ||
    popover.getAttribute("data-open") === "";

  if (isOpen) {
    popover.setAttribute("data-open", "");
    return;
  }

  popover.removeAttribute("data-open");
}

function installPopoverStateMirror() {
  document.querySelectorAll("[popover]").forEach((popover) => {
    syncPopoverDataOpen(popover);
  });

  document.addEventListener(
    "toggle",
    (event) => {
      const target = event.target;
      if (!(target instanceof HTMLElement)) {
        return;
      }
      syncPopoverDataOpen(target);
    },
    true,
  );
}

installPopoverStateMirror(); */

function formatRelativeTime(date) {
  const now = new Date();
  const diffInSeconds = Math.floor((now - date) / 1000);

  // Define units and their thresholds
  const units = [
    { unit: 'year', seconds: 31536000 },
    { unit: 'month', seconds: 2592000 },
    { unit: 'day', seconds: 86400 },
    { unit: 'hour', seconds: 3600 },
    { unit: 'minute', seconds: 60 },
    { unit: 'second', seconds: 1 },
  ];

  for (const { unit, seconds } of units) {
    if (Math.abs(diffInSeconds) >= seconds || unit === 'second') {
      const value = Math.floor(diffInSeconds / seconds);
      const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' }); // Use the user's locale
      return rtf.format(-value, unit);
    }
  }
}

class RelativeTime extends HTMLTimeElement {
  connectedCallback() {
    this.update();
    this.interval = setInterval(() => this.update(), 60000);
  }

  disconnectedCallback() {
    clearInterval(this.interval);
  }

  update() {
    const datetimeAttr = this.getAttribute('datetime');
    if (!datetimeAttr) return;
    const date = new Date(datetimeAttr);
    this.textContent = formatRelativeTime(date);
  }
}

customElements.define('relative-time', RelativeTime, { extends: 'time' });