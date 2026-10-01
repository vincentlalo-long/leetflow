import { getRoadmapProgress } from "./roadmaps.js";
import { loadState, saveState } from "./storage.js";

function send(type, payload = {}) {
  return new Promise((resolve) => {
    try {
      chrome.runtime.sendMessage({ type, ...payload }, (response) => {
        const failure = chrome.runtime.lastError;
        if (failure) {
          resolve({ ok: false, error: failure.message || String(failure) });
          return;
        }
        resolve(response || { ok: false, error: "No response from the background." });
      });
    } catch (error) {
      resolve({ ok: false, error: error?.message || String(error) });
    }
  });
}

function relativeTime(iso) {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  if (Number.isNaN(diff)) return "";
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function escapeHtml(value) {
  return String(value ?? "").replace(/[&<>"']/g, (character) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;"
  })[character]);
}

function daysUntil(dateString) {
  if (!dateString) return "";
  const target = new Date(`${dateString}T00:00:00Z`).getTime();
  if (Number.isNaN(target)) return "";
  const days = Math.round((target - Date.now()) / 86400000);
  if (days <= 0) return "due now";
  return `in ${days} day${days === 1 ? "" : "s"}`;
}

let feedbackTimer = 0;

function showFeedback(text, kind = "ok") {
  const node = document.querySelector("#reviewFeedback");
  if (!node) return;
  node.hidden = false;
  node.className = `feedback ${kind}`;
  node.textContent = text;
  clearTimeout(feedbackTimer);
  feedbackTimer = setTimeout(() => {
    node.hidden = true;
  }, 8000);
}

async function refresh() {
  const response = await send("get-state");
  if (!response.ok) throw new Error(response.error || "Could not read extension state.");
  const state = response.state;
  const queue = state.queue || [];
  const pending = queue.filter((job) => job.status === "pending").length;
  const failed = queue.filter((job) => job.status === "failed");
  const problems = Object.values(state.problems || {});

  document.querySelector("#status").textContent = state.settings.repo
    ? `${state.settings.owner}/${state.settings.repo}`
    : "Not configured";

  const lastSync = state.lastSync;
  document.querySelector("#lastsync").textContent = lastSync
    ? `Last sync ${relativeTime(lastSync.at)} · ${String(lastSync.sha || "").slice(0, 7)} · ${lastSync.count} file set${lastSync.count === 1 ? "" : "s"}`
    : "Nothing synced yet.";

  const notes = [];
  if (failed.length > 0 && failed[0].error) {
    notes.push(`<div class="note error">⚠️ ${escapeHtml(failed[0].error)}</div>`);
  } else if (state.lastError) {
    notes.push(`<div class="note error">⚠️ ${escapeHtml(state.lastError)}</div>`);
  }

  document.querySelector("#summary").innerHTML = `
    <div class="row"><span>Pending</span><strong>${pending}</strong></div>
    <div class="row"><span>Failed</span><strong class="${failed.length ? "failed" : ""}">${failed.length}</strong></div>
    <div class="row"><span>Synced records</span><strong>${problems.length}</strong></div>
    ${notes.join("")}`;

  const roadmapStats = getRoadmapProgress(problems);

  document.querySelector("#b75-count").textContent =
    `${roadmapStats.blind75.solved}/${roadmapStats.blind75.total} (${roadmapStats.blind75.percent}%)`;
  document.querySelector("#b75-bar").style.width = `${roadmapStats.blind75.percent}%`;

  document.querySelector("#nc150-count").textContent =
    `${roadmapStats.neetcode150.solved}/${roadmapStats.neetcode150.total} (${roadmapStats.neetcode150.percent}%)`;
  document.querySelector("#nc150-bar").style.width = `${roadmapStats.neetcode150.percent}%`;

  const review = await send("get-review");
  if (!review.ok) throw new Error(review.error || "Could not read the review queue.");
  const first = review.due?.[0];
  const today = new Date().toISOString().slice(0, 10);
  const titleHtml = first
    ? first.url
      ? `<a href="${escapeHtml(first.url)}" target="_blank" rel="noreferrer" style="color: #0969da; text-decoration: none; font-weight: 500;">${escapeHtml(first.number)}. ${escapeHtml(first.title)} ↗</a>`
      : `<span>${escapeHtml(first.number)}. ${escapeHtml(first.title)}</span>`
    : "";

  const upcoming = review.upcoming || [];
  const skippedCount = review.skipped || 0;
  const scheduleHtml = upcoming.length
    ? `<div class="schedule-label">Schedule</div><ul class="schedule">${upcoming
        .map(
          (item) => `<li class="${item.due <= today ? "due" : ""}">
            <span class="schedule-title">${escapeHtml(item.number)}. ${escapeHtml(item.title)}</span>
            <span class="schedule-date">${escapeHtml(item.due)} · ${escapeHtml(daysUntil(item.due))}</span>
          </li>`
        )
        .join("")}</ul>`
    : `<div class="schedule-label">Schedule</div><small class="schedule-empty">Nothing scheduled yet.</small>`;

  const skippedHtml =
    skippedCount > 0
      ? `<div class="schedule-actions"><span>${skippedCount} skipped</span><button type="button" id="restoreReviews">Restore</button></div>`
      : "";

  document.querySelector("#review").innerHTML = first
    ? `<strong>Review now</strong><div class="review-current">${titleHtml}</div><small>Due ${escapeHtml(first.review.due)} · ${escapeHtml(daysUntil(first.review.due))} · ${review.due.length} due</small>${scheduleHtml}${skippedHtml}`
    : `<strong>Review queue</strong><div>Nothing due right now.</div><small>${review.total || 0} tracked problems</small>${scheduleHtml}${skippedHtml}`;

  document.querySelector("#restoreReviews")?.addEventListener("click", async (event) => {
    event.currentTarget.disabled = true;
    const response = await send("restore-reviews");
    if (!response.ok) {
      showFeedback(`⚠️ Could not restore — ${response.error || "unknown error"}`, "error");
    } else {
      showFeedback(`↺ Restored ${response.restored} problem(s) — they are due now.`);
    }
    await refresh().catch(() => {});
  });

  document.querySelectorAll("#grades button").forEach((button) => {
    button.disabled = !first;
    button.onclick = async () => {
      if (!first) return;
      const grade = button.dataset.grade;
      const label = button.textContent.trim();
      const key = `${first.number}:${first.language}`;
      button.disabled = true;
      const response =
        grade === "skip"
          ? await send("skip-review", { key })
          : await send("grade-review", { key, grade });
      if (!response.ok) {
        showFeedback(`⚠️ ${label} not saved — ${response.error || "unknown error"}`, "error");
      } else if (grade === "skip") {
        showFeedback(`⏭ ${first.number}. ${first.title} · no review needed — removed from schedule`);
      } else {
        const due = response.item?.review?.due || "";
        showFeedback(`✅ ${first.number}. ${first.title} · ${label} → next review ${due} (${daysUntil(due)})`);
      }
      await refresh().catch(() => {});
    };
  });

  const interviewCheckbox = document.querySelector("#interviewMode");
  interviewCheckbox.checked = Boolean(state.settings.interviewMode);
  interviewCheckbox.onchange = async () => {
    const current = await loadState();
    current.settings.interviewMode = interviewCheckbox.checked;
    await saveState(current);
    await refresh();
  };
}

function exportToAnki(problems) {
  if (!problems.length) {
    alert("No synced problems to export yet! Solve some problems first.");
    return;
  }

  const rows = problems.map((problem) => {
    const front = `<h2>${escapeHtml(problem.number)}. ${escapeHtml(problem.title)}</h2><p><b>Difficulty:</b> ${escapeHtml(problem.difficulty)}</p><p><b>Tags:</b> ${escapeHtml((problem.tags || []).join(", ") || "None")}</p><p><a href="${escapeHtml(problem.url)}">LeetCode Link</a></p>`;
    const notesHtml = problem.notes
      ? `<div style="background:#eef;padding:8px;border-radius:4px;"><b>Insight / Notes:</b><br>${escapeHtml(problem.notes).replace(/\n/g, "<br>")}</div>`
      : "";
    const back = `${notesHtml}<h3>Solution (${escapeHtml(problem.language)})</h3><pre style="background:#f4f4f4;padding:8px;border-radius:4px;overflow-x:auto;"><code>${escapeHtml(problem.code || "No code")}</code></pre><p><small>Time spent: ${escapeHtml(problem.timeSpent || "N/A")} · Attempts: ${escapeHtml(problem.attemptsSummary || "Clean AC")}</small></p>`;
    return `${front.replace(/\t/g, " ").replace(/\r?\n/g, "")}\t${back.replace(/\t/g, " ").replace(/\r?\n/g, "")}`;
  });

  const blob = new Blob([rows.join("\n")], { type: "text/tab-separated-values;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = `LeetFlow_Anki_Deck_${new Date().toISOString().slice(0, 10)}.txt`;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  URL.revokeObjectURL(url);
}

document.querySelector("#retry").addEventListener("click", async () => {
  const button = document.querySelector("#retry");
  button.disabled = true;
  button.textContent = "Syncing…";
  await send("retry-queue");
  await refresh().catch(() => {});
  button.disabled = false;
  button.textContent = "Sync / Retry now";
});

document.querySelector("#exportAnki").addEventListener("click", async () => {
  const response = await send("get-state");
  if (!response.ok) return;
  exportToAnki(Object.values(response.state.problems || {}));
});

document.querySelector("#options").addEventListener("click", () => chrome.runtime.openOptionsPage());

refresh().catch((error) => {
  document.querySelector("#status").textContent = "Error";
  document.querySelector("#summary").innerHTML =
    `<div class="note error">⚠️ ${escapeHtml(error.message)}</div>`;
});
