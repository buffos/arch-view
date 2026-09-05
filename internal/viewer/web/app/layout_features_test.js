const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const moduleURL = (name) => pathToFileURL(path.join(__dirname, name)).href;
const metadata = JSON.parse(fs.readFileSync(path.join(__dirname, "../../layout/features.json"), "utf8"));
const definition = (id, order = 0, prerequisites = []) => ({ id, name: id, order, prerequisites, owns: [id],
  algorithms: ["layered"], surfaces: ["browser"], status: "supported", required_options: {} });
const handler = (id) => Object.fromEntries(["id", "negotiate", "prepare", "normalize", "validate", "render", "fallback"]
  .map((key) => [key, key === "id" ? id : (context) => context]));

test("SC-AER-011 registry uses dependencies, priority and stable ID", async () => {
  const { createFeatureRegistry } = await import(moduleURL("layout_features.js"));
  const definitions = [definition("a", 0, ["b"]), definition("b", 20), definition("c", 10), definition("d", 10)];
  const handlers = definitions.map((item) => handler(item.id));
  const registry = createFeatureRegistry(definitions, handlers);
  assert.deepEqual(registry.order, ["c", "d", "b", "a"]);
  assert.deepEqual(createFeatureRegistry([...definitions].reverse(), [...handlers].reverse()).order, registry.order);
  const trace = [];
  handlers.forEach((item) => { item.prepare = (context) => { trace.push(item.id); return context; }; });
  await registry.run("prepare", {}, registry.order);
  assert.deepEqual(trace, registry.order);
});

test("SC-AER-011 rejects duplicate, cyclic, missing and conflicting registrations", async () => {
  const { createFeatureRegistry } = await import(moduleURL("layout_features.js"));
  const a = definition("a");
  assert.throws(() => createFeatureRegistry([a, a]), /Duplicate feature/);
  assert.throws(() => createFeatureRegistry([a], [handler("a"), handler("a")]), /Duplicate handler/);
  assert.throws(() => createFeatureRegistry([a], [handler("unknown")]), /Unknown handler/);
  assert.throws(() => createFeatureRegistry([a]), /no handler/);
  assert.throws(() => createFeatureRegistry([a], [{ id: "a" }]), /Incomplete/);
  assert.throws(() => createFeatureRegistry([definition("a", 0, ["missing"])], [handler("a")]), /Unknown feature prerequisite/);
  assert.throws(() => createFeatureRegistry([definition("a", 0, ["b"]), definition("b", 0, ["a"])], [handler("a"), handler("b")]), /cycle/);
  assert.throws(() => createFeatureRegistry([a, { ...definition("b"), owns: ["a.child"] }], [handler("a"), handler("b")]), /ownership/);
});

test("SC-AER-001 known feature preferences remain stable", async () => {
  const { sharedFeatureRegistry, featureProblems } = await import(moduleURL("layout_features.js"));
  assert.equal(metadata.length, 5);
  const registry = sharedFeatureRegistry({ features: metadata });
  const profile = { algorithm: "layered", features: ["compound"], options: {} };
  const negotiated = registry.negotiate(profile);
  assert.deepEqual(negotiated.effective, ["compound"]);
  assert.equal(negotiated.diagnostics.length, 0);
  assert.deepEqual(profile.features, ["compound"]);
  assert.equal(featureProblems(profile, { features: metadata }).length, 0);
  assert.throws(() => registry.negotiate({ features: ["unknown"] }), /Unknown/);
});

test("SC-AER-008 compatibility and fallback hooks remain handler-owned", async () => {
  const { createFeatureRegistry, featureProblems } = await import(moduleURL("layout_features.js"));
  const a = { ...definition("a"), required_options: { routing: "SPLINES" } };
  const implementation = handler("a");
  implementation.validate = () => { throw new Error("invalid"); };
  implementation.fallback = (context, error, phase) => ({ ...context, error: error.message, phase });
  const registry = createFeatureRegistry([a], [implementation]);
  assert.deepEqual(registry.negotiate({ algorithm: "layered", options: { routing: "SPLINES" }, features: ["a"] }).effective, ["a"]);
  assert.equal(featureProblems({ algorithm: "mrtree", options: {}, features: ["a"] }, { features: [a] }).length, 2);
  assert.deepEqual(await registry.run("validate", {}, ["a"]), { error: "invalid", phase: "validate" });
});
