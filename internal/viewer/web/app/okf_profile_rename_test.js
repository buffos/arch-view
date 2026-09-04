const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "okf_profile_lifecycle.js")).href).then(async ({ renameEditorProfile }) => {
  globalThis.prompt = () => { throw new Error("prompt() is not supported."); };
  const state = { profileID: "project:old", bundleID: ".okf", configurationRevision: "r1", editorProfile: { profile_id: "new", name: "New name" } };
  const elements = { editorJSON: { value: "{}" }, editorStatus: { textContent: "" } };
  let closed = 0, selected = 0, writes = 0;
  const api = {
    renameProfile: async (oldID, newID, name, revision) => {
      assert.deepEqual([oldID, newID, name, revision], ["project:old", "new", "New name", "r1"]);
      writes++;
      return { revision: "r2" };
    },
    getProfiles: async () => ({ profiles: [{ profile_id: "project:new" }] }),
    getCatalog: async () => ({ diagnostics: [], bundles: [] })
  };
  const editor = { read: () => state.editorProfile, renderSelectors: () => {}, close: () => { closed++; } };
  await renameEditorProfile(state, api, elements, { selectProfile: async () => { selected++; } }, editor);
  assert.equal(state.profileID, "project:new");
  assert.equal(state.configurationRevision, "r2");
  assert.deepEqual([writes, closed, selected], [1, 1, 1]);
  state.editorProfile.profile_id = "builtin:neutral";
  await renameEditorProfile(state, api, elements, {}, editor);
  assert.equal(writes, 1);
  assert.match(elements.editorStatus.textContent, /Enter a project Profile ID/);
  state.editorProfile = { profile_id: "occupied", name: "Keep my draft" };
  api.renameProfile = async () => {
    throw Object.assign(new Error("Configuration changed."), { code: "okf_revision_conflict" });
  };
  await renameEditorProfile(state, api, elements, {}, editor);
  assert.equal(state.profileID, "project:new", "failed rename retains selected source identity");
  assert.equal(state.configurationRevision, "r2");
  assert.equal(state.editorProfile.name, "Keep my draft");
  assert.equal(closed, 1, "failed rename leaves the editor open");
  assert.match(elements.editorStatus.textContent, /Your draft has not been saved/);
  assert.equal(state.editorSavePending, false);
  delete globalThis.prompt;
}).catch((error) => { console.error(error); process.exitCode = 1; });
