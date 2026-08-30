export function createAPI(context) {
  async function getJSON(path) {
    if (context.embeddedExport) return embeddedJSON(path);
    const response = await fetch(path, { headers: { Accept: "application/json" } });
    const body = await response.json();
    if (!response.ok) throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    return body;
  }

  function embeddedJSON(path) {
    const url = new URL(path, window.location.href);
    const modelPath = "/v1/models/" + encodeURIComponent(context.modelID);
    if (url.pathname === "/v1/quality/profiles") return { schema_version: "arch-view.quality-profiles/v1", status: "unavailable", profiles: [], message: "Quality evaluation is unavailable in a self-contained export." };
    if (url.pathname === "/v1/quality/rules") return { schema_version: "arch-view.quality-rules/v1", status: "unavailable", rules: [], message: "Quality rule configuration is unavailable in a self-contained export." };
    if (url.pathname === modelPath) return context.embeddedExport.model;
    if (url.pathname === modelPath + "/quality") return embeddedQualityReport(context);
    if (url.pathname === modelPath + "/quality/findings") return embeddedQualityFindings(context, url);
    if (url.pathname.indexOf(modelPath + "/quality/findings/") === 0) return embeddedQualityEvidence(context, url, modelPath);
    if (url.pathname === modelPath + "/quality/coverage") return embeddedQualityCoverage(context, url);
    if (url.pathname === modelPath + "/source-index") {
      if (context.embeddedExport.model && context.embeddedExport.model.source_index) return context.embeddedExport.model.source_index;
      throw new Error("The exported model has no source-facts attachment.");
    }
    if (url.pathname === modelPath + "/source-index/files" || url.pathname === modelPath + "/source-index/symbols" || url.pathname === modelPath + "/source-index/documentation") {
      return embeddedSourcePage(context, url);
    }
    if (url.pathname.indexOf(modelPath + "/source-index/evidence/") === 0) {
      return embeddedSourceEvidence(context, url);
    }
    if (url.pathname === modelPath + "/projection") {
      const pathValue = url.searchParams.getAll("path");
      const visibility = url.searchParams.get("reference_visibility") || "hidden";
      const key = JSON.stringify(pathValue) + "|" + visibility;
      if (context.embeddedExport.scenes && context.embeddedExport.scenes[key]) return context.embeddedExport.scenes[key];
      throw new Error("The exported hierarchy path or reference view is unavailable.");
    }
    if (url.pathname === "/v1/layout/config") return { schema_version: "arch-view.config/v1", layout: { algorithm: "layered", options: {} }, origin: "session", status: "valid", can_save: false, can_save_as: false, diagnostics: [] };
    if (url.pathname === "/v1/layout/options") return { schema_version: "arch-view.config/v1", adapter: { id: "export", version: "embedded", source: "export" }, algorithms: [], categories: [], options: [] };
    throw new Error("The self-contained export does not require network access.");
  }

  async function postJSON(path, value) {
    const response = await fetch(path, { method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(value) });
    const body = await response.json();
    if (!response.ok) throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    return body;
  }

  async function putJSON(path, value) {
    const response = await fetch(path, { method: "PUT", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(value) });
    const body = await response.json();
    if (!response.ok) throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    return body;
  }

  function currentModelID() {
    const state = context.state;
    if (state.model && state.model.model_id) return state.model.model_id;
    if (state.scene && state.scene.model_id) return state.scene.model_id;
    return context.modelID;
  }

  return { currentModelID, getJSON, postJSON, putJSON };
}

function embeddedQualityValue(context) {
  return context.embeddedExport && context.embeddedExport.model && context.embeddedExport.model.quality_report || null;
}

function embeddedQualityReport(context) {
  const report = embeddedQualityValue(context);
  if (!report) return { schema_version: "arch-view.quality-query/v1", status: "missing", coverage: [], message: "No quality report is attached to this model revision." };
  return { schema_version: "arch-view.quality-query/v1", status: "available", report: report, coverage: report.coverage || [] };
}

function embeddedQualityPage(items, url, snapshotIDs, scopeID) {
  const limitText = url.searchParams.get("limit") || "50";
  if (!/^\d+$/.test(limitText)) throw new Error("The quality query limit must be an integer.");
  const limit = Number(limitText);
  if (limit < 1 || limit > 200) throw new Error("The quality query limit is outside the allowed bound.");
  let start = 0;
  const cursor = url.searchParams.get("cursor");
  if (cursor) {
    try {
      const decoded = atob(cursor.replace(/-/g, "+").replace(/_/g, "/"));
      if (!/^qv1:\d+$/.test(decoded)) throw new Error("invalid cursor");
      start = Number(decoded.slice(4));
    } catch (_) { throw new Error("The quality query cursor is malformed."); }
  }
  if (start > items.length) throw new Error("The quality query cursor is outside the result set.");
  const end = Math.min(items.length, start + limit);
  const next = end < items.length ? btoa("qv1:" + end).replace(/=/g, "").replace(/\+/g, "-").replace(/\//g, "_") : "";
  return { schema_version: "arch-view.quality-query/v1", source_snapshot_ids: snapshotIDs || [], scope_id: scopeID || "", items: items.slice(start, end), total: items.length, next_cursor: next || undefined };
}

function embeddedQualityCoverageMatchesScope(coverage, scopeID) {
  if (!scopeID) return true;
  const evidenceIDs = coverage && coverage.provenance && coverage.provenance.evidence_ids || [];
  if (!evidenceIDs.length) return true;
  const prefix = "scope:" + scopeID + ":";
  return evidenceIDs.some(function (evidenceID) { return evidenceID === "scope:" + scopeID || String(evidenceID).indexOf(prefix) === 0; });
}

function embeddedQualityFindings(context, url) {
  const report = embeddedQualityValue(context);
  const scopeID = url.searchParams.get("scope") === "all" ? "" : url.searchParams.get("scope") || "";
  if (!report) return { schema_version: "arch-view.quality-query/v1", report_id: context.modelID, evaluation_id: "", source_snapshot_ids: [], scope_id: scopeID, items: [], total: 0, coverage: [], status: "missing", message: "No quality report is attached to this model revision." };
  const matches = (report.findings || []).filter(function (finding) {
    const subject = finding.subject_ref || {};
    return (!url.searchParams.get("report_id") || url.searchParams.get("report_id") === report.evaluation_id) &&
      (!url.searchParams.get("snapshot_id") || subject.snapshot_id === url.searchParams.get("snapshot_id")) &&
      (!scopeID || subject.scope_id === scopeID) &&
      (!url.searchParams.get("subject_id") || subject.id === url.searchParams.get("subject_id")) &&
      (!url.searchParams.get("subject_kind") || subject.kind === url.searchParams.get("subject_kind")) &&
      (!url.searchParams.get("file_id") || subject.kind === "file" && subject.id === url.searchParams.get("file_id")) &&
      (!url.searchParams.get("rule_id") || finding.rule_id === url.searchParams.get("rule_id")) &&
      (!url.searchParams.get("assessment_kind") || finding.assessment_kind === url.searchParams.get("assessment_kind")) &&
      (!url.searchParams.get("severity") || finding.severity === url.searchParams.get("severity")) &&
      (!url.searchParams.get("status") || finding.status === url.searchParams.get("status"));
  }).slice().sort(function (left, right) {
    return String(left.finding_key || "").localeCompare(String(right.finding_key || "")) || String(left.rule_id || "").localeCompare(String(right.rule_id || "")) || String(left.id || "").localeCompare(String(right.id || ""));
  });
  const page = embeddedQualityPage(matches, url, report.source_snapshot_ids || [], scopeID);
  page.report_id = report.evaluation_id;
  page.evaluation_id = report.evaluation_id;
  page.coverage = (report.coverage || []).filter(function (coverage) {
    return embeddedQualityCoverageMatchesScope(coverage, scopeID) && (!url.searchParams.get("rule_id") || coverage.rule_id === url.searchParams.get("rule_id"));
  });
  page.status = "available";
  return page;
}

function embeddedQualityCoverage(context, url) {
  const report = embeddedQualityValue(context);
  if (!report) return { schema_version: "arch-view.quality-query/v1", report_id: context.modelID, evaluation_id: "", items: [], total: 0, status: "missing", message: "No quality report is attached to this model revision." };
  const ruleID = url.searchParams.get("rule_id") || "";
  const scopeID = url.searchParams.get("scope") === "all" ? "" : url.searchParams.get("scope") || "";
  const items = (report.coverage || []).filter(function (coverage) { return embeddedQualityCoverageMatchesScope(coverage, scopeID) && (!ruleID || coverage.rule_id === ruleID); });
  const page = embeddedQualityPage(items, url, [], "");
  page.report_id = report.evaluation_id;
  page.evaluation_id = report.evaluation_id;
  page.status = "available";
  return page;
}

function embeddedQualityEvidence(context, url, modelPath) {
  const report = embeddedQualityValue(context);
  if (!report) throw new Error("No quality report is attached to this model revision.");
  const prefix = modelPath + "/quality/findings/";
  const suffix = url.pathname.slice(prefix.length);
  if (!suffix.endsWith("/evidence")) throw new Error("The quality evidence route is unavailable.");
  const findingID = decodeURIComponent(suffix.slice(0, -"/evidence".length));
  const finding = (report.findings || []).find(function (value) { return value.id === findingID || value.finding_key === findingID; });
  if (!finding) throw new Error("The quality finding was not found in the exported report.");
  const include = url.searchParams.get("include_source_context") || url.searchParams.get("include_source") || "false";
  const includeSource = include === "true";
  const result = { schema_version: "arch-view.quality-query/v1", report_id: report.evaluation_id, evaluation_id: report.evaluation_id, finding: finding };
  if (includeSource) {
    const lines = Number(url.searchParams.get("max_lines") || "120");
    const bytes = Number(url.searchParams.get("max_bytes") || String(4 * 1024 * 1024));
    if (!Number.isInteger(lines) || !Number.isInteger(bytes) || lines < 1 || lines > 120 || bytes < 1 || bytes > 4 * 1024 * 1024) throw new Error("The source-context budget is outside the allowed bound.");
    result.source_context_requested = true;
    result.source_context_max_lines = lines;
    result.source_context_max_bytes = bytes;
    throw new Error("Bounded source context is unavailable in this export because source contents were not embedded.");
  }
  return result;
}

function embeddedSourceIndex(context) {
  const model = context.embeddedExport && context.embeddedExport.model;
  if (!model || !model.source_index) throw new Error("The exported model has no source-facts attachment.");
  return model.source_index;
}

function embeddedSourceSnapshot(index, url) {
  const requestedScope = url.searchParams.get("scope") || "";
	const requestedSnapshot = url.searchParams.get("snapshot_id") || "";
	if (!requestedSnapshot && (!requestedScope || requestedScope === "all") && index.projection) return index.projection;
  const snapshots = Array.isArray(index.snapshots) ? index.snapshots : [];
	const matches = snapshots.filter(function (snapshot) {
		if (requestedSnapshot && snapshot.snapshot_id !== requestedSnapshot) return false;
		return !requestedScope || requestedScope === "all" || snapshot.scope_context && snapshot.scope_context.scope_id === requestedScope;
	});
	return matches.length === 1 ? matches[0] : null;
}

function embeddedIDs(url, key) {
  return Array.from(new Set(url.searchParams.getAll(key).map(function (value) { return value.trim(); }).filter(Boolean)));
}

function embeddedMembership(snapshot, url) {
  const moduleIDs = new Set(embeddedIDs(url, "module_id"));
  const fileIDs = new Set(embeddedIDs(url, "file_id"));
  if (moduleIDs.size) {
    (snapshot.relations || []).forEach(function (relation) {
      if (relation.category !== "contains" || !relation.from_ref || relation.from_ref.kind !== "module" || !relation.to_ref || relation.to_ref.kind !== "file") return;
      if (moduleIDs.has(relation.from_ref.id)) fileIDs.add(relation.to_ref.id);
    });
  }
  if (!moduleIDs.size && !fileIDs.size) return null;
  return fileIDs;
}

function embeddedSymbolMembership(snapshot, files) {
  if (!files) return null;
  const symbols = new Set();
  (snapshot.relations || []).forEach(function (relation) {
    if ((relation.category !== "contains" && relation.category !== "declares") || !relation.from_ref || relation.from_ref.kind !== "file" || !relation.to_ref || relation.to_ref.kind !== "symbol") return;
    if (files.has(relation.from_ref.id)) symbols.add(relation.to_ref.id);
  });
  return symbols;
}

function embeddedPage(items, snapshot, url) {
	const limitText = url.searchParams.get("limit") || "50";
	if (!/^\d+$/.test(limitText)) throw new Error("The source-facts query limit must be an integer.");
	const limit = Number(limitText);
	if (limit < 1 || limit > 200) throw new Error("The source-facts query limit is outside the allowed bound.");
  let start = 0;
  const cursor = url.searchParams.get("cursor");
  if (cursor) {
    try {
      const decoded = atob(cursor.replace(/-/g, "+").replace(/_/g, "/"));
	  if (!/^v1:\d+$/.test(decoded)) throw new Error("invalid cursor");
	  start = Number(decoded.slice(3));
	} catch (_) { throw new Error("The source-facts query cursor is malformed."); }
  }
	if (start > items.length) throw new Error("The source-facts query cursor is outside the result set.");
  const end = Math.min(items.length, start + limit);
  const next = end < items.length ? btoa("v1:" + end).replace(/=/g, "").replace(/\+/g, "-").replace(/\//g, "_") : "";
  return { snapshot_id: snapshot.snapshot_id, scope_id: snapshot.scope_context && snapshot.scope_context.scope_id || "", items: items.slice(start, end), total: items.length, next_cursor: next || undefined, coverage: snapshot.coverage || undefined };
}

function embeddedSourcePage(context, url) {
  const index = embeddedSourceIndex(context);
  const snapshot = embeddedSourceSnapshot(index, url);
  if (!snapshot) throw new Error("The exported source-facts scope is unavailable.");
  const files = embeddedMembership(snapshot, url);
  const symbols = embeddedSymbolMembership(snapshot, files);
  let items;
  if (url.pathname.endsWith("/files")) {
    items = (snapshot.files || []).filter(function (file) {
      return !files || files.has(file.id);
    }).filter(function (file) {
      const language = url.searchParams.get("language");
      const prefix = url.searchParams.get("path_prefix");
      return (!language || String(file.language && file.language.id || "").toLowerCase().indexOf(language.toLowerCase()) >= 0) && (!prefix || String(file.path || "").toLowerCase().indexOf(prefix.toLowerCase()) === 0);
    }).sort(function (left, right) { return String(left.path).localeCompare(String(right.path)) || String(left.id).localeCompare(String(right.id)); });
  } else if (url.pathname.endsWith("/symbols")) {
    const paths = {};
    (snapshot.files || []).forEach(function (file) { paths[file.id] = file.path; });
    items = (snapshot.symbols || []).filter(function (symbol) {
      if (!symbols) return true;
      return symbols.has(symbol.id);
    }).filter(function (symbol) {
      const name = url.searchParams.get("name");
      const languageKind = url.searchParams.get("language_kind");
      return (!name || String(symbol.name || "").toLowerCase().indexOf(name.toLowerCase()) >= 0) &&
        (!languageKind || String(symbol.language_kind || "").toLowerCase().indexOf(languageKind.toLowerCase()) >= 0);
    }).sort(function (left, right) {
      const leftPath = left.locations && left.locations[0] ? paths[left.locations[0].span.file_id] || "" : "";
      const rightPath = right.locations && right.locations[0] ? paths[right.locations[0].span.file_id] || "" : "";
      return leftPath.localeCompare(rightPath) || String(left.name).localeCompare(String(right.name)) || String(left.id).localeCompare(String(right.id));
    });
  } else {
    const subjectIDs = new Set(embeddedIDs(url, "subject_id"));
    items = (snapshot.documentation || []).filter(function (record) {
      if (subjectIDs.size && (!record.subject_ref || !subjectIDs.has(record.subject_ref.id))) return false;
      if (!files) return true;
      if (!record.subject_ref) return false;
      return record.subject_ref.kind === "file" ? files.has(record.subject_ref.id) : record.subject_ref.kind === "symbol" && symbols.has(record.subject_ref.id);
    }).map(function (record) {
	  return compactEmbeddedDocumentation(record, url.searchParams.get("include_documentation_text") === "true");
	}).sort(function (left, right) { return String(left.id).localeCompare(String(right.id)); });
  }
  return embeddedPage(items, snapshot, url);
}

function embeddedSourceEvidence(context, url) {
  const index = embeddedSourceIndex(context);
  const snapshot = embeddedSourceSnapshot(index, url);
  if (!snapshot) throw new Error("The exported source-facts scope is unavailable.");
  const entityID = decodeURIComponent(url.pathname.slice(url.pathname.lastIndexOf("/") + 1));
	const includeDocumentationText = url.searchParams.get("include_documentation_text") === "true";
  const result = { snapshot_id: snapshot.snapshot_id, scope_id: snapshot.scope_context && snapshot.scope_context.scope_id || "", entity_id: entityID, spans: [], source_reference_ids: [], documentation: [], relations: [] };
  (snapshot.symbols || []).forEach(function (symbol) {
    if (symbol.id !== entityID) return;
    (symbol.locations || []).forEach(function (location) {
      result.spans.push(location.span);
      (location.source_reference_ids || []).forEach(function (id) { result.source_reference_ids.push(id); });
    });
  });
  (snapshot.documentation || []).forEach(function (record) {
    if (record.id === entityID || record.subject_ref && record.subject_ref.id === entityID) {
		result.documentation.push(compactEmbeddedDocumentation(record, includeDocumentationText));
      (record.spans || []).forEach(function (span) { result.spans.push(span); });
	  (record.source_reference_ids || []).forEach(function (id) { result.source_reference_ids.push(id); });
    }
  });
	(snapshot.occurrences || []).forEach(function (occurrence) {
		if (occurrence.id === entityID && occurrence.source_span) result.spans.push(occurrence.source_span);
	});
  (snapshot.relations || []).forEach(function (relation) {
    if (relation.id === entityID || relation.from_ref && relation.from_ref.id === entityID || relation.to_ref && relation.to_ref.id === entityID) {
      result.relations.push(relation);
      (relation.evidence_spans || []).forEach(function (span) { result.spans.push(span); });
    }
  });
  if (!result.spans.length && !result.documentation.length && !result.relations.length) throw new Error("The source-fact evidence was not found in the exported model.");
  result.source_reference_ids = Array.from(new Set(result.source_reference_ids)).sort();
  return result;
}

function compactEmbeddedDocumentation(record, includeText) {
	if (includeText) return record;
	const copy = Object.assign({}, record);
	delete copy.raw_text;
	const characters = Array.from(String(copy.normalized_text || ""));
	if (characters.length > 512) {
		copy.normalized_text = characters.slice(0, 512).join("");
		copy.completeness = "truncated";
	}
	return copy;
}
