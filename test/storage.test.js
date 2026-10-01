import assert from "node:assert/strict";
import test from "node:test";
import { DEFAULT_SETTINGS, loadState, migrateLegacyState, saveState } from "../extension/storage.js";
import { installChromeMock } from "./helpers/mock.js";

test("loadState returns defaults when storage is empty", async () => {
  installChromeMock({});
  const state = await loadState();
  assert.deepEqual(state.settings, DEFAULT_SETTINGS);
  assert.deepEqual(state.queue, []);
  assert.deepEqual(state.problems, {});
  assert.equal(state.lastSync, null);
  assert.equal(state.migrated, false);
});

test("legacy synced/syncedProblems/reviews data is migrated into problems", async () => {
  const { store } = installChromeMock({
    settings: { githubToken: "t", owner: "o", repo: "r" },
    synced: { "1:Python3": "2024-01-01T00:00:00.000Z" },
    syncedProblems: {
      "1:Python3": {
        number: "1",
        language: "Python3",
        title: "Two Sum",
        slug: "two-sum",
        tags: ["Array"],
        difficulty: "Easy",
        code: "x"
      }
    },
    reviews: {
      "1:Python3": {
        number: "1",
        language: "Python3",
        title: "Two Sum",
        slug: "two-sum",
        review: { due: "2024-01-05", stability: 4, reviews: 1, lastGrade: "good" }
      }
    }
  });

  const state = await loadState();
  assert.equal(state.migrated, true);
  const record = state.problems["1:Python3"];
  assert.equal(record.title, "Two Sum");
  assert.equal(record.difficulty, "Easy");
  assert.equal(record.syncedAt, "2024-01-01T00:00:00.000Z");
  assert.deepEqual(record.review, { due: "2024-01-05", stability: 4, reviews: 1, lastGrade: "good" });
  assert.equal(state.settings.repo, "r");

  await saveState(state);
  assert.deepEqual(store.synced, {}, "legacy keys are cleared after migration");
  assert.deepEqual(store.syncedProblems, {});
  assert.deepEqual(store.reviews, {});

  const second = await loadState();
  assert.equal(second.migrated, false, "migration must only run once");
  assert.equal(second.problems["1:Python3"].title, "Two Sum");
});

test("migrateLegacyState keeps a stub record for sync markers without metadata", () => {
  const problems = migrateLegacyState({ synced: { "42:Go": "2024-02-02T00:00:00.000Z" } });
  assert.equal(problems["42:Go"].number, "42");
  assert.equal(problems["42:Go"].language, "Go");
  assert.equal(problems["42:Go"].syncedAt, "2024-02-02T00:00:00.000Z");
});
