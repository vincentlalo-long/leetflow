import assert from "node:assert/strict";
import test from "node:test";
import {
  compareVersions,
  fetchLatestRelease,
  getExtensionVersion,
  isUpdateAvailable,
  parseVersion
} from "../extension/update.js";
import { installChromeMock } from "./helpers/mock.js";

test("parseVersion handles v-prefixes and partial versions", () => {
  assert.deepEqual(parseVersion("v0.8.3"), [0, 8, 3]);
  assert.deepEqual(parseVersion("1.2"), [1, 2, 0]);
  assert.deepEqual(parseVersion(""), [0, 0, 0]);
});

test("compareVersions orders semver correctly", () => {
  assert.equal(compareVersions("0.8.3", "0.7.2"), 1);
  assert.equal(compareVersions("0.7.2", "0.8.3"), -1);
  assert.equal(compareVersions("v0.8.3", "0.8.3"), 0);
});

test("isUpdateAvailable detects a newer release", () => {
  assert.equal(isUpdateAvailable("0.7.2", "0.8.3"), true);
  assert.equal(isUpdateAvailable("0.8.3", "0.8.3"), false);
  assert.equal(isUpdateAvailable("0.8.3", ""), false);
});

test("getExtensionVersion reads the manifest", () => {
  installChromeMock({});
  assert.equal(getExtensionVersion(), "0.8.7");
});

test("fetchLatestRelease parses the GitHub payload", async () => {
  const fakeFetch = async () => ({
    ok: true,
    status: 200,
    json: async () => ({
      tag_name: "v0.8.3",
      html_url: "https://github.com/vincentlalo-long/leetflow/releases/tag/v0.8.3",
      body: "fixes",
      published_at: "2026-01-01T00:00:00Z"
    })
  });
  const latest = await fetchLatestRelease(fakeFetch);
  assert.equal(latest.version, "0.8.3");
  assert.match(latest.url, /releases/);
});

test("fetchLatestRelease throws a friendly rate-limit error", async () => {
  const fakeFetch = async () => ({ ok: false, status: 403, json: async () => ({}) });
  await assert.rejects(() => fetchLatestRelease(fakeFetch), /rate limit/i);
});
