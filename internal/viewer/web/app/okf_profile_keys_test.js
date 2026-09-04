const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "okf_profile_form.js")).href).then(({ readProfileForm }) => {
  for (const list of ["state_mapping", "style_state_mapping", "style_tokens"]) {
    const row = { querySelector: (selector) => {
      const key = selector.match(/data-repeat-value="([^"]+)"/)[1];
      return { value: ({ source: "__proto__", target: "custom", id: "__proto__", fill: "red" })[key] || "" };
    } };
    const rows = [row];
    const root = {
      querySelector: () => null,
      querySelectorAll: (selector) => selector === '[data-repeat-row="' + list + '"]' ? rows : []
    };
    const result = readProfileForm(root);
    const mapping = list === "state_mapping" ? result.state.mapping : list === "style_state_mapping" ? result.style.state_tokens : result.style.tokens;
    assert.equal(Object.getPrototypeOf(mapping), Object.prototype);
    assert.equal(Object.hasOwn(mapping, "__proto__"), true);
    assert.equal(Object.hasOwn(JSON.parse(JSON.stringify(mapping)), "__proto__"), true);
    rows.push(row);
    assert.throws(() => readProfileForm(root), /Duplicate mapping or token key/);
  }
}).catch((error) => { console.error(error); process.exitCode = 1; });
