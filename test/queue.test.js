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

const {
  MAX_JOB_ATTEMPTS,
  enqueue,
  resetQueueForTests,
  resolveProblemMetadata,
  runQueue
} = await import("../extension/queue.js");
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
  if (beforeGithubRoutes) beforeGithubRoutes(router);
  registerGraphqlRoute(router, (slug) => ({
    questionFrontendId: slug === "two-sum" ? "1" : "2",
    title: slug === "two-sum" ? "Two Sum" : "Add Two Numbers",
    difficulty: "Easy",
    topicTags: [{ name: "Array" }]
  }));
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

test("filling in complexity after a sync updates the stored problem", async () => {
  const { git } = setup();

  await enqueue(solution());
  const response = await enqueue(solution({ timeComplexity: "O(N)", spaceComplexity: "O(1)" }));

  assert.equal(response.status, "queued-update");

  const readme = git.read("LeetCode/0001-two-sum/README.md");
  assert.match(readme, /\| Time Complexity \| `O\(N\)` \|/);
  assert.match(readme, /\| Space Complexity \| `O\(1\)` \|/);

  const state = await loadState();
  assert.equal(state.problems["1:Python3"].timeComplexity, "O(N)");

  const unchanged = await enqueue(solution({ timeComplexity: "O(N)", spaceComplexity: "O(1)" }));
  assert.equal(unchanged.status, "unchanged");
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

test("a second approach is stored alongside the first and synced as its own file", async () => {
  const { git } = setup();

  await enqueue(solution());
  const response = await enqueue(
    solution({ approach: "HashMap", code: "def twoSum(nums, target):\n    seen = {}" })
  );

  assert.equal(response.status, "queued-update");
  assert.equal(response.syncResult.synced, 1);

  const files = git.files();
  assert.ok(files.includes("LeetCode/0001-two-sum/two-sum.py"), "default file keeps its name");
  assert.ok(files.includes("LeetCode/0001-two-sum/two-sum-hashmap.py"), "labeled approach gets a suffix");

  const readme = git.read("LeetCode/0001-two-sum/README.md");
  assert.match(readme, /## Solutions/);
  assert.match(readme, /### HashMap/);
  assert.match(readme, /\[`two-sum-hashmap\.py`\]\(\.\/two-sum-hashmap\.py\)/);

  const state = await loadState();
  const record = state.problems["1:Python3"];
  assert.equal(record.solutions.length, 2);
  assert.equal(record.solutions[0].approach, "");
  assert.equal(record.solutions[1].approach, "HashMap");
  assert.ok(record.review, "the review schedule stays attached to the shared record");
});

test("re-syncing the same approach updates it instead of adding a duplicate", async () => {
  const { git } = setup();

  await enqueue(solution({ approach: "Brute Force", code: "brute v1" }));
  const response = await enqueue(solution({ approach: "brute-force", code: "brute v2" }));

  assert.equal(response.status, "queued-update");

  const state = await loadState();
  const record = state.problems["1:Python3"];
  assert.equal(record.solutions.length, 1, "labels that slugify the same are one approach");
  assert.equal(record.solutions[0].approach, "brute-force");
  assert.equal(record.solutions[0].code, "brute v2");
  assert.equal(
    git.read("LeetCode/0001-two-sum/two-sum-brute-force.py"),
    "brute v2"
  );
});

test("an unchanged second approach is reported as unchanged", async () => {
  setup();

  await enqueue(solution());
  await enqueue(solution({ approach: "HashMap", code: "def twoSum(): return {}" }));
  const repeated = await enqueue(solution({ approach: "HashMap", code: "def twoSum(): return {}" }));

  assert.equal(repeated.status, "unchanged");
  assert.ok(!repeated.syncResult);
});

test("an unlabeled resubmission of a labeled approach does not duplicate it", async () => {
  const { git } = setup();

  await enqueue(solution({ approach: "HashMap", code: "def twoSum(): return {}" }));
  const again = await enqueue(solution({ approach: "", code: "def twoSum(): return {}" }));

  assert.equal(again.status, "unchanged");
  const state = await loadState();
  const record = state.problems["1:Python3"];
  assert.equal(record.solutions.length, 1, "no unlabeled duplicate is pushed");
  assert.equal(record.solutions[0].approach, "HashMap");
  assert.ok(!git.files().includes("LeetCode/0001-two-sum/two-sum.py"));
});

test("resubmitting without notes keeps the stored notes and complexity", async () => {
  const { git } = setup();

  await enqueue(
    solution({ notes: "aha", timeComplexity: "O(N)", spaceComplexity: "O(1)" })
  );
  const again = await enqueue(solution());

  assert.equal(again.status, "unchanged", "empty prompts are treated as no change");

  const state = await loadState();
  const record = state.problems["1:Python3"];
  assert.equal(record.notes, "aha");
  assert.equal(record.timeComplexity, "O(N)");
  assert.match(git.read("LeetCode/0001-two-sum/README.md"), /aha/);
});

test("a rate limited drain schedules one retry at the reset time", async () => {
  const { chromeMock } = setup({}, (target) => {
    target.route("POST", /\/git\/trees$/, () => ({
      status: 403,
      body: { message: "API rate limit exceeded" },
      headers: {
        "x-ratelimit-remaining": "0",
        "x-ratelimit-reset": String(Math.ceil(Date.now() / 1000) + 1800)
      }
    }));
  });

  const response = await enqueue(solution());

  assert.equal(response.syncResult.rateLimited, true);
  const queueAlarms = chromeMock.alarms.filter(
    (alarm) => alarm.name === "leetflow-sync-queue"
  );
  assert.equal(queueAlarms.length, 1, "the backoff is not overwritten by a 1 minute alarm");
  assert.ok(
    queueAlarms[0].options.delayInMinutes >= 25,
    `expected the reset window, got ${queueAlarms[0].options.delayInMinutes}`
  );
});

test("a permanently failing job stops retrying after the attempt cap", async () => {
  const { chromeMock, router } = setup({}, (target) => {
    target.route("POST", /\/git\/trees$/, () => ({
      status: 403,
      body: { message: "Resource not accessible by personal access token" }
    }));
  });

  const first = await enqueue(solution());
  assert.equal(first.syncResult.failed, 1);

  const delays = [];
  for (let attempt = 1; attempt < MAX_JOB_ATTEMPTS; attempt++) {
    chromeMock.alarms.length = 0;
    await runQueue();
    const alarm = chromeMock.alarms.find((item) => item.name === "leetflow-sync-queue");
    delays.push(alarm?.options?.delayInMinutes);
  }

  assert.deepEqual(delays, [2, 4, 8, undefined], "each retry backs off, the last one stops");

  const state = await loadState();
  assert.equal(state.queue.length, 1);
  assert.equal(state.queue[0].status, "failed");
  assert.equal(state.queue[0].attempts, MAX_JOB_ATTEMPTS);

  chromeMock.alarms.length = 0;
  const exhausted = await runQueue();
  const stateAfter = await loadState();
  assert.equal(stateAfter.queue[0].attempts, MAX_JOB_ATTEMPTS, "no further attempts");
  assert.equal(
    chromeMock.alarms.filter((item) => item.name === "leetflow-sync-queue").length,
    0,
    "an exhausted queue schedules no more retries"
  );
  assert.equal(exhausted.ok, true, "an empty eligible queue is not an error");

  assert.ok(
    router.calls.filter((call) => call.pathname.endsWith("/git/trees")).length >= 1
  );
});

test("a problem whose number resolves during drain leaves no placeholder record", async () => {
  let graphqlCalls = 0;
  const { git } = setup({}, (target) => {
    target.route("POST", /\/graphql$/, () => {
      graphqlCalls += 1;
      if (graphqlCalls === 1) return { status: 500, body: { message: "temporary" } };
      return {
        status: 200,
        body: {
          data: {
            question: {
              questionFrontendId: "1",
              title: "Two Sum",
              difficulty: "Easy",
              topicTags: [{ name: "Array" }]
            }
          }
        }
      };
    });
  });

  const response = await enqueue(solution());
  assert.equal(response.key, ":Python3", "the number was unknown at enqueue time");

  const state = await loadState();
  assert.ok(state.problems["1:Python3"], "the record moved to the resolved key");
  assert.equal(state.problems[":Python3"], undefined, "the placeholder is gone");
  assert.equal(state.queue.length, 0);
  assert.ok(git.read("LeetCode/0001-two-sum/README.md").includes("Two Sum"));
});

test("syncing a second language keeps one folder, one README and one index row", async () => {
  const { git } = setup();

  await enqueue(solution());
  await enqueue(
    solution({ language: "cpp", code: "class Solution { public: vector<int> twoSum(vector<int>& n, int t) { return {}; } };" })
  );

  const files = git.files();
  assert.ok(files.includes("LeetCode/0001-two-sum/two-sum.py"));
  assert.ok(files.includes("LeetCode/0001-two-sum/two-sum.cpp"));

  const readme = git.read("LeetCode/0001-two-sum/README.md");
  assert.match(readme, /\[`two-sum\.py`\]\(\.\/two-sum\.py\)/);
  assert.match(readme, /\[`two-sum\.cpp`\]\(\.\/two-sum\.cpp\)/);
  assert.match(readme, /### Python3/);
  assert.match(readme, /### C\+\+/);

  const root = git.read("LeetCode/README.md");
  const rows = root.split("\n").filter((line) => /^\| 1 \|/.test(line));
  assert.equal(rows.length, 1, "the index does not list the same problem twice");
  assert.match(rows[0], /Python3, C\+\+/);
});
