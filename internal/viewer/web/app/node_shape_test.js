const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "node_shape.js")).href).then(({ nodeShapeMarkup, shapeDimensions, shapeContentBox }) => {
  const box = { x: 10, y: 20, width: 200, height: 80 };
  assert.equal(nodeShapeMarkup("rounded_rectangle", box, ' class="node-shape"'), '<rect class="node-shape" x="10" y="20" width="200" height="80" rx="12"></rect>');
  assert.match(nodeShapeMarkup("rectangle", box), /rx="0"/);
  assert.match(nodeShapeMarkup("pill", box), /rx="40"/);
  assert.match(nodeShapeMarkup("ellipse", box), /cx="110" cy="60" rx="100" ry="40"/);
  assert.equal(nodeShapeMarkup("unknown", box), nodeShapeMarkup("rounded_rectangle", box));
  assert.match(nodeShapeMarkup("diamond", box), /points="110,20 210,60 110,100 10,60"/);
  assert.match(nodeShapeMarkup("hexagon", box), /points="60,20 160,20 210,60 160,100 60,100 10,60"/);
  for (const shape of ["rectangle", "rounded_rectangle", "pill", "diamond", "hexagon", "ellipse"]) {
    const dimensions = shapeDimensions(shape, { width: 210, height: 112 });
    const content = shapeContentBox(shape, { x: 0, y: 0, ...dimensions });
    assert.ok(Math.abs(content.width - 210) < 1e-9);
    assert.ok(Math.abs(content.height - 112) < 1e-9);
    assert.ok(content.x >= 0 && content.y >= 0);
  }
}).catch((error) => { console.error(error); process.exitCode = 1; });
