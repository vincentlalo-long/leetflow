import test from "node:test";
import assert from "node:assert/strict";
import { configureRepository, installChromeMock } from "./helpers/mock.js";

const { listeners, store } = installChromeMock(configureRepository());
await import("../extension/background.js");

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
