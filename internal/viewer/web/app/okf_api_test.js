const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "okf_api.js"), "utf8");

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(async function (apiModule) {
  const calls = [];
  const fetchImplementation = async function (url, options) {
    calls.push({ url, options });
    if (url.endsWith("/catalog")) {
      return { ok: true, status: 200, json: async function () { return { data: { bundles: [] }, diagnostics: [], meta: { request_id: "req-1" } }; } };
    }
    if (url.endsWith("/extensions")) {
      return { ok: false, status: 409, json: async function () { return { error: { code: "okf_revision_conflict", message: "stale", diagnostics: [{ code: "okf_revision_conflict" }] }, meta: { request_id: "req-2" } }; } };
    }
    return { ok: true, status: 200, json: async function () { return { data: { saved: true }, diagnostics: [], meta: { request_id: "req-3" } }; } };
  };

  const api = apiModule.createOKFAPI(fetchImplementation, "/api/okf");
  const catalog = await api.getCatalog();
  assert.deepEqual(catalog, { bundles: [] });
  assert.equal(calls[0].url, "/api/okf/catalog");
  assert.equal(calls[0].options.headers.Accept, "application/json");

  const saved = await api.saveProfile({ profile_id: "project:review" }, "rev-1", "op-1");
  assert.deepEqual(saved, { saved: true });
  assert.equal(calls[1].url, "/api/okf/profiles/project%3Areview");
  assert.equal(calls[1].options.method, "PUT");
  assert.equal(calls[1].options.headers.Accept, "application/json");
  assert.equal(calls[1].options.headers["If-Match"], "rev-1");
  assert.equal(calls[1].options.headers["Idempotency-Key"], "op-1");
  assert.deepEqual(JSON.parse(calls[1].options.body), { profile_id: "project:review" });

  await assert.rejects(api.getExtensions(), function (error) {
    assert.equal(error.code, "okf_revision_conflict");
    assert.equal(error.status, 409);
    assert.equal(error.diagnostics[0].code, "okf_revision_conflict");
    return true;
  });
  const draft = { name: "Retry me", extension: { retained: true } };
  await api.saveProfileAs(draft, "copy", "builtin:neutral", "rev-2", "same-creation");
  await api.saveProfileAs(draft, "copy", "builtin:neutral", "rev-2", "same-creation");
  const retries = calls.slice(-2);
  assert.deepEqual(retries[0], retries[1], "explicit retransmission preserves exact command and operation identity");
  assert.equal(retries[0].options.headers["Idempotency-Key"], "same-creation");
  assert.equal(JSON.parse(retries[0].options.body).operation_id, "same-creation");
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
