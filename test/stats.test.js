import assert from "node:assert/strict";
import test from "node:test";
import {
  collectActivity,
  difficultyStats,
  gradeStats,
  heatmapWeeks,
  streakStats,
  tagStats
} from "../extension/stats.js";

function problem(overrides = {}) {
  return {
    number: "1",
    title: "Two Sum",
    difficulty: "Easy",
    tags: ["Array"],
    acceptedAt: "2026-10-01T10:00:00.000Z",
    ...overrides
  };
}

test("collectActivity counts solves and reviews per day", () => {
  const days = collectActivity([
    problem({ acceptedAt: "2026-10-29T08:00:00.000Z" }),
    problem({ acceptedAt: "2026-10-29T12:00:00.000Z" }),
    problem({
      acceptedAt: "2026-10-28T08:00:00.000Z",
      review: { history: [{ at: "2026-10-29T18:00:00.000Z", grade: "good" }] }
    })
  ]);

  assert.deepEqual(days.get("2026-10-29"), { solves: 2, reviews: 1 });
  assert.deepEqual(days.get("2026-10-28"), { solves: 1, reviews: 0 });
});

test("heatmapWeeks returns seven cells per week and marks activity", () => {
  const weeks = heatmapWeeks(
    [
      problem({
        acceptedAt: "2026-10-29T08:00:00.000Z",
        review: { history: [{ at: "2026-10-29T09:00:00.000Z", grade: "good" }] }
      })
    ],
    { today: "2026-10-29", weeks: 4 }
  );

  assert.equal(weeks.length, 4);
  assert.ok(weeks.every((week) => week.length === 7));
  const cells = weeks.flat();
  assert.equal(cells.length, 28);

  const todayCell = cells.find((cell) => cell.date === "2026-10-29");
  assert.equal(todayCell.solves, 1);
  assert.equal(todayCell.reviews, 1);
  assert.equal(todayCell.total, 2);
  assert.equal(todayCell.future, false);
});

test("streakStats reports current and longest streaks", () => {
  const stats = streakStats(
    [
      problem({ acceptedAt: "2026-10-20T08:00:00.000Z" }),
      problem({ acceptedAt: "2026-10-21T08:00:00.000Z" }),
      problem({ acceptedAt: "2026-10-27T08:00:00.000Z" }),
      problem({ acceptedAt: "2026-10-28T08:00:00.000Z" }),
      problem({ acceptedAt: "2026-10-29T08:00:00.000Z" })
    ],
    "2026-10-29"
  );

  assert.equal(stats.current, 3);
  assert.equal(stats.longest, 3);
  assert.equal(stats.activeDays, 5);
  assert.equal(stats.todayActive, true);
});

test("streakStats keeps the streak alive until the end of today", () => {
  const stats = streakStats(
    [
      problem({ acceptedAt: "2026-10-27T08:00:00.000Z" }),
      problem({ acceptedAt: "2026-10-28T08:00:00.000Z" })
    ],
    "2026-10-29"
  );

  assert.equal(stats.current, 2);
  assert.equal(stats.todayActive, false);
});

test("gradeStats combines history with legacy lastGrade records", () => {
  const stats = gradeStats([
    { review: { history: [{ at: "2026-10-01T00:00:00.000Z", grade: "again" }, { at: "2026-10-02T00:00:00.000Z", grade: "good" }] } },
    { review: { history: [], lastGrade: "easy", reviews: 1 } },
    { review: { lastGrade: "skip", skipped: true } }
  ]);

  assert.deepEqual(stats.counts, { again: 1, hard: 0, good: 1, easy: 1, skip: 1 });
  assert.equal(stats.total, 3);
  assert.ok(Math.abs(stats.recallRate - 2 / 3) < 1e-9);
});

test("difficultyStats buckets unknown difficulties", () => {
  const counts = difficultyStats([
    { difficulty: "Easy" },
    { difficulty: "Medium" },
    { difficulty: "Hard" },
    { difficulty: "Unknown" },
    { difficulty: "Weird" }
  ]);

  assert.deepEqual(counts, { Easy: 1, Medium: 1, Hard: 1, Unknown: 2 });
});

test("tagStats sorts by frequency and respects the limit", () => {
  const stats = tagStats(
    [
      { tags: ["Array", "Hash Table"] },
      { tags: ["Array"] },
      { tags: ["DP"] }
    ],
    2
  );

  assert.deepEqual(stats, [
    { tag: "Array", count: 2 },
    { tag: "DP", count: 1 }
  ]);
});
