const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

Promise.all(["okf_detail_form.js", "okf_profile_form.js"].map((name) => import(pathToFileURL(path.join(__dirname, name)).href))).then(([form, profile]) => {
  const inputs = { id: { value: "test.custom" }, version: { value: "2" }, parameters: { value: '{"field":"status"}' } };
  const root = { querySelector: (selector) => inputs[selector.match(/renderer\.(\w+)/)[1]] };
  const details = { show_raw_markdown: true };
  form.readDetailRenderer(root, details);
  assert.deepEqual(details.renderer, { id: "test.custom", version: "2", parameters: { field: "status" } });
  assert.equal(details.show_raw_markdown, true);
  const markup = form.detailRendererMarkup(details);
  assert.ok(markup.includes('value="test.custom"'));
  assert.ok(markup.includes('data-profile="details.renderer.parameters"'));
  assert.ok(!form.detailRendererMarkup({ renderer: { id: '<script>' } }).includes('<script>'));
  for (const invalid of ["[]", "null", "false", "{broken"]) {
    inputs.parameters.value = invalid;
    assert.throws(() => form.readDetailRenderer(root, details));
  }
  inputs.parameters.value = "{}";
  inputs.version.value = "";
  assert.throws(() => form.readDetailRenderer(root, details), /version/);
  inputs.id.value = "";
  form.readDetailRenderer(root, details);
  assert.equal(Object.hasOwn(details, "renderer"), false);
  const original = { details: { renderer: { id: "old", version: "1", parameters: { old: true } } }, extension: { keep: 1 } };
  const after = { details: { renderer: { id: "new", version: "2", parameters: {} } } };
  const changed = profile.profileFormChanges(original, original, after);
  assert.deepEqual(changed.details.renderer, after.details.renderer, "switching renderer must not leak parameters from the old renderer");
}).catch((error) => { console.error(error); process.exitCode = 1; });
