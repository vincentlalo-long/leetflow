import { loadState, saveState } from "./storage.js";
import { putFile } from "./github.js";
import { problemPath, renderProblemReadme, renderRootReadme, slugify } from "./templates.js";
import { ensureReview, dueReviews, gradeReview } from "./review.js";

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === "accepted-solution") {
    enqueue(message.problem).then(() => sendResponse({ ok: true })).catch((error) => sendResponse({ ok: false, error: error.message }));
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

async function enqueue(problem) {
  const state = await loadState();
  const key = `${problem.number}:${problem.language}`;
  if (state.synced[key] || state.queue.some((job) => job.key === key)) return;
  const reviewed = ensureReview(problem);
  state.queue.push({ key, problem: reviewed, attempts: 0, status: "pending", createdAt: new Date().toISOString() });
  state.reviews[key] = reviewed;
  await saveState(state);
  if (state.settings.autoSync) await processQueue();
}

async function processQueue() {
  const state = await loadState();
  const result = { synced: 0, failed: 0 };
  for (const job of state.queue) {
    if (job.status === "done") continue;
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
    }
  }
  state.queue = state.queue.filter((job) => job.status !== "done");
  await saveState(state);
  return result;
}

async function syncJob(job, state) {
  const settings = state.settings;
  if (!settings.githubToken || !settings.owner || !settings.repo) {
    throw new Error("Configure GitHub token, owner, and repository in Options first.");
  }
  const base = problemPath(job.problem, settings);
  const ext = job.problem.extension || "txt";
  const solutionFileName = `${slugify(job.problem.slug || job.problem.title)}.${ext}`;
  await putFile({
    ...settings,
    path: `${base}/${solutionFileName}`,
    content: job.problem.code,
    message: `sync: ${job.problem.number}. ${job.problem.title}`
  });
  await putFile({
    ...settings,
    path: `${base}/README.md`,
    content: renderProblemReadme(job.problem, settings),
    message: `docs: add README for ${job.problem.number}. ${job.problem.title}`
  });
  if (settings.updateRootReadme) {
    const problems = Object.values(state.syncedProblems || {});
    const merged = [...problems.filter((p) => p.number !== job.problem.number), job.problem];
    state.syncedProblems = Object.fromEntries(merged.map((p) => [`${p.number}:${p.language}`, p]));
    state.reviews[job.key] = ensureReview(job.problem);
    await putFile({
      ...settings,
      path: `${settings.rootDir}/README.md`,
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
