import { loadState, saveState } from "./storage.js";
import { checkRepository } from "./github.js";

const fields = ["githubToken", "owner", "repo", "branch", "rootDir", "autoSync", "updateRootReadme", "interviewMode", "promptNotes"];
const $ = (id) => document.getElementById(id);

async function load() {
  const state = await loadState();
  const settings = state?.settings || {};
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
  if (settings.githubToken && settings.owner && settings.repo) {
    await checkRepository(settings.githubToken, settings.owner, settings.repo);
  }
  const state = await loadState();
  await saveState({ ...(state || {}), settings });
  $("message").style.color = "#1a7f37";
  $("message").textContent = "Saved and verified.";

  try {
    const queueRes = await new Promise((resolve) =>
      chrome.runtime.sendMessage({ type: "retry-queue" }, resolve)
    );
    if (queueRes?.synced > 0) {
      $("message").textContent = `Saved and verified! Synced ${queueRes.synced} pending solution(s) to GitHub.`;
    }
  } catch {
    // Handled
  }
}

$("save").addEventListener("click", () => save().catch((error) => {
  $("message").textContent = error.message;
  $("message").style.color = "#cf222e";
}));
load();
