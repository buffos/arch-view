const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { createRequire } = require("node:module");
const { pathToFileURL } = require("node:url");

const requireFromTest = createRequire(__filename);
const ELK = requireFromTest("../vendor/elk.bundled.js");
const moduleURL = (name) => pathToFileURL(path.join(__dirname, name)).href;
const features = JSON.parse(fs.readFileSync(path.join(__dirname, "../../layout/features.json"), "utf8"));
const catalog = {
  features,
  options: [
    { id: "org.eclipse.elk.edgeRouting", editable: true, renderer_support: "supported", targets: ["PARENTS"], supported_targets: ["PARENTS"], algorithms: ["layered"] },
    { id: "org.eclipse.elk.layered.edgeRouting.splines.mode", editable: true, renderer_support: "supported", targets: ["PARENTS"], supported_targets: ["PARENTS"], algorithms: ["layered"] }
  ]
};

const scene = {
  visible_nodes: [{ id: "a" }, { id: "b" }, { id: "c" }],
  visible_relationships: [
    { id: "e1", from_visible_id: "a", to_visible_id: "b", count: 3 },
    { id: "e2", from_visible_id: "a", to_visible_id: "c", count: 2 }
  ]
};

test("SC-AER-002/003 pinned ELK yields count-label bounds and shared junctions", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  const profile = { algorithm: "layered", features: ["edge_labels", "junctions"], options: { "org.eclipse.elk.edgeRouting": "ORTHOGONAL" } };
  const source = { kind: "architecture", id: "model-1", revision: "revision-1", navigation_scope: { hierarchy_path: [] } };
  const result = await runFeatureLayout(scene, profile, catalog, null, ELK, source);
  const geometry = normalizeGeometrySnapshot(scene, result, profile, source);
  assert.equal(geometry.schema_version, "arch-view.geometry/v1");
  assert.deepEqual(geometry.source, source);
  assert.equal(geometry.edges.length, 2);
  assert.equal(geometry.edges[0].labels[0].text, "3");
  assert.ok(Object.values(geometry.edges[0].labels[0].bounds).every(Number.isFinite));
  assert.equal(geometry.junctions.length, 1);
  assert.deepEqual(geometry.junctions[0].incident_geometry_edge_ids, ["e1", "e2"]);
});

test("SC-AER-006 pinned ELK preserves connected cubic sections", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  const profile = { algorithm: "layered", features: ["spline_refinement"], options: {
    "org.eclipse.elk.edgeRouting": "SPLINES",
    "org.eclipse.elk.layered.edgeRouting.splines.mode": "SLOPPY"
  } };
  const source = { kind: "okf", id: ".okf", revision: "projection-1", navigation_scope: { focus_root: "" } };
  const result = await runFeatureLayout(scene, profile, catalog, null, ELK, source);
  const geometry = normalizeGeometrySnapshot(scene, result, profile, source);
  assert.equal(geometry.source.kind, "okf");
  geometry.edges.forEach((edge) => {
    assert.equal(edge.route.kind, "spline");
    edge.route.sections.forEach((section) => section.segments.forEach((segment) => {
      assert.equal(segment.kind, "cubic");
      assert.ok([segment.control1, segment.control2, segment.to].flatMap(Object.values).every(Number.isFinite));
    }));
  });
});

test("SC-AER-007 malformed spline output falls back for its edge", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  class InvalidEngine {
    async layout(graph) {
      return { ...graph, width: 400, height: 200,
        children: graph.children.map((node, index) => ({ ...node, x: index * 160, y: 20 })),
        edges: graph.edges.map((edge) => ({ ...edge, sections: [{ startPoint: { x: 100, y: 40 }, bendPoints: [{ x: NaN, y: 40 }], endPoint: { x: 160, y: 40 } }] })) };
    }
  }
  const profile = { algorithm: "layered", features: ["spline_refinement"], options: { "org.eclipse.elk.edgeRouting": "SPLINES" } };
  const result = await runFeatureLayout(scene, profile, catalog, null, InvalidEngine, { kind: "architecture", id: "m", revision: "r", navigation_scope: {} });
  const geometry = normalizeGeometrySnapshot(scene, result, profile, result.geometrySource);
  assert.ok(geometry.edges.every((edge) => edge.route.kind === "orthogonal"));
  assert.ok(geometry.diagnostics.some((item) => item.code === "geometry_route_invalid"));
});

test("SC-AER-010 relationships without an explicit count get no invented label", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const noCount = { ...scene, visible_relationships: scene.visible_relationships.map(({ count, ...relationship }) => relationship) };
  const profile = { algorithm: "layered", features: ["edge_labels"], options: {} };
  const result = await runFeatureLayout(noCount, profile, catalog, null, ELK, { kind: "okf", id: "bundle", revision: "r", navigation_scope: {} });
  assert.ok(result.edges.every((edge) => !edge.labels));
});

test("SC-AER-002/003 shared presentation emits labels and non-semantic junction markers", async () => {
  const { geometryLabelMarkup, geometryJunctionMarkup } = await import(moduleURL("edge_presentation.js"));
  const labelMarkup = geometryLabelMarkup([{ id: "label::e1::count", text: "3", visible: true, position: { x: 12, y: 34 } }]);
  const junctionMarkup = geometryJunctionMarkup([{ id: "junction::1:2", position: { x: 1, y: 2 } }]);
  assert.match(labelMarkup, /data-edge-label-id="label::e1::count"/);
  assert.match(labelMarkup, />3<\/text>/);
  assert.match(junctionMarkup, /<circle class="edge-junction"/);
  assert.doesNotMatch(junctionMarkup, /marker-|role=/);
});
