const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

const moduleURL = pathToFileURL(path.join(__dirname, "okf_state.js")).href;

import(moduleURL).then(async function (stateModule) {
  const state = stateModule.createOKFState("session-1");
  assert.equal(state.sessionID, "session-1");
  assert.equal(state.depth, 2);
  assert.equal(state.full, false);
  assert.equal(state.configurationRevision, "");
  assert.equal(state.layoutProfile.algorithm, "layered");
  assert.deepEqual(state.layoutProfile.features, ["junctions", "ports"]);
  assert.equal(state.layoutOverride, null);

  const firstRequest = stateModule.beginRequest(state);
  const secondRequest = stateModule.beginRequest(state);
  assert.equal(stateModule.isCurrent(state, firstRequest), false);
  assert.equal(stateModule.isCurrent(state, secondRequest), true);

  const catalog = { profiles: [{ profile_id: "builtin:neutral" }, { profile_id: "project:review" }] };
  assert.equal(stateModule.profileByID(catalog, "project:review").profile_id, "project:review");
  assert.equal(stateModule.profileByID(catalog, "project:missing"), null);
  const invalidDiagnostic = Object.freeze({ code: "okf_frontmatter_invalid", path: "target.md", message: "Invalid YAML", recovery: "Repair frontmatter." });
  const report = stateModule.catalogDiagnostics({
    diagnostics: [{ code: "catalog_warning" }],
    bundles: [
      { bundle_id: "broken/.okf", selectable: false, diagnostics: [invalidDiagnostic] },
      { bundle_id: "healthy/.okf", selectable: true, diagnostics: [{ code: "projection_warning" }] }
    ]
  });
  assert.deepEqual(report.map((value) => value.code), ["catalog_warning", "okf_frontmatter_invalid"]);
  assert.equal(report[1].bundle_id, "broken/.okf");
  assert.equal(report[1].path, "target.md");
  assert.equal(report[1].recovery, "Repair frontmatter.");
  assert.equal(invalidDiagnostic.bundle_id, undefined, "report must not mutate catalog records");
  assert.deepEqual(stateModule.catalogDiagnostics(null), []);
  let releaseProfiles;
  let profilesStarted;
  const started = new Promise((resolve) => { profilesStarted = resolve; });
  const pendingProfiles = new Promise((resolve) => { releaseProfiles = resolve; });
  const oldRequest = stateModule.beginRequest(state);
  const oldLoad = stateModule.refreshCatalogState(state, {
    refreshCatalog: async () => ({ id: "old" }),
    getProfiles: () => { profilesStarted(); return pendingProfiles; },
    getLayoutOptions: async () => ({ id: "old" })
  }, oldRequest);
  await started;
  const newRequest = stateModule.beginRequest(state);
  await stateModule.refreshCatalogState(state, {
    refreshCatalog: async () => ({ id: "new" }),
    getProfiles: async () => ({ id: "new" }),
    getLayoutOptions: async () => ({ id: "new" })
  }, newRequest);
  releaseProfiles({ id: "old" });
  assert.equal(await oldLoad, null);
  assert.equal(state.catalog.id, "new");
  assert.equal(state.profiles.id, "new");
  assert.equal(state.layoutCatalog.id, "new");
  const failedRequest = stateModule.beginRequest(state);
  await assert.rejects(stateModule.refreshCatalogState(state, {
    refreshCatalog: async () => ({ id: "partial" }),
    getProfiles: async () => { throw new Error("profiles unavailable"); },
    getLayoutOptions: async () => ({ id: "partial" })
  }, failedRequest), /profiles unavailable/);
  assert.equal(state.catalog.id, "new", "failed refresh must not publish a partial catalog");
  assert.equal(state.profiles.id, "new");
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
