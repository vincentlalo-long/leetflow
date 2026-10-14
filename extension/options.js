import { loadState } from "./storage.js";
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
  const hint = kind === "error" ? "" : " You can close this tab.";
  node.textContent = text + hint;
  node.style.color = kind === "error" ? "#cf222e" : kind === "warn" ? "#9a6700" : "#1a7f37";
  const close = $("closeTab");
  if (close) close.hidden = kind === "error";
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
  try {
    const version = await send("get-version");
    if (version?.ok && version.version) $("version").textContent = version.version;
    else $("version").textContent = chrome.runtime.getManifest?.().version || "…";
  } catch {
    try {
      $("version").textContent = chrome.runtime.getManifest?.().version || "…";
    } catch {
      /* cosmetic */
    }
  }
  try {
    const update = await send("check-update");
    const node = $("updateStatus");
    if (update?.ok && update.updateAvailable) {
      node.innerHTML = "";
      node.append(`Update ${update.current} → ${update.latest} available — `);
      const link = document.createElement("a");
      link.href = update.url || "https://github.com/vincentlalo-long/leetflow/releases";
      link.target = "_blank";
      link.rel = "noreferrer";
      link.textContent = "download here";
      node.append(link);
    } else if (update?.ok) {
      node.textContent = `You're on the latest version (${update.current}).`;
    } else {
      node.textContent = "Could not check for updates right now.";
    }
  } catch {
    const node = $("updateStatus");
    if (node) node.textContent = "Could not check for updates right now.";
  }
}

function downloadBackup(backup) {
  const blob = new Blob([JSON.stringify(backup, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = `leetflow-backup-${new Date().toISOString().slice(0, 10)}.json`;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  URL.revokeObjectURL(url);
}

async function save() {
  const settings = {};
  for (const field of fields) {
    const node = $(field);
    if (!node) continue;
    settings[field] = node.type === "checkbox" ? node.checked : node.value.trim();
  }

  const saved = await send("save-settings", { settings });
  if (!saved.ok) {
    throw new Error(saved.error || "Could not save the settings.");
  }
  setMessage("Settings saved.", "ok");

  if (!(settings.githubToken && settings.owner && settings.repo)) {
    return;
  }

  try {
    await checkRepository(settings.githubToken, settings.owner, settings.repo);
  } catch (error) {
    const status = error?.status;
    const raw = error?.message || String(error);
    const friendly =
      status === 401 ? "GitHub rejected the token (401). It is invalid or expired."
      : status === 404 ? "Repository not found (404). Check the owner/repository names."
      : status === 403 ? `${raw} — you may have hit the GitHub rate limit.`
      : raw;
    setMessage(`Saved, but the repository check failed: ${friendly}`, "warn");
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

$("closeTab").addEventListener("click", () => {
  setMessage("Settings saved. Close this tab manually (Ctrl+W / Cmd+W).", "ok");
  try {
    window.close();
  } catch {
    /* the browser refused to close the tab — the hint above stays visible */
  }
});

$("exportBackup")?.addEventListener("click", async () => {
  const response = await send("export-backup");
  if (!response?.ok) {
    setMessage(response?.error || "Could not export the backup.", "error");
    return;
  }
  downloadBackup(response.backup);
  setMessage(`Backup exported (${Object.keys(response.backup?.state?.problems || {}).length} problems).`, "ok");
});

$("importBackup")?.addEventListener("change", async (event) => {
  const file = event.target?.files?.[0];
  if (!file) return;
  try {
    const text = await file.text();
    const backup = JSON.parse(text);
    const response = await send("import-backup", { backup });
    if (!response?.ok) throw new Error(response?.error || "Could not import the backup.");
    await load();
    setMessage("Backup imported. Your data has been restored.", "ok");
  } catch (error) {
    setMessage(error?.message || String(error), "error");
  } finally {
    event.target.value = "";
  }
});

load().catch((error) => setMessage(error.message || String(error), "error"));
