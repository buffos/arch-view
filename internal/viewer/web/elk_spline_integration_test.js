const assert = require("node:assert/strict");
const path = require("node:path");
const { createRequire } = require("node:module");

const requireFromTest = createRequire(__filename);
const { pathToFileURL } = require("node:url");

Promise.all([
  import(pathToFileURL(path.join(__dirname, "layout_request.js")).href),
  import(pathToFileURL(path.join(__dirname, "graph_route.js")).href)
]).then(async function (modules) {
  const request = modules[0];
  const routing = modules[1];
  const ELK = requireFromTest("./vendor/elk.bundled.js");
  const elk = new ELK();
  const catalog = {
    options: [{
      id: "org.eclipse.elk.edgeRouting",
      editable: true,
      renderer_support: "supported",
      targets: ["PARENTS"],
      algorithms: ["layered"]
    }]
  };
  const scene = {
    visible_nodes: [{ id: "a" }, { id: "b" }, { id: "c" }],
    visible_relationships: [
      { id: "a-b", from_visible_id: "a", to_visible_id: "b" },
      { id: "a-c", from_visible_id: "a", to_visible_id: "c" },
      { id: "b-c", from_visible_id: "b", to_visible_id: "c" }
    ]
  };
  const profile = { algorithm: "layered", options: { "org.eclipse.elk.edgeRouting": "SPLINES" } };
  const graph = request.buildELKGraph(scene, profile, catalog);
  assert.equal(graph.layoutOptions["elk.edgeRouting"], "SPLINES");
  const result = await elk.layout(graph);
  assert.equal(result.edges.length, 3);
  result.edges.forEach(function (edge) {
    const route = routing.fromELKSplineSections(edge.sections, 24);
    assert.equal(route.kind, "spline");
    assert.ok(route.sections.length > 0);
    route.sections.forEach(function (section) {
      assert.ok(section.segments.length > 0);
      section.segments.forEach(function (segment) {
        assert.equal(segment.kind, "cubic");
        [segment.control1, segment.control2, segment.to].forEach(function (point) {
          assert.ok(Number.isFinite(point.x));
          assert.ok(Number.isFinite(point.y));
        });
      });
    });
    assert.match(routing.pathForRoute(route), /^M /);
  });
  try {
    if (typeof elk.terminateWorker === "function") elk.terminateWorker();
  } catch (error) {
    // The pinned non-worker adapter can expose a no-op terminator with an
    // incompatible worker implementation; the layout result is still valid.
  }
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
