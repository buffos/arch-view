const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "api.js"), "utf8");

global.window = { location: { href: "http://127.0.0.1/" } };

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(async function (apiModule) {
  const requested = [];
  global.fetch = async function (url, options) {
    requested.push({ url: url, options: options });
    return {
      ok: true,
      json: async function () { return { ok: true }; }
    };
  };

  const context = { liveEnabled: true, embeddedExport: null, modelID: "session:api", state: { liveRevision: 7 } };
  const api = apiModule.createAPI(context);
  await api.getJSON("/v1/models/session%3Aapi/source-index/files?limit=25");
  await api.postJSON("/v1/live/session%3Aapi/status", {});

  assert.equal(requested[0].url, "/v1/models/session%3Aapi/source-index/files?limit=25&revision=7");
  assert.equal(requested[1].url, "/v1/live/session%3Aapi/status");
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
