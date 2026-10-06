import { loadState, saveState } from "./storage.js";
import { commitFiles, readRemoteFile } from "./github.js";
import {
  mergeFolderRecords,
  mergeRootReadme,
  normalizeLanguage,
  problemPath,
  renderProblemReadme,
  renderRootReadme,
  slugify,
  solutionFiles
} from "./templates.js";
import { dueReviews, ensureReview, gradeReview, restoreReview, skipReview } from "./review.js";

export const QUEUE_ALARM = "leetflow-sync-queue";
export const MAX_JOB_ATTEMPTS = 5;
const METADATA_TIMEOUT_MS = 8000;

let activeRun = null;
let stateLock = Promise.resolve();

function withStateLock(worker) {
  const run = stateLock.then(
    () => worker(),
    () => worker()
  );
  stateLock = run.then(
    () => undefined,
    () => undefined
  );
  return run;
}

export function resetQueueForTests() {
  activeRun = null;
  stateLock = Promise.resolve();
}

export function problemKey(problem) {
  return `${problem?.number ?? ""}:${problem?.language ?? ""}`;
}

export function isConfigured(settings) {
  return Boolean(settings?.githubToken && settings?.owner && settings?.repo);
}

function hasProblemNumber(value) {
  return /^\d+$/.test(String(value ?? "").trim());
}

function normalizeProblemNumber(value) {
  const raw = String(value ?? "").trim();
  if (/^\d+$/.test(raw)) return raw;
  const leading = raw.match(/^\d+/);
  return leading ? leading[0] : "";
}

function eligibleForRetry(job) {
  if (!job || job.status === "done") return false;
  if (job.status === "pending") return true;
  return job.status === "failed" && (job.attempts || 0) < MAX_JOB_ATTEMPTS;
}

function backoffMinutes(job) {
  const attempts = Math.max(1, job.attempts || 1);
  return Math.min(60, 2 ** (attempts - 1));
}

function nextRetryDelay(queue) {
  const delays = (queue || []).filter(eligibleForRetry).map(backoffMinutes);
  return delays.length ? Math.min(...delays) : null;
}

function scheduleRetry(delayMinutes = 1) {
  try {
    globalThis.chrome?.alarms?.create?.(QUEUE_ALARM, { delayInMinutes: delayMinutes });
  } catch {
    // Alarms are optional; the next message or startup also drains the queue.
  }
}

function endpointFor(problem) {
  const isCn =
    String(problem?.url || "").includes("leetcode.cn") ||
    String(problem?.site || "").includes("cn");
  return isCn ? "https://leetcode.cn/graphql" : "https://leetcode.com/graphql";
}

export async function resolveProblemMetadata(problem) {
  if (hasProblemNumber(problem?.number)) return problem;
  if (!problem?.slug) return problem;

  try {
    const controller = typeof AbortController === "function" ? new AbortController() : null;
    const timer = controller ? setTimeout(() => controller.abort(), METADATA_TIMEOUT_MS) : null;
    const response = await fetch(endpointFor(problem), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: `
          query questionData($titleSlug: String!) {
            question(titleSlug: $titleSlug) {
              questionFrontendId
              title
              difficulty
              topicTags { name }
            }
          }
        `,
        variables: { titleSlug: problem.slug }
      }),
      signal: controller?.signal
    });
    if (timer) clearTimeout(timer);

    if (!response.ok) return problem;
    const payload = await response.json();
    const question = payload?.data?.question;
    if (!question?.questionFrontendId) return problem;

    return {
      ...problem,
      number: normalizeProblemNumber(question.questionFrontendId) || String(problem.number || ""),
      title: question.title || problem.title || problem.slug,
      difficulty:
        question.difficulty && (!problem.difficulty || problem.difficulty === "Unknown")
          ? question.difficulty
          : problem.difficulty,
      tags:
        question.topicTags?.length && (!problem.tags || problem.tags.length === 0)
          ? question.topicTags.map((tag) => tag.name)
          : problem.tags
    };
  } catch (error) {
    console.warn("[LeetFlow] Failed to resolve problem metadata:", error);
    return problem;
  }
}

export function normalizeIncoming(problem) {
  const source = problem || {};
  const language = normalizeLanguage(source.language);
  return {
    ...source,
    language: language.name,
    extension: source.extension || language.ext,
    approach: typeof source.approach === "string" ? source.approach.trim() : ""
  };
}

function solutionsOf(existing) {
  if (Array.isArray(existing?.solutions) && existing.solutions.length) {
    return [...existing.solutions];
  }
  if (!existing) return [];
  return [
    {
      approach: "",
      code: existing.code || "",
      notes: existing.notes || "",
      timeComplexity: existing.timeComplexity || "",
      spaceComplexity: existing.spaceComplexity || "",
      acceptedAt: existing.acceptedAt || existing.syncedAt || ""
    }
  ];
}

function findSolutionIndex(existing, incoming) {
  const solutions = solutionsOf(existing);
  const approach = slugify(incoming?.approach);
  const code = String(incoming?.code ?? "");

  if (approach) {
    const byLabel = solutions.findIndex((solution) => slugify(solution.approach) === approach);
    if (byLabel >= 0) return byLabel;
    // An unlabeled entry that carries the same code is the same solution being
    // labeled for the first time; adopt the label instead of duplicating.
    if (code) {
      const byCode = solutions.findIndex((solution) => String(solution.code || "") === code);
      if (byCode >= 0 && !String(solutions[byCode].approach || "").trim()) return byCode;
    }
    return -1;
  }

  // An unlabeled resubmission first matches on identical code so it does not
  // duplicate a labeled approach, then falls back to the default entry.
  if (code) {
    const byCode = solutions.findIndex((solution) => String(solution.code || "") === code);
    if (byCode >= 0) return byCode;
  }
  return solutions.findIndex((solution) => !String(solution.approach || "").trim());
}

function inheritMissing(existing, incoming) {
  if (!existing) return incoming;
  const index = findSolutionIndex(existing, incoming);
  if (index < 0) return incoming;
  const previous = solutionsOf(existing)[index];
  const effective = { ...incoming };
  for (const field of ["approach", "notes", "timeComplexity", "spaceComplexity"]) {
    if (!String(effective[field] || "").trim() && String(previous[field] || "").trim()) {
      effective[field] = previous[field];
    }
  }
  return effective;
}

export function upsertSolution(existing, incoming) {
  const solutions = solutionsOf(existing);
  const index = findSolutionIndex(existing, incoming);
  const previous = index >= 0 ? solutions[index] : null;
  const approach =
    String(incoming?.approach || "").trim() || String(previous?.approach || "").trim();
  const pick = (field) => {
    const value = String(incoming?.[field] || "");
    return value.trim() ? value : String(previous?.[field] || "");
  };
  const entry = {
    approach,
    code: String(incoming?.code ?? ""),
    notes: pick("notes"),
    timeComplexity: pick("timeComplexity"),
    spaceComplexity: pick("spaceComplexity")
  };

  if (index >= 0) {
    solutions[index] = {
      ...previous,
      ...entry,
      acceptedAt: incoming?.acceptedAt || previous.acceptedAt || ""
    };
  } else {
    solutions.push({ ...entry, acceptedAt: incoming?.acceptedAt || "" });
  }
  return solutions;
}

function sameSolution(existing, incoming) {
  if (!existing?.syncedAt) return false;
  const index = findSolutionIndex(existing, incoming);
  if (index < 0) return false;
  const current = solutionsOf(existing)[index];
  return (
    current.code === String(incoming.code ?? "") &&
    (current.notes || "") === (incoming.notes || "") &&
    (current.timeComplexity || "") === (incoming.timeComplexity || "") &&
    (current.spaceComplexity || "") === (incoming.spaceComplexity || "")
  );
}

export async function enqueue(rawProblem) {
  const incoming = normalizeIncoming(rawProblem);
  const resolved = await resolveProblemMetadata(incoming);

  const response = await withStateLock(async () => {
    const state = await loadState();
    const configured = isConfigured(state.settings);
    const key = problemKey(resolved);
    const existing = state.problems[key];
    const effective = inheritMissing(existing, resolved);
    const result = { ok: true, configured, key, status: "queued", syncResult: null };
    result.autoSync = Boolean(state.settings.autoSync);

    if (!resolved.slug && !resolved.title) {
      throw new Error("Could not read the problem from the LeetCode page.");
    }

    if (sameSolution(existing, effective)) {
      result.status = "unchanged";
      await saveState(state);
      return result;
    }

    const merged = { ...(existing || {}), ...effective };
    if (existing?.review) merged.review = existing.review;
    merged.solutions = upsertSolution(existing, effective);
    const reviewed = ensureReview(merged);
    state.problems[key] = reviewed;
    result.status = existing?.syncedAt ? "queued-update" : "queued";

    const now = new Date().toISOString();
    const job = state.queue.find((item) => item?.key === key);
    if (job) {
      job.problem = reviewed;
      job.status = "pending";
      job.error = "";
      job.attempts = 0;
      job.updatedAt = now;
    } else {
      state.queue.push({ key, problem: reviewed, attempts: 0, status: "pending", createdAt: now });
    }
    state.lastError = "";
    await saveState(state);
    return result;
  });

  if (response.status !== "unchanged" && response.configured && response.autoSync) {
    response.syncResult = await runQueue();
  }
  return response;
}

export function runQueue() {
  if (activeRun) return activeRun;
  activeRun = withStateLock(drainQueue).finally(() => {
    activeRun = null;
  });
  return activeRun;
}

export function retryQueue() {
  if (activeRun) return activeRun;
  activeRun = withStateLock(async () => {
    const state = await loadState();
    let changed = false;
    for (const job of state.queue || []) {
      if (job?.status === "failed") {
        job.status = "pending";
        job.attempts = 0;
        job.error = "";
        changed = true;
      }
    }
    if (changed) await saveState(state);
    return drainQueue();
  }).finally(() => {
    activeRun = null;
  });
  return activeRun;
}

export function clearFailedJobs() {
  return withStateLock(async () => {
    const state = await loadState();
    const before = state.queue.length;
    state.queue = state.queue.filter((job) => job?.status !== "failed");
    const removed = before - state.queue.length;
    if (removed > 0) {
      state.lastError = "";
      await saveState(state);
    }
    return { removed };
  });
}

export function saveSettings(patch) {
  return withStateLock(async () => {
    const state = await loadState();
    state.settings = { ...state.settings, ...(patch || {}) };
    await saveState(state);
    return { settings: state.settings };
  });
}

export async function findSolutionFor({ slug, code } = {}) {
  const state = await loadState();
  const wanted = String(code ?? "");
  for (const record of Object.values(state.problems || {})) {
    if (!record || record.slug !== slug) continue;
    const solutions = Array.isArray(record.solutions) && record.solutions.length
      ? record.solutions
      : [
          {
            approach: "",
            code: record.code || "",
            notes: record.notes || "",
            timeComplexity: record.timeComplexity || "",
            spaceComplexity: record.spaceComplexity || ""
          }
        ];
    const match =
      solutions.find((solution) => solution.code === wanted) ||
      (solutions.length === 1 ? solutions[0] : null);
    if (match) {
      return {
        solution: {
          approach: match.approach || "",
          notes: match.notes || "",
          timeComplexity: match.timeComplexity || "",
          spaceComplexity: match.spaceComplexity || ""
        }
      };
    }
  }
  return { solution: null };
}

function rootReadmePath(settings) {
  const root = settings.rootDir ? `${String(settings.rootDir).replace(/\/+$/, "")}/` : "";
  return `${root}README.md`;
}

function buildCommitMessage(prepared) {
  if (prepared.length === 1) {
    const problem = prepared[0].problem;
    const number = hasProblemNumber(problem.number) ? `${problem.number}. ` : "";
    const title = String(problem.title || problem.slug || "solution").replace(/^\d+\.\s*/, "");
    return `sync: ${number}${title}`;
  }
  return `sync: ${prepared.length} solutions + README index`;
}

async function buildFiles(prepared, problems, settings) {
  const files = [];
  const readmeFolders = new Set();

  for (const entry of prepared) {
    const problem = entry.problem;
    const base = problemPath(problem, settings);
    for (const { solution, file } of solutionFiles(problem)) {
      files.push({ path: `${base}/${file}`, content: String(solution.code ?? "") });
    }
    readmeFolders.add(base);
  }

  for (const base of readmeFolders) {
    const records = Object.values(problems).filter(
      (problem) =>
        problem && (problem.title || problem.slug) && problem.syncedAt &&
        problemPath(problem, settings) === base
    );
    const fallback = prepared
      .map((entry) => entry.problem)
      .filter((problem) => problemPath(problem, settings) === base);
    const merged = mergeFolderRecords(records.length ? records : fallback);
    if (merged) {
      files.push({ path: `${base}/README.md`, content: renderProblemReadme(merged, settings) });
    }
  }

  if (settings.updateRootReadme) {
    const path = rootReadmePath(settings);
    const current = await readRemoteFile(
      settings.githubToken,
      settings.owner,
      settings.repo,
      settings.branch,
      path
    );
    files.push({
      path,
      content: mergeRootReadme(current, renderRootReadme(Object.values(problems), settings))
    });
  }

  return files;
}

function markFailed(job) {
  job.attempts = (job.attempts || 0) + 1;
  job.status = "failed";
  job.error = job.error || "Sync failed.";
}

async function drainQueue() {
  const state = await loadState();
  const settings = state.settings;
  const result = {
    ok: true,
    configured: isConfigured(settings),
    synced: 0,
    failed: 0,
    skipped: 0,
    errors: [],
    commit: null,
    rateLimited: false
  };

  const jobs = state.queue.filter(eligibleForRetry);
  if (jobs.length === 0) {
    await saveState(state);
    return result;
  }

  if (!result.configured) {
    result.skipped = jobs.length;
    result.ok = false;
    result.errors.push("GitHub is not connected yet. Open Settings to add your token and repository.");
    return result;
  }

  const prepared = [];
  const problems = { ...state.problems };

  for (const job of jobs) {
    try {
      let problem = normalizeIncoming(job.problem);
      if (!hasProblemNumber(problem.number)) {
        problem = await resolveProblemMetadata(problem);
      }
      if (!hasProblemNumber(problem.number)) {
        throw new Error(
          `Could not resolve the problem number for "${problem.slug || problem.title || "unknown"}".`
        );
      }
      if (!String(problem.code || "").length) {
        throw new Error("No solution code was captured for this submission.");
      }

      const key = problemKey(problem);
      const previousKey = job.key;
      const placeholder =
        previousKey && previousKey !== key ? state.problems[previousKey] : null;
      const existing = problems[key] || placeholder;
      const effective = inheritMissing(existing, problem);
      const record = { ...(existing || {}), ...effective, syncedAt: new Date().toISOString() };
      if (existing?.review) record.review = existing.review;
      else if (placeholder?.review) record.review = placeholder.review;
      record.solutions = upsertSolution(existing, effective);
      const reviewed = ensureReview(record);

      job.key = key;
      job.problem = reviewed;
      job.status = "pending";
      problems[key] = reviewed;
      if (previousKey && previousKey !== key) {
        delete problems[previousKey];
        delete state.problems[previousKey];
      }
      prepared.push({ key, problem: reviewed, job });
    } catch (error) {
      markFailed(job);
      job.error = error.message;
      result.failed++;
      result.errors.push(error.message);
    }
  }

  if (prepared.length === 0) {
    state.lastError = result.errors[0] || "Nothing to sync.";
    state.queue = state.queue.filter((job) => job.status !== "done");
    await saveState(state);
    result.ok = false;
    return result;
  }

  let retryDelay = null;
  try {
    const files = await buildFiles(prepared, problems, settings);
    const commit = await commitFiles({
      token: settings.githubToken,
      owner: settings.owner,
      repo: settings.repo,
      branch: settings.branch,
      files,
      message: buildCommitMessage(prepared)
    });

    for (const entry of prepared) {
      entry.job.status = "done";
      entry.job.error = "";
    }
    state.problems = problems;
    state.lastSync = {
      at: new Date().toISOString(),
      sha: commit.sha,
      count: prepared.length,
      message: commit.message || buildCommitMessage(prepared)
    };
    state.lastError = "";
    result.synced = prepared.length;
    result.commit = commit.sha;
  } catch (error) {
    const message = error?.message || String(error);
    result.rateLimited = Boolean(error?.rateLimited);
    result.errors.push(message);
    state.lastError = message;

    if (result.rateLimited) {
      const resetAt = Number(error.resetAt) || Date.now() + 60000;
      retryDelay = Math.max(1, Math.ceil((resetAt - Date.now()) / 60000));
    }

    for (const entry of prepared) {
      if (result.rateLimited) {
        entry.job.status = "pending";
        entry.job.error = message;
      } else {
        entry.job.attempts = (entry.job.attempts || 0) + 1;
        entry.job.status = "failed";
        entry.job.error = message;
      }
    }
    result.failed = result.rateLimited ? 0 : prepared.length;
    result.ok = false;
  }

  state.queue = state.queue.filter((job) => job.status !== "done");
  await saveState(state);

  if (!result.ok) {
    const delay = retryDelay ?? nextRetryDelay(state.queue);
    if (delay) scheduleRetry(delay);
  }

  return result;
}

function summarizeReview(problem) {
  return {
    number: problem.number,
    title: problem.title,
    language: problem.language,
    url: problem.url,
    due: problem.review?.due || "",
    stability: problem.review?.stability ?? 0,
    reviews: problem.review?.reviews ?? 0,
    lastGrade: problem.review?.lastGrade || ""
  };
}

export async function getReviewState() {
  return withStateLock(async () => {
    const state = await loadState();
    const tracked = Object.values(state.problems || {}).filter(
      (problem) => problem && (problem.title || problem.slug)
    );
    const withReview = tracked.map(ensureReview);
    const active = withReview.filter((problem) => !problem.review.skipped);
    const upcoming = [...active]
      .sort((a, b) => String(a.review.due).localeCompare(String(b.review.due)))
      .slice(0, 5)
      .map(summarizeReview);
    return {
      due: dueReviews(withReview),
      upcoming,
      total: active.length,
      skipped: withReview.length - active.length
    };
  });
}

export async function gradeReviewJob(key, grade) {
  return withStateLock(async () => {
    const state = await loadState();
    const current = state.problems?.[key];
    if (!current) throw new Error("Review item no longer exists.");
    state.problems[key] = gradeReview(current, grade);
    await saveState(state);
    const tracked = Object.values(state.problems || {})
      .filter((problem) => problem && (problem.title || problem.slug))
      .map(ensureReview);
    return { item: state.problems[key], due: dueReviews(tracked) };
  });
}

export async function skipReviewJob(key) {
  return withStateLock(async () => {
    const state = await loadState();
    const current = state.problems?.[key];
    if (!current) throw new Error("Review item no longer exists.");
    state.problems[key] = skipReview(current);
    await saveState(state);
    return summarizeReview(state.problems[key]);
  });
}

export async function restoreReviewJobs() {
  return withStateLock(async () => {
    const state = await loadState();
    let restored = 0;
    for (const [key, problem] of Object.entries(state.problems || {})) {
      if (problem?.review?.skipped) {
        state.problems[key] = restoreReview(problem);
        restored += 1;
      }
    }
    if (restored > 0) await saveState(state);
    return { restored };
  });
}
