import { loadState } from "./storage.js";
import {
  QUEUE_ALARM,
  enqueue,
  getReviewState,
  gradeReviewJob,
  restoreReviewJobs,
  runQueue,
  skipReviewJob
} from "./queue.js";

export const REVIEW_ALARM = "leetflow-review-badge";
const BADGE_COLOR = "#cf222e";

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
  } catch {
    /* alarms are optional */
  }
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
      return respond(sendResponse, withBadge(runQueue()));
    case "get-state":
      return respond(sendResponse, loadState().then((state) => ({ state })));
    case "get-review":
      return respond(sendResponse, getReviewState());
    case "grade-review":
      return respond(sendResponse, withBadge(gradeReviewJob(message.key, message.grade)));
    case "skip-review":
      return respond(sendResponse, withBadge(skipReviewJob(message.key)));
    case "restore-reviews":
      return respond(sendResponse, withBadge(restoreReviewJobs()));
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
  });
}

refreshBadge();
