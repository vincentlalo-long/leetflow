import { loadState, saveState } from "./storage.js";
import { putFile } from "./github.js";
import { problemPath, renderProblemReadme, renderRootReadme, slugify } from "./templates.js";
import { ensureReview, dueReviews, gradeReview } from "./review.js";

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === "open-options") {
    chrome.runtime.openOptionsPage();
    sendResponse({ ok: true });
    return true;
  }
  if (message.type === "accepted-solution") {
    enqueue(message.problem)
      .then((res) => sendResponse({ ok: true, ...res }))
      .catch((error) => sendResponse({ ok: false, error: error.message }));
    return true;
  }
  if (message.type === "get-state") {
    loadState().then((state) => sendResponse({ ok: true, state })).catch((error) => sendResponse({ ok: false, error: error.message }));
    return true;
  }
  if (message.type === "retry-queue") {
    processQueue().then((result) => sendResponse({ ok: true, ...result })).catch((error) => sendResponse({ ok: false, error: error.message }));
    return true;
  }
  if (message.type === "get-review") {
    getReviewState().then((review) => sendResponse({ ok: true, ...review })).catch((error) => sendResponse({ ok: false, error: error.message }));
    return true;
  }
  if (message.type === "grade-review") {
    gradeReviewJob(message.key, message.grade).then((review) => sendResponse({ ok: true, ...review })).catch((error) => sendResponse({ ok: false, error: error.message }));
    return true;
  }
});

chrome.runtime.onStartup.addListener(processQueue);
chrome.runtime.onInstalled.addListener(processQueue);

async function resolveProblemMetadata(problem) {
  if (problem?.number && problem.number !== "unknown" && /^\d+$/.test(String(problem.number))) {
    return problem;
  }
  if (!problem?.slug) return problem;

  try {
    const isCn = problem.url?.includes("leetcode.cn");
    const endpoint = isCn ? "https://leetcode.cn/graphql" : "https://leetcode.com/graphql";
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
            }
          }
        `,
        variables: { titleSlug: problem.slug }
      })
    });
    if (res.ok) {
      const data = await res.json();
      const q = data?.data?.question;
      if (q?.questionFrontendId) {
        problem.number = String(parseInt(q.questionFrontendId, 10));
        if (q.title && (!problem.title || problem.title === problem.slug)) {
          problem.title = q.title;
        }
        if (q.difficulty && (!problem.difficulty || problem.difficulty === "Unknown")) {
          problem.difficulty = q.difficulty;
        }
        if (q.topicTags?.length && (!problem.tags || problem.tags.length === 0)) {
          problem.tags = q.topicTags.map((t) => t.name);
        }
      }
    }
  } catch (err) {
    console.warn("[LeetFlow] Background failed to resolve problem metadata:", err);
  }
  return problem;
}

async function enqueue(problem) {
  problem = await resolveProblemMetadata(problem);
  const state = await loadState();
  const key = `${problem.number}:${problem.language}`;
  const configured = Boolean(state.settings.githubToken && state.settings.owner && state.settings.repo);

  if (state.synced[key]) {
    return { alreadySynced: true, configured, key };
  }

  let job = state.queue.find((j) => j.key === key);
  if (!job) {
    const reviewed = ensureReview(problem);
    job = { key, problem: reviewed, attempts: 0, status: "pending", createdAt: new Date().toISOString() };
    state.queue.push(job);
    state.reviews[key] = reviewed;
  } else {
    job.problem = { ...job.problem, ...problem };
    job.status = "pending";
    job.error = "";
  }
  await saveState(state);

  let syncResult = null;
  if (configured && state.settings.autoSync) {
    syncResult = await processQueue();
  }
  return { key, configured, syncResult };
}

async function processQueue() {
  const state = await loadState();
  const result = { synced: 0, failed: 0, errors: [] };
  for (const job of state.queue) {
    if (job.status === "done") continue;
    if (!job.problem.number || job.problem.number === "unknown" || !/^\d+$/.test(String(job.problem.number))) {
      job.problem = await resolveProblemMetadata(job.problem);
      job.key = `${job.problem.number}:${job.problem.language}`;
    }
    try {
      await syncJob(job, state);
      job.status = "done";
      job.error = "";
      state.synced[job.key] = job.problem.acceptedAt;
      result.synced++;
    } catch (error) {
      job.status = "failed";
      job.error = error.message;
      job.attempts++;
      result.failed++;
      result.errors.push(error.message);
    }
  }
  state.queue = state.queue.filter((job) => job.status !== "done");
  await saveState(state);
  return result;
}

async function syncJob(job, state) {
  if (!job.problem.number || job.problem.number === "unknown" || !/^\d+$/.test(String(job.problem.number))) {
    job.problem = await resolveProblemMetadata(job.problem);
    job.key = `${job.problem.number}:${job.problem.language}`;
  }
  const settings = state.settings;
  const token = settings.githubToken;
  if (!token || !settings.owner || !settings.repo) {
    throw new Error("Configure GitHub token, owner, and repository in Options first.");
  }
  const base = problemPath(job.problem, settings);
  const ext = job.problem.extension || "txt";
  const solutionFileName = `${slugify(job.problem.slug || job.problem.title)}.${ext}`;
  const numPrefix = /^\d+$/.test(String(job.problem.number)) ? `${job.problem.number}. ` : "";

  console.log(`[LeetFlow] Syncing problem #${job.problem.number} to ${settings.owner}/${settings.repo}`);

  await putFile({
    ...settings,
    token,
    path: `${base}/${solutionFileName}`,
    content: job.problem.code,
    message: `sync: ${numPrefix}${job.problem.title}`
  });
  await putFile({
    ...settings,
    token,
    path: `${base}/README.md`,
    content: renderProblemReadme(job.problem, settings),
    message: `docs: add README for ${numPrefix}${job.problem.title}`
  });
  if (settings.updateRootReadme) {
    const problems = Object.values(state.syncedProblems || {});
    const cleanOld = problems.filter((p) => p.number !== "unknown" && p.slug !== job.problem.slug);
    const merged = [...cleanOld, job.problem];
    state.syncedProblems = Object.fromEntries(merged.map((p) => [`${p.number}:${p.language}`, p]));
    state.reviews[job.key] = ensureReview(job.problem);
    const rootPath = settings.rootDir ? `${settings.rootDir.replace(/\/+$/, "")}/README.md` : "README.md";
    await putFile({
      ...settings,
      token,
      path: rootPath,
      content: renderRootReadme(Object.values(state.syncedProblems), settings),
      message: "docs: update LeetCode index"
    });
  }
}

async function getReviewState() {
  const state = await loadState();
  const problems = Object.values(state.reviews || {}).map(ensureReview);
  return { due: dueReviews(problems), total: problems.length };
}

async function gradeReviewJob(key, grade) {
  const state = await loadState();
  const current = state.reviews?.[key];
  if (!current) throw new Error("Review item no longer exists.");
  const updated = gradeReview(current, grade);
  state.reviews[key] = updated;
  if (state.syncedProblems?.[key]) state.syncedProblems[key] = updated;
  await saveState(state);
  return { item: updated, due: dueReviews(Object.values(state.reviews)) };
}
