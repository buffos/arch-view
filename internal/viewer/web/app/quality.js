import { escapeHTML } from "./utils.js";

export const qualityPageLimit = 25;
export const qualityEvaluationRequestSchemaVersion = "arch-view.quality-evaluation-request/v1";
export const qualityBaselineCreateSchemaVersion = "arch-view.quality-baseline-create/v1";

const coverageStateOrder = Object.freeze(["observed", "absent", "partial", "unknown", "unsupported", "not_evaluable"]);

const ruleLabels = Object.freeze({
  "source:file.max-lines": "Files over the line threshold",
  "source:callable.max-lines": "Callables over the line threshold",
  "source:callable.max-cyclomatic-complexity": "Callable complexity",
  "source:callable.max-nesting-depth": "Callable nesting depth",
  "source:public-symbol.documentation": "Public-symbol documentation",
  "architecture:module.max-efferent-coupling": "Module outgoing coupling",
  "architecture:module.max-afferent-coupling": "Module incoming coupling",
  "architecture:no-cycles": "Architecture cycles",
  "architecture:forbidden-dependency": "Forbidden dependency",
  "architecture:layer-direction": "Layer direction",
  "signal:solid.srp": "Single-responsibility signal",
  "signal:solid.ocp": "Open/closed signal",
  "signal:solid.lsp": "Substitutability signal",
  "signal:solid.isp": "Interface-segregation signal",
  "signal:solid.dip": "Dependency-inversion signal"
});

const ruleExplanations = Object.freeze({
  "architecture:module.max-efferent-coupling": "Checks how many different modules this module depends on. A high count can make changes ripple across many parts of the system.",
  "architecture:module.max-afferent-coupling": "Checks how many different modules depend on this module. A high count means changes here may affect many consumers.",
  "architecture:no-cycles": "Checks whether dependencies form a loop, such as A → B → A. Loops make ownership and change order harder to understand.",
  "architecture:forbidden-dependency": "Checks only dependency edges that you explicitly marked as forbidden. The profile chooses which modules or patterns are forbidden; this check does not invent that policy.",
  "architecture:layer-direction": "Checks whether dependencies follow the layer direction you explicitly configured. It reports only edges that cross a configured layer boundary the wrong way."
});

const ruleCategories = Object.freeze([
  { prefix: "source:", label: "Source facts" },
  { prefix: "architecture:", label: "Architecture" },
  { prefix: "signal:", label: "Design signals" }
]);

const assessmentLabels = Object.freeze({ exact: "Exact check", signal: "Advisory signal" });

function reportFromContext(context) {
  if (!context || !context.state) return null;
  if (context.state.qualityReport) return context.state.qualityReport;
  const model = context.state.model;
  return model && model.quality_report || null;
}

function activeScope(context) {
  if (!context || !context.aggregateEnabled) return "";
  const scope = String(context.state && context.state.activeScope || "");
  return scope && scope !== "all" ? scope : "";
}

function qualityEndpoint(context, api, suffix, options) {
  const query = new URLSearchParams();
  const scope = activeScope(context);
  if (scope) query.set("scope", scope);
  Object.keys(options || {}).forEach(function (key) {
    const value = options[key];
    if (value == null || value === "") return;
    query.set(key, String(value));
  });
  const queryText = query.toString();
  return "/v1/models/" + encodeURIComponent(api.currentModelID()) + "/quality" + suffix + (queryText ? "?" + queryText : "");
}

function reportCacheKey(context) {
  return activeScope(context) || "all";
}

function applyQualityReport(context, envelope) {
  const value = envelope || {};
  context.state.qualityReportEnvelope = value;
  context.state.qualityReport = value.report || null;
  context.state.qualityReportStatus = value.status === "available" && value.report ? "available" : value.status || "missing";
  context.state.qualityReportError = value.message || "";
  context.state.qualityFindingsPage = null;
  context.state.qualityFindingsCache = {};
  context.state.qualityFindingsError = "";
  context.state.qualityEvidence = null;
  if (value.report && value.report.profile_id && value.report.profile_version) {
    context.state.activeQualityProfile = { profile_id: value.report.profile_id, profile_version: value.report.profile_version };
    context.state.activeQualityProfileKey = qualityProfileKey(value.report.profile_id, value.report.profile_version);
  }
  renderQualityProfileControl(context);
}

function qualityProfileKey(profileID, profileVersion) {
  return String(profileID || "") + "\u0000" + String(profileVersion || "");
}

function qualityProfileSelectionValue(profile) {
  return encodeURIComponent(String(profile && profile.profile_id || "")) + "|" + encodeURIComponent(String(profile && profile.profile_version || ""));
}

function qualityProfileStatusText(context) {
  const state = context.state;
  if (state.qualityEvaluationStatus === "loading") return "Evaluating the loaded model…";
  if (state.qualityEvaluationStatus === "error") return state.qualityEvaluationError || "Quality evaluation failed.";
  if (state.qualityProfilesStatus === "loading") return "Looking in quality-profiles/…";
  if (state.qualityProfilesError) return state.qualityProfilesError;
  if (state.qualityProfilesStatus === "unavailable") return "Available only in a project-backed viewer.";
  if (state.qualityProfilesStatus === "empty") return "Add JSON profiles to quality-profiles/.";
  if (state.qualityProfilesStatus === "invalid") return "No usable profile was found.";
  if (state.qualityProfilesStatus === "partial") return "Some profile files are invalid.";
  if (state.qualityEvaluationTemporary) return "Applied for this session; the profile file was not changed.";
  const activeDescriptor = (state.qualityProfiles || []).find(function (profile) {
    return state.activeQualityProfile && profile && profile.profile_id === state.activeQualityProfile.profile_id && profile.profile_version === state.activeQualityProfile.profile_version;
  });
  if (activeDescriptor && activeDescriptor.baseline_id) return "Applied to the current analysis scope; baseline attached.";
  if (state.activeQualityProfile) return "Applied to the current analysis scope.";
  return "Choose a profile to run checks on the loaded model.";
}

export function renderQualityProfileControl(context) {
  if (!context || !context.elements || !context.elements.qualityProfileControl || !context.elements.qualityProfile || !context.elements.qualityProfileStatus) return;
  const control = context.elements.qualityProfileControl;
  const select = context.elements.qualityProfile;
  const status = context.elements.qualityProfileStatus;
  const configure = context.elements.qualityProfileConfigure;
  const state = context.state;
  control.hidden = Boolean(context.embeddedExport);
  if (control.hidden) return;
  const profiles = (context.state.qualityProfiles || []).filter(function (profile) { return profile && profile.status === "available" && profile.profile_id && profile.profile_version; });
  const selectedKey = context.state.activeQualityProfileKey || "";
  select.innerHTML = '<option value="">No profile selected</option>' + profiles.map(function (profile) {
    const value = qualityProfileSelectionValue(profile);
    const label = (profile.name || profile.file_name || profile.profile_id) + (profile.profile_version ? " · " + profile.profile_version : "");
    return '<option value="' + escapeHTML(value) + '" data-profile-id="' + escapeHTML(profile.profile_id) + '" data-profile-version="' + escapeHTML(profile.profile_version) + '">' + escapeHTML(label) + '</option>';
  }).join("") + (context.state.qualityProfilesStatus === "invalid" || context.state.qualityProfilesStatus === "partial" ? (context.state.qualityProfiles || []).filter(function (profile) { return profile && profile.status !== "available"; }).map(function (profile) { return '<option disabled>' + escapeHTML((profile.name || profile.file_name || "Invalid profile") + " · unavailable") + "</option>"; }).join("") : "");
  const selected = profiles.find(function (profile) { return qualityProfileKey(profile.profile_id, profile.profile_version) === selectedKey; });
  select.value = selected ? qualityProfileSelectionValue(selected) : "";
  select.disabled = state.qualityProfilesStatus === "loading" || !profiles.length || state.qualityEvaluationStatus === "loading";
  status.textContent = qualityProfileStatusText(context);
  status.className = "quality-profile-status" + ((state.qualityEvaluationStatus === "error" || state.qualityProfilesError || state.qualityProfilesStatus === "invalid") ? " quality-profile-status-error" : "");
  status.title = status.textContent;
  if (configure) {
    configure.hidden = !state.activeQualityProfile || state.qualityRuleCatalogStatus === "unavailable";
    configure.disabled = state.qualityRuleCatalogStatus === "loading" || state.qualityEvaluationStatus === "loading";
  }
}

export async function loadQualityProfiles(context, api) {
  if (!context || !context.state || !api || context.embeddedExport) {
    if (context && context.state) context.state.qualityProfilesStatus = "unavailable";
    renderQualityProfileControl(context);
    return null;
  }
  const request = (context.state.qualityProfilesRequest || 0) + 1;
  context.state.qualityProfilesRequest = request;
  context.state.qualityProfilesStatus = "loading";
  context.state.qualityProfilesError = "";
  renderQualityProfileControl(context);
  try {
    const response = await api.getJSON("/v1/quality/profiles");
    if (request !== context.state.qualityProfilesRequest) return response;
    context.state.qualityProfiles = Array.isArray(response.profiles) ? response.profiles : [];
    context.state.qualityProfilesStatus = response.status || (context.state.qualityProfiles.length ? "available" : "empty");
    context.state.qualityProfilesError = response.status === "unavailable" ? response.message || "Quality profiles are unavailable." : "";
    renderQualityProfileControl(context);
    return response;
  } catch (error) {
    if (request !== context.state.qualityProfilesRequest) return null;
    context.state.qualityProfiles = [];
    context.state.qualityProfilesStatus = "unavailable";
    context.state.qualityProfilesError = error && error.message || "Quality profiles are unavailable.";
    renderQualityProfileControl(context);
    return null;
  }
}

export async function evaluateQualityProfile(context, api, profileID, profileVersion, scope, ruleBindings) {
  if (!context || !context.state || !api || !profileID || !profileVersion) return null;
  const request = (context.state.qualityEvaluationRequest || 0) + 1;
  context.state.qualityEvaluationRequest = request;
  context.state.qualityEvaluationStatus = "loading";
  context.state.qualityEvaluationError = "";
  context.state.qualityReportStatus = "loading";
  context.state.qualityReportError = "";
  context.state.qualityFindingsPage = null;
  context.state.qualityFindingsCache = {};
  context.state.qualityEvidence = null;
  renderQualityProfileControl(context);
  try {
    const requestBody = {
      schema_version: qualityEvaluationRequestSchemaVersion,
      profile_id: profileID,
      profile_version: profileVersion,
      scope: scope || "all"
    };
    if (ruleBindings !== undefined && ruleBindings !== null) requestBody.rule_bindings = ruleBindings;
    const response = await api.postJSON("/v1/quality/evaluate", requestBody);
    if (request !== context.state.qualityEvaluationRequest) return response;
    applyQualityReport(context, response);
    const cache = context.state.qualityReportCache || {};
    cache[reportCacheKey(context)] = response;
    context.state.qualityReportCache = cache;
    context.state.qualityEvaluationStatus = "available";
    context.state.qualityEvaluationTemporary = Array.isArray(ruleBindings);
    context.state.qualityEvaluationError = "";
    renderQualityProfileControl(context);
    return response;
  } catch (error) {
    if (request !== context.state.qualityEvaluationRequest) return null;
    context.state.qualityEvaluationStatus = "error";
    context.state.qualityEvaluationTemporary = false;
    context.state.qualityEvaluationError = error && error.message || "Quality evaluation failed.";
    context.state.qualityReportStatus = "unavailable";
    context.state.qualityReportError = context.state.qualityEvaluationError;
    context.state.qualityReport = null;
    renderQualityProfileControl(context);
    return null;
  }
}

export async function saveQualityProfile(context, api, ruleBindings) {
  const profile = context && context.state && context.state.activeQualityProfile;
  if (!profile || !api || !api.putJSON) return null;
  return api.putJSON("/v1/quality/profiles/save", {
    schema_version: "arch-view.quality-profile-save/v1",
    profile_id: profile.profile_id,
    profile_version: profile.profile_version,
    rule_bindings: Array.isArray(ruleBindings) ? ruleBindings : []
  });
}

export async function saveQualityProfileAs(context, api, ruleBindings, fileName, profileID, profileVersion) {
  const source = context && context.state && context.state.activeQualityProfile;
  if (!source || !api || !api.putJSON) return null;
  return api.putJSON("/v1/quality/profiles/save-as", {
    schema_version: "arch-view.quality-profile-save/v1",
    profile_id: String(profileID || "").trim(),
    profile_version: String(profileVersion || "").trim(),
    source_profile_id: source.profile_id,
    source_profile_version: source.profile_version,
    file_name: String(fileName || "").trim(),
    rule_bindings: Array.isArray(ruleBindings) ? ruleBindings : []
  });
}

export function qualityReport(context) {
  return reportFromContext(context);
}

export function qualityReportStatus(context) {
  if (!context || !context.state) return "missing";
  return context.state.qualityReportStatus || (reportFromContext(context) ? "available" : "missing");
}

export async function loadQualityReport(context, api) {
  if (!context || !context.state || !api) return null;
  const key = reportCacheKey(context);
  const cache = context.state.qualityReportCache || {};
  context.state.qualityReportCache = cache;
  if (cache[key]) {
    applyQualityReport(context, cache[key]);
    return cache[key];
  }
  const request = (context.state.qualityReportRequest || 0) + 1;
  context.state.qualityReportRequest = request;
  context.state.qualityReportStatus = "loading";
  context.state.qualityReportError = "";
  try {
    const suffix = activeScope(context) ? "?scope=" + encodeURIComponent(activeScope(context)) : "";
    const envelope = await api.getJSON("/v1/models/" + encodeURIComponent(api.currentModelID()) + "/quality" + suffix);
    if (request !== context.state.qualityReportRequest) return envelope;
    cache[key] = envelope;
    applyQualityReport(context, envelope);
    return envelope;
  } catch (error) {
    if (request !== context.state.qualityReportRequest) return null;
    context.state.qualityReportStatus = "unavailable";
    context.state.qualityReportError = error && error.message || "Quality checks are unavailable for this model revision.";
    context.state.qualityReport = null;
    return null;
  }
}

function findingScopeMatches(context, finding) {
  const scope = activeScope(context);
  return !scope || !finding || !finding.subject_ref || !finding.subject_ref.scope_id || finding.subject_ref.scope_id === scope;
}

function coverageScopeMatches(context, coverage) {
  const scope = activeScope(context);
  if (!scope) return true;
  const evidenceIDs = coverage && coverage.provenance && coverage.provenance.evidence_ids || [];
  if (!evidenceIDs.length) return true;
  const prefix = "scope:" + scope + ":";
  return evidenceIDs.some(function (evidenceID) { return evidenceID === "scope:" + scope || String(evidenceID).indexOf(prefix) === 0; });
}

function reportFindingsFromContext(context) {
  const report = reportFromContext(context);
  return report && Array.isArray(report.findings) ? report.findings : [];
}

function findingsFromContext(context) {
  const page = context && context.state && context.state.qualityFindingsPage;
  if (page && Array.isArray(page.items)) return page.items;
  return reportFindingsFromContext(context);
}

export function qualityBaselineCandidates(context) {
  const reportFindings = reportFindingsFromContext(context);
  const source = reportFindings.length ? reportFindings : findingsFromContext(context);
  return source.filter(function (finding) {
    return findingScopeMatches(context, finding) && String(finding && finding.status || "active") === "active";
  });
}

export function buildQualityBaselineRequest(context, options) {
  options = options || {};
  const profile = context && context.state && context.state.activeQualityProfile || {};
  const allActive = options.allActive !== false;
  const request = {
    schema_version: qualityBaselineCreateSchemaVersion,
    profile_id: String(profile.profile_id || "").trim(),
    profile_version: String(profile.profile_version || "").trim(),
    baseline_id: String(options.baselineID || "").trim(),
    revision: String(options.revision || "").trim(),
    file_name: String(options.fileName || "").trim(),
    reason: String(options.reason || "").trim(),
    owner: String(options.owner || "").trim(),
    all_active: allActive,
    attach_to_profile: options.attachToProfile !== false
  };
  const scope = activeScope(context);
  if (scope) request.scope = scope;
  if (!allActive) request.finding_ids = Array.isArray(options.findingIDs) ? options.findingIDs.map(function (value) { return String(value || "").trim(); }).filter(Boolean) : [];
  return request;
}

function baselineFindingLabel(finding) {
  const subject = finding && finding.subject_ref || {};
  const subjectLabel = subject.kind === "file" ? "File finding" : subject.kind === "symbol" ? "Symbol finding" : subject.kind === "module" ? "Module finding" : "Reported finding";
  return [ruleLabel(finding && finding.rule_id), subjectLabel, finding && finding.message || ""].filter(Boolean).join(" · ");
}

export function renderQualityBaselineSelection(context) {
  const candidates = qualityBaselineCandidates(context);
  if (!candidates.length) return '<div class="quality-baseline-selection-empty"><span class="state-pill neutral">No active findings</span><p>There are no active findings in the current report to add to a baseline.</p></div>';
  if (context.state.qualityBaselineAllActive !== false) {
    return '<div class="quality-baseline-selection-summary"><strong>All ' + escapeHTML(String(candidates.length)) + ' active findings</strong><span>New findings remain visible when their exact versions do not match this baseline.</span></div>';
  }
  const selected = new Set(Array.isArray(context.state.qualityBaselineSelectedFindings) ? context.state.qualityBaselineSelectedFindings : []);
  const visible = candidates.slice(0, qualityPageLimit);
  const rows = visible.map(function (finding) {
    const value = finding && (finding.id || finding.finding_key) || "";
    return '<label class="quality-baseline-finding"><input type="checkbox" data-quality-baseline-finding value="' + escapeHTML(value) + '"' + (selected.has(value) ? " checked" : "") + '><span>' + escapeHTML(baselineFindingLabel(finding)) + '</span></label>';
  }).join("");
  const note = candidates.length > visible.length ? '<p class="muted">Showing the first ' + qualityPageLimit + ' active findings. For another finding, open it from the report and choose “Baseline this”.</p>' : "";
  return '<div class="quality-baseline-finding-list">' + rows + '</div>' + note;
}

export async function createQualityBaseline(context, api, options) {
  if (!context || !context.state || !api || !api.postJSON) return null;
  context.state.qualityBaselineStatus = "loading";
  context.state.qualityBaselineError = "";
  try {
    const response = await api.postJSON("/v1/quality/baselines/create", buildQualityBaselineRequest(context, options));
    context.state.qualityBaselineStatus = response && response.status || "created";
    context.state.qualityBaselineMessage = response && response.message || "Quality baseline created.";
    return response;
  } catch (error) {
    context.state.qualityBaselineStatus = "error";
    context.state.qualityBaselineError = error && error.message || "The quality baseline could not be created.";
    throw error;
  }
}

function qualityFindingsCacheKey(context, requested) {
  return reportCacheKey(context) + "|" + Object.keys(requested).sort().map(function (name) { return name + "=" + requested[name]; }).join("&");
}

function findingIdentity(finding, index) {
  if (finding && finding.id) return "id:" + finding.id;
  if (finding && finding.finding_key) return "key:" + finding.finding_key;
  return "index:" + index;
}

function mergeQualityFindingsPage(existing, page) {
  const previous = existing && Array.isArray(existing.items) ? existing.items : [];
  const identities = new Set(previous.map(findingIdentity));
  const nextItems = previous.slice();
  (page && page.items || []).forEach(function (finding, index) {
    const identity = findingIdentity(finding, previous.length + index);
    if (identities.has(identity)) return;
    identities.add(identity);
    nextItems.push(finding);
  });
  return Object.assign({}, page, {
    items: nextItems,
    total: page && page.total != null ? page.total : existing && existing.total || nextItems.length,
    next_cursor: page && page.next_cursor || "",
    coverage: page && Array.isArray(page.coverage) && page.coverage.length ? page.coverage : existing && existing.coverage || []
  });
}

export async function loadQualityFindings(context, api, options) {
  if (!context || !context.state || !api) return null;
  const requested = Object.assign({ limit: qualityPageLimit }, options || {});
  if (requested.rule_id == null) {
    const ruleID = qualityFindingRuleFilter(context);
    if (ruleID) requested.rule_id = ruleID;
  }
  const cache = context.state.qualityFindingsCache || {};
  context.state.qualityFindingsCache = cache;
  const key = qualityFindingsCacheKey(context, requested);
  if (cache[key]) {
    context.state.qualityFindingsError = "";
    context.state.qualityFindingsPage = cache[key];
    return cache[key];
  }
  context.state.qualityFindingsLoading = true;
  context.state.qualityFindingsError = "";
  try {
    const page = await api.getJSON(qualityEndpoint(context, api, "/findings", requested));
    cache[key] = page;
    context.state.qualityFindingsPage = page;
    return page;
  } catch (error) {
    context.state.qualityFindingsError = error && error.message || "Quality findings are unavailable.";
    throw error;
  } finally {
    context.state.qualityFindingsLoading = false;
  }
}

export async function loadMoreQualityFindings(context, api) {
  if (!context || !context.state || !api) return null;
  const existing = context.state.qualityFindingsPage;
  if (!existing) return loadQualityFindings(context, api);
  if (!existing.next_cursor) return existing;
  const page = await loadQualityFindings(context, api, { limit: qualityPageLimit, cursor: existing.next_cursor });
  const merged = mergeQualityFindingsPage(existing, page || { items: [], total: existing.total, coverage: existing.coverage });
  context.state.qualityFindingsPage = merged;
  return merged;
}

export function qualitySummary(context) {
  const report = reportFromContext(context);
  const findings = reportFindingsFromContext(context).filter(function (finding) { return findingScopeMatches(context, finding); });
  const summary = { total: findings.length, active: 0, exact: 0, signal: 0, suppressed: 0, baseline: 0, resolved: 0, notEvaluable: 0, warning: 0, error: 0, blocker: 0, affectedFiles: 0, partialCoverage: 0, coverageStates: { observed: 0, absent: 0, partial: 0, unknown: 0, unsupported: 0, not_evaluable: 0 } };
  const files = new Set();
  findings.forEach(function (finding) {
    const status = String(finding.status || "active");
    if (status === "active") summary.active++;
    if (finding.assessment_kind === "exact") summary.exact++;
    if (finding.assessment_kind === "signal") summary.signal++;
    if (status === "suppressed") summary.suppressed++;
    if (status === "baseline") summary.baseline++;
    if (status === "resolved") summary.resolved++;
    if (status === "not_evaluable") summary.notEvaluable++;
    if (status === "active" && finding.severity === "warning") summary.warning++;
    if (status === "active" && finding.severity === "error") summary.error++;
    if (status === "active" && finding.severity === "blocker") summary.blocker++;
    if (status === "active" && finding.rule_id === "source:file.max-lines" && finding.subject_ref && finding.subject_ref.kind === "file") files.add(finding.subject_ref.id);
  });
  summary.affectedFiles = files.size;
  (report && report.coverage || []).filter(function (coverage) { return coverageScopeMatches(context, coverage); }).forEach(function (coverage) {
    const state = String(coverage.status || "unknown").toLowerCase();
    if (Object.prototype.hasOwnProperty.call(summary.coverageStates, state)) summary.coverageStates[state]++;
    else summary.coverageStates.unknown++;
    if (state !== "observed") summary.partialCoverage++;
  });
  return summary;
}

export function affectedFileIDs(context) {
  const result = new Set();
  reportFindingsFromContext(context).forEach(function (finding) {
    if (!findingScopeMatches(context, finding) || finding.rule_id !== "source:file.max-lines" || finding.status !== "active") return;
    if (finding.subject_ref && finding.subject_ref.kind === "file" && finding.subject_ref.id) result.add(finding.subject_ref.id);
  });
  return Array.from(result).sort();
}

export function qualityAffectedOnly(context) {
  return Boolean(context && context.state && context.state.qualityAffectedOnly);
}

export function toggleQualityAffectedOnly(context) {
  if (!context || !context.state) return false;
  context.state.qualityAffectedOnly = !context.state.qualityAffectedOnly;
  return context.state.qualityAffectedOnly;
}

export function qualityFileFilterMarkup(context) {
  if (qualityReportStatus(context) !== "available") return "";
  const affected = affectedFileIDs(context);
  const active = qualityAffectedOnly(context);
  const label = active ? "Showing files over threshold" : "Show files over threshold";
  return '<button type="button" class="button tertiary quality-file-filter" data-quality-affected-only aria-pressed="' + String(active) + '"' + (!affected.length ? " disabled" : "") + '>' + escapeHTML(label) + (affected.length ? " · " + affected.length : "") + "</button>";
}

export function qualityFindingRuleOptions(context) {
  const report = reportFromContext(context);
  const ruleIDs = new Set();
  (report && report.findings || []).forEach(function (finding) {
    if (finding && finding.rule_id) ruleIDs.add(String(finding.rule_id));
  });
  (report && report.coverage || []).forEach(function (coverage) {
    if (coverage && coverage.rule_id) ruleIDs.add(String(coverage.rule_id));
  });
  return Array.from(ruleIDs).map(function (id) {
    const label = ruleLabel(id);
    return { id: id, label: label === "Quality check" ? id : label, category: ruleCategory(id) };
  }).sort(function (left, right) {
    const leftCategory = ruleCategories.findIndex(function (category) { return category.label === left.category; });
    const rightCategory = ruleCategories.findIndex(function (category) { return category.label === right.category; });
    return (leftCategory - rightCategory) || left.label.localeCompare(right.label) || left.id.localeCompare(right.id);
  });
}

export function qualityFindingRuleFilter(context) {
  const value = String(context && context.state && context.state.qualityFindingRuleFilter || "");
  return qualityFindingRuleOptions(context).some(function (option) { return option.id === value; }) ? value : "";
}

export function setQualityFindingRuleFilter(context, value) {
  if (!context || !context.state) return "";
  context.state.qualityFindingRuleFilter = String(value || "");
  context.state.qualityFindingsPage = null;
  context.state.qualityFindingsError = "";
  return qualityFindingRuleFilter(context);
}

function findingRuleMatches(context, finding) {
  const filter = qualityFindingRuleFilter(context);
  return !filter || String(finding && finding.rule_id || "") === filter;
}

export function qualityFindingFilterMarkup(context) {
  const options = qualityFindingRuleOptions(context);
  const current = qualityFindingRuleFilter(context);
  const groups = ruleCategories.map(function (category) {
    const entries = options.filter(function (option) { return option.category === category.label; });
    if (!entries.length) return "";
    return '<optgroup label="' + escapeHTML(category.label) + '">' + entries.map(function (option) {
      return '<option value="' + escapeHTML(option.id) + '"' + (option.id === current ? " selected" : "") + '>' + escapeHTML(option.label) + '</option>';
    }).join("") + '</optgroup>';
  }).join("");
  return '<label class="select-control quality-finding-filter"><span>Filter by check</span><select data-quality-finding-rule aria-label="Filter quality findings by check"><option value=""' + (!current ? " selected" : "") + '>All checks</option>' + groups + '</select></label>';
}

function ruleLabel(ruleID) {
  return ruleLabels[ruleID] || "Quality check";
}

function ruleDescription(entry) {
  return ruleExplanations[entry && entry.id] || entry && entry.description || "No description was reported for this check.";
}

function statusLabel(status) {
  return String(status || "active").replaceAll("_", " ").replace(/(^|\s)\S/g, function (value) { return value.toUpperCase(); });
}

function qualityPill(value, details) {
  const state = String(value || "unknown").toLowerCase();
  const tone = state === "active" || state === "unsupported" || state === "unknown" || state === "partial" || state === "not_evaluable" ? "warning" : state === "error" || state === "blocker" ? "error" : "ok";
  return '<span class="state-pill ' + tone + '" title="' + escapeHTML(details || "") + '">' + escapeHTML(statusLabel(value)) + "</span>";
}

function qualityCount(value, label, className) {
  return '<div class="quality-count ' + (className || "") + '"><strong>' + escapeHTML(value) + '</strong><span>' + escapeHTML(label) + "</span></div>";
}

function coverageCountText(value, noun, suffix) {
  const count = Number(value || 0);
  return String(count) + " " + noun + (count === 1 ? "" : "s") + (suffix ? " " + suffix : "");
}

export function qualityCoverageSummaryMarkup(states, total) {
  const values = states || {};
  const reported = Number.isFinite(Number(total)) && Number(total) > 0
    ? Number(total)
    : coverageStateOrder.reduce(function (sum, state) { return sum + Number(values[state] || 0); }, 0);
  const observed = Number(values.observed || 0);
  const headline = reported ? observed + " of " + reported + " checks observed" : "No checks observed";
  const items = coverageStateOrder.filter(function (state) { return Number(values[state] || 0) > 0; }).map(function (state) {
    const suffix = state === "observed" ? "observed" : statusLabel(state).toLowerCase();
    return '<span class="quality-coverage-state">' + qualityPill(state, "Reported coverage state") + '<strong>' + escapeHTML(coverageCountText(values[state], "check", suffix)) + '</strong></span>';
  });
  if (!items.length) return '<div class="quality-coverage-summary quality-coverage-summary-empty" aria-label="Quality coverage summary"><strong>Coverage</strong><span class="state-pill warning">Not reported</span><span class="quality-coverage-headline">No checks reported coverage.</span></div>';
  return '<div class="quality-coverage-summary" aria-label="Quality coverage summary"><strong>Coverage</strong><span class="quality-coverage-headline">' + escapeHTML(headline) + '</span><span class="quality-coverage-states">' + items.join("") + '</span></div>';
}

function coverageStatusExplanation(status) {
  switch (String(status || "unknown").toLowerCase()) {
    case "unsupported": return "This rule is implemented, but the loaded model does not provide the input capability it needs, so it was not run.";
    case "not_evaluable": return "The rule has the required capability, but the current data or policy is not sufficient to evaluate it.";
    case "partial": return "Only part of the expected input was available, so the result is incomplete.";
    case "unknown": return "The source reported an unknown state, so no conclusion was made.";
    case "absent": return "The expected input was not reported for this model revision.";
    default: return "The rule did not produce complete coverage for this model revision.";
  }
}

export function qualityCoverageDetailsMarkup(coverage) {
  const groups = [];
  const byKey = new Map();
  (coverage || []).filter(function (item) { return String(item && item.status || "unknown").toLowerCase() !== "observed"; }).forEach(function (item) {
    const status = String(item.status || "unknown").toLowerCase();
    const ruleID = String(item.rule_id || "");
    const reason = String(item.reason || "");
    const key = ruleID + "\u0000" + status + "\u0000" + reason;
    let group = byKey.get(key);
    if (!group) {
      group = { ruleID: ruleID, status: status, reason: reason, count: 0 };
      byKey.set(key, group);
      groups.push(group);
    }
    group.count++;
  });
  if (!groups.length) return "";
  const rows = groups.map(function (group) {
    const count = group.count > 1 ? " · " + group.count + " report entries" : "";
    const reason = group.reason ? " Report reason: " + group.reason : "";
    return '<li class="quality-coverage-detail"><div class="quality-coverage-detail-heading"><strong>' + escapeHTML(ruleLabel(group.ruleID)) + '</strong>' + qualityPill(group.status, group.reason) + '<span class="muted">' + escapeHTML(count) + '</span></div><p>' + escapeHTML(coverageStatusExplanation(group.status) + reason) + "</p></li>";
  }).join("");
  return '<details class="quality-coverage-details"><summary>Why is coverage incomplete?</summary><p>Coverage is counted per rule and source scope, so one enabled rule can contribute more than one report entry in a multi-scope model.</p><ul>' + rows + "</ul></details>";
}

function coverageStateMarkup(states) {
  return qualityCoverageSummaryMarkup(states);
}

function findingSubjectLabel(finding) {
  const kind = finding && finding.subject_ref && finding.subject_ref.kind;
  if (kind === "file") return "File-level observation";
  if (kind === "symbol") return "Symbol-level observation";
  if (kind === "module") return "Module-level observation";
  return kind ? kind.charAt(0).toUpperCase() + kind.slice(1) + " observation" : "Reported observation";
}

function findingRows(context, findings) {
  return findings.map(function (finding) {
    const limitation = finding.assessment_kind === "signal" ? "Advisory signal; this is not proof of a design violation." : "Exact check over reported facts.";
    const message = finding.message || limitation;
    const evidenceButton = finding.id
      ? '<button type="button" class="button tertiary" data-quality-evidence="' + escapeHTML(finding.id) + '">View evidence</button>'
      : "";
    const baselineValue = finding.id || finding.finding_key || "";
    const baselineButton = !context.liveEnabled && finding.status === "active" && baselineValue
      ? '<button type="button" class="button tertiary" data-quality-baseline-open="' + escapeHTML(baselineValue) + '">Baseline this</button>'
      : "";
    return '<li class="quality-finding-row"><div class="quality-finding-main"><div class="quality-finding-heading"><strong>' + escapeHTML(ruleLabel(finding.rule_id)) + '</strong>' + qualityPill(finding.status, limitation) + '</div><span>' + escapeHTML(findingSubjectLabel(finding)) + " · " + escapeHTML(finding.severity || "info") + '</span><p>' + escapeHTML(message) + '</p>' + (finding.limitations && finding.limitations.length ? '<small>' + escapeHTML(finding.limitations[0]) + '</small>' : '<small>' + escapeHTML(limitation) + '</small>') + '</div><div class="quality-finding-actions">' + evidenceButton + baselineButton + '</div></li>';
  }).join("");
}

function qualityFindingsPagerMarkup(context, loaded, total, hasMore) {
  if (!hasMore) return "";
  const loading = Boolean(context.state.qualityFindingsLoading);
  const error = context.state.qualityFindingsError;
  const message = error ? error : loading ? "Loading more quality findings…" : "Showing " + loaded + " of " + total + " quality findings.";
  const buttonLabel = error ? "Try again" : loading ? "Loading…" : "Load more findings";
  return '<div class="quality-findings-pager"><p class="muted">' + escapeHTML(message) + '</p><button type="button" class="button secondary" data-quality-findings-load-more' + (loading ? " disabled" : "") + '>' + escapeHTML(buttonLabel) + "</button></div>";
}

function renderQualityEvidence(context) {
  const value = context.state.qualityEvidence;
  if (!value) return "";
  if (value.loading) return '<div class="quality-evidence-panel" role="status">Loading bounded quality evidence…</div>';
  if (value.error) return '<div class="quality-evidence-panel"><span class="state-pill warning">Unavailable</span><p>' + escapeHTML(value.error) + "</p></div>";
  const finding = value.data && value.data.finding;
  if (!finding) return "";
  const spans = finding.evidence && finding.evidence.source_spans || [];
  const sourceButton = spans.length && context.sourceEnabled && !value.data.source_context
    ? '<button type="button" class="button secondary" data-quality-source="' + escapeHTML(finding.id) + '">Show bounded source excerpt</button>'
    : "";
  const source = value.data.source_context;
  const sourceMarkup = source ? '<div class="quality-source-context"><strong>' + escapeHTML(source.path || "Source file") + '</strong><span>' + escapeHTML(sourceRange(source)) + ' · read-only</span><pre><code>' + (source.lines || []).map(function (line) { return '<span class="source-line"><span class="source-number" aria-hidden="true">' + escapeHTML(line.number) + '</span><span class="source-text">' + escapeHTML(line.text) + "</span></span>"; }).join("") + "</code></pre></div>" : "";
  return '<div class="quality-evidence-panel"><div class="quality-evidence-heading"><strong>Evidence for ' + escapeHTML(ruleLabel(finding.rule_id)) + '</strong>' + qualityPill(finding.assessment_kind, finding.assessment_kind === "signal" ? "Advisory signal" : "Exact check") + '</div><p>' + escapeHTML(spans.length ? spans.length + " bounded source location(s) were reported." : "No line-level source span was reported for this finding.") + "</p>" + sourceButton + sourceMarkup + "</div>";
}

function sourceRange(source) {
  if (!source || !source.start) return "File provenance only";
  const start = Number(source.start.line || 0);
  const end = source.end && Number(source.end.line || 0) >= start ? Number(source.end.line) : start;
  if (!start) return "File provenance only";
  return start === end ? "Line " + start : "Lines " + start + "–" + end;
}

export function renderQualitySummary(context) {
  const status = qualityReportStatus(context);
  if (status === "loading") return '<div class="quality-summary quality-summary-loading"><span class="state-pill warning">Loading</span><span>Quality checks are loading for this scope.</span></div>';
  if (status !== "available") return '<div class="quality-summary quality-summary-missing"><span class="state-pill warning">' + escapeHTML(statusLabel(status)) + '</span><span>' + escapeHTML(context.state.qualityReportError || "No quality report is attached to this model revision.") + "</span></div>";
  const summary = qualitySummary(context);
  const tone = summary.blocker || summary.error || summary.active ? "warning" : summary.partialCoverage ? "warning" : "ok";
  const headline = summary.active ? "Attention" : summary.partialCoverage ? "Coverage needs review" : summary.total === 0 && summary.coverageStates.observed > 0 ? "No active findings" : "Coverage not established";
  return '<div class="quality-summary"><div class="quality-summary-heading"><strong>Quality checks</strong><span class="state-pill ' + tone + '">' + escapeHTML(headline) + '</span></div><div class="quality-summary-counts">' + qualityCount(summary.active, "active") + qualityCount(summary.exact, "exact") + qualityCount(summary.signal, "signals") + (summary.affectedFiles ? qualityCount(summary.affectedFiles, "affected files", "quality-count-attention") : "") + '</div>' + (summary.partialCoverage ? coverageStateMarkup(summary.coverageStates) : "") + '<small>Exact checks use reported facts. Signals are advisory structural indicators.</small></div>';
}

export function renderQualityOverview(context) {
  const status = qualityReportStatus(context);
  if (status !== "available") return '<div class="quality-overview-state">' + renderQualitySummary(context) + (context.state.qualityReportError ? '<p>' + escapeHTML(context.state.qualityReportError) + "</p>" : "") + "</div>";
  const report = reportFromContext(context);
  const summary = qualitySummary(context);
  const page = context.state.qualityFindingsPage;
  const reportFindings = reportFindingsFromContext(context).filter(function (finding) { return findingScopeMatches(context, finding); });
  const filteredReportFindings = reportFindings.filter(function (finding) { return findingRuleMatches(context, finding); });
  const findings = (page && Array.isArray(page.items) ? page.items : filteredReportFindings.slice(0, qualityPageLimit)).filter(function (finding) { return findingScopeMatches(context, finding) && findingRuleMatches(context, finding); });
  const coverage = (report && report.coverage || []).filter(function (item) { return coverageScopeMatches(context, item); });
  const total = page && page.total != null ? Number(page.total) : filteredReportFindings.length;
  const hasMore = page ? Boolean(page.next_cursor) : total > findings.length;
  const pager = qualityFindingsPagerMarkup(context, findings.length, total, hasMore);
  const rows = findings.length ? '<ul class="quality-findings-list">' + findingRows(context, findings) + '</ul>' + pager : summary.partialCoverage || !coverage.length ? '<div class="quality-empty"><span class="state-pill warning">Coverage incomplete</span><p>No clean result can be inferred because the active scope does not have complete quality coverage.</p></div>' : '<div class="quality-empty"><span class="state-pill ok">Clear</span><p>No quality findings were reported for the active scope.</p></div>';
  const baselineAction = !context.embeddedExport && !context.liveEnabled && summary.active
    ? '<button type="button" class="button secondary" data-quality-baseline-open="all">Create baseline</button>'
    : "";
  return '<div class="quality-overview"><div class="quality-overview-lede"><p>These checks describe reported code facts. Exact checks are configurable; SOLID entries are advisory signals, not proof of a violation.</p>' + qualityCoverageSummaryMarkup(summary.coverageStates, coverage.length) + '<p class="quality-coverage-explanation">Coverage is counted per rule and source scope. Observed means the check received the facts it needs and ran; it does not mean that no finding was found.</p>' + qualityCoverageDetailsMarkup(coverage) + '</div><div class="quality-count-grid">' + qualityCount(summary.active, "active findings", summary.active ? "quality-count-attention" : "") + qualityCount(summary.exact, "exact") + qualityCount(summary.signal, "advisory signals") + qualityCount(summary.suppressed + summary.baseline + summary.resolved, "lifecycle states") + '</div><div class="quality-findings-heading"><div><h4>Reported findings</h4><span class="muted">' + escapeHTML(String(total)) + " shown" + (qualityFindingRuleFilter(context) ? " · filtered" : "") + '</span></div><div class="quality-findings-tools">' + qualityFindingFilterMarkup(context) + baselineAction + '</div></div>' + rows + renderQualityEvidence(context) + "</div>";
}

function ruleCategory(ruleID) {
  const value = String(ruleID || "");
  const match = ruleCategories.find(function (category) { return value.indexOf(category.prefix) === 0; });
  return match ? match.label : "Other checks";
}

function ruleAssessmentLabel(kind) {
  return assessmentLabels[String(kind || "").toLowerCase()] || "Reported check";
}

function ruleCatalogSearchMatch(entry, query) {
  if (!query) return true;
  const haystack = [ruleLabel(entry.id), entry.id, ruleDescription(entry), entry.assessment_kind, ruleCategory(entry.id)].join(" ").toLowerCase();
  return haystack.indexOf(query) >= 0;
}

function catalogEntryKey(entry) {
  return String(entry && entry.id || "") + "\u0000" + String(entry && entry.version || "");
}

function draftBindingMap(context) {
  const bindings = context && context.state && Array.isArray(context.state.qualityRuleDraft) ? context.state.qualityRuleDraft : [];
  return new Map(bindings.map(function (binding) { return [String(binding.rule_id || "") + "\u0000" + String(binding.rule_version || ""), binding]; }));
}

function catalogBinding(entry) {
  return {
    rule_id: entry.id,
    rule_version: entry.version,
    enabled: Boolean(entry.enabled),
    parameters: entry.parameters || { namespace: entry.parameter_schema && entry.parameter_schema.namespace || "", schema_version: entry.parameter_schema && entry.parameter_schema.schema_version || "", payload: {} },
    severity: entry.severity || entry.default_severity || ""
  };
}

export function qualityRuleBindings(context) {
  const state = context && context.state;
  if (!state) return [];
  if (Array.isArray(state.qualityRuleDraft)) return state.qualityRuleDraft.map(function (binding) { return Object.assign({}, binding); });
  return (Array.isArray(state.qualityRuleCatalog) ? state.qualityRuleCatalog : []).map(catalogBinding);
}

export function initializeQualityRuleDraft(context) {
  if (!context || !context.state) return [];
  context.state.qualityRuleDraft = qualityRuleBindings(Object.assign({}, context, { state: Object.assign({}, context.state, { qualityRuleDraft: null }) }));
  return context.state.qualityRuleDraft;
}

export function setQualityRuleEnabled(context, ruleID, ruleVersion, enabled) {
  if (!context || !context.state) return [];
  const bindings = qualityRuleBindings(context).map(function (binding) { return Object.assign({}, binding); });
  const key = String(ruleID || "") + "\u0000" + String(ruleVersion || "");
  bindings.forEach(function (binding) {
    if (String(binding.rule_id || "") + "\u0000" + String(binding.rule_version || "") === key) binding.enabled = Boolean(enabled);
  });
  context.state.qualityRuleDraft = bindings;
  return bindings;
}

export function renderQualityRuleCatalog(context) {
  const state = context && context.state;
  if (!state) return "";
  if (state.qualityRuleCatalogStatus === "loading") return '<div class="quality-rule-catalog-state" role="status"><span class="state-pill warning">Loading</span><p>Loading the available quality checks…</p></div>';
  if (state.qualityRuleCatalogStatus === "unavailable" || state.qualityRuleCatalogStatus === "error") return '<div class="quality-rule-catalog-state"><span class="state-pill warning">Unavailable</span><p>' + escapeHTML(state.qualityRuleCatalogError || "The quality rule catalog is unavailable.") + "</p></div>";
  const catalog = (Array.isArray(state.qualityRuleCatalog) ? state.qualityRuleCatalog : []).filter(function (entry) { return entry && entry.id && ruleCatalogSearchMatch(entry, String(state.qualityRuleSearch || "").trim().toLowerCase()); });
  if (!catalog.length) return '<div class="quality-rule-catalog-state"><span class="state-pill warning">No matches</span><p>Try a broader rule name, category, or description.</p></div>';
  const bindings = draftBindingMap(context);
  const groups = ruleCategories.map(function (category) {
    const entries = catalog.filter(function (entry) { return ruleCategory(entry.id) === category.label; });
    if (!entries.length) return "";
    const enabled = entries.filter(function (entry) { const binding = bindings.get(catalogEntryKey(entry)); return binding ? binding.enabled : entry.enabled; }).length;
    return '<section class="quality-rule-group" aria-labelledby="quality-rule-group-' + escapeHTML(category.prefix.replace(":", "")) + '"><div class="quality-rule-group-heading"><div><p class="eyebrow">' + escapeHTML(category.label) + '</p><h4 id="quality-rule-group-' + escapeHTML(category.prefix.replace(":", "")) + '">' + escapeHTML(category.label) + '</h4></div><span class="muted">' + escapeHTML(String(enabled)) + ' of ' + escapeHTML(String(entries.length)) + ' enabled</span></div><div class="quality-rule-list">' + entries.map(function (entry) {
      const binding = bindings.get(catalogEntryKey(entry));
      const isEnabled = binding ? Boolean(binding.enabled) : Boolean(entry.enabled);
      const configured = Boolean(entry.configured);
      const stateLabel = isEnabled ? "Enabled" : configured ? "Configured but off" : "Available but off";
      const stateClass = isEnabled ? "ok" : "neutral";
      const description = ruleDescription(entry);
      const limitation = entry.limitations && entry.limitations.length ? " " + entry.limitations[0] : "";
      return '<label class="quality-rule-card' + (isEnabled ? " enabled" : "") + '"><input type="checkbox" data-quality-rule-toggle data-rule-id="' + escapeHTML(entry.id) + '" data-rule-version="' + escapeHTML(entry.version) + '"' + (isEnabled ? " checked" : "") + '><span class="quality-rule-card-body"><span class="quality-rule-title"><strong>' + escapeHTML(ruleLabel(entry.id)) + '</strong><span class="state-pill neutral">' + escapeHTML(ruleAssessmentLabel(entry.assessment_kind)) + '</span><span class="state-pill ' + stateClass + '">' + escapeHTML(stateLabel) + '</span></span><span class="quality-rule-description">' + escapeHTML(description + limitation) + '</span><span class="quality-rule-meta">' + escapeHTML(entry.parameter_source === "selected profile" ? "Uses the selected profile setting." : "Starts with a catalog default; changes apply for this session only.") + '</span></span></label>';
    }).join("") + "</div></section>";
  }).join("");
  return '<div class="quality-rule-catalog-heading"><div><p class="eyebrow">AVAILABLE CHECKS</p><h3>Quality rules</h3></div><span class="muted">' + escapeHTML(String(catalog.length)) + " shown</span></div>" + groups;
}

export async function loadQualityRuleCatalog(context, api, profileID, profileVersion) {
  if (!context || !context.state || !api || !profileID || !profileVersion || context.embeddedExport) {
    if (context && context.state) {
      context.state.qualityRuleCatalogStatus = "unavailable";
      context.state.qualityRuleCatalogError = "The quality rule catalog is unavailable in this viewer session.";
    }
    renderQualityProfileControl(context);
    return null;
  }
  const request = (context.state.qualityRuleCatalogRequest || 0) + 1;
  context.state.qualityRuleCatalogRequest = request;
  context.state.qualityRuleCatalogStatus = "loading";
  context.state.qualityRuleCatalogError = "";
  renderQualityProfileControl(context);
  try {
    const endpoint = "/v1/quality/rules?profile_id=" + encodeURIComponent(profileID) + "&profile_version=" + encodeURIComponent(profileVersion);
    const response = await api.getJSON(endpoint);
    if (request !== context.state.qualityRuleCatalogRequest) return response;
    context.state.qualityRuleCatalog = Array.isArray(response.rules) ? response.rules : [];
    context.state.qualityRuleCatalogStatus = response.status || (context.state.qualityRuleCatalog.length ? "available" : "empty");
    context.state.qualityRuleCatalogError = response.message || "";
    context.state.qualityRuleDraft = null;
    renderQualityProfileControl(context);
    return response;
  } catch (error) {
    if (request !== context.state.qualityRuleCatalogRequest) return null;
    context.state.qualityRuleCatalog = [];
    context.state.qualityRuleCatalogStatus = "error";
    context.state.qualityRuleCatalogError = error && error.message || "The quality rule catalog is unavailable.";
    renderQualityProfileControl(context);
    return null;
  }
}

export async function openQualityEvidence(context, api, findingID, includeSourceContext) {
  if (!context || !context.state || !api || !findingID) return;
  context.state.qualityEvidence = { loading: true };
  try {
    const suffix = "/findings/" + encodeURIComponent(findingID) + "/evidence";
    const data = await api.getJSON(qualityEndpoint(context, api, suffix, includeSourceContext ? { include_source_context: true, max_lines: 120, max_bytes: 4 * 1024 * 1024 } : {}));
    context.state.qualityEvidence = { data: data };
  } catch (error) {
    context.state.qualityEvidence = { error: error && error.message || "Quality evidence is unavailable." };
  }
}

export async function loadQualitySource(context, api, findingID) {
  return openQualityEvidence(context, api, findingID, true);
}
