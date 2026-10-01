import { loadState, saveState } from "./storage.js";
import { commitFiles, readRemoteFile } from "./github.js";
import {
  mergeRootReadme,
  normalizeLanguage,
  problemPath,
  renderProblemReadme,
  renderRootReadme,
  slugify
} from "./templates.js";
import { dueReviews, ensureReview, gradeReview, restoreReview, skipReview } from "./review.js";

export const QUEUE_ALARM = "leetflow-sync-queue";
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
      number: String(parseInt(question.questionFrontendId, 10)),
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
    extension: source.extension || language.ext
  };
}

function sameSolution(existing, incoming) {
  return (
    Boolean(existing?.syncedAt) &&
    existing.code === incoming.code &&
    (existing.notes || "") === (incoming.notes || "") &&
    (existing.timeComplexity || "") === (incoming.timeComplexity || "") &&
    (existing.spaceComplexity || "") === (incoming.spaceComplexity || "")
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
    const result = { ok: true, configured, key, status: "queued", syncResult: null };
    result.autoSync = Boolean(state.settings.autoSync);

    if (!resolved.slug && !resolved.title) {
      throw new Error("Could not read the problem from the LeetCode page.");
    }

    if (sameSolution(existing, resolved)) {
      result.status = "unchanged";
      await saveState(state);
      return result;
    }

    const merged = { ...(existing || {}), ...resolved };
    if (existing?.review) merged.review = existing.review;
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
  for (const entry of prepared) {
    const problem = entry.problem;
    const base = problemPath(problem, settings);
    const extension = problem.extension || normalizeLanguage(problem.language).ext || "txt";
    const solutionFile = `${slugify(problem.slug || problem.title)}.${extension}`;
    files.push({ path: `${base}/${solutionFile}`, content: String(problem.code ?? "") });
    files.push({ path: `${base}/README.md`, content: renderProblemReadme(problem, settings) });
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

  const jobs = state.queue.filter((job) => job && job.status !== "done");
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
      const existing = problems[key] || state.problems[job.key];
      const record = { ...(existing || {}), ...problem, syncedAt: new Date().toISOString() };
      if (existing?.review) record.review = existing.review;
      const reviewed = ensureReview(record);

      job.key = key;
      job.problem = reviewed;
      job.status = "pending";
      problems[key] = reviewed;
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
      const minutes = Math.max(1, Math.ceil((resetAt - Date.now()) / 60000));
      scheduleRetry(minutes);
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

  if (!result.ok && state.queue.length > 0) {
    scheduleRetry(result.rateLimited ? undefined : 1);
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
