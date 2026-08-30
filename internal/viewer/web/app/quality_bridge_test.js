const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "quality.js"), "utf8")
  .replace(/^import .*?;\r?\n/gm, "") + `
function escapeHTML(value) { return String(value == null ? "" : value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;"); }
`;

import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(async function (quality) {
  const select = { innerHTML: "", value: "", disabled: false };
  const context = {
    embeddedExport: null,
    state: {
      qualityProfiles: [],
      qualityProfilesStatus: "loading",
      qualityProfilesError: "",
      qualityProfilesRequest: 0,
      activeQualityProfile: null,
      activeQualityProfileKey: "",
      qualityEvaluationStatus: "idle",
      qualityEvaluationError: "",
      qualityEvaluationRequest: 0,
      qualityReportStatus: "missing",
      qualityReportError: "",
      qualityReport: null,
      qualityReportCache: {},
      qualityReportRequest: 0,
      qualityFindingsPage: null,
      qualityFindingsCache: {},
      qualityEvidence: null
    },
    elements: {
      qualityProfileControl: { hidden: false },
      qualityProfile: select,
      qualityProfileStatus: { textContent: "", className: "" }
    }
  };

  let getCalls = 0;
  const profile = { name: "human", file_name: "human.json", profile_id: "profile:human", profile_version: "1.0.0", status: "available" };
  const api = {
    getJSON: async function (endpoint) {
      getCalls += 1;
      assert.equal(endpoint, "/v1/quality/profiles");
      return { status: "available", profiles: [profile] };
    },
    postJSON: async function (endpoint, body) {
      assert.equal(endpoint, "/v1/quality/evaluate");
      assert.equal(body.schema_version, "arch-view.quality-evaluation-request/v1");
      assert.equal(body.profile_id, "profile:human");
      assert.equal(body.profile_version, "1.0.0");
      assert.equal(body.scope, "scope-go");
      if (body.rule_bindings) assert.equal(body.rule_bindings[0].enabled, false);
      return { status: "available", scope_id: "scope-go", report: { profile_id: "profile:human", profile_version: "1.0.0", findings: [], coverage: [] } };
    },
    putJSON: async function (endpoint, body) {
      assert.equal(body.schema_version, "arch-view.quality-profile-save/v1");
      assert.equal(body.rule_bindings[0].enabled, false);
      if (endpoint === "/v1/quality/profiles/save") {
        assert.equal(body.profile_id, "profile:human");
        assert.equal(body.profile_version, "1.0.0");
        return { status: "saved", profile: profile };
      }
      assert.equal(endpoint, "/v1/quality/profiles/save-as");
      assert.equal(body.source_profile_id, "profile:human");
      assert.equal(body.profile_id, "profile:review");
      assert.equal(body.file_name, "review.json");
      return { status: "created", profile: { profile_id: "profile:review", profile_version: "1.0.0", file_name: "review.json" } };
    }
  };

  await quality.loadQualityProfiles(context, api);
  assert.equal(getCalls, 1);
  assert.equal(context.state.qualityProfilesStatus, "available");
  assert.equal(select.disabled, false);
  assert.match(select.innerHTML, /profile:human/);

  const result = await quality.evaluateQualityProfile(context, api, "profile:human", "1.0.0", "scope-go");
  assert.equal(result.scope_id, "scope-go");
  assert.equal(context.state.qualityEvaluationStatus, "available");
  assert.equal(context.state.qualityReportStatus, "available");
  assert.equal(context.state.qualityReportCache.all.status, "available");
  assert.equal(context.state.activeQualityProfile.profile_id, "profile:human");
  assert.match(context.elements.qualityProfileStatus.textContent, /Applied/);
  await quality.loadQualityReport(context, api);
  assert.equal(context.state.qualityReportStatus, "available");

  await quality.evaluateQualityProfile(context, api, "profile:human", "1.0.0", "scope-go", [{
    rule_id: "source:file.max-lines",
    rule_version: "1.0.0",
    enabled: false,
    parameters: { namespace: "rule-config:source-file-size", schema_version: "1.0.0", payload: { operator: "greater_than", limit: 500, unit: "unit:line" } },
    severity: "warning"
  }]);
  assert.equal(context.state.qualityEvaluationTemporary, true);
  assert.match(context.elements.qualityProfileStatus.textContent, /session/);

  await quality.saveQualityProfile(context, api, [{
    rule_id: "source:file.max-lines",
    rule_version: "1.0.0",
    enabled: false,
    parameters: { namespace: "rule-config:source-file-size", schema_version: "1.0.0", payload: {} },
    severity: "warning"
  }]);
  const created = await quality.saveQualityProfileAs(context, api, [{
    rule_id: "source:file.max-lines",
    rule_version: "1.0.0",
    enabled: false,
    parameters: { namespace: "rule-config:source-file-size", schema_version: "1.0.0", payload: {} },
    severity: "warning"
  }], "review.json", "profile:review", "1.0.0");
  assert.equal(created.status, "created");

  context.state.qualityProfiles = [{ name: "broken", file_name: "broken.json", status: "invalid", message: "invalid" }];
  context.state.qualityProfilesStatus = "invalid";
  quality.renderQualityProfileControl(context);
  assert.equal(select.disabled, true);
  assert.match(select.innerHTML, /unavailable/);
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
