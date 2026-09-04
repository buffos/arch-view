const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "okf_shape_choices.js")).href).then(async ({ loadShapeChoices, shapeChoicesMarkup }) => {
  const list = { outerHTML: "" };
  const details = { outerHTML: "" };
  const root = { innerHTML: "unchanged draft", querySelector: (selector) => selector === '#okf-shape-choices' ? list : details };
  await loadShapeChoices(root, { getExtensions: async () => ({ extensions: [
    { kind: "rule", id: "not.a.shape", version: "1" },
    { kind: "detail_renderer", id: "test.detail", version: "3", description: "Custom detail" },
    { kind: "shape", id: "test.triangle", version: "2", description: "Triangle <safe>" },
    { kind: "shape", id: 'test."unsafe', version: "1", description: "Quoted" }
  ] }) });
  assert.match(list.outerHTML, /test.triangle@2/);
  assert.match(list.outerHTML, /Triangle &lt;safe&gt;/);
  assert.doesNotMatch(list.outerHTML, /not.a.shape|value="test\."unsafe/);
  assert.equal(root.innerHTML, "unchanged draft");
  assert.match(details.outerHTML, /test.detail/);
  assert.match(details.outerHTML, /version 3/);
  assert.equal(shapeChoicesMarkup(root), list.outerHTML + details.outerHTML, "form rerender loses cached choices");
  const before = list.outerHTML;
  await loadShapeChoices(root, { getExtensions: async () => { throw new Error("offline"); } });
  assert.equal(list.outerHTML, before);
  assert.doesNotMatch(shapeChoicesMarkup({}), /test.triangle/);
}).catch((error) => { console.error(error); process.exitCode = 1; });
