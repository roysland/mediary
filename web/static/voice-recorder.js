/**
 * voice-recorder.js
 *
 * Handles MediaRecorder-based audio capture and upload via fetch.
 * Requires the following elements in the DOM (rendered by voice_entry_section template):
 *   #voice-record-btn   — idle mic button
 *   #voice-stop-btn     — stop button (shown while recording)
 *   #voice-idle         — idle state container
 *   #voice-recording    — recording state container
 *   #voice-uploading    — uploading state container
 *   #voice-result       — replaced with server HTML on success
 *   #voice-error        — error message container
 *   #voice-timer        — countdown / elapsed timer display
 */

const MAX_RECORDING_MS = 60_000; // 60 seconds

function initVoiceRecorder() {
  const recordBtn   = document.getElementById("voice-record-btn");
  const stopBtn     = document.getElementById("voice-stop-btn");
  const voiceSection = recordBtn ? recordBtn.closest(".voice-entry-section") : null;
  const idleEl      = document.getElementById("voice-idle");
  const recordingEl = document.getElementById("voice-recording");
  const uploadingEl = document.getElementById("voice-uploading");
  const resultEl    = document.getElementById("voice-result");
  const errorEl     = document.getElementById("voice-error");
  const timerEl     = document.getElementById("voice-timer");

  if (!recordBtn) return; // voice section not present on this page
  if (!stopBtn) return;

  // Prevent duplicate listeners when HTMX swaps unrelated DOM fragments.
  if (voiceSection && voiceSection.dataset.voiceBound === "1") return;
  if (voiceSection) {
    voiceSection.dataset.voiceBound = "1";
  }

  let mediaRecorder = null;
  let audioChunks   = [];
  let timerInterval = null;
  let autoStopTimer = null;
  let elapsedMs     = 0;

  function setState(state) {
    idleEl.hidden      = state !== "idle";
    recordingEl.hidden = state !== "recording";
    uploadingEl.hidden = state !== "uploading";
    resultEl.hidden    = state !== "result";
    errorEl.hidden     = true; // clear error on state change
  }

  function showError(msg) {
    setState("idle");
    errorEl.textContent = msg;
    errorEl.hidden = false;
  }

  function formatTime(ms) {
    const totalSec = Math.floor(ms / 1000);
    const min = Math.floor(totalSec / 60);
    const sec = totalSec % 60;
    return `${min}:${sec.toString().padStart(2, "0")}`;
  }

  function startTimer() {
    elapsedMs = 0;
    timerEl.textContent = formatTime(0);
    timerInterval = setInterval(() => {
      elapsedMs += 1000;
      timerEl.textContent = formatTime(elapsedMs);
    }, 1000);
  }

  function stopTimer() {
    clearInterval(timerInterval);
    timerInterval = null;
  }

  async function startRecording() {
    if (mediaRecorder && mediaRecorder.state === "recording") {
      return;
    }

    errorEl.hidden = true;

    let stream;
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
    } catch (err) {
      showError("Microphone access denied. Please allow microphone access in your browser settings.");
      return;
    }

    audioChunks = [];

    // Pick the best supported MIME type.
    const mimeType = chooseMimeType();
    const options  = mimeType ? { mimeType } : {};

    try {
      mediaRecorder = new MediaRecorder(stream, options);
    } catch (err) {
      stream.getTracks().forEach(t => t.stop());
      showError("Audio recording is not supported in this browser.");
      return;
    }

    const activeRecorder = mediaRecorder;

    activeRecorder.addEventListener("dataavailable", (e) => {
      if (e.data && e.data.size > 0) {
        audioChunks.push(e.data);
      }
    });

    activeRecorder.addEventListener("stop", async () => {
      // Release mic immediately after recording stops.
      stream.getTracks().forEach(t => t.stop());
      stopTimer();
      clearTimeout(autoStopTimer);
      autoStopTimer = null;

      const blob = new Blob(audioChunks, { type: activeRecorder.mimeType || "audio/webm" });
      audioChunks = [];
      mediaRecorder = null;

      // If the user stops immediately, the recorder may not have emitted data yet.
      if (blob.size === 0) {
        showError("Recording was too short. Please try again and speak for at least a moment.");
        return;
      }

      setState("uploading");
      await uploadAudio(blob, activeRecorder.mimeType);
    });

    activeRecorder.start(1000); // collect chunks every 1s

    setState("recording");
    startTimer();

    // Auto-stop after MAX_RECORDING_MS.
    autoStopTimer = setTimeout(() => {
      if (activeRecorder.state === "recording") {
        activeRecorder.stop();
      }
    }, MAX_RECORDING_MS);
  }

  function stopRecording() {
    clearTimeout(autoStopTimer);
    autoStopTimer = null;

    if (mediaRecorder && mediaRecorder.state === "recording") {
      mediaRecorder.stop();
    }
  }

  async function uploadAudio(blob, mimeType) {
    const formData = new FormData();
    const ext = mimeTypeToExtension(mimeType);
    formData.append("audio", blob, `recording.${ext}`);

    let response;
    try {
      response = await fetch("/entry/voice", {
        method: "POST",
        body: formData,
      });
    } catch (err) {
      showError("Network error. Please check your connection and try again.");
      return;
    }

    if (!response.ok) {
      showError("Failed to save voice note. Please try again.");
      return;
    }

    const html = await response.text();
    resultEl.innerHTML = html;
    // Let htmx process any hx-* attributes in the server-returned fragment.
    if (window.htmx) {
      htmx.process(resultEl);
    }
    setState("result");
  }

  function chooseMimeType() {
    const candidates = [
      "audio/webm;codecs=opus",
      "audio/webm",
      "audio/ogg;codecs=opus",
      "audio/mp4",
    ];
    return candidates.find(t => MediaRecorder.isTypeSupported(t)) || "";
  }

  function mimeTypeToExtension(mimeType) {
    if (!mimeType) return "webm";
    if (mimeType.includes("ogg")) return "ogg";
    if (mimeType.includes("mp4")) return "mp4";
    return "webm";
  }

  recordBtn.addEventListener("click", startRecording);
  stopBtn.addEventListener("click", stopRecording);
  setState("idle");
}

document.addEventListener("DOMContentLoaded", initVoiceRecorder);
// Re-initialise after HTMX swaps (covers home page partial refreshes).
document.addEventListener("htmx:afterSwap", initVoiceRecorder);
