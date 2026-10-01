import assert from "node:assert/strict";
import test from "node:test";
import { commitFiles, GitHubError } from "../extension/github.js";
import { createFetchRouter, registerGithubRoutes } from "./helpers/mock.js";

const settings = { token: "token", owner: "acme", repo: "leetcode", branch: "main" };

const files = [
  { path: "LeetCode/0001-two-sum/two-sum.py", content: "class Solution: pass" },
  { path: "LeetCode/0001-two-sum/README.md", content: "# Two Sum" },
  { path: "LeetCode/README.md", content: "# LeetCode Solutions" }
];

test("commitFiles batches every file into one commit using the Git Data API", async () => {
  const router = createFetchRouter();
  const repo = { commits: [] };
  registerGithubRoutes(router, repo, { initialHead: "head-sha" });

  const result = await commitFiles({ ...settings, files, message: "sync: 1. Two Sum" });

  assert.equal(router.calls.length, 5);
  assert.equal(router.calls[0].pathname, "/repos/acme/leetcode/git/ref/heads/main");
  assert.equal(router.calls[0].method, "GET");

  const treeCall = router.calls.find((call) => call.pathname.endsWith("/git/trees"));
  assert.equal(treeCall.method, "POST");
  assert.equal(treeCall.body.tree.length, 3);
  assert.equal(treeCall.body.tree[0].content, "class Solution: pass");
  assert.ok(treeCall.body.base_tree, "existing repositories must build on the base tree");

  const commitCall = router.calls.find(
    (call) => call.pathname.endsWith("/git/commits") && call.method === "POST"
  );
  assert.deepEqual(commitCall.body.parents, ["head-sha"]);

  const patchCall = router.calls.find((call) => call.method === "PATCH");
  assert.ok(patchCall.pathname.endsWith("/git/refs/heads/main"));

  assert.equal(result.sha, repo.commits.at(-1).sha);
  assert.equal(result.files, 3);
});

test("commitFiles bootstraps an empty repository with a root commit", async () => {
  const router = createFetchRouter();
  const repo = { commits: [] };
  registerGithubRoutes(router, repo);

  const result = await commitFiles({ ...settings, files, message: "sync: initial" });

  const treeCall = router.calls.find((call) => call.pathname.endsWith("/git/trees"));
  assert.equal(treeCall.body.base_tree, undefined);

  const commitCall = router.calls.find(
    (call) => call.pathname.endsWith("/git/commits") && call.method === "POST"
  );
  assert.deepEqual(commitCall.body.parents, []);

  const createRefCall = router.calls.find(
    (call) => call.pathname.endsWith("/git/refs") && call.method === "POST"
  );
  assert.equal(createRefCall.body.ref, "refs/heads/main");
  assert.equal(router.calls.length, 4);
  assert.ok(result.sha);
});

test("commitFiles retries once when the branch head moved", async () => {
  const router = createFetchRouter();
  const repo = { commits: [] };
  registerGithubRoutes(router, repo, { initialHead: "head-sha", patchFailures: 1 });

  const result = await commitFiles({ ...settings, files, message: "sync" });

  assert.equal(result.sha, repo.commits.at(-1).sha);
  assert.equal(repo.commits.length, 2, "the first commit was orphaned and retried");
  assert.ok(router.calls.filter((call) => call.pathname.endsWith("/git/trees")).length === 2);
});

test("commitFiles surfaces GitHub rate limits without retrying", async () => {
  const router = createFetchRouter();
  router.route("GET", /\/git\/ref\/heads\/[^/]+$/, () => ({
    status: 403,
    body: { message: "API rate limit exceeded" },
    headers: { "x-ratelimit-remaining": "0", "x-ratelimit-reset": "1893456000" }
  }));

  await assert.rejects(
    () => commitFiles({ ...settings, files, message: "sync" }),
    (error) => {
      assert.ok(error instanceof GitHubError);
      assert.equal(error.rateLimited, true);
      return true;
    }
  );
  assert.equal(router.calls.length, 1, "rate limited calls must not be retried immediately");
});

test("commitFiles rejects an empty batch and missing configuration", async () => {
  await assert.rejects(() => commitFiles({ ...settings, files: [], message: "x" }), /Nothing to commit/);
  await assert.rejects(
    () => commitFiles({ token: "", owner: "acme", repo: "leetcode", files, message: "x" }),
    /Missing GitHub token/
  );
  await assert.rejects(
    () => commitFiles({ token: "t", owner: "", repo: "", files, message: "x" }),
    /owner and repository/
  );
});
