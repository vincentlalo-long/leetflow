import { getRoadmapProgress } from "./roadmaps.js";
import { loadState, saveState } from "./storage.js";

function send(type, payload = {}) {
  return new Promise((resolve) => chrome.runtime.sendMessage({ type, ...payload }, resolve));
}

async function refresh() {
  const response = await send("get-state");
  if (!response.ok) throw new Error(response.error);
  const state = response.state;
  const queue = state.queue || [];
  const failed = queue.filter((job) => job.status === "failed");
  
  document.querySelector("#status").textContent = state.settings.repo ? `${state.settings.owner}/${state.settings.repo}` : "Not configured";
  document.querySelector("#summary").innerHTML = `
    <div class="row"><span>Pending</span><strong>${queue.filter((job) => job.status === "pending").length}</strong></div>
    <div class="row"><span>Failed</span><strong class="${failed.length ? "failed" : ""}">${failed.length}</strong></div>
    <div class="row"><span>Synced records</span><strong>${Object.keys(state.synced || {}).length}</strong></div>`;

  // Roadmap progress
  const syncedList = Object.values(state.syncedProblems || {});
  const roadmapStats = getRoadmapProgress(syncedList);
  
  document.querySelector("#b75-count").textContent = `${roadmapStats.blind75.solved}/${roadmapStats.blind75.total} (${roadmapStats.blind75.percent}%)`;
  document.querySelector("#b75-bar").style.width = `${roadmapStats.blind75.percent}%`;
  
  document.querySelector("#nc150-count").textContent = `${roadmapStats.neetcode150.solved}/${roadmapStats.neetcode150.total} (${roadmapStats.neetcode150.percent}%)`;
  document.querySelector("#nc150-bar").style.width = `${roadmapStats.neetcode150.percent}%`;

  // Review state
  const review = await send("get-review");
  const first = review.due?.[0];
  const titleHtml = first
    ? (first.url
        ? `<a href="${first.url}" target="_blank" style="color: #0969da; text-decoration: none; font-weight: 500;">${first.number}. ${first.title} ↗</a>`
        : `<span>${first.number}. ${first.title}</span>`)
    : "";
    
  document.querySelector("#review").innerHTML = first
    ? `<strong>Review now</strong><div style="margin: 4px 0;">${titleHtml}</div><small>Due ${first.review.due} · ${review.due.length} due</small>`
    : `<strong>Review queue</strong><div>Nothing due right now.</div><small>${review.total || 0} tracked problems</small>`;

  document.querySelectorAll("#grades button").forEach((button) => {
    button.disabled = !first;
    button.onclick = async () => {
      await send("grade-review", { key: `${first.number}:${first.language}`, grade: button.dataset.grade });
      await refresh();
    };
  });

  // Interview Mode toggle
  const interviewCheckbox = document.querySelector("#interviewMode");
  interviewCheckbox.checked = Boolean(state.settings.interviewMode);
  interviewCheckbox.onchange = async () => {
    const currentState = await loadState();
    currentState.settings.interviewMode = interviewCheckbox.checked;
    await saveState(currentState);
  };
}

function exportToAnki(syncedProblems) {
  const problems = Object.values(syncedProblems || {});
  if (!problems.length) {
    alert("No synced problems to export yet! Solve some problems first.");
    return;
  }

  // Format: Front \t Back (TSV format compatible with Anki import)
  const rows = problems.map((p) => {
    const front = `<h2>${p.number}. ${p.title}</h2><p><b>Difficulty:</b> ${p.difficulty}</p><p><b>Tags:</b> ${(p.tags || []).join(", ") || "None"}</p><p><a href="${p.url}">LeetCode Link</a></p>`;
    const notesHtml = p.notes ? `<div style="background:#eef;padding:8px;border-radius:4px;"><b>Insight / Notes:</b><br>${p.notes.replace(/\n/g, "<br>")}</div>` : "";
    const back = `${notesHtml}<h3>Solution (${p.language})</h3><pre style="background:#f4f4f4;padding:8px;border-radius:4px;overflow-x:auto;"><code>${p.code ? p.code.replace(/</g, "&lt;").replace(/>/g, "&gt;") : "No code"}</code></pre><p><small>Time spent: ${p.timeSpent || "N/A"} · Attempts: ${p.attemptsSummary || "Clean AC"}</small></p>`;
    // Strip tabs and newlines from TSV columns
    const cleanFront = front.replace(/\t/g, " ").replace(/\r?\n/g, "");
    const cleanBack = back.replace(/\t/g, " ").replace(/\r?\n/g, "");
    return `${cleanFront}\t${cleanBack}`;
  });

  const content = rows.join("\n");
  const blob = new Blob([content], { type: "text/tab-separated-values;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `LeetFlow_Anki_Deck_${new Date().toISOString().slice(0, 10)}.txt`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

document.querySelector("#retry").addEventListener("click", async () => {
  await send("retry-queue");
  await refresh();
});

document.querySelector("#exportAnki").addEventListener("click", async () => {
  const { state } = await send("get-state");
  exportToAnki(state.syncedProblems);
});

document.querySelector("#options").addEventListener("click", () => chrome.runtime.openOptionsPage());
refresh();
