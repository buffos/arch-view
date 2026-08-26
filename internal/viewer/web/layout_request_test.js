const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "layout_request.js"), "utf8");
import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (request) {
assert.ok(request.buildELKGraph, "ELK request helper should be exposed");

const catalog = {
  options: [
    { id: "org.eclipse.elk.layered.spacing.baseValue", editable: true, renderer_support: "supported", targets: ["PARENTS"], algorithms: ["layered"] },
    { id: "org.eclipse.elk.alignment", editable: true, renderer_support: "supported", targets: ["NODES"], algorithms: ["layered"] },
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
    "org.eclipse.elk.layered.spacing.baseValue": 18,
    "org.eclipse.elk.layered.nodePlacement.strategy": "SIMPLE",
    "org.eclipse.elk.alignment": "CENTER",
    "org.eclipse.elk.spacing.baseValue": 24
  }
};

const graph = request.buildELKGraph(scene, profile, catalog);
assert.equal(graph.layoutOptions["org.eclipse.elk.layered.spacing.baseValue"], "18");
assert.equal(graph.layoutOptions["org.eclipse.elk.layered.nodePlacement.strategy"], "SIMPLE");
assert.equal(graph.layoutOptions["org.eclipse.elk.alignment"], undefined, "node options must not be attached to the root");
assert.equal(graph.layoutOptions["org.eclipse.elk.spacing.baseValue"], undefined, "non-catalogued options must not be attached to the root");
assert.equal(graph.children.length, 2);
assert.equal(graph.edges[0].sources[0], "a");
assert.equal(graph.edges[0].targets[0], "b");

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
