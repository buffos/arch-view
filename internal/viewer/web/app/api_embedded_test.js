const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "api.js"), "utf8");

global.window = { location: { href: "http://127.0.0.1/" } };

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(async function (apiModule) {
  const context = {
    modelID: "model-1",
    embeddedExport: {
      model: {
        model_id: "model-1",
        source_index: {
          snapshots: [
            {
              snapshot_id: "snapshot-1",
              scope_context: { scope_id: "scope-1" },
              coverage: [{ capability: "source:files", status: "observed" }],
              files: [
                { id: "file-a", path: "same.go" },
                { id: "file-b", path: "same.go" }
              ],
              symbols: [
                { id: "symbol-a", name: "Main", language_kind: "go:function", locations: [{ span: { file_id: "file-a" } }], documentation_ids: ["doc-a"] },
                { id: "symbol-b", name: "Thing", language_kind: "go:struct", locations: [{ span: { file_id: "file-b" } }], documentation_ids: ["doc-b"] }
              ],
              documentation: [
				{ id: "doc-a", subject_ref: { kind: "symbol", id: "symbol-a" }, status: "present", completeness: "complete", normalized_text: "x".repeat(520), raw_text: "sensitive full documentation" },
                { id: "doc-b", subject_ref: { kind: "symbol", id: "symbol-b" }, status: "present", normalized_text: "Thing docs", raw_text: "Thing docs" }
              ],
              relations: [
                { category: "contains", from_ref: { kind: "module", id: "module-a" }, to_ref: { kind: "file", id: "file-a" } },
                { category: "contains", from_ref: { kind: "module", id: "module-b" }, to_ref: { kind: "file", id: "file-b" } },
                { category: "declares", from_ref: { kind: "file", id: "file-a" }, to_ref: { kind: "symbol", id: "symbol-a" } },
                { category: "declares", from_ref: { kind: "file", id: "file-b" }, to_ref: { kind: "symbol", id: "symbol-b" } }
              ]
            }
          ]
        }
      }
    }
  };
  const api = apiModule.createAPI(context);
  const page = await api.getJSON("/v1/models/model-1/source-index/files?scope=scope-1&module_id=module-a&module_id=module-a&limit=1");
  assert.equal(page.total, 1);
  assert.equal(page.items[0].id, "file-a");
  assert.equal(page.coverage[0].status, "observed");

  const kindPage = await api.getJSON("/v1/models/model-1/source-index/symbols?scope=scope-1&language_kind=go:struct&limit=25");
  assert.equal(kindPage.total, 1);
  assert.equal(kindPage.items[0].id, "symbol-b");

  const documentation = await api.getJSON("/v1/models/model-1/source-index/documentation?scope=scope-1&subject_id=symbol-a&subject_id=symbol-b&limit=25");
  assert.equal(documentation.total, 2);
  assert.deepEqual(documentation.items.map(function (item) { return item.id; }), ["doc-a", "doc-b"]);
  assert.equal(documentation.items[0].raw_text, undefined, "embedded default documentation must omit raw text");
	assert.equal(Array.from(documentation.items[0].normalized_text).length, 512);
	assert.equal(documentation.items[0].completeness, "truncated");

	const evidence = await api.getJSON("/v1/models/model-1/source-index/evidence/symbol-a?scope=scope-1");
	assert.equal(evidence.documentation[0].raw_text, undefined, "embedded default evidence must omit raw text");
	assert.equal(Array.from(evidence.documentation[0].normalized_text).length, 512);

  await assert.rejects(
    api.getJSON("/v1/models/model-1/source-index/files?scope=missing-scope&module_id=module-a"),
    /source-facts scope is unavailable/
  );
	await assert.rejects(
		api.getJSON("/v1/models/model-1/source-index/files?scope=scope-1&limit=201"),
		/outside the allowed bound/
	);
	await assert.rejects(
		api.getJSON("/v1/models/model-1/source-index/files?scope=scope-1&cursor=invalid"),
		/cursor is malformed/
	);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
