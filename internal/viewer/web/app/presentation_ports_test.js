const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { createRequire } = require("node:module");
const { pathToFileURL } = require("node:url");

const ELK = createRequire(__filename)("../vendor/elk.bundled.js");
const moduleURL = (name) => pathToFileURL(path.join(__dirname, name)).href;
const features = JSON.parse(fs.readFileSync(path.join(__dirname, "../../layout/features.json"), "utf8"));
const catalog = {
  features,
  options: [{
    id: "org.eclipse.elk.portConstraints", editable: true, renderer_support: "supported",
    targets: ["NODES"], supported_targets: ["NODES"], algorithms: ["layered"]
  }]
};
const scene = {
  visible_nodes: [{ id: "a" }, { id: "b" }],
  visible_relationships: [{ id: "edge-a-b", from_visible_id: "a", to_visible_id: "b" }]
};

test("SC-AER-004 pinned ELK returns deterministic presentation ports and endpoints", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  const profile = { algorithm: "layered", features: ["ports"], options: {} };
  const source = { kind: "architecture", id: "model", revision: "r1", navigation_scope: {} };
  const output = await runFeatureLayout(scene, profile, catalog, null, ELK, source);
  const geometry = normalizeGeometrySnapshot(scene, output, profile, source);

  assert.deepEqual(geometry.provenance.features, ["ports"]);
  assert.deepEqual(geometry.nodes.map((node) => node.ports.map((port) => [port.id, port.role, port.side])), [
    [["port::a::in", "in", "WEST"], ["port::a::out", "out", "EAST"]],
    [["port::b::in", "in", "WEST"], ["port::b::out", "out", "EAST"]]
  ]);
  assert.equal(geometry.edges[0].source_port_id, "port::a::out");
  assert.equal(geometry.edges[0].target_port_id, "port::b::in");
  assert.equal(geometry.edges[0].route.sections[0].start.x, geometry.nodes[0].ports[1].position.x);
  assert.equal(geometry.edges[0].route.sections[0].segments.at(-1).to.x, geometry.nodes[1].ports[0].position.x);
  assert.ok(geometry.nodes.flatMap((node) => node.ports).every((port) => port.label && Object.values(port.bounds).every(Number.isFinite)));
});

test("SC-AER-004 port sides follow every supported layout direction", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  const expected = {
    RIGHT: ["WEST", "EAST"], LEFT: ["EAST", "WEST"],
    DOWN: ["NORTH", "SOUTH"], UP: ["SOUTH", "NORTH"]
  };
  for (const [direction, sides] of Object.entries(expected)) {
    const profile = { algorithm: "layered", features: ["ports"], options: { "org.eclipse.elk.direction": direction } };
    const output = await runFeatureLayout(scene, profile, catalog, null, ELK, { kind: "architecture", id: "model", revision: "r", navigation_scope: {} });
    const geometry = normalizeGeometrySnapshot(scene, output, profile, output.geometrySource);
    assert.deepEqual(geometry.nodes[0].ports.map((port) => port.side), sides, direction);
    assert.equal(geometry.edges[0].source_port_id, "port::a::out", direction);
    assert.equal(geometry.edges[0].target_port_id, "port::b::in", direction);
  }
});

test("SC-AER-007 malformed port geometry is omitted with affected-edge fallback", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  class InvalidPortEngine {
    async layout(graph) {
      return {
        ...graph, width: 500, height: 200,
        children: graph.children.map((node, index) => ({ ...node, x: 30 + index * 260, y: 40,
          ports: node.ports.map((port) => {
            const role = port.id.endsWith("::in") ? "in" : "out";
            return { ...port, x: role === "in" ? -10 : (node.id === "a" ? Number.NaN : node.width), y: 36,
              labels: port.labels.map((label) => ({ ...label, x: role === "in" ? -label.width : 10, y: -1 })) };
          }) })),
        edges: graph.edges.map((edge) => ({ ...edge, sections: [{ startPoint: { x: 220, y: 81 }, endPoint: { x: 290, y: 81 } }] }))
      };
    }
  }
  const profile = { algorithm: "layered", features: ["ports"], options: {} };
  const output = await runFeatureLayout(scene, profile, catalog, null, InvalidPortEngine, { kind: "okf", id: "bundle", revision: "r", navigation_scope: {} });
  const geometry = normalizeGeometrySnapshot(scene, output, profile, output.geometrySource);
  assert.ok(geometry.diagnostics.some((item) => item.code === "geometry_port_invalid"));
  assert.ok(geometry.diagnostics.some((item) => item.code === "geometry_port_endpoint_invalid"));
  assert.deepEqual(geometry.nodes[0].ports.map((port) => port.role), ["in"]);
  assert.equal(geometry.edges[0].source_port_id, undefined);
  assert.equal(geometry.edges[0].route.kind, "orthogonal");
});

test("SC-AER-007 a valid port on the wrong node is rejected as an endpoint", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  class WrongEndpointEngine {
    async layout(graph) {
      const engine = new ELK();
      const output = await engine.layout(graph);
      output.edges[0].sources = ["port::b::out"];
      return output;
    }
  }
  const profile = { algorithm: "layered", features: ["ports"], options: {} };
  const output = await runFeatureLayout(scene, profile, catalog, null, WrongEndpointEngine, { kind: "architecture", id: "model", revision: "r", navigation_scope: {} });
  const geometry = normalizeGeometrySnapshot(scene, output, profile, output.geometrySource);
  assert.ok(geometry.diagnostics.some((item) => item.code === "geometry_port_endpoint_invalid"));
  assert.equal(geometry.edges[0].source_port_id, undefined);
  assert.equal(geometry.edges[0].target_port_id, undefined);
  assert.equal(geometry.edges[0].route.kind, "orthogonal");
});

test("SC-AER-007 a route detached from valid ports is rejected", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  class DetachedRouteEngine {
    async layout(graph) {
      const output = await new ELK().layout(graph);
      output.edges[0].sections[0].startPoint.x += 5;
      return output;
    }
  }
  const profile = { algorithm: "layered", features: ["ports"], options: {} };
  const output = await runFeatureLayout(scene, profile, catalog, null, DetachedRouteEngine, { kind: "architecture", id: "model", revision: "r", navigation_scope: {} });
  const geometry = normalizeGeometrySnapshot(scene, output, profile, output.geometrySource);
  assert.ok(geometry.diagnostics.some((item) => item.code === "geometry_port_endpoint_invalid"));
  assert.equal(geometry.edges[0].source_port_id, undefined);
  assert.equal(geometry.edges[0].route.kind, "orthogonal");
});

test("SC-AER-007 a missing ELK edge is diagnosed and uses ordinary geometry", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  class MissingEdgeEngine {
    async layout(graph) {
      const output = await new ELK().layout(graph);
      return { ...output, edges: [] };
    }
  }
  const profile = { algorithm: "layered", features: ["ports"], options: {} };
  const output = await runFeatureLayout(scene, profile, catalog, null, MissingEdgeEngine, { kind: "okf", id: "bundle", revision: "r", navigation_scope: {} });
  const geometry = normalizeGeometrySnapshot(scene, output, profile, output.geometrySource);
  assert.ok(geometry.diagnostics.some((item) => item.code === "geometry_port_endpoint_invalid"));
  assert.equal(geometry.edges[0].source_port_id, undefined);
  assert.equal(geometry.edges[0].route.kind, "orthogonal");
});

test("SC-AER-011 ports are stable across repeats and feature request order", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  const countedScene = { ...scene, visible_relationships: [{ ...scene.visible_relationships[0], count: 2 }] };
  const source = { kind: "architecture", id: "model", revision: "r1", navigation_scope: {} };
  const run = async (features) => {
    const profile = { algorithm: "layered", features, options: {} };
    const output = await runFeatureLayout(countedScene, profile, catalog, null, ELK, source);
    return normalizeGeometrySnapshot(countedScene, output, profile, source);
  };
  const first = await run(["ports", "edge_labels"]);
  const repeated = await run(["ports", "edge_labels"]);
  const reordered = await run(["edge_labels", "ports"]);
  assert.deepEqual(repeated, first);
  assert.deepEqual(reordered, first);
});

test("SC-AER-004 shared port markup remains presentation-only", async () => {
  const { geometryPortMarkup, geometryPortRoute } = await import(moduleURL("port_presentation.js"));
  const markup = geometryPortMarkup([{ id: "port::a::in", role: "in", side: "WEST", label: "in",
    bounds: { x: 10, y: 20, width: 10, height: 10 }, label_bounds: { x: -5, y: 21, width: 14, height: 12 } }]);
  assert.match(markup, /class="presentation-port in"/);
  assert.match(markup, />in<\/text>/);
  assert.match(markup, /aria-hidden="true"/);
  assert.doesNotMatch(markup, /tabindex|role="button"|data-node-id/);

  const snapshot = {
    nodes: [
      { id: "a", bounds: { x: 0, y: 0 }, ports: [{ id: "port::a::out", position: { x: 100, y: 40 }, bounds: { x: 90, y: 35, width: 10, height: 10 }, label_bounds: { x: 101, y: 46, width: 20, height: 12 } }] },
      { id: "b", bounds: { x: 200, y: 0 }, ports: [{ id: "port::b::in", position: { x: 190, y: 40 }, bounds: { x: 190, y: 35, width: 10, height: 10 }, label_bounds: { x: 175, y: 46, width: 14, height: 12 } }] }
    ]
  };
  const edge = { source_node_id: "a", target_node_id: "b", source_port_id: "port::a::out", target_port_id: "port::b::in" };
  const moved = geometryPortRoute(snapshot, edge, { a: { x: 25, y: 15 }, b: { x: 200, y: 0 } }, null);
  assert.equal(moved.route.sections[0].start.x, 125);
  assert.equal(moved.route.sections[0].start.y, 55);
  assert.equal(moved.preserveEndpoints, true);
});

test("SC-AER-004 port-bound self-loops retain ELK geometry after a manual move", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { normalizeGeometrySnapshot } = await import(moduleURL("geometry_snapshot.js"));
  const { geometryPortRoute } = await import(moduleURL("port_presentation.js"));
  const { edgeGeometry } = await import(pathToFileURL(path.join(__dirname, "../graph_route.js")).href);
  const selfScene = {
    visible_nodes: [{ id: "a" }],
    visible_relationships: [{ id: "self", from_visible_id: "a", to_visible_id: "a" }]
  };
  const profile = { algorithm: "layered", features: ["ports"], options: {} };
  const source = { kind: "architecture", id: "model", revision: "r", navigation_scope: {} };
  const output = await runFeatureLayout(selfScene, profile, catalog, null, ELK, source);
  const snapshot = normalizeGeometrySnapshot(selfScene, output, profile, source);
  const edge = snapshot.edges[0];
  const movedBounds = { ...snapshot.nodes[0].bounds, x: snapshot.nodes[0].bounds.x + 25, y: snapshot.nodes[0].bounds.y + 15 };
  const routed = geometryPortRoute(snapshot, edge, { a: movedBounds }, null);
  const rendered = edgeGeometry(selfScene.visible_relationships[0], movedBounds, movedBounds, routed.route, routed.preserveEndpoints);
  assert.equal(routed.route.sections[0].start.x, edge.route.sections[0].start.x + 25);
  assert.equal(routed.route.sections[0].start.y, edge.route.sections[0].start.y + 15);
  assert.equal(rendered.path.includes(" C "), edge.route.kind === "spline");
  assert.notEqual(rendered.path, edgeGeometry(selfScene.visible_relationships[0], movedBounds, movedBounds, null).path);
});

test("SC-AER-004 OKF uses the shared live port renderer without adding semantic items", async () => {
  const graph = await import(moduleURL("okf_graph.js"));
  const container = { innerHTML: "", classList: { add() {} }, querySelector() { return null; } };
  globalThis.ELK = ELK;
  try {
    const snapshot = { projection_revision: "p1", nodes: [{ id: "a", label: "A" }, { id: "b", label: "B" }],
      relationships: [{ id: "e", kind: "containment", from: "a", to: "b" }] };
    const layout = await graph.renderOKFGraph(container, snapshot, "", {
      layoutProfile: { algorithm: "layered", features: ["ports"], options: {} }, layoutCatalog: catalog
    });
    assert.match(container.innerHTML, /class="presentation-port out"/);
    assert.match(container.innerHTML, /class="presentation-port in"/);
    assert.equal(layout.geometry.edges[0].source_port_id, "port::a::out");
    assert.equal(layout.geometry.edges[0].target_port_id, "port::b::in");
    assert.doesNotMatch(container.innerHTML, /presentation-port[^>]*(?:tabindex|role="button")/);
  } finally { delete globalThis.ELK; }
});

test("SC-AER-004 architecture uses the same port geometry and renderer", async () => {
  const { runFeatureLayout } = await import(moduleURL("elk_runtime.js"));
  const { adaptELKLayout } = await import(moduleURL("layout.js"));
  const { renderGraph } = await import(moduleURL("graph.js"));
  const { sceneLayoutKey } = await import(moduleURL("view.js"));
  const node = (id, layer) => ({ id, label: id.toUpperCase(), kind: "module", cycle_state: "none", diagnostic_state: "none",
    confidence_state: "not_applicable", identity_state: "stable", layer, counts: { module_count: 1, internal_relationship_count: 0 }, accessible_label: id });
  const architectureScene = {
    model_revision: "r1", model_id: "m1", hierarchy_path: [], reference_visibility: "local",
    accessibility: { reading_order: [] }, project: {},
    visible_nodes: [node("a", 0), node("b", 1)],
    visible_relationships: [{ id: "edge-a-b", from_visible_id: "a", to_visible_id: "b", count: 1, cycle_state: "none", accessible_label: "A to B" }]
  };
  const profile = { algorithm: "layered", features: ["ports"], options: {} };
  const output = await runFeatureLayout(architectureScene, profile, catalog, null, ELK, { kind: "architecture", id: "m1", revision: "r1", navigation_scope: {} });
  const layout = adaptELKLayout(architectureScene, output, sceneLayoutKey(architectureScene), "layered", profile);
  const graphElement = { innerHTML: "", querySelector() { return null; } };
  renderGraph({ embeddedExport: null, elements: { graph: graphElement }, state: { scene: architectureScene, layout, viewport: { zoom: 1, panX: 0, panY: 0, positions: {} } } }, {});
  assert.match(graphElement.innerHTML, /class="presentation-port out"/);
  assert.match(graphElement.innerHTML, /class="presentation-port in"/);
  const start = layout.geometry.edges[0].route.sections[0].start;
  assert.ok(graphElement.innerHTML.includes('d="M ' + start.x + " " + start.y), "route must begin at the ELK output port");
});
