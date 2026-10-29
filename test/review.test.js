import assert from "node:assert/strict";
import test from "node:test";
import { dateOnly, ensureReview, gradeReview, skipReview } from "../extension/review.js";

test("gradeReview appends to history and schedules the next review", () => {
  const problem = ensureReview({ number: "1", title: "Two Sum", durationSeconds: 300, attemptsCount: 1 });
  assert.equal(problem.review.due, dateOnly());

  const graded = gradeReview(problem, "good");
  assert.equal(graded.review.reviews, 1);
  assert.equal(graded.review.lastGrade, "good");
  assert.equal(graded.review.history.length, 1);
  assert.equal(graded.review.history[0].grade, "good");
  assert.ok(graded.review.due > dateOnly());

  const again = gradeReview(graded, "again");
  assert.deepEqual(again.review.history.map((entry) => entry.grade), ["good", "again"]);
});

test("ensureReview preserves existing history", () => {
  const graded = gradeReview(ensureReview({ number: "1", title: "Two Sum" }), "hard");
  const reviewed = ensureReview(graded);
  assert.equal(reviewed.review.history.length, 1);
  assert.equal(reviewed.review.history[0].grade, "hard");
});

test("skipReview clears the due date but keeps history", () => {
  const graded = gradeReview(ensureReview({ number: "1", title: "Two Sum" }), "easy");
  const skipped = skipReview(graded);
  assert.equal(skipped.review.skipped, true);
  assert.equal(skipped.review.due, "");
  assert.equal(skipped.review.lastGrade, "skip");
  assert.equal(skipped.review.history.length, 1);
});
