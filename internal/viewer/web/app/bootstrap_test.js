const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "bootstrap.js"), "utf8")
  .replace(/^import .*?;\r?\n/gm, "");

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(async function (bootstrap) {
  let modelResolve;
  let sceneCalls = 0;
  let scopesStarted = false;
  const modelLoad = new Promise(function (resolve) { modelResolve = resolve; });
  const navigation = {
    loadModel: function () { return modelLoad; },
    loadScopes: function () { scopesStarted = true; return Promise.resolve(true); },
    loadScene: function (pathValue) {
      sceneCalls += 1;
      assert.deepEqual(pathValue, ["internal"]);
      return Promise.resolve(true);
    }
  };
  const startup = bootstrap.loadInitialState(navigation, { embeddedExport: { initial_path: ["internal"] } });
  assert.equal(scopesStarted, true, "scope discovery must start before model loading finishes");
  modelResolve();
  await startup;
  assert.equal(sceneCalls, 1);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
