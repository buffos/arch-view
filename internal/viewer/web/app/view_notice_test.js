const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "view.js"), "utf8")
  .replace(/^import .*?;\r?\n/gm, "")
  .replace(/^export \{ classForState, formatList, referenceScopeLabel, referenceVisibilityLabel \};\r?\n/m, "");

function classList() {
  const values = new Set(["notice", "error"]);
  return {
    add: function (value) { values.add(value); },
    remove: function (value) { values.delete(value); },
    toggle: function (value, force) {
      if (force === undefined ? !values.has(value) : force) values.add(value);
      else values.delete(value);
      return values.has(value);
    },
    contains: function (value) { return values.has(value); }
  };
}

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (view) {
  const banner = {
    classList: classList(),
    hidden: true,
    textContent: "",
    role: "",
    setAttribute: function (name, value) { if (name === "role") this.role = value; }
  };
  const context = { elements: { errorBanner: banner } };

  view.showNotice(context, "The selected node is not present in this scope.");
  assert.equal(banner.hidden, false);
  assert.equal(banner.textContent, "The selected node is not present in this scope.");
  assert.equal(banner.classList.contains("info"), true);
  assert.equal(banner.classList.contains("error"), false);
  assert.equal(banner.role, "status");

  view.showError(context, "The inspection link is invalid.");
  assert.equal(banner.classList.contains("error"), true);
  assert.equal(banner.classList.contains("info"), false);
  assert.equal(banner.role, "alert");

  view.hideError(context);
  assert.equal(banner.hidden, true);
  assert.equal(banner.classList.contains("error"), false);
  assert.equal(banner.classList.contains("info"), false);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
