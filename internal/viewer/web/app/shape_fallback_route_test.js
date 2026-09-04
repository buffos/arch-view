const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "../graph_route.js")).href).then((routing) => {
  const shape = { geometry: "polygon", points: [
    { x: 0.2, y: 0.2 }, { x: 0.8, y: 0.2 }, { x: 1, y: 1 }, { x: 0, y: 1 }
  ], content: { x: 0.3, y: 0.3, width: 0.4, height: 0.4 } };
  const box = { x: 0, y: 0, width: 200, height: 100, shape };
  const loop = routing.selfLoopRoute(box);
  assert.deepEqual(loop.sections[0].start, { x: 100, y: 20 });
  assert.deepEqual(loop.sections[0].segments[0].to, { x: 100, y: 100 });
  const relationship = { from_visible_id: "a", to_visible_id: "b" };
  const target = { ...box, y: -200 };
  const fallback = routing.orthogonalRoute(box, target);
  assert.deepEqual(fallback.sections[0].start, { x: 100, y: 20 });
  assert.deepEqual(routing.edgeGeometry(relationship, box, target, null), routing.geometryFromRoute(fallback));
  assert.deepEqual(routing.edgeGeometry({ ...relationship, to_visible_id: "a" }, box, box, null), routing.geometryFromRoute(loop));
  const rectangle = { ...box, shape: undefined };
  assert.deepEqual(routing.selfLoopRoute(rectangle).sections[0].start, { x: 100, y: 0 });
}).catch((error) => { console.error(error); process.exitCode = 1; });
