import assert from "node:assert/strict";
import test from "node:test";
import {
  ROADMAPS,
  getRoadmapBadges,
  getRoadmapProgress,
  getRoadmaps
} from "../extension/roadmaps.js";

test("official LeetCode study plans are recognized by problem number", () => {
  assert.ok(getRoadmaps(1768).includes("LeetCode 75"));
  assert.ok(getRoadmaps(88).includes("Top Interview 150"));
  assert.ok(getRoadmaps(271).includes("Blind 75"));
  assert.ok(getRoadmaps(25).includes("NeetCode 150"));

  const twoSum = getRoadmaps(1);
  assert.ok(twoSum.includes("Top Interview 150"));
  assert.ok(twoSum.includes("Top 100 Liked"));
  assert.ok(twoSum.includes("Blind 75"));
  assert.ok(twoSum.includes("NeetCode 150"));

  assert.deepEqual(getRoadmaps(999999), []);
  assert.deepEqual(getRoadmaps(""), []);
  assert.deepEqual(getRoadmaps("not-a-number"), []);
});

test("README badges cover every plan a problem belongs to", () => {
  const badges = getRoadmapBadges(1);
  assert.match(badges, /Roadmap-Top%20Interview%20150-ef4743/);
  assert.match(badges, /Roadmap-Top%20100%20Liked-1a7f37/);
  assert.match(badges, /Roadmap-Blind%2075-007acc/);
  assert.match(badges, /Roadmap-NeetCode%20150-6366f1/);
  assert.doesNotMatch(badges, /LeetCode%2075/);

  assert.match(getRoadmapBadges(1768), /Roadmap-LeetCode%2075-f89f1b/);
  assert.equal(getRoadmapBadges(999999), "");
});

test("progress counts unique solved problems per plan", () => {
  const progress = getRoadmapProgress([
    { number: "1" },
    { number: 1768 },
    { number: "88" },
    { number: "88" },
    { number: "" },
    { number: "999999" }
  ]);
  const byName = Object.fromEntries(progress.map((plan) => [plan.name, plan]));

  assert.equal(progress.length, ROADMAPS.length);
  assert.deepEqual(
    progress.map((plan) => plan.name),
    ["LeetCode 75", "Top Interview 150", "Top 100 Liked", "Blind 75", "NeetCode 150"]
  );
  assert.equal(byName["LeetCode 75"].solved, 1);
  assert.equal(byName["Top Interview 150"].solved, 2);
  assert.equal(byName["Top 100 Liked"].solved, 1);
  assert.equal(byName["Blind 75"].solved, 1);
  assert.equal(byName["NeetCode 150"].solved, 1);
  assert.deepEqual(
    progress.map((plan) => plan.total),
    [75, 150, 100, 75, 150]
  );
  assert.equal(byName["Top Interview 150"].percent, Math.round((2 / 150) * 100));
});

test("progress starts at zero with no synced problems", () => {
  const progress = getRoadmapProgress();
  assert.ok(progress.every((plan) => plan.solved === 0 && plan.percent === 0));
  assert.ok(getRoadmapProgress([]).every((plan) => plan.solved === 0));
});
