const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "quality.js"), "utf8")
  .replace(/^import .*?;\r?\n/gm, "") + `
function escapeHTML(value) { return String(value == null ? "" : value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;"); }
function classForState(value) { return String(value || "none").replace(/[^a-z0-9_-]/gi, "-"); }
`;

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(async function (quality) {
  const context = {
    aggregateEnabled: false,
    sourceEnabled: true,
    state: {
      activeScope: "",
      qualityReportStatus: "available",
      qualityReport: {
        findings: [
          { id: "finding:one", rule_id: "source:file.max-lines", status: "active", severity: "warning", assessment_kind: "exact", subject_ref: { kind: "file", id: "file-a" }, message: "File is over the threshold." },
          { id: "finding:two", rule_id: "signal:solid.srp", status: "active", severity: "info", assessment_kind: "signal", subject_ref: { kind: "symbol", id: "symbol-a" }, message: "Structural signal." },
          { id: "finding:three", rule_id: "source:file.max-lines", status: "suppressed", severity: "warning", assessment_kind: "exact", subject_ref: { kind: "file", id: "file-b" }, message: "Suppressed." }
        ],
        coverage: [{ rule_id: "source:file.max-lines", status: "observed" }]
      },
      qualityFindingsPage: null,
      qualityFindingsCache: {},
      qualityAffectedOnly: false
    }
  };

  const summary = quality.qualitySummary(context);
  assert.equal(summary.active, 2);
  assert.equal(summary.exact, 2);
  assert.equal(summary.signal, 1);
  assert.equal(summary.affectedFiles, 1);
  assert.deepEqual(quality.affectedFileIDs(context), ["file-a"]);

  const compact = quality.renderQualitySummary(context);
  assert.match(compact, /Quality checks/);
  assert.match(compact, /affected files/);
  assert.doesNotMatch(compact, /finding:one/);
  assert.doesNotMatch(compact, /file-a/);
  const overview = quality.renderQualityOverview(context);
  assert.match(overview, /Create baseline/);
  assert.match(overview, /Baseline this/);
  assert.match(overview, /data-quality-finding-rule/);
  assert.match(overview, /Files over the line threshold/);
  assert.match(overview, /Single-responsibility signal/);
  assert.deepEqual(quality.qualityFindingRuleOptions(context).map(function (option) { return option.id; }), ["source:file.max-lines", "signal:solid.srp"]);

  context.state.qualityFindingRuleFilter = "signal:solid.srp";
  const filteredOverview = quality.renderQualityOverview(context);
  assert.match(filteredOverview, /Structural signal\./);
  assert.doesNotMatch(filteredOverview, /File is over the threshold\./);
  context.state.qualityFindingRuleFilter = "";

  const filter = quality.qualityFileFilterMarkup(context);
  assert.match(filter, /Show files over threshold/);
  quality.toggleQualityAffectedOnly(context);
  assert.match(quality.qualityFileFilterMarkup(context), /Showing files over threshold/);

  let calls = 0;
  const requests = [];
  const api = {
    currentModelID: function () { return "model-1"; },
    getJSON: async function (endpoint) {
      calls += 1;
      requests.push(endpoint);
      if (endpoint.endsWith("/quality")) return { status: "available", report: context.state.qualityReport };
      if (endpoint.includes("cursor=next-page")) return { items: [context.state.qualityReport.findings[2]], total: 3, next_cursor: "", coverage: context.state.qualityReport.coverage };
      return { items: context.state.qualityReport.findings.slice(0, 2), total: 3, next_cursor: "next-page", coverage: context.state.qualityReport.coverage };
    }
  };
  const page = await quality.loadQualityFindings(context, api);
  assert.equal(page.total, 3);
  assert.equal(page.items.length, 2);
  assert.match(requests[0], /limit=25/);
  const callsAfterFirst = calls;
  await quality.loadQualityFindings(context, api, { limit: 25 });
  assert.equal(calls, callsAfterFirst, "bounded quality findings should be cached");
  const expanded = await quality.loadMoreQualityFindings(context, api);
  assert.equal(expanded.items.length, 3);
  assert.equal(expanded.next_cursor, "");
  assert.match(requests[1], /cursor=next-page/);
  const callsAfterExpansion = calls;
  await quality.loadMoreQualityFindings(context, api);
  assert.equal(calls, callsAfterExpansion, "quality findings should not fetch after the final page");

  context.state.qualityFindingRuleFilter = "signal:solid.srp";
  context.state.qualityFindingsPage = null;
  context.state.qualityFindingsCache = {};
  await quality.loadQualityFindings(context, api);
  assert.match(requests[2], /rule_id=signal%3Asolid\.srp/);
  context.state.qualityFindingRuleFilter = "";

  context.state.qualityFindingsPage = {
    items: context.state.qualityReport.findings.slice(0, 2),
    total: 3,
    next_cursor: "next-page",
    coverage: context.state.qualityReport.coverage
  };
  const pagedOverview = quality.renderQualityOverview(context);
  assert.match(pagedOverview, /Showing 2 of 3 quality findings/);
  assert.match(pagedOverview, /data-quality-findings-load-more/);

  context.state.qualityFindingsPage = null;
  context.state.qualityReport.findings = [];
  context.state.qualityReport.coverage = [
    { rule_id: "source:file.max-lines", status: "partial" },
    { rule_id: "source:callable.max-lines", status: "unknown" },
    { rule_id: "source:callable.max-cyclomatic-complexity", status: "unsupported" },
    { rule_id: "source:public-symbol.documentation", status: "not_evaluable" }
  ];
  const incomplete = quality.renderQualityOverview(context);
  assert.match(incomplete, /Partial/);
  assert.match(incomplete, /Unknown/);
  assert.match(incomplete, /Unsupported/);
  assert.match(incomplete, /Not Evaluable/);
  assert.match(incomplete, /Coverage incomplete/);
  assert.match(incomplete, /Coverage is counted per rule and source scope/);
  const coverageDetails = quality.qualityCoverageDetailsMarkup([
    { rule_id: "signal:solid.srp", status: "unsupported", reason: "the loaded source index does not provide SOLID structural facts", scope_id: "scope-a" },
    { rule_id: "signal:solid.srp", status: "unsupported", reason: "the loaded source index does not provide SOLID structural facts", scope_id: "scope-b" },
    { rule_id: "architecture:forbidden-dependency", status: "not_evaluable", reason: "no explicit forbidden-dependency constraint is configured", scope_id: "all" }
  ]);
  assert.match(coverageDetails, /Single-responsibility signal/);
  assert.match(coverageDetails, /2 report entries/);
  assert.match(coverageDetails, /loaded source index does not provide SOLID structural facts/);
  const coverage = quality.qualityCoverageSummaryMarkup({ observed: 2, partial: 1 }, 3);
  assert.match(coverage, /2 checks observed/);
  assert.match(coverage, /1 check partial/);
  assert.doesNotMatch(coverage, /Observed<\/span><strong>2<\/strong>/);
  assert.doesNotMatch(incomplete, /class="state-pill ok">Clear/);

  const baselineContext = {
    aggregateEnabled: false,
    state: {
      activeScope: "",
      activeQualityProfile: { profile_id: "profile:human", profile_version: "1.0.0" },
      qualityReport: {
        findings: [
          { id: "finding:one", rule_id: "source:file.max-lines", status: "active", subject_ref: { kind: "file", id: "file-a" }, message: "File is over the threshold." },
          { id: "finding:two", rule_id: "signal:solid.srp", status: "active", subject_ref: { kind: "symbol", id: "symbol-a" }, message: "Structural signal." }
        ]
      },
      qualityFindingsPage: null
    }
  };
  const baselineRequest = quality.buildQualityBaselineRequest(baselineContext, {
    baselineID: "baseline:human",
    revision: "1.0.0",
    fileName: "human.json",
    reason: "accepted for remediation",
    owner: "architecture",
    allActive: true,
    attachToProfile: true
  });
  assert.deepEqual(baselineRequest, {
    schema_version: "arch-view.quality-baseline-create/v1",
    profile_id: "profile:human",
    profile_version: "1.0.0",
    baseline_id: "baseline:human",
    revision: "1.0.0",
    file_name: "human.json",
    reason: "accepted for remediation",
    owner: "architecture",
    all_active: true,
    attach_to_profile: true
  });
  assert.deepEqual(quality.qualityBaselineCandidates(baselineContext).map(function (finding) { return finding.id; }), ["finding:one", "finding:two"]);
  const selectedBaselineRequest = quality.buildQualityBaselineRequest(baselineContext, {
    baselineID: "baseline:one",
    fileName: "one.json",
    reason: "accepted one finding",
    allActive: false,
    findingIDs: ["finding:one"],
    attachToProfile: false
  });
  assert.deepEqual(selectedBaselineRequest.finding_ids, ["finding:one"]);
  assert.equal(selectedBaselineRequest.all_active, false);
  assert.equal(selectedBaselineRequest.attach_to_profile, false);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
