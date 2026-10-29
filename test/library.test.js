import assert from "node:assert/strict";
import test from "node:test";
import {
  configureRepository,
  createFetchRouter,
  installChromeMock,
  registerGithubRoutes,
  registerGraphqlRoute
} from "./helpers/mock.js";

const { enqueue, resetQueueForTests, saveProblemEdits } = await import("../extension/queue.js");
const { loadState } = await import("../extension/storage.js");

function solution(overrides = {}) {
  return {
    number: "",
    slug: "two-sum",
    title: "Two Sum",
    url: "https://leetcode.com/problems/two-sum/",
    language: "python3",
    code: "class Solution:\n    def twoSum(self, nums, target):\n        return []",
    description: "Given an array of integers.",
    tags: ["Array", "Hash Table"],
    difficulty: "Easy",
    acceptedAt: "2024-01-01T00:00:00.000Z",
    timeSpent: "3m 20s",
    attemptsSummary: "1st try (Clean AC)",
    ...overrides
  };
}

function setup(settings = {}) {
  installChromeMock(configureRepository(settings));
  const router = createFetchRouter();
  const repo = { commits: [] };
  registerGraphqlRoute(router, (slug) => ({
    questionFrontendId: slug === "two-sum" ? "1" : "2",
    title: slug === "two-sum" ? "Two Sum" : "Add Two Numbers",
    difficulty: "Easy",
    topicTags: [{ name: "Array" }]
  }));
  const git = registerGithubRoutes(router, repo, { initialHead: "head-sha" });
  resetQueueForTests();
  return { router, repo, git };
}

test("editing notes and complexity re-commits the problem README", async () => {
  const { repo, git } = setup();
  await enqueue(solution());

  const key = "1:Python3";
  const response = await saveProblemEdits(key, 0, {
    notes: "Use a hash map",
    timeComplexity: "O(N)",
    spaceComplexity: "O(N)"
  });

  assert.equal(response.ok, true);
  assert.equal(response.status, "queued-update");
  assert.equal(response.syncResult.synced, 1);
  assert.equal(repo.commits.length, 2);

  const readme = git.read("LeetCode/0001-two-sum/README.md");
  assert.match(readme, /Use a hash map/);
  assert.match(readme, /O\(N\)/);

  const state = await loadState();
  assert.equal(state.problems[key].solutions[0].notes, "Use a hash map");
  assert.equal(state.problems[key].solutions[0].timeComplexity, "O(N)");
  assert.equal(state.queue.length, 0);
});

test("clearing a note removes it from the README", async () => {
  const { git } = setup();
  await enqueue(solution({ notes: "First insight" }));

  const key = "1:Python3";
  const response = await saveProblemEdits(key, 0, {
    notes: "",
    timeComplexity: "",
    spaceComplexity: ""
  });

  assert.equal(response.syncResult.synced, 1);
  const state = await loadState();
  assert.equal(state.problems[key].solutions[0].notes, "");

  const readme = git.read("LeetCode/0001-two-sum/README.md");
  assert.doesNotMatch(readme, /First insight/);
  assert.match(readme, /_Add your notes here\._/);
});

test("saving identical fields reports unchanged and skips the commit", async () => {
  const { repo } = setup();
  await enqueue(solution({ notes: "Keep me", timeComplexity: "O(N)" }));

  const response = await saveProblemEdits("1:Python3", 0, {
    notes: "Keep me",
    timeComplexity: "O(N)",
    spaceComplexity: ""
  });

  assert.equal(response.status, "unchanged");
  assert.equal(repo.commits.length, 1);
});

test("editing a missing problem rejects", async () => {
  setup();
  await assert.rejects(
    () => saveProblemEdits("missing", 0, { notes: "x" }),
    /no longer in your local archive/
  );
});
