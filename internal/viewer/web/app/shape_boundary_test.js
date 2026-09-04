const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "shape_boundary.js")).href).then(({ shapeBoundaryPoint, attachShapeEndpoints }) => {
  const box = { x: 10, y: 20, width: 200, height: 100 };
  const near = (actual, expected) => assert.ok(Math.abs(actual - expected) < 1e-8, `${actual} != ${expected}`);
  for (const shape of ["diamond", "hexagon", "ellipse", "pill"]) {
    for (const [dx, dy] of [[100, 50], [-100, 50], [-100, -50], [100, -50], [100, 0], [0, 50]]) {
      const point = shapeBoundaryPoint({ ...box, shape }, dx, dy);
      const x = Math.abs(point.x - 110), y = Math.abs(point.y - 70);
      const equations = {
        diamond: () => x / 100 + y / 50,
        hexagon: () => Math.max(y / 50, x / 100 + y / 100),
        ellipse: () => Math.hypot(x / 100, y / 50),
        pill: () => Math.hypot(Math.max(0, x - 50), y) / 50
      };
      near(equations[shape](), 1);
    }
  }
  const route = { sections: [{ start: { x: 210, y: 120 }, segments: [
    { kind: "line", to: { x: 250, y: 140 } },
    { kind: "cubic", control1: { x: 260, y: 140 }, control2: { x: 280, y: 100 }, to: { x: 300, y: 100 } }
  ] }] };
  const original = JSON.stringify(route);
  const adjusted = attachShapeEndpoints(route, { ...box, shape: "diamond" }, { x: 300, y: 100, width: 200, height: 100, shape: "ellipse" });
  assert.notDeepEqual(adjusted.sections[0].start, route.sections[0].start);
  assert.deepEqual(adjusted.sections[0].segments[0], route.sections[0].segments[0]);
  assert.deepEqual(adjusted.sections[0].segments[1].control2, route.sections[0].segments[1].control2);
  assert.equal(JSON.stringify(route), original, "cached route mutated");
  assert.equal(attachShapeEndpoints(route, box, box), route, "architecture rectangles changed");
  for (const invalid of [{ sections: "bad" }, { sections: [null] }, { sections: [{ segments: {} }] }]) {
    assert.equal(attachShapeEndpoints(invalid, { ...box, shape: "diamond" }, box), invalid);
  }
}).catch((error) => { console.error(error); process.exitCode = 1; });
