const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { createRequire } = require("node:module");
const { pathToFileURL } = require("node:url");

const ELK = createRequire(__filename)("../vendor/elk.bundled.js");
const moduleURL = (name) => pathToFileURL(path.join(__dirname, name)).href;
const features = JSON.parse(fs.readFileSync(path.join(__dirname, "../../layout/features.json"), "utf8"));
const catalog = { features, options: [
  { id: "org.eclipse.elk.edgeRouting", editable: true, renderer_support: "supported", targets: ["PARENTS"], supported_targets: ["PARENTS"], algorithms: ["layered"], type: "ENUM" },
  { id: "org.eclipse.elk.portConstraints", editable: true, renderer_support: "supported", targets: ["NODES"], supported_targets: ["NODES"], algorithms: ["layered"], type: "ENUM" }
] };
const source = { kind: "okf", id: "bundle", revision: "r1", navigation_scope: { focus_root: "root" } };
const scene = {
  visible_nodes: [
    { id: "root", hierarchy_path: ["root"] },
    { id: "group", hierarchy_path: ["root", "group"] },
    { id: "leaf", hierarchy_path: ["root", "group", "leaf"] },
    { id: "peer", hierarchy_path: ["root", "peer"] }
  ],
  visible_relationships: [
    { id: "root-group", from_visible_id: "root", to_visible_id: "group", count: 1 },
    { id: "group-leaf", from_visible_id: "group", to_visible_id: "leaf", count: 1 },
    { id: "leaf-peer", from_visible_id: "leaf", to_visible_id: "peer", count: 2 }
  ]
};

function contains(parent, child) {
  return child.x >= parent.x && child.y >= parent.y
    && child.x + child.width <= parent.x + parent.width
    && child.y + child.height <= parent.y + parent.height;
}

test("SC-AER-005 pinned ELK produces validated nested visible hierarchy", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  const profile = { algorithm: "layered", features: ["compound"], options: {} };
  const output = await runFeatureLayout(scene, profile, catalog, null, ELK, source);
  const geometry = normalizeGeometrySnapshot(scene, output, profile, source);
  const nodes = new Map(geometry.nodes.map((node) => [node.id, node]));
  assert.deepEqual(geometry.provenance.features, ["compound"]);
  assert.equal(nodes.get("root").parent_id, null);
  assert.deepEqual(nodes.get("root").children_ids, ["group", "peer"]);
  assert.equal(nodes.get("group").parent_id, "root");
  assert.deepEqual(nodes.get("group").children_ids, ["leaf"]);
  assert.equal(nodes.get("leaf").parent_id, "group");
  const containers = new Map(geometry.containers.map((container) => [container.owner_node_id, container]));
  assert.ok(geometry.containers.every((container) => !("semantic_node_id" in container)));
  assert.equal(containers.has("root"), false, "the canvas is the top-level container");
  assert.ok(contains(containers.get("group").bounds, nodes.get("group").bounds));
  assert.ok(contains(containers.get("group").bounds, nodes.get("leaf").bounds));
  assert.notDeepEqual(containers.get("group").bounds, nodes.get("group").bounds);
  assert.equal(geometry.edges.length, scene.visible_relationships.length);
  assert.ok(geometry.edges.every((edge) => edge.route.sections.length > 0));
});

test("SC-AER-005 hidden hierarchy segments are not invented", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  const sparse = {
    visible_nodes: [scene.visible_nodes[0], { id: "leaf", hierarchy_path: ["root", "hidden", "leaf"] }],
    visible_relationships: [{ id: "root-leaf", from_visible_id: "root", to_visible_id: "leaf" }]
  };
  const profile = { algorithm: "layered", features: ["compound"], options: {} };
  const output = await runFeatureLayout(sparse, profile, catalog, null, ELK, source);
  const geometry = normalizeGeometrySnapshot(sparse, output, profile, source);
  assert.deepEqual(geometry.nodes.map((node) => node.id).sort(), ["leaf", "root"]);
  assert.equal(geometry.containers.length, 0);
});

test("SC-AER-011 compound geometry composes with labels, ports, and splines", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  const profile = {
    algorithm: "layered",
    features: ["ports", "compound", "spline_refinement", "edge_labels"],
    options: { "org.eclipse.elk.edgeRouting": "SPLINES" }
  };
  const output = await runFeatureLayout(scene, profile, catalog, null, ELK, source);
  const geometry = normalizeGeometrySnapshot(scene, output, profile, source);
  assert.deepEqual(geometry.provenance.features, ["compound", "edge_labels", "spline_refinement", "ports"]);
  assert.ok(geometry.nodes.every((node) => node.ports.length === 2));
  assert.ok(geometry.edges.every((edge) => edge.source_port_id && edge.target_port_id));
  assert.ok(geometry.edges.some((edge) => edge.route.kind === "spline"));
  const degraded = geometry.edges.find((edge) => edge.route.kind === "orthogonal");
  assert.ok(degraded);
  const sourcePort = geometry.nodes.find((node) => node.id === degraded.source_node_id).ports.find((port) => port.id === degraded.source_port_id);
  const targetPort = geometry.nodes.find((node) => node.id === degraded.target_node_id).ports.find((port) => port.id === degraded.target_port_id);
  assert.deepEqual(degraded.route.sections[0].start, sourcePort.position);
  assert.deepEqual(degraded.route.sections[0].segments.at(-1).to, targetPort.position);
  assert.ok(geometry.edges.every((edge) => edge.labels.length === 1));
  assert.ok(geometry.diagnostics.every((item) => item.code === "geometry_route_invalid"));
});

test("SC-AER-007 malformed compound bounds fall back to flat geometry", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  class InvalidCompoundEngine {
    async layout(graph) {
      const output = await new ELK().layout(graph);
      const container = output.children.find((node) => node.id.startsWith("arch-view-container::"));
      container.children[0].x = container.width + 10;
      return output;
    }
  }
  const profile = { algorithm: "layered", features: ["compound"], options: {} };
  const output = await runFeatureLayout(scene, profile, catalog, null, InvalidCompoundEngine, source);
  const geometry = normalizeGeometrySnapshot(scene, output, profile, source);
  assert.ok(geometry.diagnostics.some((item) => item.code === "geometry_hierarchy_invalid"));
  assert.ok(geometry.nodes.every((node) => node.parent_id === null && node.children_ids.length === 0));
  assert.ok(geometry.edges.every((edge) => edge.route.kind === "orthogonal"));
});

test("SC-AER-010 container presentation and movement stay non-semantic", async () => {
  const { applyGeometryMove, geometryContainerBounds, geometryContainerMarkup, geometryMoveIDs, geometryMoveStart } = await import(moduleURL("container_presentation.js"));
  const snapshot = {
    nodes: [
      { id: "root", semantic_node_id: "root", bounds: { x: 12, y: 12, width: 190, height: 82 }, parent_id: null, children_ids: ["group"] },
      { id: "group", semantic_node_id: "group", bounds: { x: 32, y: 72, width: 190, height: 82 }, parent_id: "root", children_ids: ["leaf"] },
      { id: "leaf", semantic_node_id: "leaf", bounds: { x: 40, y: 120, width: 190, height: 82 }, parent_id: "group", children_ids: [] }
    ],
    containers: [
      { id: "arch-view-container::group", owner_node_id: "group", bounds: { x: 20, y: 60, width: 300, height: 180 } }
    ]
  };
  assert.deepEqual(geometryMoveIDs(snapshot, "group"), ["group", "leaf"]);
  const positions = Object.fromEntries(snapshot.nodes.map((node) => [node.id, { ...node.bounds }]));
  const manual = {};
  applyGeometryMove(geometryMoveStart(snapshot, "group", positions), positions, manual, 15, -5);
  assert.deepEqual(manual, { group: { x: 47, y: 67 }, leaf: { x: 55, y: 115 } });
  assert.deepEqual(positions.root, snapshot.nodes[0].bounds);
  const markup = geometryContainerMarkup(snapshot, Object.fromEntries(snapshot.nodes.map((node) => [node.id, node.bounds])));
  assert.match(markup, /class="geometry-container-frame"/);
  assert.doesNotMatch(markup, /tabindex|role="button"|data-node-id|data-okf-node/);
  const before = geometryContainerBounds(snapshot, positions).group;
  applyGeometryMove(geometryMoveStart(snapshot, "leaf", positions), positions, manual, 500, 0);
  const expanded = geometryContainerBounds(snapshot, positions).group;
  assert.ok(expanded.width > before.width, "moving a child cannot leave it outside an unchanged frame");
});

test("SC-AER-005 OKF renders shared frames behind semantic nodes", async () => {
  const graph = await import(moduleURL("okf_graph.js"));
  const container = { innerHTML: "", classList: { add() {} }, querySelector() { return null; } };
  globalThis.ELK = ELK;
  try {
    const snapshot = {
      projection_revision: "p1",
      nodes: scene.visible_nodes.map((node) => ({ ...node, label: node.id, presentation_fields: [], presentation_style: {} })),
      relationships: scene.visible_relationships.map((edge) => ({ id: edge.id, kind: "containment", from: edge.from_visible_id, to: edge.to_visible_id }))
    };
    const layout = await graph.renderOKFGraph(container, snapshot, "", {
      layoutProfile: { algorithm: "layered", features: ["compound"], options: {} }, layoutCatalog: catalog
    });
    assert.equal(layout.geometry.nodes.filter((node) => node.children_ids.length).length, 2);
    assert.match(container.innerHTML, /class="geometry-container-frame"/);
    assert.ok(container.innerHTML.indexOf("geometry-containers") < container.innerHTML.indexOf("okf-edges"));
    assert.equal((container.innerHTML.match(/data-okf-node=/g) || []).length, snapshot.nodes.length);
  } finally { delete globalThis.ELK; }
});

test("SC-AER-005 architecture consumes the same compound geometry", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { adaptELKLayout } = await import(moduleURL("layout.js"));
  const { renderGraph } = await import(moduleURL("graph.js"));
  const { sceneLayoutKey } = await import(moduleURL("view.js"));
  const node = (value) => ({ ...value, label: value.id, kind: "group", cycle_state: "none", diagnostic_state: "none",
    confidence_state: "not_applicable", identity_state: "stable", layer: 0, counts: { module_count: 1, internal_relationship_count: 0 }, accessible_label: value.id });
  const architectureScene = {
    ...scene,
    visible_nodes: scene.visible_nodes.map(node),
    visible_relationships: scene.visible_relationships.map((edge) => ({ ...edge, cycle_state: "none", accessible_label: edge.id })),
    model_revision: "r1", model_id: "m1", hierarchy_path: [], reference_visibility: "local",
    accessibility: { reading_order: [] }, project: {}
  };
  const profile = { algorithm: "layered", features: ["compound"], options: {} };
  const output = await runFeatureLayout(architectureScene, profile, catalog, null, ELK, { kind: "architecture", id: "m1", revision: "r1", navigation_scope: {} });
  const layout = adaptELKLayout(architectureScene, output, sceneLayoutKey(architectureScene), "layered", profile);
  const graphElement = { innerHTML: "", querySelector() { return null; } };
  renderGraph({ embeddedExport: null, elements: { graph: graphElement }, state: { scene: architectureScene, layout, viewport: { zoom: 1, panX: 0, panY: 0, positions: {} } } }, {});
  assert.match(graphElement.innerHTML, /class="geometry-container-frame"/);
  assert.ok(graphElement.innerHTML.indexOf("geometry-containers") < graphElement.innerHTML.indexOf("class=\"edges\""));
  assert.equal((graphElement.innerHTML.match(/data-node-id=/g) || []).length, architectureScene.visible_nodes.length);
});
