const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "quality.js"), "utf8")
  .replace(/^import .*?;\r?\n/gm, "") + `
function escapeHTML(value) { return String(value == null ? "" : value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;"); }
`;

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (quality) {
  const context = {
    state: {
      qualityRuleCatalog: [
        {
          id: "source:file.max-lines",
          version: "1.0.0",
          assessment_kind: "exact",
          default_severity: "warning",
          description: "Flag files over the configured line limit.",
          enabled: true,
          configured: true,
          parameters: { namespace: "rule-config:source-file-size", schema_version: "1.0.0", payload: { limit: 500 } }
        },
        {
          id: "signal:solid.srp",
          version: "1.0.0",
          assessment_kind: "signal",
          default_severity: "info",
          description: "Advisory structural signal.",
          enabled: false,
          configured: false,
          parameters: { namespace: "rule-config:solid-signal", schema_version: "1.0.0", payload: {} }
        },
        {
          id: "architecture:forbidden-dependency",
          version: "1.0.0",
          assessment_kind: "exact",
          default_severity: "error",
          description: "Report only canonical dependency edges selected by explicit forbidden-dependency policy.",
          enabled: false,
          configured: false,
          parameters: { namespace: "rule-config:architecture-constraint", schema_version: "1.0.0", payload: {} }
        }
      ],
      qualityRuleCatalogStatus: "available",
      qualityRuleSearch: ""
    }
  };

  const bindings = quality.qualityRuleBindings(context);
  assert.deepEqual(bindings.map(function (binding) { return [binding.rule_id, binding.enabled]; }), [
    ["source:file.max-lines", true],
    ["signal:solid.srp", false],
    ["architecture:forbidden-dependency", false]
  ]);

  const markup = quality.renderQualityRuleCatalog(context);
  assert.match(markup, /Quality rules/);
  assert.match(markup, /Files over the line threshold/);
  assert.match(markup, /Single-responsibility signal/);
  assert.match(markup, /Exact/);
  assert.match(markup, /Advisory signal/);
  assert.match(markup, /data-quality-rule-toggle/);
  assert.match(markup, /checked/);
  assert.match(markup, /Available but off/);
  assert.match(markup, /Checks only dependency edges that you explicitly marked as forbidden/);
  assert.doesNotMatch(markup, /<code>source:file\.max-lines<\/code>/);
  quality.setQualityRuleEnabled(context, "signal:solid.srp", "1.0.0", true);
  assert.equal(quality.qualityRuleBindings(context)[1].enabled, true);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
