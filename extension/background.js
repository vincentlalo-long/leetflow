import { exportBackup, importBackup, loadState } from "./storage.js";
import { checkForUpdate, getExtensionVersion } from "./update.js";
import {
  QUEUE_ALARM,
  clearFailedJobs,
  enqueue,
  findSolutionFor,
  getReviewState,
  gradeReviewJob,
  restoreReviewJobs,
  retryQueue,
  runQueue,
  saveSettings,
  skipReviewJob
} from "./queue.js";

export const REVIEW_ALARM = "leetflow-review-badge";
export const UPDATE_ALARM = "leetflow-update-check";
const BADGE_COLOR = "#cf222e";
const UPDATE_CACHE_MS = 6 * 60 * 60 * 1000;
let cachedUpdate = null;

export async function refreshBadge() {
  if (!chrome.action?.setBadgeText) return;
  try {
    const { due } = await getReviewState();
    const count = Array.isArray(due) ? due.length : 0;
    await chrome.action.setBadgeText({ text: count > 0 ? String(count) : "" });
    if (count > 0) await chrome.action.setBadgeBackgroundColor?.({ color: BADGE_COLOR });
  } catch {
    /* the badge is cosmetic — never let it break the background */
  }
}

function scheduleBadgeAlarm() {
  try {
    chrome.alarms?.create?.(REVIEW_ALARM, { periodInMinutes: 60 });
    chrome.alarms?.create?.(UPDATE_ALARM, { periodInMinutes: 360 });
  } catch {
    /* alarms are optional */
  }
}

export async function getUpdateStatus(force = false) {
  const now = Date.now();
  if (!force && cachedUpdate && now - cachedUpdate.checkedAt < UPDATE_CACHE_MS) {
    return cachedUpdate;
  }
  const current = getExtensionVersion();
  const result = await checkForUpdate(current);
  cachedUpdate = { ...result, checkedAt: now };
  return cachedUpdate;
}

function withBadge(promise) {
  return promise.then((payload) => {
    refreshBadge();
    return payload;
  });
}

function respond(sendResponse, promise) {
  promise
    .then((payload) => sendResponse({ ok: true, ...(payload || {}) }))
    .catch((error) => sendResponse({ ok: false, error: error?.message || String(error) }));
  return true;
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  switch (message?.type) {
    case "open-options":
      chrome.runtime.openOptionsPage();
      sendResponse({ ok: true });
      return false;
    case "accepted-solution":
      return respond(sendResponse, withBadge(enqueue(message.problem)));
    case "retry-queue":
      return respond(sendResponse, withBadge(retryQueue()));
    case "clear-failed":
      return respond(sendResponse, clearFailedJobs());
    case "save-settings":
      return respond(sendResponse, saveSettings(message.settings));
    case "get-solution":
      return respond(sendResponse, findSolutionFor(message));
    case "get-state":
      return respond(sendResponse, loadState().then((state) => ({ state })));
    case "get-version":
      return respond(sendResponse, Promise.resolve({ version: getExtensionVersion() }));
    case "check-update":
      return respond(sendResponse, getUpdateStatus(Boolean(message?.force)));
    case "get-review":
      return respond(sendResponse, getReviewState());
    case "grade-review":
      return respond(sendResponse, withBadge(gradeReviewJob(message.key, message.grade)));
    case "skip-review":
      return respond(sendResponse, withBadge(skipReviewJob(message.key)));
    case "restore-reviews":
      return respond(sendResponse, withBadge(restoreReviewJobs()));
    case "export-backup":
      return respond(
        sendResponse,
        exportBackup(getExtensionVersion()).then((backup) => ({ backup }))
      );
    case "import-backup":
      return respond(
        sendResponse,
        withBadge(importBackup(message?.backup).then(() => ({})))
      );
    default:
      return false;
  }
});

function onWake() {
  runQueue().then(refreshBadge).catch(() => {});
}

chrome.runtime.onStartup.addListener(() => {
  scheduleBadgeAlarm();
  onWake();
});
chrome.runtime.onInstalled.addListener(() => {
  scheduleBadgeAlarm();
  onWake();
});

if (chrome.alarms?.onAlarm) {
  chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm?.name === QUEUE_ALARM) onWake();
    if (alarm?.name === REVIEW_ALARM) refreshBadge();
    if (alarm?.name === UPDATE_ALARM) getUpdateStatus().catch(() => {});
  });
}

refreshBadge();
