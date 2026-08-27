const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "graph_route.js"), "utf8");

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (routing) {

const elkRoute = routing.fromELKSections([
  {
    startPoint: { x: 10, y: 20 },
    bendPoints: [{ x: 20, y: 20 }, { x: 20, y: 20 }],
    endPoint: { x: 30, y: 40 }
  }
], 24);
assert.deepEqual(JSON.parse(JSON.stringify(elkRoute.points)), [
  { x: 34, y: 44 },
  { x: 44, y: 44 },
  { x: 54, y: 64 }
]);
assert.equal(routing.pathForRoute(elkRoute), "M 34 44 L 44 44 L 54 64");
assert.equal(elkRoute.labelX, 44);
assert.equal(elkRoute.labelY, 37);

const elkSpline = routing.fromELKSplineSections([
  {
    startPoint: { x: 10, y: 20 },
    bendPoints: [
      { x: 10, y: 20 },
      { x: 20, y: 0 },
      { x: 30, y: 40 },
      { x: 40, y: 80 },
      { x: 50, y: 60 }
    ],
    endPoint: { x: 60, y: 50 }
  }
], 24);
assert.equal(elkSpline.kind, "spline");
assert.equal(elkSpline.sections[0].segments.length, 2);
assert.deepEqual(JSON.parse(JSON.stringify(routing.routePoints(elkSpline))), [
  { x: 34, y: 44 },
  { x: 44, y: 24 },
  { x: 54, y: 64 },
  { x: 64, y: 104 },
  { x: 74, y: 84 },
  { x: 84, y: 74 }
]);
assert.equal(routing.pathForRoute(elkSpline), "M 34 44 C 34 44, 44 24, 54 64 C 64 104, 74 84, 84 74");
assert.equal(elkSpline.labelX, 69);
assert.equal(elkSpline.labelY, 80.75);

const from = { x: 40, y: 42, width: 190, height: 82 };
const horizontal = routing.orthogonalRoute(from, { x: 314, y: 42, width: 190, height: 82 });
assert.deepEqual(JSON.parse(JSON.stringify(routing.routePoints(horizontal))), [
  { x: 230, y: 83 },
  { x: 272, y: 83 },
  { x: 314, y: 83 }
]);
assert.equal(routing.geometryFromRoute(horizontal).path, "M 230 83 L 272 83 L 314 83");
assert.equal(routing.geometryFromRoute({
  points: [{ x: 10, y: 20 }, { x: 30, y: 40 }],
  label_x: 21,
  label_y: 33
}).path, "M 10 20 L 30 40");

const vertical = routing.orthogonalRoute(from, { x: 40, y: 240, width: 190, height: 82 });
assert.deepEqual(JSON.parse(JSON.stringify(routing.routePoints(vertical))), [
  { x: 135, y: 124 },
  { x: 135, y: 182 },
  { x: 135, y: 240 }
]);

const loop = routing.selfLoopRoute(from);
assert.equal(routing.routePoints(loop), null);
assert.match(routing.geometryFromRoute(loop).path, /^M 135 42 C 235 -13, 235 179, 135 124$/);

const relationship = { from_visible_id: "a", to_visible_id: "b" };
assert.equal(routing.edgeGeometry(relationship, from, { x: 400, y: 42, width: 190, height: 82 }, null).path, "M 230 83 L 315 83 L 400 83");
assert.match(routing.edgeGeometry({ from_visible_id: "a", to_visible_id: "a" }, from, from, null).path, / C /);
assert.match(routing.edgeGeometry({ from_visible_id: "a", to_visible_id: "a" }, from, from, elkSpline).path, /^M 135 42 C /);
assert.equal(routing.edgeGeometry(relationship, from, { x: 400, y: 42, width: 190, height: 82 }, null).path, "M 230 83 L 315 83 L 400 83");

assert.equal(routing.fromELKSections(null, 24), null);
assert.equal(routing.fromELKSections([{ startPoint: { x: 10, y: 20 } }], 24), null);
assert.equal(routing.fromELKSections([{
  startPoint: { x: 10, y: 20 },
  bendPoints: [{ x: "invalid", y: 20 }],
  endPoint: { x: 30, y: 40 }
}], 24), null);
assert.equal(routing.fromELKSplineSections([{ startPoint: { x: 10, y: 20 }, endPoint: { x: 30, y: 40 }, bendPoints: [] }], 24), null);
assert.equal(routing.fromELKSplineSections([{ startPoint: { x: 10, y: 20 }, endPoint: { x: 30, y: 40 }, bendPoints: [{ x: 10, y: 20 }] }], 24), null);
assert.equal(routing.fromELKSplineSections([{ startPoint: { x: 10, y: 20 }, endPoint: { x: 30, y: 40 }, bendPoints: [{ x: 10, y: 20 }, { x: 20, y: 20 }, { x: 25, y: 30 }] }], 24), null);
assert.equal(routing.fromELKSplineSections([{ startPoint: { x: 10, y: 20 }, endPoint: { x: 30, y: 40 }, bendPoints: [{ x: 10, y: 20 }, { x: Infinity, y: 20 }] }], 24), null);
assert.equal(routing.geometryFromRoute({
  sections: [{ start: { x: 10, y: 20 }, segments: [{ kind: "line", to: { x: NaN, y: 40 } }] }]
}), null);
assert.equal(routing.edgeGeometry(relationship, from, { x: 400, y: 42, width: 190, height: 82 }, {
  kind: "spline",
  sections: [{
    start: { x: 230, y: 83 },
    segments: [{ kind: "cubic", to: { x: 400, y: 83 }, control1: { x: 300, y: 20 } }]
  }]
}).path, "M 230 83 L 315 83 L 400 83");
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
