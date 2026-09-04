const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "okf_profile_lifecycle.js")).href).then(async ({ changeProfileConfiguration }) => {
  function setup() {
    const state = { editorProfile: { name: "Draft" }, profileID: "project:one", bundleID: "bundle", configurationRevision: "r1" };
    const elements = { editorJSON: { value: "{}" }, editorStatus: { textContent: "" } };
    let finish;
    const waiting = new Promise((resolve) => { finish = resolve; });
    let published = 0;
    let writes = 0;
    const operation = {
      write: ({ profileID, bundleID, revision }) => {
        assert.deepEqual([profileID, bundleID, revision], ["project:one", "bundle", "r1"]);
        writes++;
        return waiting;
      },
      publish: () => { published++; }
    };
    return { state, elements, operation, finish, run: () => changeProfileConfiguration(state, elements, operation), counts: () => ({ published, writes }) };
  }
  for (const supersede of [
    (state) => { state.editorProfile = null; },
    (state) => { state.editorProfile = { name: "Another draft" }; },
    (state) => { state.editorProfile.name = "New edit"; },
    (state) => { state.profileID = "project:two"; },
    (state) => { state.bundleID = "another"; }
  ]) {
    const test = setup();
    const running = test.run();
    await test.run();
    supersede(test.state);
    test.state.configurationRevision = "r3";
    test.finish({ revision: "r2" });
    await running;
    assert.deepEqual(test.counts(), { published: 0, writes: 1 });
    assert.equal(test.state.configurationRevision, "r3", "late write must not regress known revision");
    assert.equal(test.state.editorSavePending, false);
  }
  const delayed = setup();
  let finishPrepare;
  delayed.operation.prepare = () => new Promise((resolve) => { finishPrepare = resolve; });
  const preparing = delayed.run();
  delayed.finish({ revision: "r2" });
  await Promise.resolve();
  delayed.elements.editorJSON.value = "{";
  finishPrepare([]);
  await preparing;
  assert.equal(delayed.counts().published, 0);

  const success = setup();
  const running = success.run();
  success.finish({ revision: "r2" });
  await running;
  assert.deepEqual(success.counts(), { published: 1, writes: 1 });
  assert.equal(success.state.configurationRevision, "r2");

  const failure = setup();
  failure.operation.write = async () => { throw new Error("revision conflict"); };
  await failure.run();
  assert.equal(failure.elements.editorStatus.textContent, "revision conflict");
  assert.equal(failure.state.editorProfile.name, "Draft");
  assert.equal(failure.state.editorSavePending, false);
  const refreshFailure = setup();
  refreshFailure.operation.prepare = async () => { throw new Error("offline"); };
  const refresh = refreshFailure.run();
  refreshFailure.finish({ revision: "r2" });
  await refresh;
  assert.match(refreshFailure.elements.editorStatus.textContent, /was saved.*offline/);
  assert.equal(refreshFailure.state.configurationRevision, "r2");
}).catch((error) => { console.error(error); process.exitCode = 1; });
