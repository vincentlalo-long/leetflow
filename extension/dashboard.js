import {
  collectActivity,
  difficultyStats,
  gradeStats,
  heatmapWeeks,
  streakStats,
  tagStats
} from "./stats.js";

const GRADE_ROWS = [
  { key: "again", label: "Again", color: "#cf222e" },
  { key: "hard", label: "Hard", color: "#bf8700" },
  { key: "good", label: "Good", color: "#1a7f37" },
  { key: "easy", label: "Easy", color: "#0969da" }
];

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

function percent(value) {
  return `${Math.round(value * 100)}%`;
}

function renderKpis(problems, today) {
  const streak = streakStats(problems, today);
  const grades = gradeStats(problems);
  const activeDays = collectActivity(problems).size;
  const items = [
    { label: "Current streak", value: `${streak.current} day${streak.current === 1 ? "" : "s"}` },
    { label: "Longest streak", value: `${streak.longest} day${streak.longest === 1 ? "" : "s"}` },
    { label: "Active days", value: activeDays },
    { label: "Reviews graded", value: grades.total },
    { label: "Recall rate", value: grades.total ? percent(grades.recallRate) : "—" },
    { label: "Problems tracked", value: problems.length }
  ];

  setHtml(
    document.querySelector("#kpis"),
    items
      .map((item) => `<div class="kpi"><span>${escapeHtml(item.label)}</span><strong>${escapeHtml(item.value)}</strong></div>`)
      .join("")
  );
}

function heatLevel(total) {
  if (total <= 0) return 0;
  if (total === 1) return 1;
  if (total === 2) return 2;
  if (total <= 4) return 3;
  return 4;
}

function renderHeatmap(problems, today) {
  const weeks = heatmapWeeks(problems, { today });
  const cells = weeks
    .map(
      (week) =>
        `<div class="week">${week
          .map((day) => {
            const title = `${day.date}: ${day.solves} solved · ${day.reviews} reviewed`;
            return `<i class="cell level${heatLevel(day.total)}${day.future ? " future" : ""}" title="${escapeHtml(title)}"></i>`;
          })
          .join("")}</div>`
    )
    .join("");

  setHtml(document.querySelector("#heatmap"), `<div class="weeks">${cells}</div>`);
  document.querySelector("#activityRange").textContent =
    `${weeks[0][0].date} → ${weeks[weeks.length - 1][6].date}`;
}

function renderGrades(problems) {
  const grades = gradeStats(problems);
  document.querySelector("#recall").textContent = grades.total
    ? `${percent(grades.recallRate)} recalled without "Again"`
    : "No graded reviews yet";

  if (!grades.total) {
    setHtml(document.querySelector("#grades"), `<p class="empty">Grade a due review in the popup to start tracking recall.</p>`);
    return;
  }

  const denominator = grades.total + grades.counts.skip;
  const rows = GRADE_ROWS.map((row) => {
    const count = grades.counts[row.key] || 0;
    const width = (count / denominator) * 100;
    return `<div class="bar-row"><span>${row.label}</span><div class="bar-track"><div class="bar-fill" style="width:${width}%;background:${row.color}"></div></div><span class="bar-value">${count}</span></div>`;
  });

  if (grades.counts.skip > 0) {
    rows.push(
      `<div class="bar-row"><span>Skipped</span><div class="bar-track"><div class="bar-fill" style="width:${(grades.counts.skip / denominator) * 100}%;background:#8c959f"></div></div><span class="bar-value">${grades.counts.skip}</span></div>`
    );
  }

  setHtml(document.querySelector("#grades"), rows.join(""));
}

function renderDifficulty(problems) {
  const counts = difficultyStats(problems);
  const total = Object.values(counts).reduce((sum, value) => sum + value, 0);
  if (!total) {
    setHtml(document.querySelector("#difficulty"), `<p class="empty">No problems tracked yet.</p>`);
    return;
  }

  const colors = { Easy: "#1a7f37", Medium: "#bf8700", Hard: "#cf222e", Unknown: "#8c959f" };
  const rows = ["Easy", "Medium", "Hard", "Unknown"]
    .filter((key) => counts[key] > 0)
    .map(
      (key) =>
        `<div class="bar-row"><span>${key}</span><div class="bar-track"><div class="bar-fill" style="width:${(counts[key] / total) * 100}%;background:${colors[key]}"></div></div><span class="bar-value">${counts[key]}</span></div>`
    );

  setHtml(document.querySelector("#difficulty"), rows.join(""));
}

function renderTags(problems) {
  const tags = tagStats(problems);
  if (!tags.length) {
    setHtml(document.querySelector("#tags"), `<p class="empty">No topic tags yet.</p>`);
    return;
  }

  setHtml(
    document.querySelector("#tags"),
    `<div class="chips">${tags
      .map((entry) => `<span class="chip">${escapeHtml(entry.tag)} · ${entry.count}</span>`)
      .join("")}</div>`
  );
}

async function load() {
  const errorNode = document.querySelector("#error");
  errorNode.hidden = true;
  try {
    const response = await send("get-state");
    if (!response.ok) throw new Error(response.error || "Could not read extension state.");
    const problems = Object.values(response.state.problems || {}).filter(Boolean);
    const today = new Date().toISOString().slice(0, 10);

    renderKpis(problems, today);
    renderHeatmap(problems, today);
    renderGrades(problems);
    renderDifficulty(problems);
    renderTags(problems);
    document.querySelector("#empty").hidden = problems.length > 0;
  } catch (error) {
    errorNode.hidden = false;
    errorNode.textContent = error?.message || String(error);
  }
}

document.querySelector("#refresh").addEventListener("click", () => load());
load();
