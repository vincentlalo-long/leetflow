let lastKey = "";
const sessionStartTime = Date.now();
const failAttempts = { wa: 0, tle: 0, re: 0, mle: 0, total: 0 };
let lastFailureKey = "";
let isPrompting = false;

function injectInpageScript() {
  if (document.getElementById("leetflow-inpage-script")) return;
  try {
    const script = document.createElement("script");
    script.id = "leetflow-inpage-script";
    script.src = chrome.runtime.getURL("inpage.js");
    (document.head || document.documentElement).appendChild(script);
  } catch {
    // Handled if blocked or already injected
  }
}
injectInpageScript();

function normalizeLanguage(raw) {
  if (!raw) return { name: "Unknown", ext: "txt" };
  const clean = String(raw).trim().toLowerCase();

  const map = {
    "c++": { name: "C++", ext: "cpp" },
    cpp: { name: "C++", ext: "cpp" },
    c: { name: "C", ext: "c" },
    java: { name: "Java", ext: "java" },
    python: { name: "Python", ext: "py" },
    python3: { name: "Python3", ext: "py" },
    py: { name: "Python3", ext: "py" },
    csharp: { name: "C#", ext: "cs" },
    "c#": { name: "C#", ext: "cs" },
    cs: { name: "C#", ext: "cs" },
    javascript: { name: "JavaScript", ext: "js" },
    js: { name: "JavaScript", ext: "js" },
    typescript: { name: "TypeScript", ext: "ts" },
    ts: { name: "TypeScript", ext: "ts" },
    golang: { name: "Go", ext: "go" },
    go: { name: "Go", ext: "go" },
    rust: { name: "Rust", ext: "rs" },
    rs: { name: "Rust", ext: "rs" },
    swift: { name: "Swift", ext: "swift" },
    kotlin: { name: "Kotlin", ext: "kt" },
    kt: { name: "Kotlin", ext: "kt" },
    ruby: { name: "Ruby", ext: "rb" },
    rb: { name: "Ruby", ext: "rb" },
    scala: { name: "Scala", ext: "scala" },
    php: { name: "PHP", ext: "php" },
    dart: { name: "Dart", ext: "dart" },
    elixir: { name: "Elixir", ext: "ex" },
    erlang: { name: "Erlang", ext: "erl" },
    racket: { name: "Racket", ext: "rkt" },
    mysql: { name: "MySQL", ext: "sql" },
    sql: { name: "SQL", ext: "sql" },
    "ms sql server": { name: "MS SQL Server", ext: "sql" },
    oracle: { name: "Oracle", ext: "sql" },
    postgresql: { name: "PostgreSQL", ext: "sql" },
    bash: { name: "Bash", ext: "sh" }
  };

  if (map[clean]) return map[clean];

  for (const [key, val] of Object.entries(map)) {
    if (clean.includes(key)) return val;
  }

  return { name: raw, ext: "txt" };
}

function text(selector) {
  return document.querySelector(selector)?.textContent?.trim() || "";
}

function formatDuration(sec) {
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  if (m === 0) return `${s}s`;
  return `${m}m ${s < 10 ? "0" : ""}${s}s`;
}

function formatAttempts() {
  if (failAttempts.total === 0) return "1st try (Clean AC)";
  const parts = [];
  if (failAttempts.wa > 0) parts.push(`${failAttempts.wa} WA`);
  if (failAttempts.tle > 0) parts.push(`${failAttempts.tle} TLE`);
  if (failAttempts.re > 0) parts.push(`${failAttempts.re} RE`);
  if (failAttempts.mle > 0) parts.push(`${failAttempts.mle} MLE`);
  parts.push("1 AC");
  return `${failAttempts.total + 1} (${parts.join(", ")})`;
}

function checkSubmissionFailure() {
  const statusSelectors = [
    '[data-e2e-locator="submission-result"]',
    '[data-cypress="SubmissionResult"]',
    'div[class*="status__"][class*="error__"]',
    'span[class*="status__"][class*="error__"]',
    '[class*="result-status-"]',
    'div[class*="status-column"]'
  ];

  for (const sel of statusSelectors) {
    const el = document.querySelector(sel);
    if (el) {
      if (el.closest('[class*="discuss"], [class*="solution"], [class*="comment"]')) continue;
      const t = el.textContent.trim().toLowerCase();
      let type = "";
      if (t.includes("wrong answer") || t.includes("解答错误")) type = "wa";
      else if (t.includes("time limit") || t.includes("超出时间限制")) type = "tle";
      else if (t.includes("runtime error") || t.includes("执行出错")) type = "re";
      else if (t.includes("memory limit") || t.includes("超出内存限制")) type = "mle";

      if (type) {
        const failKey = `${type}:${el.textContent.slice(0, 30)}`;
        if (failKey !== lastFailureKey) {
          lastFailureKey = failKey;
          failAttempts[type]++;
          failAttempts.total++;
        }
      }
    }
  }
}

function getEditorCode() {
  if (document.documentElement.dataset.leetflowCode) {
    return document.documentElement.dataset.leetflowCode;
  }
  const textarea = document.querySelector(".monaco-editor textarea, textarea.inputarea");
  if (textarea?.value) return textarea.value;
  const lines = [...document.querySelectorAll(".monaco-editor .view-line")]
    .map((line) => line.textContent || "")
    .join("\n");
  return lines;
}

function getEditorLanguage() {
  const fromMonaco = document.documentElement.dataset.leetflowLang;
  if (fromMonaco) return fromMonaco;

  const candidates = [
    text('[data-cy="lang-select"]'),
    text('button[id*="headlessui-listbox-button"]'),
    text('.ant-select-selection-selected-value'),
    text('[class*="select-language"]')
  ];
  return candidates.find(Boolean) || "unknown";
}

function isAccepted() {
  const statusSelectors = [
    '[data-e2e-locator="submission-result"]',
    '[data-cypress="SubmissionResult"]',
    'div[class*="status__"][class*="success__"]',
    'span[class*="status__"][class*="success__"]',
    '[class*="result-status-accepted"]',
    'div[class*="status-column"]'
  ];

  for (const selector of statusSelectors) {
    const el = document.querySelector(selector);
    if (el) {
      const t = el.textContent.trim().toLowerCase();
      if (t.includes("accepted") || t.includes("通过")) {
        return true;
      }
    }
  }

  const containers = document.querySelectorAll(
    '[class*="submission-result"], [class*="result-container"], [data-layout-path*="console"], [class*="run-code-result"]'
  );
  for (const container of containers) {
    if (container.closest('[class*="discuss"], [class*="solution"], [class*="comment"]')) {
      continue;
    }
    const t = container.textContent.toLowerCase();
    if ((t.includes("accepted") || t.includes("通过")) &&
        (t.includes("runtime") || t.includes("memory") || t.includes("beats") || t.includes("ms") || t.includes("mb"))) {
      return true;
    }
  }

  return false;
}

const problemCache = {};

function getSlug() {
  const match = location.pathname.match(/\/problems\/([^/]+)/);
  return match ? match[1] : "";
}

async function fetchGraphQLProblem(slug) {
  if (!slug) return null;
  try {
    const endpoint = location.hostname.includes("leetcode.cn")
      ? "https://leetcode.cn/graphql"
      : "https://leetcode.com/graphql";

    const res = await fetch(endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: `
          query questionData($titleSlug: String!) {
            question(titleSlug: $titleSlug) {
              questionFrontendId
              title
              difficulty
              topicTags {
                name
              }
              content
            }
          }
        `,
        variables: { titleSlug: slug }
      })
    });

    if (!res.ok) return null;
    const json = await res.json();
    return json?.data?.question || null;
  } catch (e) {
    console.warn("[LeetFlow] Failed to fetch GraphQL problem info:", e);
    return null;
  }
}

function extractFromDOM(slug) {
  try {
    const nextDataEl = document.getElementById("__NEXT_DATA__");
    if (nextDataEl?.textContent) {
      const data = JSON.parse(nextDataEl.textContent);
      const findQuestion = (obj, depth = 0) => {
        if (!obj || depth > 8 || typeof obj !== "object") return null;
        if (obj.questionFrontendId && (!slug || obj.titleSlug === slug)) {
          return obj;
        }
        for (const k of Object.keys(obj)) {
          const found = findQuestion(obj[k], depth + 1);
          if (found) return found;
        }
        return null;
      };
      const q = findQuestion(data);
      if (q?.questionFrontendId) {
        return {
          number: String(parseInt(q.questionFrontendId, 10)),
          title: q.title || "",
          difficulty: q.difficulty || "",
          tags: q.topicTags?.map((t) => t.name) || []
        };
      }
    }
  } catch {
    // Ignore JSON parse error
  }

  const candidateSelectors = [
    'div[class*="text-title-large"]',
    `a[href*="/problems/${slug}"]`,
    'div[data-cy="question-title"]',
    'div[class*="title__"]',
    '[data-track-load="description_content"] h1',
    '[data-track-load="description_content"] h4',
    'h1',
    'h4'
  ];

  for (const sel of candidateSelectors) {
    const elements = document.querySelectorAll(sel);
    for (const el of elements) {
      const t = el.textContent?.trim() || "";
      const match = t.match(/^(\d+)\s*[\.\:\-]?\s+(.+)$/);
      if (match) {
        return {
          number: String(parseInt(match[1], 10)),
          title: match[2].trim()
        };
      }
    }
  }

  const links = document.querySelectorAll('a[href^="/problems/"]');
  for (const link of links) {
    const t = link.textContent?.trim() || "";
    const match = t.match(/^(\d+)\s*[\.\:\-]?\s+(.+)$/);
    if (match) {
      return {
        number: String(parseInt(match[1], 10)),
        title: match[2].trim()
      };
    }
  }

  const titleMatch = document.title.match(/^(\d+)\s*[\.\:\-]?\s+(.*?)(?:\s*-\s*LeetCode.*)?$/i);
  if (titleMatch) {
    return {
      number: String(parseInt(titleMatch[1], 10)),
      title: titleMatch[2].trim()
    };
  }

  return null;
}

async function loadProblemMetadata(slug) {
  if (!slug) return null;
  if (problemCache[slug]?.number && /^\d+$/.test(problemCache[slug].number)) {
    return problemCache[slug];
  }

  const domInfo = extractFromDOM(slug);
  if (domInfo?.number) {
    problemCache[slug] = { ...(problemCache[slug] || {}), ...domInfo };
  }

  const gqlData = await fetchGraphQLProblem(slug);
  if (gqlData) {
    const num = gqlData.questionFrontendId
      ? String(parseInt(gqlData.questionFrontendId, 10))
      : (problemCache[slug]?.number || "");

    problemCache[slug] = {
      number: num,
      title: gqlData.title || problemCache[slug]?.title || slug,
      difficulty: gqlData.difficulty || problemCache[slug]?.difficulty || "Unknown",
      tags: gqlData.topicTags?.map((t) => t.name) || problemCache[slug]?.tags || [],
      description: gqlData.content || problemCache[slug]?.description || ""
    };
  }

  return problemCache[slug];
}

let lastSeenSlug = "";
function checkSlugChange() {
  const slug = getSlug();
  if (slug && slug !== lastSeenSlug) {
    lastSeenSlug = slug;
    loadProblemMetadata(slug);
  }
}
window.addEventListener("popstate", checkSlugChange);

async function metadata() {
  const slug = getSlug();
  if (!slug) return null;

  let meta = problemCache[slug];
  if (!meta || !meta.number || !/^\d+$/.test(meta.number)) {
    meta = await loadProblemMetadata(slug);
  }

  let number = meta?.number || "";
  let title = meta?.title || "";

  if (!number || !/^\d+$/.test(number)) {
    const dom = extractFromDOM(slug);
    if (dom?.number) {
      number = dom.number;
      if (!title) title = dom.title;
    }
  }

  if (!title) {
    let rawTitle = text('div[class*="text-title-large"]') ||
      text('[data-cy="question-title"]') ||
      text("h1") ||
      document.title.replace(/\s*-\s*LeetCode.*$/i, "").trim();
    title = rawTitle.replace(/^\d+\.\s*/, "").trim() || slug;
  }

  const difficultyEl = document.querySelector(
    '[class*="text-difficulty"], [data-cy="difficulty"], [class*="difficulty-"]'
  );
  let difficulty = meta?.difficulty || difficultyEl?.textContent?.trim() || "Unknown";
  if (!["Easy", "Medium", "Hard"].includes(difficulty)) {
    if (/easy|简单/i.test(difficulty)) difficulty = "Easy";
    else if (/medium|中等/i.test(difficulty)) difficulty = "Medium";
    else if (/hard|困难/i.test(difficulty)) difficulty = "Hard";
    else difficulty = "Unknown";
  }

  const domTags = [...document.querySelectorAll('a[href*="/tag/"], a[href*="/topics/"]')]
    .map((node) => node.textContent.trim())
    .filter(Boolean);
  const tags = [...new Set([...(meta?.tags || []), ...domTags])];

  const rawLang = getEditorLanguage();
  const normalized = normalizeLanguage(rawLang);
  const code = getEditorCode();

  const durationSeconds = Math.round((Date.now() - sessionStartTime) / 1000);
  const timeSpent = formatDuration(durationSeconds);
  const attemptsCount = failAttempts.total + 1;
  const attemptsSummary = formatAttempts();

  const description = meta?.description ||
    document.querySelector('[data-track-load="description_content"], [class*="question-content"]')?.innerHTML || "";

  return {
    number: number || "",
    title,
    slug,
    url: location.href.split("?")[0].replace(/\/submissions\/.*$/, ""),
    difficulty,
    tags,
    language: normalized.name,
    extension: normalized.ext,
    code,
    description,
    acceptedAt: new Date().toISOString(),
    durationSeconds,
    timeSpent,
    attemptsCount,
    attemptsSummary,
    notes: ""
  };
}

function showToast(contentHtml) {
  let toast = document.getElementById("leetflow-toast");
  if (!toast) {
    toast = document.createElement("div");
    toast.id = "leetflow-toast";
    document.body.appendChild(toast);
  }
  toast.innerHTML = contentHtml;
  return toast;
}

function closeToast() {
  const toast = document.getElementById("leetflow-toast");
  if (toast) {
    toast.remove();
  }
  isPrompting = false;
}

function showUnconfiguredNotice() {
  isPrompting = true;
  const toast = showToast(`
    <div class="leetflow-header">
      <span class="leetflow-title" style="color: #bf8700;">⚠️ GitHub Not Connected</span>
      <button class="leetflow-close-x" id="leetflow-x">✕</button>
    </div>
    <p style="margin: 8px 0; font-size: 12px; color: #57606a; line-height: 1.4;">
      Solution saved locally! Connect your GitHub token & repository to enable auto-sync.
    </p>
    <div class="leetflow-footer">
      <button class="leetflow-btn-secondary" id="leetflow-dismiss-btn">Later</button>
      <button class="leetflow-btn-primary" id="leetflow-connect-btn">Connect GitHub</button>
    </div>
  `);

  toast.querySelector("#leetflow-x")?.addEventListener("click", closeToast);
  toast.querySelector("#leetflow-dismiss-btn")?.addEventListener("click", closeToast);
  toast.querySelector("#leetflow-connect-btn")?.addEventListener("click", () => {
    chrome.runtime.sendMessage({ type: "open-options" });
    closeToast();
  });
}

function showNotePrompt(problem, isConfigured) {
  isPrompting = true;

  const toast = showToast(`
    <div class="leetflow-header">
      <span class="leetflow-title">🎉 Accepted! Add note & sync</span>
      <button class="leetflow-close-x" id="leetflow-x">✕</button>
    </div>
    <div class="leetflow-metrics">
      <span>⏱️ ${problem.timeSpent}</span>
      <span>🎯 ${problem.attemptsSummary}</span>
    </div>
    <div class="leetflow-body">
      <textarea id="leetflow-note-input" placeholder="Aha! moment, key trick, or mistake to remember..."></textarea>
    </div>
    <div class="leetflow-footer">
      <button class="leetflow-btn-secondary" id="leetflow-skip-btn">Skip Note & Sync</button>
      <button class="leetflow-btn-primary" id="leetflow-sync-btn">Save & Sync</button>
    </div>
  `);

  const textarea = toast.querySelector("#leetflow-note-input");
  textarea.focus();

  const handleSync = (note) => {
    problem.notes = (note || "").trim();
    toast.innerHTML = `<div class="leetflow-header"><span class="leetflow-title">🚀 Syncing to GitHub...</span></div>`;

    sendSolution(problem, (res) => {
      if (!res?.configured) {
        showUnconfiguredNotice();
        return;
      }

      if (res?.syncResult?.failed > 0) {
        const errMsg = res.syncResult.errors?.[0] || "Sync failed. Check your settings.";
        toast.innerHTML = `
          <div class="leetflow-header"><span class="leetflow-title" style="color:#cf222e;">❌ Sync failed</span><button class="leetflow-close-x" id="leetflow-x">✕</button></div>
          <p style="margin: 6px 0; font-size: 12px; color: #57606a;">${errMsg}</p>
          <div class="leetflow-footer">
            <button class="leetflow-btn-secondary" id="leetflow-dismiss-btn">Dismiss</button>
            <button class="leetflow-btn-primary" id="leetflow-connect-btn">Settings</button>
          </div>
        `;
        toast.querySelector("#leetflow-x")?.addEventListener("click", closeToast);
        toast.querySelector("#leetflow-dismiss-btn")?.addEventListener("click", closeToast);
        toast.querySelector("#leetflow-connect-btn")?.addEventListener("click", () => {
          chrome.runtime.sendMessage({ type: "open-options" });
          closeToast();
        });
      } else {
        toast.innerHTML = `
          <div class="leetflow-header"><span class="leetflow-title" style="color:#1a7f37;">✅ Synced to GitHub!</span></div>
          <p style="margin: 4px 0; font-size: 12px; color: #57606a;">Saved to your repository archive.</p>
        `;
        setTimeout(closeToast, 2500);
      }
    });
  };

  toast.querySelector("#leetflow-x")?.addEventListener("click", closeToast);
  toast.querySelector("#leetflow-sync-btn")?.addEventListener("click", () => handleSync(textarea.value));
  toast.querySelector("#leetflow-skip-btn")?.addEventListener("click", () => handleSync(""));

  textarea.addEventListener("keydown", (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      handleSync(textarea.value);
    }
  });
}

function sendSolution(problem, callback) {
  try {
    chrome.runtime.sendMessage({ type: "accepted-solution", problem }, (response) => {
      if (callback) callback(response);
    });
  } catch (error) {
    console.warn("[LeetFlow] Could not send message to extension background:", error);
    if (callback) callback({ ok: false, error: error.message });
  }
}

let isSyncing = false;

async function syncIfAccepted() {
  checkSubmissionFailure();
  if (isPrompting || isSyncing) return;
  if (!isAccepted()) return;

  isSyncing = true;
  try {
    const problem = await metadata();
    if (!problem || !problem.code) {
      isSyncing = false;
      return;
    }

    const key = `${problem.number}:${problem.language}:${problem.code.length}`;
    if (key === lastKey) {
      isSyncing = false;
      return;
    }
    lastKey = key;

    chrome.storage?.local?.get(["settings"], (res) => {
      const settings = res?.settings || {};
      const isConfigured = Boolean(settings.githubToken && settings.owner && settings.repo);
      const shouldPrompt = settings.promptNotes !== false;

      if (!isConfigured) {
        sendSolution(problem);
        showUnconfiguredNotice();
        isSyncing = false;
        return;
      }

      if (shouldPrompt) {
        showNotePrompt(problem, isConfigured);
        isSyncing = false;
      } else {
        sendSolution(problem, (syncRes) => {
          isSyncing = false;
          if (syncRes?.syncResult?.synced > 0) {
            showToast(`
              <div class="leetflow-header"><span class="leetflow-title" style="color:#1a7f37;">✅ Synced to GitHub!</span></div>
            `);
            setTimeout(closeToast, 2000);
          }
        });
      }
    });
  } catch (err) {
    console.error("[LeetFlow] syncIfAccepted error:", err);
    isSyncing = false;
  }
}

function setupTimerBadge() {
  if (document.getElementById("leetflow-session-timer") || !document.body) return;
  const timer = document.createElement("div");
  timer.id = "leetflow-session-timer";
  timer.title = "LeetFlow Session Timer (Click to toggle Interview Mode)";
  document.body.appendChild(timer);

  setInterval(() => {
    const elapsed = Math.floor((Date.now() - sessionStartTime) / 1000);
    const m = Math.floor(elapsed / 60);
    const s = elapsed % 60;
    timer.textContent = `⏱️ ${m}:${s < 10 ? "0" : ""}${s}`;
  }, 1000);

  timer.addEventListener("click", () => {
    document.body.classList.toggle("leetflow-interview-mode");
  });
}

try {
  chrome.storage?.local?.get(["settings"], (data) => {
    if (data?.settings?.interviewMode) {
      document.body.classList.add("leetflow-interview-mode");
    }
  });
} catch {
  // Ignored if storage is unavailable
}

const observer = new MutationObserver(() => {
  checkSlugChange();
  checkSubmissionFailure();
  syncIfAccepted();
});
observer.observe(document.documentElement, { childList: true, subtree: true });

checkSlugChange();

setTimeout(() => {
  setupTimerBadge();
  checkSlugChange();
  syncIfAccepted();
}, 1500);
