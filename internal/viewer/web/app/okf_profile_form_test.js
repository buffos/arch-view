const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "okf_profile_form.js")).href).then(function (form) {
  const markup = form.profileFormMarkup({
    profile_id: "project:review",
    name: "Review",
    bases: ["builtin:neutral"],
    node_fields: [{ source: "frontmatter.state", label: "State", max_length: 20 }],
    state: { field: "state", mapping: { implemented: "implemented" }, roll_up: true },
    style: { default_token: "state.unknown", state_tokens: { implemented: "state.implemented" }, tokens: { "state.implemented": { fill: "#fff" } }, decorations: { root: { fill: "#4338ca" } } },
    layout: { algorithm: "layered", options: {} }
  }, { algorithms: [{ id: "layered", name: "Layered" }] });
  assert.match(markup, /Profile ID/);
  assert.match(markup, /frontmatter\.state/);
  assert.match(markup, /State token mapping/);
  assert.match(markup, /root decoration/);
  assert.match(markup, /Edit layout options/);
  const original = { profile_id: "project:child", name: "Child", bases: ["project:base"], extension: { retained: true } };
  const displayed = { ...original, hierarchy: { use_explicit: false }, navigation: { default_depth: 2 }, rules: [], node_fields: [] };
  assert.deepEqual(form.profileFormChanges(original, displayed, { ...displayed, name: "Renamed" }), { ...original, name: "Renamed" });
  assert.deepEqual(form.profileFormChanges(original, displayed, displayed), original);
  const changed = form.profileFormChanges(original, displayed, { ...displayed, navigation: { default_depth: 3 } });
  assert.deepEqual(changed.navigation, { default_depth: 3 });
  const inheritedMaps = { state: { mapping: { ready: "done", waiting: "pending" } }, style: { tokens: { red: { fill: "red" }, blue: { fill: "blue" } } } };
  const editedMaps = JSON.parse(JSON.stringify(inheritedMaps));
  editedMaps.state.mapping.ready = "complete";
  editedMaps.style.tokens.red.fill = "pink";
  assert.deepEqual(form.profileFormChanges({}, inheritedMaps, editedMaps), editedMaps, "map overrides retain unchanged inherited entries");
  assert.equal(Object.hasOwn(changed, "hierarchy"), false);
  const withRows = { node_fields: [{ source: "frontmatter.state" }] };
  assert.deepEqual(form.profileFormChanges(withRows, withRows, { node_fields: [] }), { node_fields: [] });
  const withMap = { style: { tokens: { first: { fill: "red" }, second: { fill: "blue" } } } };
  assert.deepEqual(form.profileFormChanges(withMap, withMap, { style: { tokens: { second: { fill: "blue" } } } }), { style: { tokens: { second: { fill: "blue" } } } });
  const controls = {
    profile_id: { value: original.profile_id }, name: { value: original.name },
    bases: { value: original.bases.join(",") }
  };
  const root = {
    innerHTML: "",
    querySelector: (selector) => controls[selector.match(/data-profile="([^"]+)"/)[1]] || null,
    querySelectorAll: () => []
  };
  form.renderProfileForm(root, original);
  controls.name.value = "Edited";
  const edited = form.readProfileForm(root, original);
  assert.deepEqual(edited, { ...original, name: "Edited" });
  controls.name.value = original.name;
  assert.deepEqual(form.readProfileForm(root, edited), original, "reverting a form edit restores sparse declaration");
  const layoutEdited = { ...edited, layout: { algorithm: "radial", options: {} } };
  assert.deepEqual(form.readProfileForm(root, layoutEdited).layout, layoutEdited.layout, "shared layout editor changes survive form synchronization");
  controls["navigation.default_depth"] = { value: "7" };
  form.renderProfileForm(root, original, { ...original, navigation: { default_depth: 7 } });
  assert.match(root.innerHTML, /navigation.default_depth[^>]+value="7"/);
  assert.deepEqual(form.readProfileForm(root, original), original, "inherited display stays sparse when untouched");
  controls["navigation.default_depth"].value = "8";
  assert.equal(form.readProfileForm(root, original).navigation.default_depth, 8);
  controls["navigation.default_depth"].value = "1.5";
  assert.throws(() => form.readProfileForm(root, original), /whole number/);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
