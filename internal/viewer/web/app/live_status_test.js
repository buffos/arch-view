const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "live_status.js"), "utf8");
const statusStyles = fs.readFileSync(path.join(__dirname, "..", "styles", "02-typography.css"), "utf8");

function statusElement() {
  return {
    dataset: {},
    hidden: true,
    focusCalls: 0,
    focus: function () { this.focusCalls += 1; },
    setAttribute: function (name, value) { this[name] = value; }
  };
}

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(async function (liveStatus) {
  const element = statusElement();
  const context = {
    liveEnabled: true,
    liveSessionID: "session/live",
    elements: { liveStatus: element },
    state: {}
  };
  const responses = [
    {
      revision: 1,
      freshness: { status: "current" },
      result: { state: "ready", snapshot: { revision: 1 } }
    },
    {
      revision: 2,
      freshness: { status: "stale" },
      result: { state: "updating", diagnostics: [{ message: "The input changed during analysis." }] }
    }
  ];
  const requested = [];
  const api = {
    getJSON: async function (endpoint) {
      requested.push(endpoint);
      return responses.shift();
    }
  };
  const controller = liveStatus.createLiveController(context, api);

  const ready = await controller.waitForReady(1);
  assert.equal(ready.state, "ready");
  assert.equal(ready.freshness, "current");
  assert.equal(context.state.liveStatus.revision, 1);
  assert.equal(context.state.liveRevision, 1);
  assert.equal(element.hidden, false);
  assert.equal(element.dataset.status, "current");
  assert.match(element.textContent, /Live · ready · current · revision 1/);
  assert.deepEqual(requested, ["/v1/live/session%2Flive/status"]);

  let refreshedRevision = 0;
  controller.bindRevisionHandler(async function (revision) { refreshedRevision = revision; });
  const updating = await controller.refresh();
  assert.equal(updating.state, "updating");
  assert.equal(updating.freshness, "stale");
  assert.equal(refreshedRevision, 2);
  assert.equal(context.state.liveRevision, 2);
  assert.equal(element.dataset.status, "stale");
  assert.match(element.textContent, /1 diagnostic/);
  assert.match(element["aria-label"], /The input changed during analysis/);
  assert.equal(element.focusCalls, 0, "status refresh must not move keyboard focus");
  assert.match(statusStyles, /\.session-meta\s*\{[^}]*flex-wrap:\s*wrap/s);
  assert.match(statusStyles, /@media \(max-width:\s*680px\)[\s\S]*\.live-status\s*\{[^}]*white-space:\s*normal/s);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
