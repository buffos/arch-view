const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "details.js"), "utf8").replace(/^import .*?;\r?\n/, "");

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (details) {
  const snapshot = { coverage: [{ capability: "source:declarations", status: "observed" }] };
  assert.equal(details.sourceFactStatus(null, []), "unavailable");
  assert.equal(details.sourceFactStatus({ coverage: [{ status: "unsupported" }] }, []), "unsupported");
  assert.equal(details.sourceFactStatus({ coverage: [{ status: "unknown" }] }, []), "unknown");
  assert.equal(details.sourceFactStatus({ coverage: [{ status: "partial" }] }, []), "partial");
  assert.equal(details.sourceFactStatus(snapshot, []), "missing");
  assert.equal(details.sourceFactStatus(snapshot, [{ analysis_status: "complete" }]), "populated");

  const file = { id: "file-a", path: "same.go" };
  const unrelated = { id: "file-b", path: "same.go" };
  const contained = details.sourceFactModuleData({
    files: [file, unrelated],
    symbols: [],
    documentation: [],
    relations: [{ category: "contains", from_ref: { kind: "module", id: "module-a" }, to_ref: { kind: "file", id: "file-a" } }]
  }, { module_ids: ["module-a"] });
  assert.deepEqual(contained.files.map(function (item) { return item.id; }), ["file-a"]);

  const summary = details.compactSummaryFields({
    id: "opaque-node-id",
    label: "internal",
    kind: "group",
    hierarchy_path: ["internal"],
    counts: { module_count: 3, relationship_count: 4, evidence_count: 2 },
    cycle_state: "none",
    diagnostic_state: "none",
    confidence_state: "high",
    identity_state: "stable"
  });
  assert.equal(summary.label, "internal");
  assert.deepEqual(summary.counts, { modules: 3, relationships: 4, evidence: 2 });
  assert.equal(Object.prototype.hasOwnProperty.call(summary, "id"), false, "compact summary must not expose stable ids");
  assert.equal(Object.prototype.hasOwnProperty.call(summary, "files"), false, "compact summary must not expose source lists");
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
