const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const { test } = require("node:test");

test("HTTP revision conflict preserves the editor draft and allows Save As recovery", async () => {
  const { createOKFAPI } = await import(pathToFileURL(path.join(__dirname, "okf_api.js")).href);
  const { saveEditorProfile } = await import(pathToFileURL(path.join(__dirname, "okf_profile_save.js")).href);
  const draft = { profile_id: "project:shared", name: "Unsaved work", extension: { retained: true } };
  const state = {
    editorProfile: draft, profileID: draft.profile_id, bundleID: ".okf",
    configurationRevision: "r1", layoutOverride: { algorithm: "mrtree" }
  };
  let closed = 0;
  let selected = 0;
  let rendered = 0;
  const elements = {
    editorJSON: { value: JSON.stringify(draft) }, editorStatus: { textContent: "" },
    editor: { close: () => closed++ }, profile: { value: draft.profile_id }
  };
  const requests = [];
  const respond = (status, payload) => ({ ok: status < 400, status, json: async () => payload });
  const api = createOKFAPI(async (url, options) => {
    requests.push({ url, options });
    if (options.method === "PUT") {
      assert.equal(options.headers["If-Match"], "r1");
      assert.deepEqual(JSON.parse(options.body), draft);
      return respond(409, { error: {
        code: "okf_revision_conflict", message: "Project configuration changed since it was read",
        details: { expected_revision: "r1", actual_revision: "r2" }
      } });
    }
    if (options.method === "POST") {
      const body = JSON.parse(options.body);
      assert.deepEqual(body.profile, draft);
      assert.equal(body.new_profile_id, "recovered");
      assert.equal(body.expected_revision, "r2");
      assert.equal(body.operation_id, options.headers["Idempotency-Key"]);
      return respond(201, { data: { revision: "r3" } });
    }
    if (url.endsWith("/catalog")) return respond(200, { data: { diagnostics: [], bundles: [] } });
    return respond(200, { data: { configuration_revision: "r3", profiles: [] } });
  });
  const editor = {
    read: () => structuredClone(draft), validate: async (value) => value,
    renderSelectors: () => rendered++
  };
  const services = { selectProfile: async () => selected++ };
  await saveEditorProfile(state, api, elements, services, false, editor);
  assert.equal(requests.length, 1, "a conflict must not trigger an automatic overwrite or refresh");
  assert.equal(state.editorProfile, draft);
  assert.equal(elements.editorJSON.value, JSON.stringify(draft));
  assert.deepEqual(state.layoutOverride, { algorithm: "mrtree" });
  assert.equal(state.configurationRevision, "r1", "conflict metadata must not silently authorize a stale overwrite");
  assert.equal(state.profileID, "project:shared");
  assert.equal(state.editorSavePending, false);
  assert.equal(closed + selected + rendered, 0);
  assert.match(elements.editorStatus.textContent, /configuration changed/);
  assert.match(elements.editorStatus.textContent, /Copy Advanced JSON before choosing Cancel/);
  assert.match(elements.editorStatus.textContent, /Refresh, reopen Edit profile, and merge/);

  // Simulate an explicit refresh of revision before the user chooses Save As.
  // This tests the save boundary, not the browser's refresh controls.
  state.configurationRevision = "r2";
  state.editorProfile.profile_id = "recovered";
  await saveEditorProfile(state, api, elements, services, true, editor);
  assert.equal(requests.length, 4);
  assert.equal(state.profileID, "project:recovered");
  assert.equal(state.configurationRevision, "r3");
  assert.equal(state.editorProfile, null);
  assert.equal(closed, 1);
  assert.equal(selected, 1);
  assert.equal(rendered, 1);
});
