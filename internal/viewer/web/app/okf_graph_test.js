const assert = require("node:assert/strict");
const path = require("node:path");
const { readFileSync } = require("node:fs");
const { pathToFileURL } = require("node:url");

const stylesheet = readFileSync(path.join(__dirname, "../styles/14-okf.css"), "utf8");
assert.doesNotMatch(stylesheet, /marker(?:-start|-mid|-end)?\s*:/, "OKF CSS must not override renderer-owned arrowheads");

import(pathToFileURL(path.join(__dirname, "okf_graph.js")).href).then(async (graph) => {
  const container = {
    innerHTML: "",
    classList: { add() {} },
    querySelector() { return null; }
  };
  const snapshot = {
    nodes: [{ id: "one", label: "One", shape: "ellipse", presentation_style: { shape: "rounded_rectangle", fill: "#abcdef" } }],
    relationships: []
  };
  await graph.renderOKFGraph(container, snapshot, "", {});
  assert.match(container.innerHTML, /<ellipse/);
  assert.doesNotMatch(container.innerHTML, /<rect/);
  assert.match(container.innerHTML, /#abcdef/);
  delete snapshot.nodes[0].shape;
  await graph.renderOKFGraph(container, snapshot, "", {});
  assert.match(container.innerHTML, /<rect/);
  const polygonLayout = await graph.renderOKFGraph(container, {
    nodes: [
      { id: "diamond", label: "Diamond", shape: "diamond", presentation_fields: [] },
      { id: "hexagon", label: "Hexagon", shape: "hexagon", presentation_fields: [] }
    ], relationships: []
  }, "", {});
  assert.match(container.innerHTML, /<polygon/);
  assert.match(container.innerHTML, /<g transform="translate\(105 29\)">/);
  assert.equal(polygonLayout.positions.diamond.width, 420);
  assert.equal(polygonLayout.positions.diamond.height, 116);
  assert.ok(polygonLayout.positions.hexagon.x > polygonLayout.positions.diamond.x + polygonLayout.positions.diamond.width, "fallback columns must not overlap enlarged shapes");
  let layoutInput;
  let worker;
  let terminated = false;
  globalThis.ELK = class {
    constructor(options) { worker = options.workerUrl; }
    async layout(input) {
      layoutInput = input;
      return { ...input, width: 600, height: 160, children: input.children.map((node, index) => ({ ...node, x: index * 300, y: 0 })) };
    }
    terminateWorker() { terminated = true; }
  };
  try {
    snapshot.nodes.push({ id: "two", label: "Two" });
    snapshot.relationships = [
      { id: "containment", kind: "containment", from: "one", to: "two" },
      { id: "semantic", kind: "semantic_link", from: "two", to: "one" }
    ];
    const renderedLayout = await graph.renderOKFGraph(container, snapshot, "", {}, "/shared-elk-worker.js");
    assert.equal(worker, "/shared-elk-worker.js");
    assert.equal(terminated, true);
    assert.deepEqual(layoutInput.edges.map((edge) => edge.id), ["containment"], "semantic overlays must not participate in ELK layout");
    assert.doesNotMatch(container.innerHTML, /data-okf-edge="semantic"/);
    const edgeLayer = { innerHTML: "" };
    const overlayContainer = { querySelector: () => edgeLayer };
    graph.updateOKFSemanticLinks(overlayContainer, snapshot, "one", renderedLayout, {});
    const semanticPath = edgeLayer.innerHTML.match(/<path[^>]*data-okf-edge="semantic"[^>]*>/)[0];
    const containmentPath = edgeLayer.innerHTML.match(/<path[^>]*data-okf-edge="containment"[^>]*>/)[0];
    assert.doesNotMatch(semanticPath, /marker-(?:start|end)/, "semantic links must have no arrowheads");
    assert.match(containmentPath, /marker-end="url\(#okf-arrow\)"/);
    assert.doesNotMatch(containmentPath, /marker-start/, "containment arrows point only toward the child");
    graph.updateOKFSemanticLinks(overlayContainer, snapshot, "", renderedLayout, {});
    assert.doesNotMatch(edgeLayer.innerHTML, /data-okf-edge="semantic"/);
    const previousMarkup = container.innerHTML;
    const failure = new Error("ELK failed");
    globalThis.ELK.prototype.layout = async () => { throw failure; };
    let recovery;
    await graph.renderOKFGraph(container, snapshot, "", { onLayoutError: (error, result) => { assert.equal(error, failure); recovery = result; } }, "/shared-elk-worker.js");
    assert.equal(recovery.retained, true);
    assert.equal(container.innerHTML, previousMarkup, "failed relayout must retain safe node placement");
    snapshot.nodes.push({ id: "three", label: "Three" });
    await graph.renderOKFGraph(container, snapshot, "", { onLayoutError: (_error, result) => { recovery = result; } }, "/shared-elk-worker.js");
    assert.equal(recovery.retained, false, "changed scene must not reuse incompatible positions");
    assert.match(container.innerHTML, /Three/);
    const currentMarkup = container.innerHTML;
    let notified = false;
    await graph.renderOKFGraph(container, snapshot, "", { isCurrent: () => false, onLayoutError: () => { notified = true; } }, "/shared-elk-worker.js");
    assert.equal(notified, false, "superseded failures must not replace current status");
    assert.equal(container.innerHTML, currentMarkup);
  } finally {
    delete globalThis.ELK;
  }
}).catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
