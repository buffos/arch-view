import { classForState, confidenceState, escapeHTML, formatLanguage, nodeLanguageBadge, nodeLanguageText, referenceScopeLabel } from "./utils.js";
import { samePath } from "./utils.js";
import { showError, showNotice } from "./view.js";
import { affectedFileIDs, loadMoreQualityFindings, loadQualityFindings, qualityAffectedOnly, qualityFileFilterMarkup, qualityReportStatus, renderQualityOverview, setQualityFindingRuleFilter, toggleQualityAffectedOnly } from "./quality.js";

export const inspectionSections = Object.freeze([
  "overview",
  "structure",
  "files",
  "symbols",
  "dependencies",
  "evidence",
  "technical"
]);

const collectionSections = new Set(["files", "symbols"]);

export function parseInspectionRoute(search) {
  const params = new URLSearchParams(search || "");
  if (params.get("view") !== "inspection") return null;
  const kind = params.get("kind") || "";
  const id = params.get("id") || "";
  if (!kind || !id) return { invalid: true, message: "This inspection link is missing the node kind or id." };
  if (kind !== "node") return { invalid: true, message: "Only nodes and groups can be opened in the full inspection view." };
  const requestedSection = params.get("section") || "overview";
  return {
    kind: kind,
    id: id,
    scope: params.get("scope") || "",
    path: params.getAll("path"),
    referenceVisibility: params.get("reference_visibility") || "",
    section: inspectionSections.includes(requestedSection) ? requestedSection : "overview"
  };
}

export function serializeInspectionRoute(route) {
  const params = new URLSearchParams();
  params.set("view", "inspection");
  params.set("kind", route.kind || "node");
  params.set("id", route.id || "");
  if (route.scope) params.set("scope", route.scope);
  (route.path || []).forEach(function (segment) { params.append("path", segment); });
  if (route.referenceVisibility) params.set("reference_visibility", route.referenceVisibility);
  if (route.section && route.section !== "overview") params.set("section", route.section);
  return "/?" + params.toString();
}

export function parseGraphRoute(search) {
  const params = new URLSearchParams(search || "");
  if (params.get("view") === "inspection") return null;
  const kind = params.get("selected_kind") || "";
  const id = params.get("selected_id") || "";
  return {
    path: params.getAll("path"),
    scope: params.get("scope") || "",
    referenceVisibility: params.get("reference_visibility") || "",
    selected: kind && id ? { kind: kind, id: id } : null
  };
}

export function serializeGraphRoute(route) {
  const params = new URLSearchParams();
  if (route.scope) params.set("scope", route.scope);
  (route.path || []).forEach(function (segment) { params.append("path", segment); });
  if (route.referenceVisibility) params.set("reference_visibility", route.referenceVisibility);
  if (route.selected && route.selected.kind && route.selected.id) {
    params.set("selected_kind", route.selected.kind);
    params.set("selected_id", route.selected.id);
  }
  const query = params.toString();
  return query ? "/?" + query : "/";
}

export function formatReadableLocation(start, end, path) {
  const file = String(path || "File location unavailable");
  if (!start || !start.line || start.line < 1) return file + " · File provenance only";
  const startLine = Number(start.line);
  const endLine = end && end.line && Number(end.line) >= startLine ? Number(end.line) : startLine;
  const range = endLine === startLine ? "Line " + startLine : "Lines " + startLine + "–" + endLine;
  return file + " · " + range;
}

export function formatSpanLocation(span, filePath) {
  if (!span || !filePath) return "File provenance only";
  return formatReadableLocation(span.start, span.end, filePath);
}

export function inspectionState(state, details) {
  const value = String(state || "unknown").toLowerCase();
  if (["available", "complete", "observed", "populated"].includes(value)) return { label: "Available", className: "ok", copy: details || "Available in the active scope." };
  if (["partial", "loading"].includes(value)) return { label: value === "loading" ? "Loading" : "Partial", className: "warning", copy: details || "Some information is available, but coverage is incomplete." };
  if (value === "unsupported") return { label: "Unsupported", className: "warning", copy: details || "The active analyzer does not provide this information." };
  if (value === "missing") return { label: "Missing", className: "warning", copy: details || "No matching facts were reported for this item." };
  if (value === "unavailable") return { label: "Unavailable", className: "warning", copy: details || "This model revision does not provide this information." };
  return { label: "Unknown", className: "warning", copy: details || "The available evidence does not establish this state." };
}

export function inspectionScopeStatus(scene) {
  const value = scene || {};
  if (String(value.scope_status || "").toLowerCase() === "aggregate") return value.aggregate_status || value.status || "unknown";
  return value.scope_status || value.status || "unknown";
}

export function sourceExcerptAvailable(context) {
  return Boolean(context && context.sourceEnabled);
}

export function symbolKindOptions(items, selected) {
  const values = new Set();
  (items || []).forEach(function (item) {
    const value = typeof item === "string" ? item : item && item.language_kind;
    if (value) values.add(String(value));
  });
  if (selected) values.add(String(selected));
  return [{ value: "", label: "All kinds" }].concat(Array.from(values).sort(function (left, right) { return left.localeCompare(right); }).map(function (value) {
    return { value: value, label: value };
  }));
}

export function inspectionFilterFocusState(input) {
  if (!input || !input.dataset || !input.dataset.inspectionFilter) return null;
  return {
    section: input.dataset.inspectionFilter,
    selectionStart: typeof input.selectionStart === "number" ? input.selectionStart : null,
    selectionEnd: typeof input.selectionEnd === "number" ? input.selectionEnd : null
  };
}

export function restoreInspectionFilterFocus(root, focusState) {
  if (!root || !focusState || !root.querySelector) return;
  const input = root.querySelector('[data-inspection-filter="' + focusState.section + '"]');
  if (!input || !input.focus) return;
  input.focus();
  if (input.setSelectionRange && focusState.selectionStart != null && focusState.selectionEnd != null) input.setSelectionRange(focusState.selectionStart, focusState.selectionEnd);
}

export function mergeCollectionPage(existing, page, append, filter) {
	const items = append && existing ? (existing.items || []).concat(page.items || []) : (page.items || []);
	return {
		items: items,
		total: page.total,
		next_cursor: page.next_cursor || "",
		snapshot_id: page.snapshot_id,
		scope_id: page.scope_id,
		coverage: page.coverage || existing && existing.coverage || [],
		filter: filter || ""
	};
}

function detailRow(label, value, extraClass) {
  return '<div class="inspection-detail-row"><span class="inspection-detail-key">' + escapeHTML(label) + '</span><span class="inspection-detail-value ' + (extraClass || "") + '">' + escapeHTML(value == null ? "—" : value) + "</span></div>";
}

function statePill(state, details) {
  const descriptor = inspectionState(state, details);
  return '<span class="state-pill ' + descriptor.className + '">' + escapeHTML(descriptor.label) + "</span>";
}

function collectionCoverage(data, section) {
	const relevant = section === "files"
		? new Set(["source:files", "source:size"])
		: section === "symbols"
			? new Set(["source:declarations", "source:visibility"])
			: null;
	return (data && data.coverage || []).filter(function (item) { return !relevant || relevant.has(item.capability); });
}

function collectionState(data, items, section) {
	const states = collectionCoverage(data, section).map(function (item) { return String(item.status || "unknown").toLowerCase(); });
  (items || []).forEach(function (item) {
    if (item && item.analysis_status) states.push(String(item.analysis_status).toLowerCase());
  });
  if (states.includes("unsupported")) return "unsupported";
  if (states.includes("unknown") || states.includes("unparsed")) return "unknown";
  if (states.includes("partial")) return "partial";
  if (states.includes("unavailable")) return "unavailable";
  return items && items.length ? "available" : "missing";
}

function coverageMarkup(data, section) {
	return collectionCoverage(data, section).map(function (item) {
    return '<span class="inspection-coverage-item"><strong>' + escapeHTML(item.capability || "Source facts") + '</strong> ' + statePill(item.status || "unknown", item.reason || "") + '</span>';
  }).join("");
}

function sectionCard(title, content, className) {
  return '<section class="inspection-card ' + (className || "") + '"><div class="inspection-card-heading"><h3>' + escapeHTML(title) + "</h3></div>" + content + "</section>";
}

function currentNode(context) {
  const scene = context.state.scene;
  const route = context.state.inspectionRoute;
  if (!scene || !route) return null;
  return (scene.visible_nodes || []).find(function (node) { return node.id === route.id; }) || null;
}

function modelModule(context, moduleID) {
  const modules = context.state.model && context.state.model.modules || [];
  const exact = modules.find(function (module) { return module.id === moduleID; });
  if (exact || !context.aggregateEnabled || !context.state.activeScope || context.state.activeScope === "all") return exact || null;
  const prefix = context.state.activeScope + "::";
  return modules.find(function (module) {
    if (!module.id || !module.id.startsWith(prefix)) return false;
    try {
      return decodeURIComponent(module.id.slice(prefix.length)) === moduleID;
    } catch (_) {
      return false;
    }
  }) || null;
}

function moduleLabel(context, moduleID) {
  const module = modelModule(context, moduleID);
  return module ? (module.display_name || module.name || "Unnamed module") : "Module name unavailable";
}

function activeScopeLabel(context) {
  if (!context.aggregateEnabled) return "Model scope";
  if ((context.state.activeScope || "all") === "all") return "All scopes";
  const scope = (context.state.scopes || []).find(function (item) { return item.scope_id === context.state.activeScope; });
  return scope && scope.project_root ? scope.project_root : context.state.activeScope;
}

function scopeOptions(context) {
  if (!context.aggregateEnabled) return [{ value: "", label: "Model scope" }];
  const options = [{ value: "all", label: "All scopes" }];
  (context.state.scopes || []).slice().sort(function (left, right) { return String(left.scope_id).localeCompare(String(right.scope_id)); }).forEach(function (scope) {
    options.push({ value: scope.scope_id, label: (scope.project_root || scope.scope_id) + " · " + formatLanguage(scope.analyzer && scope.analyzer.language) });
  });
  return options;
}

function nodeStatusBadges(node) {
  const badges = [];
  if (node.cycle_state && node.cycle_state !== "none") badges.push('<span class="state-pill error">' + escapeHTML(node.cycle_state) + " cycle</span>");
  if (node.diagnostic_state && node.diagnostic_state !== "none") badges.push('<span class="state-pill warning">' + escapeHTML(node.diagnostic_state) + " diagnostics</span>");
  if (node.confidence_state && node.confidence_state !== "not_applicable" && node.confidence_state !== "high") badges.push('<span class="state-pill warning">' + escapeHTML(node.confidence_state) + " confidence</span>");
  if (node.identity_state && node.identity_state !== "stable") badges.push('<span class="state-pill warning">' + escapeHTML(node.identity_state) + " identity</span>");
  return badges.join("");
}

function cacheKey(context, section) {
  const route = context.state.inspectionRoute || {};
  const filter = context.state.inspectionFilters && context.state.inspectionFilters[section] || "";
  const kind = section === "symbols" ? context.state.inspectionSymbolKind || "" : "";
  const affected = section === "files" && qualityAffectedOnly(context) ? affectedFileIDs(context).join(",") : "";
  return [section, context.state.activeScope || "", route.id || "", filter, kind, affected].join("|");
}

export function sourceQuery(context, node, limit, cursor, section) {
	const query = new URLSearchParams();
	const scope = context.state.activeScope || "";
	if (scope) query.set("scope", scope);
	query.set("limit", String(limit || 25));
	if (cursor) query.set("cursor", cursor);
	const affectedOnly = section === "files" && qualityAffectedOnly(context);
	// The source-index boundary unions module_id and file_id selectors. An
	// affected-only request must therefore send only the report's exact file
	// subjects; including the inspected node's modules would broaden the page
	// back to every contained file.
	if (!affectedOnly) (node.module_ids || []).forEach(function (moduleID) { query.append("module_id", moduleID); });
	if (affectedOnly) {
		const affected = affectedFileIDs(context);
    if (affected.length) affected.forEach(function (fileID) { query.append("file_id", fileID); });
    else query.set("file_id", "quality:no-matching-files");
  }
  if (section === "symbols" && context.state.inspectionSymbolKind) query.set("language_kind", context.state.inspectionSymbolKind);
  return query;
}

function sourceEndpoint(context, api, section, node, limit, cursor) {
  const query = sourceQuery(context, node, limit, cursor, section);
  const filter = context.state.inspectionFilters && context.state.inspectionFilters[section] || "";
  if (filter && section === "files") query.set("path_prefix", filter);
  if (filter && section === "symbols") query.set("name", filter);
  if (section === "documentation") query.set("include_documentation_text", "false");
  return "/v1/models/" + encodeURIComponent(api.currentModelID()) + "/source-index/" + section + "?" + query.toString();
}

async function getCollection(context, api, section, node, cursor) {
  return api.getJSON(sourceEndpoint(context, api, section, node, 25, cursor));
}

function documentationEndpoint(context, api, node, subjectIDs) {
  const query = sourceQuery(context, node, 200, "", "documentation");
  query.set("include_documentation_text", "false");
  Array.from(new Set((subjectIDs || []).map(function (id) { return String(id || "").trim(); }).filter(Boolean))).forEach(function (id) {
    query.append("subject_id", id);
  });
  return "/v1/models/" + encodeURIComponent(api.currentModelID()) + "/source-index/documentation?" + query.toString();
}

function filePathMap(context) {
  return context.state.inspectionFilePaths || {};
}

async function loadFilePaths(context, api, node) {
  if (context.state.inspectionFilePathsLoaded) return;
  try {
    const page = await api.getJSON(sourceEndpoint(context, api, "files", node, 200, ""));
    const paths = filePathMap(context);
    (page.items || []).forEach(function (file) { paths[file.id] = file.path; });
    context.state.inspectionFilePathsLoaded = true;
  } catch (_) {
    // Symbols still render their bounded line state when file metadata is not available.
  }
}

async function loadDocumentation(context, api, node, items) {
  const requested = context.state.inspectionDocumentationRequested || {};
  const subjectIDs = (items || []).map(function (item) { return item && item.id; }).filter(Boolean);
  const pending = subjectIDs.filter(function (id) { return !requested[id]; });
  if (!pending.length) return;
  try {
    const page = await api.getJSON(documentationEndpoint(context, api, node, pending));
    const statuses = context.state.inspectionDocumentation || {};
    (page.items || []).forEach(function (record) {
      if (!record.subject_ref || !record.subject_ref.id) return;
      statuses[record.subject_ref.id] = record.status || "unknown";
    });
    pending.forEach(function (id) { requested[id] = true; });
  } catch (_) {
    // Documentation is supplementary; the primary bounded list remains useful.
  }
}

function renderOverview(context, node) {
  const counts = node.counts || {};
  const sourceCopy = context.sourceEnabled || context.embeddedExport
    ? "Source facts are available on demand. Open Files or Symbols to load bounded results."
    : "Source facts are unavailable for this model-only session.";
  const statusBadges = nodeStatusBadges(node);
  return '<div class="inspection-overview-grid">' +
    sectionCard("What this represents", '<div class="inspection-detail-table">' + detailRow("Kind", node.kind) + detailRow("Language", nodeLanguageText(context, node)) + detailRow("Hierarchy", (node.hierarchy_path || []).join(" / ") || "Top level") + detailRow("Active scope", activeScopeLabel(context)) + "</div>" + (statusBadges ? '<div class="inspection-status-row" aria-label="Non-neutral status">' + statusBadges + "</div>" : "")) +
    sectionCard("At a glance", '<div class="inspection-count-grid">' + countTile(counts.module_count, "modules") + countTile(counts.relationship_count, "visible relationships") + countTile(counts.internal_relationship_count, "collapsed relationships") + countTile(counts.evidence_count, "source references") + "</div>") +
    sectionCard("Source facts", '<div class="inspection-state-copy">' + statePill(context.sourceEnabled || context.embeddedExport ? "available" : "unavailable") + '<p>' + escapeHTML(sourceCopy) + "</p></div>") +
    sectionCard("Quality checks", renderQualityOverview(context), "quality-overview-card") +
    '</div><p class="inspection-guidance">This page uses reported facts, counts, statuses, and locations. It does not invent an explanation when the model has not supplied one.</p>';
}

function countTile(value, label) {
  return '<div class="inspection-count-tile"><strong>' + escapeHTML(value == null ? "—" : value) + '</strong><span>' + escapeHTML(label) + "</span></div>";
}

function renderStructure(context, node) {
  const ids = node.module_ids || [];
  if (!ids.length) return emptySection("missing", "No canonical module membership was reported for this node.");
  const modules = ids.map(function (moduleID) { return modelModule(context, moduleID); }).filter(Boolean);
  if (!modules.length) return emptySection("unknown", "The current model does not contain display metadata for this node's modules.");
  const rows = modules.map(function (module) {
    return '<li class="inspection-list-row"><div><strong>' + escapeHTML(module.display_name || module.name || "Unnamed module") + '</strong><span>' + escapeHTML([module.kind, formatLanguage(module.language), (module.tags || []).join(", ") || "No tags"].filter(Boolean).join(" · ")) + '</span></div></li>';
  }).join("");
  return '<div class="inspection-section-intro"><p>Canonical modules are the explicit structural membership reported for this node.</p><span class="muted">' + escapeHTML(modules.length + " module(s)") + '</span></div><ul class="inspection-list">' + rows + '</ul>';
}

function docsStatus(context, subjectID) {
  return context.state.inspectionDocumentation && context.state.inspectionDocumentation[subjectID] || "not reported";
}

function renderFiles(context, data) {
  const items = data.items || [];
  const state = collectionState(data, items, "files");
  if (!items.length && !data.error) return collectionMarkup(context, "files", data, state, emptySection(state, qualityAffectedOnly(context) ? "No files over the active line threshold were reported for this node." : "No explicitly contained files were reported for this node."));
  const rows = items.map(function (file) {
    const language = file.language && (file.language.dialect || file.language.id) || "unknown";
    return '<li class="inspection-list-row"><div><strong>' + escapeHTML(file.path || "Unnamed file") + '</strong><span>' + escapeHTML(formatLanguage(String(language).replace(/^language:/, "")) + " · " + (file.size && file.size.line_count || 0) + " lines · " + (file.size && file.size.byte_count || 0) + " bytes") + '</span></div><div class="inspection-row-aside">' + statePill(file.analysis_status || "unknown") + '<small>docs ' + escapeHTML(docsStatus(context, file.id)) + '</small></div></li>';
  }).join("");
  return collectionMarkup(context, "files", data, state, '<ul class="inspection-list">' + rows + "</ul>");
}

function symbolLocation(context, symbol) {
  const location = symbol.locations && symbol.locations[0];
  if (!location) return "File provenance only";
  return formatSpanLocation(location.span, filePathMap(context)[location.span && location.span.file_id]);
}

export function renderSymbols(context, data) {
  const items = data.items || [];
  const state = collectionState(data, items, "symbols");
  rememberSymbolKinds(context, items);
  if (!items.length && !data.error) return collectionMarkup(context, "symbols", data, state, emptySection(state, "No explicitly declared symbols were reported for this node."));
  const rows = items.map(function (symbol) {
    const title = symbol.qualified_name || symbol.name || "Unnamed symbol";
    const location = symbolLocation(context, symbol);
    const hasExactLocation = symbol.locations && symbol.locations.some(function (item) { return item.span && item.span.start && item.span.start.line > 0; });
    const action = hasExactLocation && sourceExcerptAvailable(context)
      ? '<button type="button" class="button tertiary" data-open-source-fact="' + escapeHTML(symbol.id) + '">View source</button>'
      : hasExactLocation && context.embeddedExport
        ? '<span class="inspection-source-unavailable">Excerpt unavailable in export</span>'
        : "";
    return '<li class="inspection-list-row"><div><strong>' + escapeHTML(title) + '</strong><span>' + escapeHTML([symbol.category, symbol.language_kind, symbol.visibility && symbol.visibility.classification, "docs " + docsStatus(context, symbol.id)].filter(Boolean).join(" · ")) + '</span><small>' + escapeHTML(location) + '</small></div><div class="inspection-row-aside">' + action + '</div></li>';
  }).join("");
  return collectionMarkup(context, "symbols", data, state, '<ul class="inspection-list">' + rows + "</ul>") + renderSourcePanel(context, items.map(function (symbol) { return symbol.id; }));
}

function rememberSymbolKinds(context, items) {
  if (!context || !context.state || !items || !items.length) return;
  const known = (context.state.inspectionSymbolKinds || []).concat(items.map(function (item) { return item && item.language_kind; }));
  context.state.inspectionSymbolKinds = symbolKindOptions(known).slice(1).map(function (option) { return option.value; });
}

function symbolKindFilterMarkup(context) {
  const selected = context.state.inspectionSymbolKind || "";
  const options = symbolKindOptions(context.state.inspectionSymbolKinds || [], selected);
  const optionMarkup = options.map(function (option) {
    return '<option value="' + escapeHTML(option.value) + '"' + (option.value === selected ? " selected" : "") + '>' + escapeHTML(option.label) + "</option>";
  }).join("");
  return '<label class="inspection-filter select-control inspection-kind-filter"><span>Symbol kind</span><select data-inspection-kind-filter="symbols" aria-label="Filter symbols by kind">' + optionMarkup + "</select></label>";
}

function collectionMarkup(context, section, data, state, content) {
  const value = context.state.inspectionFilters && context.state.inspectionFilters[section] || "";
  const filter = '<label class="inspection-filter"><span>Filter ' + escapeHTML(section) + '</span><input type="search" data-inspection-filter="' + escapeHTML(section) + '" value="' + escapeHTML(value) + '" placeholder="Search ' + escapeHTML(section) + '" autocomplete="off"></label>';
  const kindFilter = section === "symbols" ? symbolKindFilterMarkup(context) : "";
  const qualityFilter = section === "files" ? qualityFileFilterMarkup(context) : "";
  const more = data.next_cursor ? '<button type="button" class="button secondary" data-inspection-load-more="' + escapeHTML(section) + '">Load more</button>' : "";
	const coverage = coverageMarkup(data, section);
  const total = data.total == null ? "" : String(data.total) + " total";
  return '<div class="inspection-section-intro"><p>Results are explicitly bounded to 25 items per request.</p><span class="muted">' + escapeHTML(total) + '</span></div><div class="inspection-collection-state">' + statePill(state) + coverage + '</div><div class="inspection-filter-row">' + filter + kindFilter + qualityFilter + '</div>' + content + (more ? '<div class="inspection-load-more">' + more + "</div>" : "");
}

function emptySection(state, copy) {
  const descriptor = inspectionState(state, copy);
  return '<div class="inspection-empty"><div>' + statePill(state) + '</div><p>' + escapeHTML(descriptor.copy) + "</p></div>";
}

function renderDependencies(context, node) {
  const scene = context.state.scene || {};
  const nodesByID = {};
  (scene.visible_nodes || []).forEach(function (item) { nodesByID[item.id] = item; });
  const rows = [];
  (scene.visible_relationships || []).filter(function (relationship) { return relationship.from_visible_id === node.id; }).forEach(function (relationship) {
    const target = nodesByID[relationship.to_visible_id];
    rows.push('<li class="inspection-list-row"><div><strong>' + escapeHTML(target ? target.label : "Target outside this projection") + '</strong><span>' + escapeHTML([relationship.type, relationship.count + " occurrence(s)", relationship.confidence_state + " confidence"].join(" · ")) + '</span></div></li>');
  });
  (scene.reference_details || []).filter(function (reference) { return (reference.from_visible_ids || []).includes(node.id); }).forEach(function (reference) {
    rows.push('<li class="inspection-list-row"><div><strong>' + escapeHTML(reference.name || "External reference") + '</strong><span>' + escapeHTML([referenceScopeLabel(reference.scope), reference.count + " occurrence(s)", reference.confidence_state + " confidence"].join(" · ")) + '</span></div></li>');
  });
  if (!rows.length) return emptySection("missing", "No outgoing dependencies are visible for this node in the current projection.");
  return '<div class="inspection-section-intro"><p>Dependencies are grouped by the current graph projection and reference visibility.</p><span class="muted">' + escapeHTML(rows.length + " dependency group(s)") + '</span></div><ul class="inspection-list">' + rows.join("") + '</ul>';
}

function evidenceLinkMap(context) {
  const map = {};
  ((context.state.scene && context.state.scene.evidence_links) || []).forEach(function (link) { map[link.id] = link; });
  return map;
}

function evidenceHasLocation(link) {
  return Boolean(link && link.start && link.end && link.start.line > 0);
}

function evidenceIsFileProvenance(link) {
  return !evidenceHasLocation(link) && String(link && link.kind || "").toLowerCase() === "file";
}

function sourceKindLabel(link) {
  const value = String(link && link.kind || "source").replace(/[_-]+/g, " ").trim();
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : "Source";
}

function uniqueFileProvenance(links) {
  const seen = {};
  return links.filter(function (link) {
    const key = link.path || "Location unavailable";
    if (seen[key]) return false;
    seen[key] = true;
    return true;
  });
}

function evidenceSubsection(title, copy, count, content) {
  return '<section class="inspection-evidence-subsection"><div class="inspection-evidence-heading"><h4>' + escapeHTML(title) + '</h4><span class="muted">' + escapeHTML(count) + '</span></div><p class="inspection-evidence-copy">' + escapeHTML(copy) + '</p>' + content + '</section>';
}

export function renderEvidence(context, node) {
  const map = evidenceLinkMap(context);
  const links = (node.evidence_ids || []).map(function (id) { return map[id] || { id: id, kind: "Source reference", path: "Location unavailable" }; });
  if (!links.length) return emptySection("missing", "No source evidence is attached to this node.");
  const located = links.filter(evidenceHasLocation);
  const fileProvenance = uniqueFileProvenance(links.filter(evidenceIsFileProvenance));
  const unlocated = links.filter(function (link) { return !evidenceHasLocation(link) && !evidenceIsFileProvenance(link); });
  const fileRows = fileProvenance.slice(0, 25).map(function (link) {
    const symbol = link.symbol ? ' · ' + link.symbol : "";
    return '<li class="inspection-list-row"><div><strong>' + escapeHTML(link.path || "Location unavailable") + '</strong><span>Module/file association · file-level only' + escapeHTML(symbol) + '</span></div></li>';
  }).join("");
  const fileOverflow = fileProvenance.length > 25 ? '<p class="muted inspection-evidence-overflow">Showing the first 25 of ' + escapeHTML(fileProvenance.length) + ' associated files. Open Files for the complete bounded file list.</p>' : "";
  const locatedRows = located.slice(0, 25).map(function (link) {
    const location = formatReadableLocation(link.start, link.end, link.path);
    const action = sourceExcerptAvailable(context)
      ? '<button type="button" class="button tertiary" data-open-source-evidence="' + escapeHTML(link.id) + '">View source</button>'
      : context.embeddedExport
        ? '<span class="inspection-source-unavailable">Excerpt unavailable in export</span>'
        : "";
    return '<li class="inspection-list-row"><div><strong>' + escapeHTML(link.symbol || sourceKindLabel(link)) + '</strong><span>' + escapeHTML(sourceKindLabel(link) + " observation · reported source location") + '</span><small>' + escapeHTML(location) + '</small></div><div class="inspection-row-aside">' + action + '</div></li>';
  }).join("");
  const locatedOverflow = located.length > 25 ? '<p class="muted inspection-evidence-overflow">Showing the first 25 of ' + escapeHTML(located.length) + ' located references.</p>' : "";
  const unlocatedRows = unlocated.slice(0, 25).map(function (link) {
    return '<li class="inspection-list-row"><div><strong>' + escapeHTML(link.symbol || sourceKindLabel(link)) + '</strong><span>' + escapeHTML(sourceKindLabel(link) + " observation · no line range reported") + '</span><small>' + escapeHTML(link.path || "Location unavailable") + '</small></div></li>';
  }).join("");
  const unlocatedOverflow = unlocated.length > 25 ? '<p class="muted inspection-evidence-overflow">Showing the first 25 of ' + escapeHTML(unlocated.length) + ' references without line ranges.</p>' : "";
  const summary = [
    links.length + " source reference(s) reported",
    located.length + " with line locations",
    fileProvenance.length + " associated file(s)"
  ].join(" · ");
  const groups = [];
  if (fileProvenance.length) groups.push(evidenceSubsection("File provenance", "These files are attached to the selected node's module observations. They establish source context and module/file association, but do not identify a line or prove a separate relationship.", fileProvenance.length + " file(s)", '<ul class="inspection-list">' + fileRows + '</ul>' + fileOverflow));
  if (located.length) groups.push(evidenceSubsection("Reported source locations", "These line ranges are the analyzer's source anchors for observations associated with this node. The kind identifies the type of observation reported.", located.length + " location(s)", '<ul class="inspection-list">' + locatedRows + '</ul>' + locatedOverflow));
  if (unlocated.length) groups.push(evidenceSubsection("Other unlocated references", "The analyzer attached these references, but did not provide a trustworthy line range. Treat them as context only.", unlocated.length + " reference(s)", '<ul class="inspection-list">' + unlocatedRows + '</ul>' + unlocatedOverflow));
  return '<div class="inspection-section-intro"><p>These source references are attached to the selected node and its reported module and relationship observations. They explain where the model got its input; they do not add an unsupported claim.</p><span class="muted">' + escapeHTML(summary) + '</span></div>' + groups.join("") + renderSourcePanel(context, links.map(function (link) { return link.id; }));
}

function renderTechnical(context, node) {
  if (context.state.inspectionTechnicalLoading) return '<div class="inspection-loading" role="status">Loading technical details…</div>';
  const index = context.state.sourceIndex;
  const snapshot = index && (index.projection || index.snapshots && index.snapshots[0]);
  const queriedSnapshot = context.state.inspectionSnapshot;
  const analyzer = context.state.model && context.state.model.analyzer || {};
  const sourceDetails = snapshot ? [
    detailRow("Snapshot", snapshot.snapshot_id || "Unavailable"),
    detailRow("Snapshot digest", snapshot.snapshot_digest && snapshot.snapshot_digest.value || "Unavailable"),
    detailRow("Provider", snapshot.producer && snapshot.producer.analyzer_id || "Unavailable"),
    detailRow("Provider version", snapshot.producer && snapshot.producer.analyzer_version || "Unavailable")
  ].join("") : [
    detailRow("Snapshot", queriedSnapshot && queriedSnapshot.snapshot_id || "Loaded on demand by bounded sections"),
    detailRow("Scope", queriedSnapshot && queriedSnapshot.scope_id || activeScopeLabel(context)),
    detailRow("Provider", analyzer.id || "Unavailable"),
    detailRow("Provider version", analyzer.version || "Unavailable")
  ].join("");
  const moduleIDs = (node.module_ids || []).map(function (id) { return detailRow("Module ID", id); }).join("");
  const relationshipIDs = (node.internal_relationship_ids || []).map(function (id) { return detailRow("Internal relationship ID", id); }).join("");
  const technicalData = context.state.inspectionTechnical || {};
  const technicalFiles = technicalData.items || [];
  const hashes = technicalFiles.map(function (file) {
    return detailRow((file.path || "File") + " hash", file.size && file.size.content_hash && file.size.content_hash.value || "Unavailable");
  }).join("");
  const metadata = technicalData.error
    ? detailRow("File metadata", "Unavailable")
    : detailRow("File metadata", technicalFiles.length + " bounded file(s)") + hashes;
  const technicalState = technicalData.error
    ? '<div class="inspection-state-copy">' + statePill("unavailable") + '<p>' + escapeHTML(technicalData.error) + "</p></div>"
    : "";
  const technical = '<details class="technical-disclosure"><summary>Show technical identifiers and provenance</summary><div class="inspection-detail-table">' + detailRow("Node ID", node.id) + moduleIDs + relationshipIDs + sourceDetails + detailRow("Model revision", context.state.scene && context.state.scene.model_revision || "Unavailable") + metadata + "</div></details>";
  const copyValue = JSON.stringify({ node_id: node.id, module_ids: node.module_ids || [], internal_relationship_ids: node.internal_relationship_ids || [], snapshot_id: snapshot && snapshot.snapshot_id || "" }, null, 2);
  return '<div class="inspection-section-intro"><p>Technical details support debugging and machine-assisted workflows. They are secondary to the human-readable sections.</p></div>' + technicalState + technical + '<button type="button" class="button secondary copy-button" data-copy-value="' + escapeHTML(copyValue) + '">Copy technical details</button>';
}

function renderSourcePanel(context, allowedIDs) {
  const source = context.state.source;
  if (!source) return "";
  if (allowedIDs && source.link && source.link.id && !allowedIDs.some(function (id) { return String(id) === String(source.link.id); })) return "";
  if (source.loading) return '<section class="inspection-source-panel"><h3>Read-only source</h3><p class="muted">Loading bounded source excerpt…</p></section>';
  if (source.error) return '<section class="inspection-source-panel"><h3>Read-only source</h3><div class="notice error">' + escapeHTML(source.error) + "</div></section>";
  const excerpt = source.data;
  if (!excerpt) return "";
  const lines = (excerpt.lines || []).map(function (line) {
    return '<span class="source-line"><span class="source-number" aria-hidden="true">' + escapeHTML(line.number) + '</span><span class="source-text">' + escapeHTML(line.text) + "</span></span>";
  }).join("");
  const range = formatReadableLocation(excerpt.start, excerpt.end, excerpt.path);
  return '<section class="inspection-source-panel"><h3>Read-only source</h3><div class="source-meta"><strong>' + escapeHTML(excerpt.path) + '</strong><span>' + escapeHTML(range.replace(excerpt.path + " · ", "")) + ' · read-only</span></div><pre class="source-excerpt"><code>' + lines + '</code></pre></section>';
}

function renderSection(context, section, node) {
  if (section === "overview") return renderOverview(context, node);
  if (section === "structure") return renderStructure(context, node);
  if (section === "dependencies") return renderDependencies(context, node);
  if (section === "evidence") return renderEvidence(context, node);
  if (section === "technical") return renderTechnical(context, node);
  const key = cacheKey(context, section);
  const data = context.state.inspectionCache[key];
  if (context.state.inspectionLoading === key) return collectionMarkup(context, section, { items: [] }, "loading", '<div class="inspection-loading" role="status">Loading ' + escapeHTML(section) + "…</div>");
  if (!data) return collectionMarkup(context, section, { items: [] }, "loading", '<div class="inspection-loading" role="status">Preparing ' + escapeHTML(section) + "…</div>");
  if (data.error) return collectionMarkup(context, section, data, "unavailable", emptySection("unavailable", data.error));
  return section === "files" ? renderFiles(context, data) : renderSymbols(context, data);
}

function renderNavigation(context) {
  const node = currentNode(context);
  const counts = {
    structure: node && node.module_ids ? node.module_ids.length : 0,
    dependencies: node ? dependencyCount(context, node) : 0,
    evidence: node && node.evidence_ids ? node.evidence_ids.length : 0
  };
  context.elements.inspectionNavigation.querySelectorAll("[data-inspection-section]").forEach(function (item) {
    const section = item.dataset.inspectionSection;
    const active = context.state.inspectionSection === section;
    item.classList.toggle("active", active);
    if (active) item.setAttribute("aria-current", "page");
    else item.removeAttribute("aria-current");
    const count = item.querySelector("[data-inspection-count]");
    if (count) count.textContent = counts[section] == null ? "" : String(counts[section]);
  });
}

function dependencyCount(context, node) {
  const scene = context.state.scene || {};
  return (scene.visible_relationships || []).filter(function (relationship) { return relationship.from_visible_id === node.id; }).length + (scene.reference_details || []).filter(function (reference) { return (reference.from_visible_ids || []).includes(node.id); }).length;
}

function renderHeader(context, node) {
  context.elements.inspectionTitle.textContent = node.label;
  context.elements.inspectionSubtitle.textContent = [node.kind, nodeLanguageText(context, node), (node.hierarchy_path || []).join(" / ") || "Top level", activeScopeLabel(context)].join(" · ");
  context.elements.inspectionScope.innerHTML = scopeOptions(context).map(function (option) { return '<option value="' + escapeHTML(option.value) + '">' + escapeHTML(option.label) + "</option>"; }).join("");
  context.elements.inspectionScope.value = context.state.activeScope || "";
  if (!Array.from(context.elements.inspectionScope.options).some(function (option) { return option.value === context.elements.inspectionScope.value; })) context.elements.inspectionScope.value = context.aggregateEnabled ? "all" : "";
  context.elements.inspectionScopeStatus.innerHTML = statePill(inspectionScopeStatus(context.state.scene));
}

function renderContent(context, node) {
  const section = context.state.inspectionSection;
  const focusState = typeof document !== "undefined" ? inspectionFilterFocusState(document.activeElement) : null;
  context.elements.inspectionContent.innerHTML = '<div class="inspection-section-heading"><p class="eyebrow">' + escapeHTML(section === "technical" ? "TECHNICAL DETAILS" : section.toUpperCase()) + '</p><h3>' + escapeHTML(section === "technical" ? "Technical details" : section.charAt(0).toUpperCase() + section.slice(1)) + "</h3></div>" + renderSection(context, section, node);
  bindContentActions(context);
  restoreInspectionFilterFocus(context.elements.inspectionContent, focusState);
}

function bindContentActions(context) {
  context.elements.inspectionContent.querySelectorAll("[data-inspection-filter]").forEach(function (input) {
    input.addEventListener("input", function () {
      const section = input.dataset.inspectionFilter;
      context.state.inspectionFilters[section] = input.value.trim().toLowerCase();
      invalidateCollectionCache(context, section);
      renderInspectionView(context);
      void ensureSection(context, context._inspectionAPI, section);
    });
  });
  context.elements.inspectionContent.querySelectorAll("[data-inspection-kind-filter]").forEach(function (select) {
    select.addEventListener("change", function () {
      context.state.inspectionSymbolKind = select.value || "";
      invalidateCollectionCache(context, "symbols");
      renderInspectionView(context);
      void ensureSection(context, context._inspectionAPI, "symbols");
    });
  });
  context.elements.inspectionContent.querySelectorAll("[data-quality-finding-rule]").forEach(function (select) {
    select.addEventListener("change", function () {
      const value = select.value || "";
      setQualityFindingRuleFilter(context, value);
      renderInspectionView(context);
      const next = context.elements.inspectionContent.querySelector("[data-quality-finding-rule]");
      if (next) next.focus();
      void ensureSection(context, context._inspectionAPI, "overview");
    });
  });
  context.elements.inspectionContent.querySelectorAll("[data-inspection-load-more]").forEach(function (button) {
    button.addEventListener("click", function () { void loadCollection(context, context._inspectionAPI, button.dataset.inspectionLoadMore, true); });
  });
  context.elements.inspectionContent.querySelectorAll("[data-quality-affected-only]").forEach(function (button) {
    button.addEventListener("click", function () {
      toggleQualityAffectedOnly(context);
      invalidateCollectionCache(context, "files");
      renderInspectionView(context);
      void ensureSection(context, context._inspectionAPI, "files");
    });
  });
  context.elements.inspectionContent.querySelectorAll("[data-quality-findings-load-more]").forEach(function (button) {
    button.addEventListener("click", function () {
      void loadMoreQualityFindings(context, context._inspectionAPI).then(function () {
        renderInspectionView(context);
      }).catch(function () {
        renderInspectionView(context);
      });
      renderInspectionView(context);
    });
  });
  context.elements.inspectionContent.querySelectorAll("[data-quality-evidence]").forEach(function (button) {
    button.addEventListener("click", function () {
      void context._inspectionServices.openQualityEvidence(button.dataset.qualityEvidence);
      renderInspectionView(context);
    });
  });
  context.elements.inspectionContent.querySelectorAll("[data-quality-source]").forEach(function (button) {
    button.addEventListener("click", function () {
      void context._inspectionServices.openQualitySource(button.dataset.qualitySource);
      renderInspectionView(context);
    });
  });
  context.elements.inspectionContent.querySelectorAll("[data-quality-baseline-open]").forEach(function (button) {
    button.addEventListener("click", function () {
      if (context._inspectionServices.openQualityBaseline) context._inspectionServices.openQualityBaseline(button.dataset.qualityBaselineOpen);
    });
  });
  context.elements.inspectionContent.querySelectorAll("[data-open-source-evidence]").forEach(function (button) {
    button.addEventListener("click", function () { context._inspectionServices.openSource(button.dataset.openSourceEvidence); });
  });
  context.elements.inspectionContent.querySelectorAll("[data-open-source-fact]").forEach(function (button) {
    button.addEventListener("click", function () { context._inspectionServices.openSourceFact(button.dataset.openSourceFact); });
  });
  context.elements.inspectionContent.querySelectorAll("[data-copy-value]").forEach(function (button) {
    button.addEventListener("click", function () {
      const value = button.dataset.copyValue || "";
      const done = function () { button.textContent = "Copied"; window.setTimeout(function () { button.textContent = "Copy technical details"; }, 1200); };
      if (navigator.clipboard && navigator.clipboard.writeText) navigator.clipboard.writeText(value).then(done).catch(function () { button.textContent = "Copy unavailable"; });
      else button.textContent = "Copy unavailable";
    });
  });
}

function invalidateCollectionCache(context, section) {
  Object.keys(context.state.inspectionCache).forEach(function (key) {
    if (key.indexOf(section + "|") === 0) delete context.state.inspectionCache[key];
  });
}

async function loadCollection(context, api, section, append) {
  const node = currentNode(context);
  if (!node || !collectionSections.has(section)) return;
  const key = cacheKey(context, section);
  const existing = context.state.inspectionCache[key];
  if (append && (!existing || !existing.next_cursor)) return;
  if (!append && existing) return;
  context.state.inspectionLoading = key;
  renderInspectionView(context);
  try {
    if (section === "symbols") await loadFilePaths(context, api, node);
    const page = await getCollection(context, api, section, node, append && existing ? existing.next_cursor : "");
    await loadDocumentation(context, api, node, page.items || []);
	context.state.inspectionSnapshot = { snapshot_id: page.snapshot_id || "", scope_id: page.scope_id || "" };
	context.state.inspectionCache[key] = mergeCollectionPage(existing, page, append, context.state.inspectionFilters[section]);
    context.state.inspectionLoading = "";
    renderInspectionView(context);
  } catch (error) {
    context.state.inspectionCache[key] = { items: [], total: 0, next_cursor: "", error: error.message || "This bounded source query is unavailable.", filter: context.state.inspectionFilters[section] || "" };
    context.state.inspectionLoading = "";
    renderInspectionView(context);
  }
}

async function ensureSection(context, api, section) {
  if (!api) return;
  if (section === "overview") {
    if (qualityReportStatus(context) !== "available" || context.state.qualityFindingsPage || context.state.qualityFindingsLoading) return;
    try {
      await loadQualityFindings(context, api);
    } catch (_) {
      // The report summary remains useful when the bounded findings query is unavailable.
    }
    renderInspectionView(context);
    return;
  }
  if (section === "technical") {
    if (context.state.inspectionTechnical || context.state.inspectionTechnicalLoading) return;
    const node = currentNode(context);
    if (!node) return;
    context.state.inspectionTechnicalLoading = true;
    renderInspectionView(context);
    try {
      const page = await getCollection(context, api, "files", node);
      context.state.inspectionTechnical = { items: page.items || [], total: page.total || 0 };
      context.state.inspectionSnapshot = { snapshot_id: page.snapshot_id || "", scope_id: page.scope_id || "" };
    } catch (error) {
      context.state.inspectionTechnical = { items: [], total: 0, error: error.message || "Technical source metadata is unavailable." };
    } finally {
      context.state.inspectionTechnicalLoading = false;
      renderInspectionView(context);
    }
    return;
  }
  if (!collectionSections.has(section)) return;
  const key = cacheKey(context, section);
  if (context.state.inspectionCache[key] || context.state.inspectionLoading === key) return;
  await loadCollection(context, api, section, false);
}

function renderInspectionView(context) {
  const graphVisible = context.state.viewMode !== "inspection";
  context.elements.graphView.hidden = !graphVisible;
  context.elements.inspectionView.hidden = graphVisible;
  if (graphVisible) return;
  const node = currentNode(context);
  if (!node) {
    context.elements.inspectionContent.innerHTML = '<div class="inspection-empty"><p>The selected node is not visible in the current scope.</p></div>';
    return;
  }
  renderHeader(context, node);
  renderNavigation(context);
  renderContent(context, node);
}

export function createInspectionController(context, api, services) {
  let navigation = null;
  let initialRoute = null;
  context._inspectionAPI = api;
  context._inspectionServices = services;
  context.state.inspectionFilters = { files: "", symbols: "" };
  context.state.inspectionSymbolKind = "";
  context.state.inspectionSymbolKinds = [];
  context.state.inspectionFilePaths = {};
  context.state.inspectionFilePathsLoaded = false;
  context.state.inspectionDocumentation = {};
  context.state.inspectionDocumentationRequested = {};

  function graphRouteForCurrentState() {
    return {
      path: context.state.scene ? (context.state.scene.hierarchy_path || []).slice() : [],
      scope: context.state.activeScope || "",
      referenceVisibility: context.state.referenceVisibility || "",
      selected: context.state.selected
    };
  }

  function graphURLForRoute(route) {
    return serializeGraphRoute(route);
  }

  function setGraphMode(route, message, tone) {
    context.state.viewMode = "graph";
    context.state.inspectionRoute = null;
    context.state.inspectionLoading = "";
    if (route && route.scope) context.state.activeScope = route.scope;
    if (route && route.referenceVisibility) context.state.referenceVisibility = route.referenceVisibility;
    if (route && Object.prototype.hasOwnProperty.call(route, "selected")) context.state.selected = route.selected;
    if (message) tone === "info" ? showNotice(context, message) : showError(context, message);
    services.renderAll();
    renderInspectionView(context);
  }

  async function applyGraphRoute(route) {
    context.state.viewMode = "graph";
    context.state.inspectionRoute = null;
    if (route.scope && context.aggregateEnabled) context.state.activeScope = route.scope;
    if (route.referenceVisibility) context.state.referenceVisibility = route.referenceVisibility;
    const scene = context.state.scene;
    const needsScene = !scene || !samePath(scene.hierarchy_path || [], route.path || []) || (context.aggregateEnabled && route.scope && context.state.activeScope !== scene.scope_id && route.scope !== "all");
    if (needsScene && navigation) await navigation.loadScene(route.path || []);
    if (route.selected && context.state.scene && context.state.scene.visible_nodes.some(function (node) { return node.id === route.selected.id; })) context.state.selected = route.selected;
    services.renderAll();
  }

  async function fallback(route, message, tone) {
    let scope = route && route.scope || context.state.activeScope || "";
    if (context.aggregateEnabled && scope && scope !== "all" && !(context.state.scopes || []).some(function (item) { return item.scope_id === scope; })) scope = "all";
    context.state.activeScope = scope;
    const path = context.state.scene && Array.isArray(context.state.scene.hierarchy_path)
      ? context.state.scene.hierarchy_path.slice()
      : route && route.path || [];
    const graphRoute = { path: path, scope: scope, referenceVisibility: route && route.referenceVisibility || context.state.referenceVisibility || "", selected: null };
    if (!context.state.scene && navigation) await navigation.loadScene([]);
    if (window.history && window.history.replaceState) window.history.replaceState({ archView: "graph", route: graphRoute }, "", graphURLForRoute(graphRoute));
    setGraphMode(graphRoute, message, tone);
  }

  async function applyRoute(route, options) {
    options = options || {};
    if (!route || route.invalid) {
      await fallback(route || {}, route && route.message || "This inspection link is invalid. The architecture graph is shown instead.");
      return false;
    }
    if (!navigation) {
      initialRoute = route;
      return true;
    }
    if (route.scope && context.aggregateEnabled) context.state.activeScope = route.scope;
    if (route.referenceVisibility) context.state.referenceVisibility = route.referenceVisibility;
    context.state.viewMode = "inspection";
    context.state.inspectionRoute = Object.assign({}, route);
    context.state.inspectionSection = route.section || "overview";
    context.state.inspectionCache = {};
    context.state.inspectionFilePaths = {};
    context.state.inspectionFilePathsLoaded = false;
    context.state.inspectionDocumentation = {};
    context.state.inspectionDocumentationRequested = {};
    context.state.inspectionSymbolKinds = [];
    context.state.inspectionSnapshot = null;
    context.state.inspectionTechnical = null;
    context.state.inspectionTechnicalLoading = false;
    const scene = context.state.scene;
    const needsScene = !scene || !samePath(scene.hierarchy_path || [], route.path || []) || (context.aggregateEnabled && route.scope && route.scope !== "all" && scene.scope_id !== route.scope);
    if (needsScene && !(await navigation.loadScene(route.path || []))) {
      await fallback(route, "The inspection scope or hierarchy is no longer available. The architecture graph is shown instead.");
      return false;
    }
    const node = currentNode(context);
    if (!node) {
      const scopeChange = options.scopeChange === true;
      await fallback(route, scopeChange
        ? "This node is not present in the selected analysis scope. Showing that scope's graph overview."
        : "This inspection link does not identify a visible node in the selected scope. The architecture graph is shown instead.", scopeChange ? "info" : "error");
      return false;
    }
    context.state.selected = { kind: "node", id: node.id };
    services.renderAll();
    renderInspectionView(context);
    if (services.loadQualityReport) void services.loadQualityReport();
    void ensureSection(context, api, context.state.inspectionSection);
    if (options.focus !== false && context.elements.inspectionContent) context.elements.inspectionContent.focus();
    return true;
  }

  function openInspection(kind, id) {
    if (kind !== "node") return;
    const node = context.state.scene && context.state.scene.visible_nodes.find(function (item) { return item.id === id; });
    if (!node) return;
    const route = { kind: "node", id: id, scope: context.state.activeScope || "", path: (context.state.scene.hierarchy_path || []).slice(), referenceVisibility: context.state.referenceVisibility || "", section: "overview" };
    const graphRoute = graphRouteForCurrentState();
    if (window.history && window.history.replaceState) window.history.replaceState({ archView: "graph", route: graphRoute }, "", graphURLForRoute(graphRoute));
    if (window.history && window.history.pushState) window.history.pushState({ archView: "inspection", route: route }, "", serializeInspectionRoute(route));
    void applyRoute(route);
  }

  function selectSection(section) {
    if (!inspectionSections.includes(section)) return;
    context.state.inspectionSection = section;
    if (context.state.inspectionRoute) {
      context.state.inspectionRoute.section = section;
      if (window.history && window.history.replaceState) window.history.replaceState({ archView: "inspection", route: context.state.inspectionRoute }, "", serializeInspectionRoute(context.state.inspectionRoute));
    }
    renderInspectionView(context);
    void ensureSection(context, api, section);
  }

  function handlePopState() {
    const route = parseInspectionRoute(window.location.search);
    if (route) {
      void applyRoute(route, { focus: false });
      return;
    }
    void applyGraphRoute(parseGraphRoute(window.location.search) || { path: [], selected: null });
  }

  function bindNavigation(value) {
    navigation = value;
    window.addEventListener("popstate", handlePopState);
  }

  function initialize() {
    const route = initialRoute || parseInspectionRoute(window.location.search);
    if (route) {
      if (route.invalid) {
        void fallback(route, route.message);
        return;
      }
      const graphRoute = { path: route.path || [], scope: route.scope || context.state.activeScope || "", referenceVisibility: route.referenceVisibility || context.state.referenceVisibility || "", selected: { kind: route.kind, id: route.id } };
      if (window.history && window.history.replaceState) {
        window.history.replaceState({ archView: "graph", route: graphRoute }, "", graphURLForRoute(graphRoute));
        window.history.pushState({ archView: "inspection", route: route }, "", serializeInspectionRoute(route));
      }
      void applyRoute(route, { initial: true, focus: false });
      return;
    }
    void applyGraphRoute(parseGraphRoute(window.location.search) || { path: [], selected: null });
  }

  context.elements.inspectionBack.addEventListener("click", function () {
    if (window.history && window.history.state && window.history.state.archView === "inspection") window.history.back();
    else void applyGraphRoute(graphRouteForCurrentState());
  });
  context.elements.inspectionNavigation.addEventListener("click", function (event) {
    const item = event.target.closest("[data-inspection-section]");
    if (item) selectSection(item.dataset.inspectionSection);
  });
  context.elements.inspectionScope.addEventListener("change", function () {
    const route = context.state.inspectionRoute;
    context.state.activeScope = context.elements.inspectionScope.value || "";
    if (route) {
      route.scope = context.state.activeScope;
      if (window.history && window.history.replaceState) window.history.replaceState({ archView: "inspection", route: route }, "", serializeInspectionRoute(route));
      void applyRoute(route, { focus: false, scopeChange: true });
    }
  });

  return { applyRoute, bindNavigation, initialize, openInspection, render: function () { renderInspectionView(context); }, selectSection };
}
