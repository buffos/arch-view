const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "okf_hidden_concepts.js")).href).then(async ({ renderHiddenConcepts }) => {
  function element() { return { innerHTML: "", disabled: true, events: {}, addEventListener(name, callback) { this.events[name] = callback; } }; }
  const parts = { details: element(), p: element(), select: element(), button: element() };
  const container = { innerHTML: "", querySelector: (name) => parts[name] };
  const state = { snapshot: { source: { bundle_id: "bundle", source_revision: "r1" }, counts: { hidden_nodes: 1 }, nodes: [{ id: "visible" }] } };
  let selected;
  let calls = 0;
  const api = { getSummary: async () => { calls++; return { source_revision: "r1", files: ["visible", "hidden<&"] }; } };
  renderHiddenConcepts(container, state, api, (id) => { selected = id; });
  assert.equal(calls, 0, "collapsed section must not fetch");
  parts.details.open = true;
  await parts.details.events.toggle();
  assert.equal(parts.button.disabled, false);
  assert.doesNotMatch(parts.select.innerHTML, /visible/);
  assert.match(parts.select.innerHTML, /hidden&lt;&amp;/);
  parts.select.value = "hidden<&";
  parts.button.events.click();
  assert.equal(selected, "hidden<&");
  renderHiddenConcepts(container, state, { getSummary: async () => ({ source_revision: "r2", files: [] }) }, () => {});
  await parts.details.events.toggle();
  assert.match(parts.p.textContent, /Source changed/);
  let finish;
  renderHiddenConcepts(container, state, { getSummary: () => new Promise((resolve) => { finish = resolve; }) }, () => {});
  const pending = parts.details.events.toggle();
  const before = parts.select.innerHTML;
  state.snapshot = { ...state.snapshot };
  finish({ source_revision: "r1", files: ["stale"] });
  await pending;
  assert.equal(parts.select.innerHTML, before);
}).catch((error) => { console.error(error); process.exitCode = 1; });
