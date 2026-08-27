const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "layout_request.js"), "utf8");
import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (request) {
assert.ok(request.buildELKGraph, "ELK request helper should be exposed");

const catalog = {
  options: [
    { id: "org.eclipse.elk.edgeRouting", editable: true, renderer_support: "supported", targets: ["PARENTS"], algorithms: ["layered"] },
    { id: "org.eclipse.elk.layered.spacing.baseValue", editable: true, renderer_support: "supported", targets: ["PARENTS"], algorithms: ["layered"] },
    { id: "org.eclipse.elk.priority", editable: true, renderer_support: "supported", targets: ["NODES", "EDGES"], algorithms: ["layered", "force"] },
    { id: "org.eclipse.elk.layered.priority.direction", editable: true, renderer_support: "supported", targets: ["EDGES"], algorithms: ["layered"] },
    { id: "org.eclipse.elk.layered.priority.shortness", editable: true, renderer_support: "supported", targets: ["EDGES"], algorithms: ["layered"] },
    { id: "org.eclipse.elk.layered.priority.straightness", editable: true, renderer_support: "supported", targets: ["EDGES"], algorithms: ["layered"] },
    { id: "org.eclipse.elk.alignment", editable: false, renderer_support: "catalog-only", targets: ["NODES"], algorithms: ["layered"] },
    { id: "org.eclipse.elk.layered.nodePlacement.strategy", editable: true, renderer_support: "supported", targets: ["PARENTS"], algorithms: ["layered"] },
    { id: "org.eclipse.elk.layered.spacing.baseValue", editable: true, renderer_support: "supported", targets: ["PARENTS"], algorithms: ["layered"] }
  ]
};
const scene = {
  visible_nodes: [{ id: "a" }, { id: "b" }],
  visible_relationships: [{ id: "a-b", from_visible_id: "a", to_visible_id: "b" }]
};
const profile = {
  algorithm: "layered",
  options: {
    "org.eclipse.elk.edgeRouting": "SPLINES",
    "org.eclipse.elk.layered.spacing.baseValue": 18,
    "org.eclipse.elk.layered.nodePlacement.strategy": "SIMPLE",
    "org.eclipse.elk.priority": 4,
    "org.eclipse.elk.layered.priority.direction": 5,
    "org.eclipse.elk.layered.priority.shortness": 6,
    "org.eclipse.elk.layered.priority.straightness": 7,
    "org.eclipse.elk.alignment": "CENTER",
    "org.eclipse.elk.spacing.baseValue": 24
  }
};

const graph = request.buildELKGraph(scene, profile, catalog);
assert.equal(graph.layoutOptions["elk.edgeRouting"], "SPLINES", "spline routing must override the default root routing key");
assert.equal(graph.layoutOptions["org.eclipse.elk.edgeRouting"], undefined, "spline routing must not be sent twice under conflicting aliases");
assert.equal(graph.layoutOptions["org.eclipse.elk.layered.spacing.baseValue"], "18");
assert.equal(graph.layoutOptions["org.eclipse.elk.layered.nodePlacement.strategy"], "SIMPLE");
assert.equal(graph.layoutOptions["org.eclipse.elk.alignment"], undefined, "node options must not be attached to the root");
assert.equal(graph.layoutOptions["org.eclipse.elk.priority"], undefined, "node/edge options must not be attached to the root");
assert.equal(graph.layoutOptions["org.eclipse.elk.spacing.baseValue"], undefined, "non-catalogued options must not be attached to the root");
assert.equal(graph.children.length, 2);
assert.deepEqual(graph.children[0].layoutOptions, { "org.eclipse.elk.priority": "4" });
assert.deepEqual(graph.children[1].layoutOptions, { "org.eclipse.elk.priority": "4" });
assert.equal(graph.edges[0].sources[0], "a");
assert.equal(graph.edges[0].targets[0], "b");
assert.deepEqual(graph.edges[0].layoutOptions, {
  "org.eclipse.elk.layered.priority.direction": "5",
  "org.eclipse.elk.layered.priority.shortness": "6",
  "org.eclipse.elk.layered.priority.straightness": "7",
  "org.eclipse.elk.priority": "4"
});

assert.deepEqual(request.buildTargetLayoutOptions(profile, catalog, "NODES"), { "org.eclipse.elk.priority": "4" });
assert.deepEqual(request.buildTargetLayoutOptions(profile, catalog, "EDGES"), {
  "org.eclipse.elk.layered.priority.direction": "5",
  "org.eclipse.elk.layered.priority.shortness": "6",
  "org.eclipse.elk.layered.priority.straightness": "7",
  "org.eclipse.elk.priority": "4"
});

const parentOnlyGraph = request.buildELKGraph(scene, {
  algorithm: "layered",
  options: { "org.eclipse.elk.layered.spacing.baseValue": 18 }
}, catalog);
assert.equal(parentOnlyGraph.children[0].layoutOptions, undefined, "empty node options should preserve the existing request shape");
assert.equal(parentOnlyGraph.edges[0].layoutOptions, undefined, "empty edge options should preserve the existing request shape");

const forceGraph = request.buildELKGraph(scene, {
  algorithm: "force",
  options: { "org.eclipse.elk.priority": 3, "org.eclipse.elk.layered.priority.direction": 9 }
}, catalog);
assert.deepEqual(forceGraph.children[0].layoutOptions, { "org.eclipse.elk.priority": "3" });
assert.deepEqual(forceGraph.edges[0].layoutOptions, { "org.eclipse.elk.priority": "3" }, "algorithm-incompatible edge options must be omitted");

const profileSnapshot = JSON.stringify(profile);
request.buildELKGraph(scene, profile, catalog);
assert.equal(JSON.stringify(profile), profileSnapshot, "building an ELK request must not mutate the profile");

const forceProfile = {
  algorithm: "force",
  options: { "org.eclipse.elk.layered.spacing.baseValue": 18 }
};
assert.equal(request.buildRootLayoutOptions(forceProfile, catalog)["org.eclipse.elk.layered.spacing.baseValue"], undefined, "algorithm-incompatible options must not be attached to the root");

const baseSpacingOptions = request.buildRootLayoutOptions(profile, catalog);
assert.equal(baseSpacingOptions["elk.spacing.nodeNode"], undefined, "explicit spacing defaults must not shadow base spacing");
assert.equal(baseSpacingOptions["elk.layered.spacing.nodeNodeBetweenLayers"], undefined, "layer spacing defaults must not shadow base spacing");

}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
