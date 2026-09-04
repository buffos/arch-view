const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "okf_profile_lifecycle.js")).href).then(async ({ deleteEditorProfile }) => {
  function setup(confirm) {
    const state = { editorProfile: { name: "Draft" }, profileID: "project:one", bundleID: ".okf", configurationRevision: "r1" };
    const elements = { editorJSON: { value: "{}" }, editorStatus: { textContent: "" } };
    const calls = [];
    const api = {
      deleteProfile: async (...args) => { calls.push(args.slice(0, 4)); return { revision: "r2" }; },
      getProfiles: async () => ({ profiles: [] }),
      getCatalog: async () => ({ diagnostics: [], bundles: [] })
    };
    const editor = { confirm, renderSelectors: () => {}, close: () => { state.editorProfile = null; } };
    const services = { selectProfile: async () => {} };
    return { state, elements, calls, run: () => deleteEditorProfile(state, api, elements, services, editor) };
  }
  for (const confirmation of [async () => false, async () => { throw new Error("Dialog unavailable"); }]) {
    const test = setup(confirmation);
    await test.run();
    assert.deepEqual(test.calls, []);
    assert.equal(test.state.profileID, "project:one");
    assert.equal(test.state.editorProfile.name, "Draft");
    assert.equal(test.state.editorConfirmationPending, false);
  }
  for (const supersede of [
    (state) => { state.editorProfile = null; },
    (state) => { state.editorProfile.name = "New edit"; },
    (state) => { state.profileID = "project:other"; },
    (state) => { state.bundleID = "second/.okf"; },
    (state, elements) => { elements.editorJSON.value = "new input"; },
    (state) => { state.configurationRevision = "r3"; }
  ]) {
    let finish;
    let prompts = 0;
    const test = setup(() => { prompts++; return new Promise((resolve) => { finish = resolve; }); });
    const waiting = test.run();
    await test.run();
    assert.equal(prompts, 1);
    supersede(test.state, test.elements);
    finish(true);
    await waiting;
    assert.deepEqual(test.calls, []);
  }
  const success = setup(async (question) => {
    assert.match(question.message, /project:one.*Neutral.*cannot be undone/);
    return true;
  });
  await success.run();
  assert.deepEqual(success.calls, [["project:one", "builtin:neutral", true, "r1"]]);
  assert.equal(success.state.profileID, "builtin:neutral");
  assert.equal(success.state.configurationRevision, "r2");
  assert.equal(success.state.editorProfile, null);
}).catch((error) => { console.error(error); process.exitCode = 1; });
