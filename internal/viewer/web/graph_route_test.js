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

const from = { x: 40, y: 42, width: 190, height: 82 };
const horizontal = routing.orthogonalRoute(from, { x: 314, y: 42, width: 190, height: 82 });
assert.deepEqual(JSON.parse(JSON.stringify(routing.routePoints(horizontal))), [
  { x: 230, y: 83 },
  { x: 272, y: 83 },
  { x: 314, y: 83 }
]);
assert.equal(routing.geometryFromRoute(horizontal).path, "M 230 83 L 272 83 L 314 83");

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
assert.equal(routing.edgeGeometry(relationship, from, { x: 400, y: 42, width: 190, height: 82 }, null, true).path, "M 230 83 L 315 83 L 400 83");
assert.match(routing.edgeGeometry({ from_visible_id: "a", to_visible_id: "a" }, from, from, null, false).path, / C /);
assert.match(routing.edgeGeometry(relationship, from, { x: 400, y: 42, width: 190, height: 82 }, null, false).path, / C /);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
