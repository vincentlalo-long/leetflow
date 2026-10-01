import { loadState } from "./storage.js";
import {
  QUEUE_ALARM,
  enqueue,
  getReviewState,
  gradeReviewJob,
  runQueue
} from "./queue.js";

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
      return respond(sendResponse, enqueue(message.problem));
    case "retry-queue":
      return respond(sendResponse, runQueue());
    case "get-state":
      return respond(sendResponse, loadState().then((state) => ({ state })));
    case "get-review":
      return respond(sendResponse, getReviewState());
    case "grade-review":
      return respond(sendResponse, gradeReviewJob(message.key, message.grade));
    default:
      return false;
  }
});

chrome.runtime.onStartup.addListener(runQueue);
chrome.runtime.onInstalled.addListener(runQueue);

if (chrome.alarms?.onAlarm) {
  chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm?.name === QUEUE_ALARM) runQueue();
  });
}
