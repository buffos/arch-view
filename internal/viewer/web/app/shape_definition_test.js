const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

Promise.all(["node_shape.js", "shape_boundary.js", "okf_graph.js"].map((file) => import(pathToFileURL(path.join(__dirname, file)).href))).then(async ([shapes, boundaries, graph]) => {
  const triangle = { id: "test.triangle", version: "2", geometry: "polygon", points: [{ x: 0.5, y: 0 }, { x: 1, y: 1 }, { x: 0, y: 1 }], content: { x: 0.4, y: 0.5, width: 0.2, height: 0.3 } };
  assert.match(shapes.nodeShapeMarkup(triangle, { x: 0, y: 0, width: 200, height: 100 }), /points="100,0 200,100 0,100"/);
  assert.deepEqual(shapes.shapeContentBox(triangle, { x: 0, y: 0, width: 200, height: 100 }), { x: 80, y: 50, width: 40, height: 30 });
  assert.deepEqual(boundaries.shapeBoundaryPoint({ x: 0, y: 0, width: 200, height: 100, shape: triangle }, 100, 0), { x: 150, y: 50 });
  for (const field of ["width", "height"]) {
    const unusable = { ...triangle, content: { ...triangle.content, [field]: Number.MIN_VALUE } };
    assert.equal(shapes.validatedShapeDefinition(unusable), null, "subnormal content fraction must not reach layout");
    assert.deepEqual(shapes.shapeDimensions(unusable, { width: 210, height: 58 }), { width: 210, height: 58 });
  }
  const precisionBoundary = { geometry: "rectangle", content: { x: 0, y: 0, width: Number.EPSILON, height: Number.EPSILON } };
  assert.equal(shapes.validatedShapeDefinition(precisionBoundary), precisionBoundary);
  assert.ok(Object.values(shapes.shapeDimensions(precisionBoundary, { width: 210, height: 112 })).every(Number.isFinite));
  for (const invalid of [{ ...triangle, geometry: "<script>" }, { ...triangle, points: [{ x: '" onload="evil()', y: 0 }, ...triangle.points.slice(1)] }]) {
    assert.equal(shapes.validatedShapeDefinition(invalid), null);
    assert.doesNotMatch(shapes.nodeShapeMarkup(invalid, { x: 0, y: 0, width: 200, height: 100 }), /script|onload|evil/);
  }
  const container = { innerHTML: "", classList: { add() {} }, querySelector() { return null; } };
  const layout = await graph.renderOKFGraph(container, { nodes: [{ id: "custom", shape: "test.triangle@2", shape_definition: triangle, presentation_fields: [], label: "Custom" }], relationships: [] }, "", {});
  assert.match(container.innerHTML, /<polygon/);
  assert.equal(layout.positions.custom.width, 1050);
  assert.ok(Math.abs(layout.positions.custom.height - 58 / 0.3) < 1e-9);
  assert.match(container.innerHTML, /translate\(420 /);
}).catch((error) => { console.error(error); process.exitCode = 1; });
