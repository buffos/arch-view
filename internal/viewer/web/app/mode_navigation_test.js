const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "mode_navigation.js"), "utf8");

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(async function (navigation) {
  assert.equal(navigation.selectedView(""), "architecture");
  assert.equal(navigation.selectedView("?view=architecture"), "architecture");
  assert.equal(navigation.selectedView("?view=okf"), "okf");
  const available = await navigation.discoverSelectableOKF(async function () {
    return { ok: true, json: async function () { return { data: { bundles: [{ bundle_id: "one", selectable: false }, { bundle_id: "two", selectable: true }] } }; } };
  });
  assert.equal(available, true);
  const unavailable = await navigation.discoverSelectableOKF(async function () {
    return { ok: true, json: async function () { return { data: { bundles: [{ bundle_id: "one", selectable: false }] } }; } };
  });
  assert.equal(unavailable, false);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
