const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

function deferred() {
  let resolve;
  const promise = new Promise((done) => { resolve = done; });
  return { promise, resolve };
}

import(pathToFileURL(path.join(__dirname, "okf_profile_save.js")).href).then(async ({ saveEditorProfile }) => {
  function fixture() {
    const state = { editorProfile: { profile_id: "project:one" }, profileID: "project:one", configurationRevision: "r1", layoutOverride: {} };
    const calls = { writes: 0, closed: 0, selected: 0 };
    const elements = { editorJSON: { value: "{}" }, editorStatus: { textContent: "" }, editor: { close: () => calls.closed++ }, profile: {} };
    const api = { saveProfile: async (_value, revision) => { assert.equal(revision, "r1"); calls.writes++; return { revision: "r2" }; }, getProfiles: async () => [], getCatalog: async () => ({ diagnostics: [], bundles: [] }) };
    const editor = { read: () => structuredClone(state.editorProfile), validate: async (value) => value, renderSelectors: () => {} };
    const services = { selectProfile: async () => { calls.selected++; } };
    return { state, calls, elements, api, editor, run: (saveAs = false) => saveEditorProfile(state, api, elements, services, saveAs, editor) };
  }
  const closed = fixture();
  const validation = deferred();
  closed.editor.validate = () => validation.promise;
  const cancelled = closed.run();
  closed.state.editorProfile = null;
  validation.resolve({ profile_id: "project:one" });
  await cancelled;
  assert.equal(closed.calls.writes, 0, "closing during validation must cancel the write");
  assert.equal(closed.state.editorSavePending, false);

  const edited = fixture();
  const write = deferred();
  edited.api.saveProfile = () => { edited.calls.writes++; return write.promise; };
  const saving = edited.run();
  await Promise.resolve();
  await edited.run();
  assert.equal(edited.calls.writes, 1, "double Save must not submit concurrent writes");
  edited.elements.editorJSON.value = '{"unfinished":';
  write.resolve({ revision: "r2" });
  await saving;
  assert.equal(edited.calls.closed, 0);
  assert.equal(edited.calls.selected, 0);
  assert.equal(edited.state.configurationRevision, "r2");
  assert.match(edited.elements.editorStatus.textContent, /newer edits remain unsaved/);

  const reopened = fixture();
  const profiles = deferred();
  reopened.api.getProfiles = () => profiles.promise;
  const oldSave = reopened.run();
  await Promise.resolve();
  await Promise.resolve();
  const newDraft = { profile_id: "project:two" };
  reopened.state.editorProfile = newDraft;
  profiles.resolve([]);
  await oldSave;
  assert.equal(reopened.state.editorProfile, newDraft);
  assert.equal(reopened.calls.closed, 0);

  const success = fixture();
  await success.run();
  assert.equal(success.calls.closed, 1);
  assert.equal(success.calls.selected, 1);
  assert.equal(success.state.editorProfile, null);
  assert.equal(success.state.configurationRevision, "r2");
  const originalNow = Date.now;
  const operationIDs = [];
  try {
    Date.now = () => 123456789;
    for (let tab = 0; tab < 2; tab++) {
      const concurrent = fixture();
      concurrent.api.saveProfile = async (_value, _revision, operationID) => {
        operationIDs.push(operationID);
        return { revision: "r2" };
      };
      await concurrent.run();
    }
    assert.equal(operationIDs.length, 2);
    assert.notEqual(operationIDs[0], operationIDs[1], "independent editors must not share an operation ID when the clock is identical");
    for (const id of operationIDs) {
      assert.match(id, /^okf-profile-[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    }
  } finally { Date.now = originalNow; }
  const conflict = fixture();
  globalThis.prompt = () => { throw new Error("prompt() is not supported."); };
  const unchangedID = fixture();
  await unchangedID.run(true);
  assert.match(unchangedID.elements.editorStatus.textContent, /Enter a new project Profile ID/);
  assert.equal(unchangedID.calls.closed, 0);
  conflict.state.editorProfile.profile_id = "copy";
  conflict.api.saveProfileAs = async () => { throw new Error("revision conflict"); };
  await conflict.run(true);
  assert.equal(conflict.state.profileID, "project:one");
  assert.equal(conflict.calls.closed, 0);
  assert.equal(conflict.elements.editorStatus.textContent, "revision conflict");
  const copied = fixture();
  copied.state.editorProfile.profile_id = "copy";
  copied.api.saveProfileAs = async (_value, newID, sourceID, revision) => {
    assert.deepEqual([newID, sourceID, revision], ["copy", "project:one", "r1"]);
    return { revision: "r2" };
  };
  await copied.run(true);
  assert.equal(copied.state.profileID, "project:copy");
  assert.equal(copied.calls.closed, 1);
  delete globalThis.prompt;
  const offline = fixture();
  offline.api.getProfiles = async () => { throw new Error("offline"); };
  await offline.run();
  assert.match(offline.elements.editorStatus.textContent, /was saved.*offline/);
  assert.equal(offline.state.configurationRevision, "r2");
  const repaired = fixture();
  const staleCatalog = { diagnostics: [{ code: "okf_profile_not_found" }], bundles: [] };
  const refreshedCatalog = { diagnostics: [], bundles: [{ bundle_id: "invalid/.okf", selectable: false, diagnostics: [{ code: "okf_bundle_invalid" }] }] };
  repaired.state.catalog = staleCatalog;
  repaired.api.getCatalog = async () => refreshedCatalog;
  await repaired.run();
  assert.equal(repaired.state.catalog, refreshedCatalog, "successful Save replaces stale warnings without dropping source diagnostics");

  const delayed = fixture();
  const catalogRead = deferred();
  delayed.state.catalog = staleCatalog;
  delayed.api.getCatalog = () => catalogRead.promise;
  const delayedSave = delayed.run();
  await Promise.resolve();
  await Promise.resolve();
  delayed.state.editorProfile = { profile_id: "project:new-editor" };
  catalogRead.resolve(refreshedCatalog);
  await delayedSave;
  assert.equal(delayed.state.catalog, staleCatalog, "late catalog reads must not publish into a different editor");
  assert.equal(delayed.calls.selected, 0);

  const failedRead = fixture();
  const priorProfiles = { profiles: [] };
  failedRead.state.catalog = staleCatalog;
  failedRead.state.profiles = priorProfiles;
  failedRead.api.getCatalog = async () => { throw new Error("catalog offline"); };
  await failedRead.run();
  assert.equal(failedRead.state.catalog, staleCatalog);
  assert.equal(failedRead.state.profiles, priorProfiles, "partial metadata refresh must not publish half the catalogs");
  assert.match(failedRead.elements.editorStatus.textContent, /was saved.*catalog offline/);
  assert.equal(failedRead.calls.closed, 0);
}).catch((error) => { console.error(error); process.exitCode = 1; });
