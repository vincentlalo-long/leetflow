import assert from "node:assert/strict";
import test from "node:test";
import { ROOT_START } from "../extension/templates.js";
import {
  configureRepository,
  createFetchRouter,
  installChromeMock,
  registerGithubRoutes,
  registerGraphqlRoute
} from "./helpers/mock.js";

const { enqueue, resetQueueForTests } = await import("../extension/queue.js");
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

function setup(settings = {}, beforeGithubRoutes) {
  const chromeMock = installChromeMock(configureRepository(settings));
  const router = createFetchRouter();
  const repo = { commits: [] };
  registerGraphqlRoute(router, (slug) => ({
    questionFrontendId: slug === "two-sum" ? "1" : "2",
    title: slug === "two-sum" ? "Two Sum" : "Add Two Numbers",
    difficulty: "Easy",
    topicTags: [{ name: "Array" }]
  }));
  if (beforeGithubRoutes) beforeGithubRoutes(router);
  const git = registerGithubRoutes(router, repo, { initialHead: "head-sha" });
  resetQueueForTests();
  return { chromeMock, router, repo, git };
}

function treeCalls(router) {
  return router.calls.filter(
    (call) => call.method === "POST" && call.pathname.endsWith("/git/trees")
  );
}

test("enqueue pushes the solution, its README and the root index in one commit", async () => {
  const { router, repo, git, chromeMock } = setup();

  const response = await enqueue(solution());

  assert.equal(response.ok, true);
  assert.equal(response.configured, true);
  assert.equal(response.status, "queued");
  assert.equal(response.syncResult.synced, 1);
  assert.equal(treeCalls(router).length, 1, "one batch means one commit");
  assert.equal(repo.commits.length, 1);

  const files = git.files();
  assert.deepEqual(files, [
    "LeetCode/0001-two-sum/README.md",
    "LeetCode/0001-two-sum/two-sum.py",
    "LeetCode/README.md"
  ]);

  const root = git.read("LeetCode/README.md");
  assert.match(root, /Two Sum/);
  assert.equal(root.split(ROOT_START).length, 2, "root index is delimited by markers");

  const problemReadme = git.read("LeetCode/0001-two-sum/README.md");
  assert.match(problemReadme, /\[`two-sum\.py`\]\(\.\/two-sum\.py\)/);

  const state = await loadState();
  assert.equal(state.queue.length, 0);
  assert.equal(state.lastError, "");
  assert.equal(state.lastSync.count, 1);
  assert.ok(state.problems["1:Python3"].syncedAt);
  assert.ok(chromeMock.store.problems["1:Python3"]);
});

test("an identical resubmission is reported as unchanged and does not commit", async () => {
  const { router, repo } = setup();

  await enqueue(solution());
  const second = await enqueue(solution());

  assert.equal(second.status, "unchanged");
  assert.ok(!second.syncResult, "nothing to run for an unchanged solution");
  assert.equal(treeCalls(router).length, 1, "no extra commit for unchanged content");
  assert.equal(repo.commits.length, 1);
});

test("a changed solution is re-committed and the root index stays intact", async () => {
  const { router, git } = setup();

  await enqueue(solution());
  const updated = await enqueue(solution({ code: "class Solution:\n    pass  # improved" }));

  assert.equal(updated.status, "queued-update");
  assert.equal(updated.syncResult.synced, 1);
  assert.equal(treeCalls(router).length, 2);
  assert.equal(git.read("LeetCode/0001-two-sum/two-sum.py"), "class Solution:\n    pass  # improved");

  const root = git.read("LeetCode/README.md");
  assert.equal(root.split(ROOT_START).length, 2);
  assert.match(root, /Two Sum/);
});

test("concurrent enqueues are serialized into a single commit without losing problems", async () => {
  const { router, git } = setup();

  await Promise.all([
    enqueue(solution()),
    enqueue(
      solution({
        number: "2",
        slug: "add-two-numbers",
        title: "Add Two Numbers",
        url: "https://leetcode.com/problems/add-two-numbers/",
        code: "class Solution:\n    def addTwoNumbers(self, a, b):\n        return None"
      })
    )
  ]);

  const state = await loadState();
  assert.equal(state.queue.length, 0, "the queue drains once");
  assert.equal(Object.keys(state.problems).length, 2);

  const root = git.read("LeetCode/README.md");
  assert.match(root, /Two Sum/);
  assert.match(root, /Add Two Numbers/);
  assert.equal(root.split(ROOT_START).length, 2);
  assert.ok(git.files().includes("LeetCode/0002-add-two-numbers/add-two-numbers.py"));
});

test("an unconfigured extension queues the solution locally without touching GitHub", async () => {
  const { router, git } = setup({ githubToken: "", owner: "", repo: "" });

  const response = await enqueue(solution());

  assert.equal(response.configured, false);
  assert.ok(!response.syncResult, "no sync is attempted before Settings are filled in");
  assert.equal(
    router.calls.filter((call) => call.url.includes("api.github.com")).length,
    0,
    "no GitHub request is issued"
  );
  assert.equal(git.files().length, 0);

  const state = await loadState();
  assert.equal(state.queue.length, 1, "the job waits for Settings");
  assert.ok(state.problems["1:Python3"], "the review record is still created locally");
});

test("a repository failure is surfaced and keeps the job retryable", async () => {
  const { router } = setup({}, (target) => {
    target.route("POST", /\/git\/trees$/, () => ({
      status: 403,
      body: { message: "Resource not accessible by personal access token" }
    }));
  });

  const response = await enqueue(solution());

  assert.equal(response.syncResult.ok, false);
  assert.equal(response.syncResult.failed, 1);
  assert.match(response.syncResult.errors[0], /Resource not accessible/);

  const state = await loadState();
  assert.equal(state.queue.length, 1);
  assert.equal(state.queue[0].status, "failed");
  assert.match(state.lastError, /Resource not accessible/);
});
