const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "okf_markup.js"), "utf8");

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (markup) {
  assert.equal(markup.escapeOKF(`<script>alert("x")</script>`), "&lt;script&gt;alert(&quot;x&quot;)&lt;/script&gt;");
  const diagnostics = markup.diagnosticMarkup([{ code: "okf_unsafe_link", message: "<blocked>" }]);
  assert.match(diagnostics, /okf_unsafe_link/);
  assert.doesNotMatch(diagnostics, /<blocked>/);
  const grouped = markup.diagnosticMarkup([
    { code: "okf_bundle_boundary_violation", message: "Local link escapes the selected bundle boundary." },
    { code: "okf_bundle_boundary_violation", message: "Local link escapes the selected bundle boundary." }
  ]);
  assert.match(grouped, /2×/);
  assert.equal((grouped.match(/Local link escapes/g) || []).length, 1, "repeated diagnostics should be grouped");
  const fullReport = markup.diagnosticMarkup(Array.from({ length: 17 }, (_, i) => ({
    code: "boundary", message: "Repeated warning", path: "source-" + i,
    bundle_id: "bundle", concept_id: "concept-" + i, operation_id: "request-1",
    recovery: "Review <unsafe> paths", details: { target: "<script>" }
  })));
  assert.equal((fullReport.match(/Repeated warning/g) || []).length, 1);
  assert.equal((fullReport.match(/Review &lt;unsafe&gt; paths/g) || []).length, 1);
  assert.match(fullReport, /source-16/);
  assert.match(fullReport, /concept-16/);
  assert.match(fullReport, /request-1/);
  assert.match(fullReport, /Affected sources and diagnostic details/);
  assert.doesNotMatch(fullReport, /<script>|<unsafe>/);

  const container = { innerHTML: "" };
  markup.renderDetail(container, {
    concept_id: "concept-1",
    overview: { title: "Concept", type: "reference", source_path: "concept.md" },
    mapped_metadata: {},
    rendered_markdown: { content: "<p>safe backend output</p>" },
    raw_markdown: "# Concept"
  });
  assert.match(container.innerHTML, /safe backend output/);
  assert.match(container.innerHTML, /Raw Markdown/);
  markup.renderDetail(container, { concept_id: "hidden", rendered_markdown: { content: "<p>Visible</p>" } });
  assert.doesNotMatch(container.innerHTML, /Raw Markdown|Unknown frontmatter/);
  assert.match(container.innerHTML, /Visible/);
  markup.renderDetail(container, {
    concept_id: "aggregate",
    declared_state: "specified",
    effective_state: "implemented",
    containment: { parent: "root", children: ["child"] },
    semantic_links: [{ raw_target: "<script>unsafe</script>", safe: false, resolved: false }],
    provenance: [{ source: "frontmatter", path: "aggregate.md" }],
    frontmatter: { private_unknown: "must stay hidden by profile" }
  });
  assert.match(container.innerHTML, /Declared state<\/dt><dd>specified/);
  assert.match(container.innerHTML, /Effective state<\/dt><dd>implemented/);
  assert.match(container.innerHTML, /Containment/);
  assert.match(container.innerHTML, /Semantic link outcomes/);
  assert.match(container.innerHTML, /Source provenance/);
  assert.match(container.innerHTML, /&lt;script&gt;/);
  assert.doesNotMatch(container.innerHTML, /<script>|must stay hidden by profile/);
  markup.renderDetail(container, { concept_id: "plain" });
  assert.doesNotMatch(container.innerHTML, /State interpretation|Source provenance|Semantic link outcomes/);

  const list = {
    innerHTML: "",
    querySelectorAll: () => []
  };
  markup.renderAccessibleItems(list, { nodes: [{ id: "x", label: "Example", presentation_fields: [{ label: "state", value: "implemented" }] }] }, () => {});
  assert.match(list.innerHTML, /Example/);
  assert.match(list.innerHTML, /state: implemented/);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
