import test from "node:test";
import assert from "node:assert/strict";
import { configureRepository, installChromeMock } from "./helpers/mock.js";

const { listeners, store, alarms, badge } = installChromeMock(configureRepository());
const background = await import("../extension/background.js");
const { refreshBadge, REVIEW_ALARM } = background;

function dispatch(message) {
  return new Promise((resolve) => {
    let settled = false;
    const finish = (value) => {
      if (settled) return;
      settled = true;
      resolve(value);
    };
    const keepAlive = listeners.message(message, {}, finish);
    if (!keepAlive) finish({ ok: false, error: `unhandled: ${message?.type}` });
  });
}

test("get-state returns the state nested under a state key", async () => {
  const response = await dispatch({ type: "get-state" });
  assert.equal(response.ok, true);
  assert.equal(typeof response.state, "object");
  assert.equal(typeof response.state.settings, "object");
  assert.ok(Array.isArray(response.state.queue));
  assert.equal(typeof response.state.problems, "object");
});

test("get-review returns the due list and the tracked total", async () => {
  const response = await dispatch({ type: "get-review" });
  assert.equal(response.ok, true);
  assert.ok(Array.isArray(response.due));
  assert.ok(Array.isArray(response.upcoming));
  assert.equal(typeof response.total, "number");
});

test("grading a due review reports the next due date", async () => {
  store.problems = {
    "1:cpp": {
      number: "1",
      title: "Two Sum",
      language: "cpp",
      code: "int main(){}",
      review: { due: "2020-01-01", stability: 2, reviews: 0, lastGrade: "" }
    }
  };

  const before = await dispatch({ type: "get-review" });
  assert.equal(before.total, 1);
  assert.equal(before.due.length, 1);
  assert.equal(before.upcoming.length, 1);
  assert.equal(before.upcoming[0].title, "Two Sum");

  const graded = await dispatch({ type: "grade-review", key: "1:cpp", grade: "good" });
  assert.equal(graded.ok, true);
  assert.equal(graded.item.review.lastGrade, "good");
  assert.equal(graded.item.review.reviews, 1);
  assert.ok(graded.item.review.due > "2020-01-01");

  const after = await dispatch({ type: "get-review" });
  assert.equal(after.upcoming[0].lastGrade, "good");
  assert.ok(after.upcoming[0].due > "2020-01-01");

  delete store.problems["1:cpp"];
});

test("a freshly synced problem is reviewable immediately", async () => {
  store.problems = {
    "2:python3": {
      number: "2",
      title: "Add Two Numbers",
      language: "python3",
      code: "x",
      review: { due: "2999-01-01", stability: 2, reviews: 0, lastGrade: "" }
    }
  };

  const review = await dispatch({ type: "get-review" });
  assert.equal(review.due.length, 1);
  assert.equal(review.due[0].title, "Add Two Numbers");
  assert.equal(review.upcoming[0].due, new Date().toISOString().slice(0, 10));
  assert.equal(review.skipped, 0);
});

test("skip-review drops a problem and restore brings it back", async () => {
  const skipped = await dispatch({ type: "skip-review", key: "2:python3" });
  assert.equal(skipped.ok, true);
  assert.equal(skipped.lastGrade, "skip");
  assert.equal(skipped.due, "");

  const afterSkip = await dispatch({ type: "get-review" });
  assert.equal(afterSkip.due.length, 0);
  assert.equal(afterSkip.upcoming.length, 0);
  assert.equal(afterSkip.total, 0);
  assert.equal(afterSkip.skipped, 1);

  const restored = await dispatch({ type: "restore-reviews" });
  assert.equal(restored.ok, true);
  assert.equal(restored.restored, 1);

  const afterRestore = await dispatch({ type: "get-review" });
  assert.equal(afterRestore.total, 1);
  assert.equal(afterRestore.due.length, 1);
  assert.equal(afterRestore.skipped, 0);

  delete store.problems["2:python3"];
});

test("retry-queue reports the drain result", async () => {
  const response = await dispatch({ type: "retry-queue" });
  assert.equal(response.ok, true);
  assert.equal(typeof response.synced, "number");
  assert.equal(typeof response.failed, "number");
});

test("unknown message types are rejected", async () => {
  const response = await dispatch({ type: "does-not-exist" });
  assert.equal(response.ok, false);
});

test("the action badge shows how many reviews are due", async () => {
  store.problems = {
    "7:cpp": {
      number: "7",
      title: "Reverse Integer",
      language: "cpp",
      code: "x",
      review: { due: "2020-01-01", stability: 1, reviews: 0, lastGrade: "" }
    },
    "8:cpp": {
      number: "8",
      title: "String to Integer (atoi)",
      language: "cpp",
      code: "y",
      review: { due: "2020-01-02", stability: 1, reviews: 0, lastGrade: "" }
    }
  };

  await refreshBadge();
  assert.equal(badge.text, "2");
  assert.equal(badge.color, "#cf222e");

  await dispatch({ type: "skip-review", key: "7:cpp" });
  await dispatch({ type: "skip-review", key: "8:cpp" });
  await refreshBadge();
  assert.equal(badge.text, "", "no due reviews means no badge");

  delete store.problems["7:cpp"];
  delete store.problems["8:cpp"];
});

test("the hourly badge alarm is scheduled on install", () => {
  alarms.length = 0;
  listeners.installed();
  const scheduled = alarms.find((alarm) => alarm.name === REVIEW_ALARM);
  assert.ok(scheduled, "expected the review badge alarm");
  assert.equal(scheduled.options.periodInMinutes, 60);
});

test("save-settings updates only the settings through the shared lock", async () => {
  store.queue = [
    { key: "1:Python3", problem: { slug: "two-sum" }, attempts: 2, status: "failed", createdAt: "2024-01-01T00:00:00.000Z" }
  ];

  const response = await dispatch({
    type: "save-settings",
    settings: { interviewMode: true, branch: "develop" }
  });

  assert.equal(response.ok, true);
  assert.equal(response.settings.interviewMode, true);
  assert.equal(response.settings.branch, "develop");
  assert.equal(store.settings.interviewMode, true);
  assert.equal(store.queue.length, 1, "the queue is left untouched");

  delete store.queue[0];
  store.queue = [];
  store.settings.interviewMode = false;
  store.settings.branch = "main";
});

test("clear-failed discards failed jobs and clears the last error", async () => {
  store.queue = [
    { key: "1:Python3", problem: { slug: "two-sum" }, attempts: 5, status: "failed", error: "boom" },
    { key: "2:Python3", problem: { slug: "add-two-numbers" }, attempts: 0, status: "pending" }
  ];
  store.lastError = "boom";

  const response = await dispatch({ type: "clear-failed" });

  assert.equal(response.ok, true);
  assert.equal(response.removed, 1);
  assert.equal(store.queue.length, 1, "only the failed job is removed");
  assert.equal(store.queue[0].status, "pending");
  assert.equal(store.lastError, "");

  store.queue = [];
});

test("get-solution returns the stored entry that matches a resubmission", async () => {
  store.problems = {
    "1:Python3": {
      number: "1",
      slug: "two-sum",
      title: "Two Sum",
      language: "Python3",
      syncedAt: "2024-01-01T00:00:00.000Z",
      solutions: [
        { approach: "HashMap", code: "def twoSum(): return {}", notes: "aha", timeComplexity: "O(N)", spaceComplexity: "O(N)" }
      ]
    }
  };

  const match = await dispatch({ type: "get-solution", slug: "two-sum", code: "def twoSum(): return {}" });
  assert.equal(match.ok, true);
  assert.equal(match.solution.approach, "HashMap");
  assert.equal(match.solution.notes, "aha");

  const miss = await dispatch({ type: "get-solution", slug: "add-two-numbers", code: "x" });
  assert.equal(miss.solution, null, "a record-less slug has nothing to prefill");

  const changed = await dispatch({ type: "get-solution", slug: "two-sum", code: "improved v2" });
  assert.equal(changed.solution.approach, "HashMap", "a lone solution still prefills");

  delete store.problems["1:Python3"];
});
