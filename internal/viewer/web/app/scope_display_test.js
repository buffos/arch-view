const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "utils.js"), "utf8");
import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (utils) {
  assert.equal(utils.displayProjectRoot("."), "Repository root");
  assert.equal(utils.formatLanguage("typescript"), "TypeScript");

  const allContext = {
    state: {
      activeScope: "all",
      model: {
        modules: [
          { id: "scope-go::main", language: "go" },
          { id: "scope-ts::main", language: "typescript" }
        ]
      },
      scene: { project: { language: "mixed" } }
    }
  };
  const mixedNode = { kind: "group", module_ids: ["scope-go::main", "scope-ts::main"] };
  assert.equal(utils.nodeLanguageBadge(allContext, mixedNode), "multi");
  assert.equal(utils.nodeLanguageText(allContext, mixedNode), "Go + TypeScript");

  const scopeContext = {
    state: {
      activeScope: "scope-python",
      model: { modules: [{ id: "scope-python::app", language: "python" }] },
      scene: { project: { language: "python" } }
    }
  };
  assert.equal(utils.nodeLanguageBadge(scopeContext, { kind: "group", module_ids: ["app"] }), "python");
  assert.equal(utils.nodeLanguageText(scopeContext, { kind: "group", module_ids: ["app"] }), "Python");
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
