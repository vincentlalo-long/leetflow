import test from "node:test";
import assert from "node:assert/strict";
import { configureRepository, installChromeMock } from "./helpers/mock.js";

const { listeners } = installChromeMock(configureRepository());
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
  assert.equal(typeof response.total, "number");
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
