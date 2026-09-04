const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const ELK = require("./vendor/elk.bundled.js");

(async () => {
  const catalog = JSON.parse(fs.readFileSync(0, "utf8"));
  const { buildELKGraph } = await import(pathToFileURL(path.join(__dirname, "layout_request.js")).href);
  const engine = new ELK();
  const known = await engine.knownLayoutOptions();
  const admitted = ["mrtree.edgeRoutingMode", "mrtree.searchOrder", "mrtree.weighting", "spacing.componentComponent", "padding"];
  for (const short of admitted) {
    const id = "org.eclipse.elk." + short;
    const published = catalog.options.find((item) => item.id === id);
    const runtime = known.find((item) => item.id === id);
    assert.equal(published.editable, true, id);
    assert.equal(published.type, runtime.type, id);
    assert.deepEqual(published.targets, runtime.targets, id);
  }
  const scene = {
    visible_nodes: ["a", "b", "c", "d", "e", "f"].map((id) => ({ id })),
    visible_relationships: [["a", "b"], ["a", "c"], ["b", "d"], ["c", "d"], ["c", "e"]]
      .map(([from_visible_id, to_visible_id], i) => ({ id: "e" + i, from_visible_id, to_visible_id }))
  };
  const run = (algorithm, options) => engine.layout(buildELKGraph(scene, { algorithm, options }, catalog));
  const key = (short) => "org.eclipse.elk." + short;
  const first = await run("mrtree", { [key("mrtree.edgeRoutingMode")]: "MIDDLE_TO_MIDDLE" });
  const second = await run("mrtree", { [key("mrtree.edgeRoutingMode")]: "AVOID_OVERLAP" });
  assert.notDeepEqual(first.edges.map((edge) => edge.sections), second.edges.map((edge) => edge.sections));
  const positions = (result) => result.children.map((node) => [node.id, node.x, node.y]);
  const ordered = await run("mrtree", { [key("mrtree.weighting")]: "MODEL_ORDER" });
  for (const value of ["DESCENDANTS", "FAN"]) {
    assert.notDeepEqual(positions(await run("mrtree", { [key("mrtree.weighting")]: value })), positions(ordered));
  }
  const dfs = await run("mrtree", { [key("mrtree.searchOrder")]: "DFS" });
  assert.deepEqual(positions(dfs), positions(ordered));
  // These catalogued values fail in the actual pinned runtime. They must not
  // be offered as usable values by either settings dialog.
  await assert.rejects(run("mrtree", { [key("mrtree.searchOrder")]: "BFS" }), /call stack/);
  await assert.rejects(run("mrtree", { [key("mrtree.edgeRoutingMode")]: "NONE" }), /vector chain/);
  const small = await run("layered", { [key("spacing.componentComponent")]: 20 });
  const large = await run("layered", { [key("spacing.componentComponent")]: 160 });
  assert.notDeepEqual(positions(small), positions(large));
  for (const algorithm of ["mrtree", "layered"]) {
    const padding = { top: 10, left: 20, bottom: 30, right: 40 };
    const input = buildELKGraph(scene, { algorithm, options: { [key("padding")]: padding } }, catalog);
    assert.equal(input.children.some((node) => node.layoutOptions?.[key("padding")]), false);
    const before = await engine.layout(input);
    const after = await run(algorithm, { [key("padding")]: { top: 40, left: 50, bottom: 60, right: 70 } });
    assert.ok(Math.abs(after.width - before.width - 60) < 1e-8);
    assert.ok(Math.abs(after.height - before.height - 60) < 1e-8);
    for (let index = 0; index < before.children.length; index++) {
      assert.ok(Math.abs(after.children[index].x - before.children[index].x - 30) < 1e-8);
      assert.ok(Math.abs(after.children[index].y - before.children[index].y - 30) < 1e-8);
    }
  }
  console.log("Pinned runtime verified: routing, weighting, DFS, component spacing, graph padding; broken NONE/BFS excluded.");
})().catch((error) => { console.error(error); process.exitCode = 1; });
