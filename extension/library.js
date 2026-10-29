import { ensureReview } from "./review.js";
import { getRoadmaps } from "./roadmaps.js";

let problems = [];
let feedbackTimer = 0;

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

function escapeHtml(value) {
  return String(value ?? "").replace(/[&<>"']/g, (character) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;"
  })[character]);
}

function setHtml(element, markup) {
  const doc = new DOMParser().parseFromString(markup, "text/html");
  element.replaceChildren(...doc.body.childNodes);
}

function solutionsOf(problem) {
  if (Array.isArray(problem.solutions) && problem.solutions.length) {
    return problem.solutions;
  }
  return [
    {
      approach: "",
      code: problem.code || "",
      notes: problem.notes || "",
      timeComplexity: problem.timeComplexity || "",
      spaceComplexity: problem.spaceComplexity || ""
    }
  ];
}

function today() {
  return new Date().toISOString().slice(0, 10);
}

function difficultyOf(problem) {
  return ["Easy", "Medium", "Hard"].includes(problem.difficulty)
    ? problem.difficulty
    : "Unknown";
}

function reviewState(problem, day) {
  const review = problem.review || {};
  if (review.skipped) return "skipped";
  if (!review.due || review.due <= day) return "due";
  return "scheduled";
}

function daysBetween(from, to) {
  const start = new Date(`${from}T00:00:00Z`).getTime();
  const end = new Date(`${to}T00:00:00Z`).getTime();
  if (Number.isNaN(start) || Number.isNaN(end)) return 0;
  return Math.max(0, Math.round((end - start) / 86400000));
}

function reviewLabel(problem, day) {
  const review = problem.review || {};
  if (review.skipped) return "No review";
  const days = daysBetween(day, review.due || day);
  if (days <= 0) return "Due now";
  return days === 1 ? "Due tomorrow" : `Due in ${days}d`;
}

function safeUrl(value) {
  const text = String(value || "");
  return /^https?:\/\//i.test(text) ? text : "";
}

function solutionMarkup(problem, solution, problemIndex, solutionIndex) {
  const approach = String(solution.approach || "").trim() || "Default solution";
  const language = solution.language || problem.language || "";
  return `<div class="solution-card" data-problem="${problemIndex}" data-solution="${solutionIndex}">
    <div class="solution-head"><strong>${escapeHtml(approach)}</strong><span class="muted">${escapeHtml(language)}</span></div>
    <div class="solution-grid">
      <label>Time complexity<input class="cx-time" type="text" value="${escapeHtml(solution.timeComplexity || "")}" placeholder="O(N)"></label>
      <label>Space complexity<input class="cx-space" type="text" value="${escapeHtml(solution.spaceComplexity || "")}" placeholder="O(1)"></label>
    </div>
    <label class="notes-label">Notes<textarea class="notes" rows="4" placeholder="Aha! moment, key trick, or mistake to remember…">${escapeHtml(solution.notes || "")}</textarea></label>
    <details class="code-toggle"><summary>Show code</summary><pre class="code"><code>${escapeHtml(solution.code || "No code")}</code></pre></details>
    <div class="solution-actions"><button type="button" class="save">Save &amp; sync</button></div>
  </div>`;
}

function problemMarkup(problem, index) {
  const day = today();
  const state = reviewState(problem, day);
  const difficulty = difficultyOf(problem);
  const roadmaps = getRoadmaps(problem.number)
    .map((name) => `<span class="badge roadmap">${escapeHtml(name)}</span>`)
    .join("");
  const solutions = solutionsOf(problem);
  const tags = (problem.tags || [])
    .map((tag) => `<span class="chip">${escapeHtml(tag)}</span>`)
    .join("");
  const number = /^\d+$/.test(String(problem.number)) ? `${problem.number}. ` : "";
  const title = `${number}${problem.title || problem.slug || "Untitled"}`;
  const url = safeUrl(problem.url);

  return `<details class="problem" data-index="${index}">
    <summary>
      <span class="problem-title"><strong>${escapeHtml(title)}</strong></span>
      <span class="problem-meta">
        <span class="badge ${difficulty.toLowerCase()}">${difficulty}</span>
        ${roadmaps}
        <span class="badge ${state}">${escapeHtml(reviewLabel(problem, day))}</span>
        <span class="muted">${escapeHtml(problem.language || "")} · ${solutions.length} solution${solutions.length === 1 ? "" : "s"}</span>
      </span>
    </summary>
    <div class="problem-body">
      <div class="chips">${tags || '<span class="muted">No topics</span>'}</div>
      ${url ? `<p><a href="${escapeHtml(url)}" target="_blank" rel="noreferrer">Open on LeetCode ↗</a></p>` : ""}
      ${solutions.map((solution, solutionIndex) => solutionMarkup(problem, solution, index, solutionIndex)).join("")}
    </div>
  </details>`;
}

function currentFilters() {
  return {
    query: document.querySelector("#search").value.trim().toLowerCase(),
    difficulty: document.querySelector("#difficulty").value,
    roadmap: document.querySelector("#roadmap").value,
    review: document.querySelector("#review").value
  };
}

function filterProblems(filters, day) {
  return problems.filter((problem) => {
    if (filters.difficulty && difficultyOf(problem) !== filters.difficulty) return false;
    if (filters.roadmap && !getRoadmaps(problem.number).includes(filters.roadmap)) return false;
    if (filters.review && reviewState(problem, day) !== filters.review) return false;
    if (!filters.query) return true;
    const haystack = [
      problem.number,
      problem.title,
      problem.slug,
      problem.language,
      ...(problem.tags || []),
      ...solutionsOf(problem).map((solution) => solution.approach)
    ]
      .join(" ")
      .toLowerCase();
    return haystack.includes(filters.query);
  });
}

function render() {
  const list = document.querySelector("#list");
  const day = today();
  const filtered = filterProblems(currentFilters(), day);

  document.querySelector("#count").textContent = problems.length
    ? `${filtered.length} of ${problems.length} problem${problems.length === 1 ? "" : "s"}`
    : "No synced problems yet.";

  if (!problems.length) {
    setHtml(list, `<p class="empty">No synced problems yet. Solve a problem on LeetCode and let LeetFlow sync it first.</p>`);
    return;
  }
  if (!filtered.length) {
    setHtml(list, `<p class="empty">No problems match these filters.</p>`);
    return;
  }

  setHtml(list, filtered.map((problem) => problemMarkup(problem, problems.indexOf(problem))).join(""));
  list.querySelectorAll(".save").forEach((button) => {
    button.addEventListener("click", () => saveSolution(button));
  });
}

function describeSave(response) {
  if (!response?.ok) {
    return { text: response?.error || "Save failed.", kind: "error" };
  }
  if (response.status === "unchanged") {
    return { text: "No changes to save.", kind: "ok" };
  }
  const sync = response.syncResult;
  if (sync?.rateLimited) {
    return { text: "Saved locally — GitHub rate limit reached; the queue will retry automatically.", kind: "ok" };
  }
  if (sync?.failed > 0) {
    return { text: sync.errors?.[0] || "Saved locally, but GitHub sync failed.", kind: "error" };
  }
  if (sync?.synced > 0) {
    return { text: "Saved and synced to GitHub.", kind: "ok" };
  }
  if (!response.configured) {
    return { text: "Saved locally — connect GitHub in Settings to enable sync.", kind: "ok" };
  }
  if (response.autoSync === false) {
    return { text: "Saved locally — auto-sync is off. Use Sync / Retry in the popup.", kind: "ok" };
  }
  return { text: "Saved to the local queue.", kind: "ok" };
}

async function saveSolution(button) {
  const card = button.closest(".solution-card");
  if (!card) return;
  const problem = problems[Number(card.dataset.problem)];
  if (!problem) return;

  button.disabled = true;
  button.textContent = "Saving…";
  const response = await send("save-problem", {
    key: problem.key,
    index: Number(card.dataset.solution),
    fields: {
      notes: card.querySelector(".notes").value,
      timeComplexity: card.querySelector(".cx-time").value.trim(),
      spaceComplexity: card.querySelector(".cx-space").value.trim()
    }
  });

  await load(describeSave(response));
}

function showFeedback(text, kind = "ok") {
  const node = document.querySelector("#feedback");
  node.hidden = false;
  node.className = `feedback ${kind}`;
  node.textContent = text;
  clearTimeout(feedbackTimer);
  feedbackTimer = setTimeout(() => {
    node.hidden = true;
  }, 8000);
}

async function load(feedback) {
  const response = await send("get-state");
  if (!response.ok) {
    showFeedback(response.error || "Could not read extension state.", "error");
    return;
  }

  problems = Object.entries(response.state.problems || {})
    .filter(([, value]) => value && (value.title || value.slug))
    .map(([key, value]) => {
      const reviewed = ensureReview(value);
      return { ...value, key, review: reviewed.review };
    })
    .sort(
      (a, b) =>
        Number(a.number || 0) - Number(b.number || 0) ||
        String(a.title || "").localeCompare(String(b.title || ""))
    );

  render();
  if (feedback) showFeedback(feedback.text, feedback.kind);
}

for (const id of ["search", "difficulty", "roadmap", "review"]) {
  const node = document.querySelector(`#${id}`);
  node.addEventListener("input", render);
  node.addEventListener("change", render);
}
document.querySelector("#refresh").addEventListener("click", () => load());
load();
