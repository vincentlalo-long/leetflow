import { loadState, saveState } from "./storage.js";
import { checkRepository } from "./github.js";

const fields = [
  "githubToken",
  "owner",
  "repo",
  "branch",
  "rootDir",
  "autoSync",
  "updateRootReadme",
  "interviewMode",
  "promptNotes"
];
const $ = (id) => document.getElementById(id);

function setMessage(text, kind) {
  const node = $("message");
  node.textContent = text;
  node.style.color = kind === "error" ? "#cf222e" : kind === "warn" ? "#9a6700" : "#1a7f37";
}

function send(type, payload = {}) {
  return new Promise((resolve) => {
    try {
      chrome.runtime.sendMessage({ type, ...payload }, (response) => {
        const failure = chrome.runtime.lastError;
        if (failure) resolve({ ok: false, error: failure.message || String(failure) });
        else resolve(response || { ok: false, error: "No response from the background." });
      });
    } catch (error) {
      resolve({ ok: false, error: error?.message || String(error) });
    }
  });
}

async function load() {
  const state = await loadState();
  const settings = state.settings || {};
  for (const field of fields) {
    const node = $(field);
    if (!node) continue;
    if (node.type === "checkbox") node.checked = Boolean(settings[field]);
    else node.value = settings[field] || "";
  }
}

async function save() {
  const settings = {};
  for (const field of fields) {
    const node = $(field);
    if (!node) continue;
    settings[field] = node.type === "checkbox" ? node.checked : node.value.trim();
  }

  const state = await loadState();
  await saveState({ ...state, settings });
  setMessage("Settings saved.", "ok");

  if (!(settings.githubToken && settings.owner && settings.repo)) {
    return;
  }

  try {
    await checkRepository(settings.githubToken, settings.owner, settings.repo);
  } catch (error) {
    setMessage(`Saved, but the repository check failed: ${error.message}`, "warn");
    return;
  }

  const queue = await send("retry-queue");
  if (!queue?.ok) {
    setMessage(`Saved, but sync failed: ${queue?.error || "unknown error"}`, "error");
    return;
  }
  if (queue.rateLimited) {
    setMessage("Saved and verified. GitHub rate limit reached; the queue will retry automatically.", "warn");
    return;
  }
  if (queue.synced > 0) {
    setMessage(`Saved and verified. Synced ${queue.synced} pending solution(s).`, "ok");
    return;
  }
  if (queue.failed > 0) {
    setMessage(`Saved and verified. ${queue.failed} job(s) failed: ${queue.errors?.[0] || ""}`, "error");
    return;
  }
  setMessage("Saved, verified, and the queue is up to date.", "ok");
}

$("save").addEventListener("click", () =>
  save().catch((error) => setMessage(error.message || String(error), "error"))
);

load().catch((error) => setMessage(error.message || String(error), "error"));
