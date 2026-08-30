const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "inspection.js"), "utf8").replace(/^import .*?;\r?\n/gm, "") + `
function escapeHTML(value) { return String(value == null ? "" : value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;"); }
function qualityAffectedOnly(context) { return Boolean(context.state.qualityAffectedOnly); }
function affectedFileIDs(context) { return context.state.affectedFileIDs || []; }
`;

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (inspection) {
  const route = inspection.parseInspectionRoute("?view=inspection&kind=node&id=opaque%2Fnode%3F1&scope=scope-go&path=internal&path=analyzers&section=files");
  assert.deepEqual(route, {
    kind: "node",
    id: "opaque/node?1",
    scope: "scope-go",
    path: ["internal", "analyzers"],
    referenceVisibility: "",
    section: "files"
  });
  const serialized = inspection.serializeInspectionRoute(route);
  assert.equal(serialized, "/?view=inspection&kind=node&id=opaque%2Fnode%3F1&scope=scope-go&path=internal&path=analyzers&section=files");

  const graph = inspection.parseGraphRoute("?scope=scope-go&path=internal&selected_kind=node&selected_id=opaque%2Fnode%3F1");
  assert.deepEqual(graph, {
    path: ["internal"],
    scope: "scope-go",
    referenceVisibility: "",
    selected: { kind: "node", id: "opaque/node?1" }
  });
  assert.equal(inspection.serializeGraphRoute(graph), "/?scope=scope-go&path=internal&selected_kind=node&selected_id=opaque%2Fnode%3F1");

  assert.equal(inspection.formatReadableLocation({ line: 13, column: 2 }, { line: 48, column: 1 }, "internal/main.go"), "internal/main.go · Lines 13–48");
  assert.equal(inspection.formatReadableLocation({ line: 13, column: 2 }, { line: 13, column: 9 }, "internal/main.go"), "internal/main.go · Line 13");
  assert.equal(inspection.formatReadableLocation(null, null, "internal/main.go"), "internal/main.go · File provenance only");
  assert.equal(inspection.inspectionState("unsupported").label, "Unsupported");
  assert.equal(inspection.inspectionState("partial").className, "warning");
  assert.equal(inspection.inspectionScopeStatus({ scope_status: "aggregate", aggregate_status: "partial", status: "complete" }), "partial");
  assert.equal(inspection.inspectionScopeStatus({ scope_status: "complete", aggregate_status: "partial", status: "complete" }), "complete");
  assert.equal(inspection.inspectionScopeStatus({ status: "complete" }), "complete");
  assert.equal(inspection.sourceExcerptAvailable({ sourceEnabled: true }), true);
  assert.equal(inspection.sourceExcerptAvailable({ sourceEnabled: false, embeddedExport: {} }), false);
	const affectedQuery = inspection.sourceQuery({ state: { activeScope: "scope-go", qualityAffectedOnly: true, affectedFileIDs: ["file-a", "file-b"] } }, { module_ids: ["module-a"] }, 25, "", "files");
	assert.deepEqual(affectedQuery.getAll("file_id"), ["file-a", "file-b"]);
	assert.deepEqual(affectedQuery.getAll("module_id"), [], "module selectors would union unrelated files into the report-backed filter");
	const regularQuery = inspection.sourceQuery({ state: { activeScope: "scope-go", qualityAffectedOnly: false } }, { module_ids: ["module-a"] }, 25, "", "files");
	assert.deepEqual(regularQuery.getAll("module_id"), ["module-a"]);
	const mergedPage = inspection.mergeCollectionPage(
		{ items: [{ id: "first" }], coverage: [{ status: "partial" }] },
		{ items: [{ id: "second" }], total: 2, snapshot_id: "snapshot", scope_id: "scope" },
		true,
		"query"
	);
	assert.deepEqual(mergedPage.items.map(function (item) { return item.id; }), ["first", "second"]);
	assert.equal(mergedPage.coverage[0].status, "partial");

  assert.deepEqual(inspection.symbolKindOptions([
    { language_kind: "go:struct" },
    { language_kind: "go:function" },
    { language_kind: "go:struct" },
    { language_kind: "" }
  ], "go:function"), [
    { value: "", label: "All kinds" },
    { value: "go:function", label: "go:function" },
    { value: "go:struct", label: "go:struct" }
  ]);

  const emptySymbols = inspection.renderSymbols({
    sourceEnabled: false,
    embeddedExport: null,
    state: {
      inspectionFilters: { symbols: "does-not-exist" },
      inspectionSymbolKind: "go:struct",
      inspectionSymbolKinds: ["go:function", "go:struct"],
      inspectionDocumentation: {}
    }
  }, { items: [], total: 0, coverage: [] });
  assert.match(emptySymbols, /data-inspection-filter="symbols"/);
  assert.match(emptySymbols, /data-inspection-kind-filter="symbols"/);
  assert.match(emptySymbols, /inspection-filter select-control inspection-kind-filter/);
  assert.match(emptySymbols, /does-not-exist/);
	const populatedSymbols = inspection.renderSymbols({
		sourceEnabled: false,
		embeddedExport: null,
		state: { inspectionFilters: { symbols: "" }, inspectionSymbolKind: "", inspectionSymbolKinds: [], inspectionDocumentation: {}, source: null, inspectionFilePaths: {} }
	}, {
		items: [{ id: "symbol", name: "Main", category: "callable", language_kind: "go:function", visibility: { classification: "public" }, locations: [] }],
		total: 1,
		coverage: [{ capability: "source:files", status: "unsupported" }, { capability: "source:declarations", status: "observed" }]
	});
	assert.match(populatedSymbols, />Available</);
	assert.doesNotMatch(populatedSymbols, />Unsupported</);

  const evidenceMarkup = inspection.renderEvidence({
    sourceEnabled: false,
    embeddedExport: null,
    state: {
      scene: {
        evidence_links: [
          { id: "file-a", kind: "file", path: "same.go" },
          { id: "file-b", kind: "file", path: "same.go" },
          { id: "import-a", kind: "import", symbol: "example.com/dep", path: "same.go", start: { line: 4, column: 2 }, end: { line: 4, column: 18 } }
        ]
      },
      source: null
    }
  }, { label: "internal", evidence_ids: ["file-a", "file-b", "import-a"] });
  assert.match(evidenceMarkup, /File provenance/);
  assert.match(evidenceMarkup, /Module\/file association/);
  assert.match(evidenceMarkup, /do not identify a line or prove a separate relationship/);
  assert.equal((evidenceMarkup.match(/same\.go/g) || []).length, 2, "file provenance should be deduplicated while retaining the located reference");
  assert.match(evidenceMarkup, /Reported source locations/);
  assert.match(evidenceMarkup, /Import observation · reported source location/);
  assert.match(evidenceMarkup, /3 source reference\(s\) reported/);

  const focus = inspection.inspectionFilterFocusState({
    dataset: { inspectionFilter: "symbols" },
    selectionStart: 2,
    selectionEnd: 5
  });
  let focused = false;
  let selection = null;
  const replacement = {
    focus: function () { focused = true; },
    setSelectionRange: function (start, end) { selection = [start, end]; }
  };
  inspection.restoreInspectionFilterFocus({
    querySelector: function (selector) {
      assert.equal(selector, '[data-inspection-filter="symbols"]');
      return replacement;
    }
  }, focus);
  assert.equal(focused, true);
  assert.deepEqual(selection, [2, 5]);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
