import { classForState, confidenceState, displayProjectRoot, escapeHTML, formatLanguage, formatList, nodeLanguageBadge, nodeLanguageText, referenceScopeLabel } from "./utils.js";
import { renderQualitySummary } from "./quality.js";

export function selectEntity(context, kind, id, preserveDoubleClick, services) {
  if (!preserveDoubleClick) context.state.lastNodeClick = null;
  context.state.selected = { kind: kind, id: id };
  context.state.source = null;
  context.state.sourceRequest += 1;
  services.renderGraph();
  services.renderAccessibleList();
  services.renderDetails();
}

function nodeListMeta(node) {
  const parts = [node.kind, "language " + (node.language || "Unknown"), "identity " + (node.identity_state || "stable")];
  if (node.confidence_state !== "not_applicable") parts.push(node.confidence_state + " confidence");
  if (node.counts.internal_relationship_count > 0) parts.push(node.counts.internal_relationship_count + " internal relationship(s)");
  return parts.join(" · ");
}

function relationshipListMeta(context, relationship) {
  const nodesByID = {};
  context.state.scene.visible_nodes.forEach(function (node) { nodesByID[node.id] = node; });
  const from = nodesByID[relationship.from_visible_id];
  const to = nodesByID[relationship.to_visible_id];
  return (from ? from.label : relationship.from_visible_id) + " → " + (to ? to.label : relationship.to_visible_id) + " · " + relationship.count + " contributor(s) · " + relationship.confidence_state + " confidence";
}

function allListItems(context) {
  const scene = context.state.scene;
  return scene.visible_nodes.map(function (node) { return { kind: "node", id: node.id, title: node.label, meta: nodeListMeta(Object.assign({}, node, { language: nodeLanguageText(context, node) })), language: nodeLanguageText(context, node), languageValue: nodeLanguageBadge(context, node), value: node }; })
    .concat(scene.visible_relationships.map(function (relationship) { return { kind: "relationship", id: relationship.id, title: relationship.type, meta: relationshipListMeta(context, relationship), value: relationship }; }))
    .concat((scene.reference_details || []).map(function (reference) { return { kind: "reference-detail", id: reference.id, title: reference.name, meta: referenceScopeLabel(reference.scope) + " · " + reference.count + " import(s) · " + reference.confidence_state + " confidence", value: reference }; }))
    .concat(scene.cycle_indicators.map(function (cycle) { return { kind: "cycle", id: cycle.id, title: cycle.label, meta: cycle.module_ids.length + " module(s) · " + cycle.relationship_ids.length + " relationship(s)", value: cycle }; }))
    .concat(scene.diagnostic_indicators.map(function (diagnostic) { return { kind: "diagnostic", id: diagnostic.id, title: diagnostic.code, meta: diagnostic.severity + " · " + diagnostic.message, value: diagnostic }; }));
}

export function renderAccessibleList(context, services) {
  const state = context.state;
  const items = allListItems(context).filter(function (item) {
    if (!state.query) return true;
    return [item.title, item.meta, item.id].join(" ").toLowerCase().includes(state.query);
  });
  context.elements.listCount.textContent = items.length + " item(s)";
  context.elements.accessibleList.innerHTML = items.length ? items.map(function (item) {
    const selected = state.selected && state.selected.kind === item.kind && state.selected.id === item.id;
    const title = '<span class="list-item-title">' + escapeHTML(item.title) + "</span>" + (item.kind === "node" ? languageBadgeMarkup(item.languageValue, item.language) : "");
    return '<button class="list-item ' + (selected ? "selected" : "") + '" type="button" data-list-kind="' + escapeHTML(item.kind) + '" data-list-id="' + escapeHTML(item.id) + '" aria-label="' + escapeHTML(item.title + ". " + item.meta) + '"><span class="list-item-title-row">' + title + '</span><span class="list-item-meta">' + escapeHTML(item.meta) + "</span></button>";
  }).join("") : '<p class="list-empty">No items match the current search.</p>';
  context.elements.accessibleList.querySelectorAll("[data-list-id]").forEach(function (element) {
    element.addEventListener("click", function () { services.selectEntity(element.dataset.listKind, element.dataset.listId); });
  });
}

function detailRow(key, value, extraClass) {
  return '<div class="detail-row"><span class="detail-key">' + escapeHTML(key) + '</span><span class="detail-value ' + (extraClass || "") + '">' + escapeHTML(value) + "</span></div>";
}

function detailSection(title, content) {
  return '<section class="detail-section"><h4>' + escapeHTML(title) + "</h4>" + content + "</section>";
}

function modelModule(context, moduleID) {
  if (!context.state.model) return null;
  const modules = context.state.model.modules || [];
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

function moduleNames(context, moduleIDs) {
  return (moduleIDs || []).map(function (moduleID) {
    const module = modelModule(context, moduleID);
    return module ? module.display_name : moduleID;
  });
}

function locationText(link) {
  if (!link) return "";
  const path = link.path || "Location unavailable";
  if (!link.start || !link.start.line || link.start.line < 1) return path + " · File provenance only";
  const endLine = link.end && link.end.line && link.end.line >= link.start.line ? link.end.line : link.start.line;
  return path + " · " + (endLine === link.start.line ? "Line " + link.start.line : "Lines " + link.start.line + "–" + endLine);
}

function evidenceLinks(context, ids) {
  const linksByID = {};
  (context.state.scene.evidence_links || []).forEach(function (link) { linksByID[link.id] = link; });
  const links = (ids || []).map(function (id) { return linksByID[id]; }).filter(Boolean);
  if (!links.length) return '<p class="muted">No source evidence is attached to this item.</p>';
  if (!context.sourceEnabled && !context.embeddedExport) return '<p class="muted">Source inspection is unavailable for this model-only session.</p>';
  if (!context.sourceEnabled) {
    return '<div class="evidence-list">' + links.map(function (link) {
      return '<div class="evidence-item" aria-label="' + escapeHTML((link.symbol || link.kind || "Source evidence") + ". " + locationText(link)) + '"><span>' + escapeHTML(link.symbol || link.kind || "Source evidence") + '</span><small>' + escapeHTML(locationText(link)) + " · source not embedded</small></div>";
    }).join("") + "</div>";
  }
  return '<div class="evidence-list">' + links.map(function (link) {
    return '<button type="button" class="evidence-item" data-evidence-id="' + escapeHTML(link.id) + '"><span>' + escapeHTML(link.symbol || link.kind || "Source evidence") + '</span><small>' + escapeHTML(locationText(link)) + "</small></button>";
  }).join("") + "</div>";
}

function relationshipTargetLabel(context, relationship) {
  const node = context.state.scene.visible_nodes.find(function (item) { return item.id === relationship.to_visible_id; });
  return node ? node.label : relationship.to_visible_id;
}

function importsForNode(context, node) {
  const imports = [];
  context.state.scene.visible_relationships.filter(function (relationship) {
    return relationship.from_visible_id === node.id && !relationship.target_scope;
  }).forEach(function (relationship) {
    imports.push({ kind: "relationship", id: relationship.id, title: relationshipTargetLabel(context, relationship), scope: relationship.target_scope || "local", count: relationship.count, confidence: relationship.confidence_state, evidenceIDs: relationship.evidence_ids || [], meta: relationship.type + " · " + relationshipListMeta(context, relationship), value: relationship });
  });
  (context.state.scene.reference_details || []).filter(function (reference) {
    return (reference.from_visible_ids || []).includes(node.id);
  }).forEach(function (reference) {
    imports.push({ kind: "reference-detail", id: reference.id, title: reference.name, scope: reference.scope, count: reference.count, confidence: reference.confidence_state, evidenceIDs: reference.evidence_ids || [], meta: "import · " + referenceScopeLabel(reference.scope) + " · " + reference.count + " occurrence(s)", value: reference });
  });
  if (context.state.model && node.internal_relationship_ids) {
    node.internal_relationship_ids.forEach(function (relationshipID) {
      const relationships = context.state.model.relationships || [];
      const relationship = relationships.find(function (item) { return item.id === relationshipID; }) || (context.aggregateEnabled && context.state.activeScope && context.state.activeScope !== "all"
        ? relationships.find(function (item) { return item.id && item.id.startsWith(context.state.activeScope + "::") && item.id.slice(context.state.activeScope.length + 2) === relationshipID; })
        : null);
      if (!relationship) return;
      imports.push({ kind: "internal-relationship", id: relationship.id, title: moduleNames(context, [relationship.to_module_id])[0] || relationship.to_module_id, scope: "internal", count: 1, confidence: relationship.confidence ? confidenceState(relationship.confidence.score) : "unknown", evidenceIDs: relationship.source_reference_ids || [], meta: relationship.type + " · collapsed internal relationship", value: relationship });
    });
  }
  return imports.filter(function (item) { return context.state.importScope === "all" || item.scope === context.state.importScope; });
}

function importSection(context, node) {
  const imports = importsForNode(context, node);
  const scopes = ["all", "local", "standard_library", "external", "unresolved", "dynamic", "internal"];
  const options = scopes.map(function (scope) {
    return '<option value="' + scope + '"' + (scope === context.state.importScope ? " selected" : "") + ">" + escapeHTML(scope === "all" ? "All scopes" : referenceScopeLabel(scope)) + "</option>";
  }).join("");
  const list = imports.length ? '<div class="import-list">' + imports.map(function (item) {
    return '<button type="button" class="import-item" data-import-kind="' + escapeHTML(item.kind) + '" data-import-id="' + escapeHTML(item.id) + '"><span class="import-item-title">' + escapeHTML(item.title) + '</span><span class="import-item-meta">' + escapeHTML(referenceScopeLabel(item.scope)) + " · " + escapeHTML(item.confidence) + " confidence · " + escapeHTML(item.count) + " occurrence(s)</span><small>" + escapeHTML(item.meta) + "</small></button>";
  }).join("") + "</div>" : '<p class="muted">No imports match this scope in the current projection.</p>';
  return detailSection("Imports & evidence", '<div class="detail-filter"><label for="detail-import-scope">Scope</label><select id="detail-import-scope">' + options + "</select></div>" + list);
}

function moduleSection(context, node) {
  if (!node.module_ids || !node.module_ids.length) return "";
  const modules = node.module_ids.map(function (moduleID) {
    const module = modelModule(context, moduleID);
    if (!module) return '<li><code>' + escapeHTML(moduleID) + "</code></li>";
    return '<li><strong>' + escapeHTML(module.display_name) + '</strong><span>' + escapeHTML(module.kind + " · " + module.language + " · " + (module.tags && module.tags.length ? module.tags.join(", ") : "no tags")) + "</span></li>";
  }).join("");
  return detailSection("Canonical modules", '<ul class="module-list">' + modules + "</ul>");
}

function activeSourceSnapshot(context) {
  const index = context.state.sourceIndex;
  if (!index) return null;
  if (context.aggregateEnabled && (context.state.activeScope || "all") === "all" && index.projection) return index.projection;
  const snapshots = Array.isArray(index.snapshots) ? index.snapshots : [];
  const activeScope = context.aggregateEnabled ? context.state.activeScope : "";
  return snapshots.find(function (snapshot) {
    return !activeScope || activeScope === "all" || snapshot.scope_context && snapshot.scope_context.scope_id === activeScope;
  }) || snapshots[0] || null;
}

export function sourceFactStatus(snapshot, files) {
	if (!snapshot) return "unavailable";
	const coverage = snapshot.coverage || [];
	const hasUnsupported = coverage.some(function (item) { return item.status === "unsupported"; });
	const hasUnknown = coverage.some(function (item) { return item.status === "unknown"; }) || files.some(function (file) { return file.analysis_status === "unknown" || file.analysis_status === "unparsed"; });
	const hasPartial = coverage.some(function (item) { return item.status === "partial"; }) || files.some(function (file) { return file.analysis_status === "partial"; });
	const hasObserved = coverage.some(function (item) { return item.status === "observed"; });
	if (!files.length && hasUnsupported && !hasObserved) return "unsupported";
	if (hasPartial || (hasUnsupported && files.length)) return "partial";
	if (hasUnknown) return "unknown";
	if (!files.length) return "missing";
	return "populated";
}

export function sourceFactModuleData(snapshot, node) {
  if (!snapshot || !node || !node.module_ids || !node.module_ids.length) return { files: [], symbols: [], documentation: [], relations: [] };
  const moduleIDs = new Set(node.module_ids);
  const fileIDs = new Set();
  const relations = (snapshot.relations || []).filter(function (relation) {
    if (relation.category !== "contains" && relation.category !== "declares") return false;
    if (!relation.to_ref) return false;
    if (relation.from_ref && relation.from_ref.kind === "module" && moduleIDs.has(relation.from_ref.id) && relation.to_ref.kind === "file") {
      fileIDs.add(relation.to_ref.id);
      return true;
    }
    return false;
  });
  const files = (snapshot.files || []).filter(function (file) { return fileIDs.has(file.id); });
  const symbolIDs = new Set();
  (snapshot.relations || []).forEach(function (relation) {
    if (!relation.to_ref || !relation.from_ref || relation.from_ref.kind !== "file" || relation.to_ref.kind !== "symbol") return;
    if (fileIDs.has(relation.from_ref.id)) symbolIDs.add(relation.to_ref.id);
  });
  const symbols = (snapshot.symbols || []).filter(function (symbol) { return symbolIDs.has(symbol.id); });
  const documentation = (snapshot.documentation || []).filter(function (record) {
    return (record.subject_ref && record.subject_ref.kind === "file" && fileIDs.has(record.subject_ref.id)) || (record.subject_ref && record.subject_ref.kind === "symbol" && symbolIDs.has(record.subject_ref.id));
  });
  (snapshot.relations || []).forEach(function (relation) {
    if (!relation.from_ref) return;
    if (relation.from_ref.kind === "file" && fileIDs.has(relation.from_ref.id) && !relations.includes(relation)) relations.push(relation);
  });
  files.sort(function (left, right) { return String(left.path).localeCompare(String(right.path)); });
  symbols.sort(function (left, right) { return String(left.id).localeCompare(String(right.id)); });
  documentation.sort(function (left, right) { return String(left.id).localeCompare(String(right.id)); });
  relations.sort(function (left, right) { return String(left.id).localeCompare(String(right.id)); });
  return { files: files, symbols: symbols, documentation: documentation, relations: relations };
}

function sourceFactHash(file) {
  const value = file && file.size && file.size.content_hash ? file.size.content_hash.value : "";
  return value ? value.slice(0, 12) + "…" : "hash unavailable";
}

function sourceFactProvenance(value) {
  const provenance = value || {};
  const provider = String(provenance.provider || "").trim();
  const version = String(provenance.provider_version || "").trim();
  const identity = provider ? provider + (version ? "@" + version : "") : "provider unavailable";
  return [provenance.status || "status unavailable", provenance.basis || "basis unavailable", identity].join(" · ");
}

function sourceFactSymbol(symbol, context, documentationBySubject) {
  const visibility = symbol.visibility && symbol.visibility.classification ? symbol.visibility.classification : "unknown";
  const title = (symbol.qualified_name || symbol.name || "Unnamed symbol") + " · " + visibility;
  const documentation = (documentationBySubject && documentationBySubject[symbol.id]) || [];
  const documentationStatus = documentation.length ? documentation.map(function (item) { return item.status || "unknown"; }).join(", ") : "not reported";
  const meta = [symbol.category || "unknown", symbol.language_kind || "kind unavailable", (symbol.locations || []).length + " location(s)", "docs " + documentationStatus, sourceFactProvenance(symbol.provenance)].join(" · ");
  const action = context.sourceEnabled && !context.embeddedExport
    ? '<button type="button" class="source-fact-action" data-source-fact-id="' + escapeHTML(symbol.id) + '">Inspect bounded source</button>'
    : "";
  return '<li class="source-fact-symbol"><div><strong>' + escapeHTML(title) + '</strong><span>' + escapeHTML(meta) + '</span></div>' + action + '</li>';
}

function sourceFactsSection(context, node) {
  const snapshot = activeSourceSnapshot(context);
  if (!snapshot) {
    const message = context.state.sourceIndexError || "This model revision does not include a source-facts attachment.";
    return detailSection("Source facts", '<div class="source-facts-state source-facts-unavailable"><span class="state-pill warning">unavailable</span><p class="muted">' + escapeHTML(message) + "</p></div>");
  }
  const data = sourceFactModuleData(snapshot, node);
  const state = sourceFactStatus(snapshot, data.files);
  const coverage = (snapshot.coverage || []).map(function (item) {
    return '<span class="source-fact-coverage"><strong>' + escapeHTML(item.capability) + '</strong> <span class="state-pill ' + (item.status === "observed" ? "ok" : item.status === "unsupported" ? "warning" : "") + '">' + escapeHTML(item.status) + '</span></span>';
  }).join("");
  if (!data.files.length) {
    const stateClass = state === "populated" ? "ok" : "warning";
    const copy = state === "unsupported" ? "The active analyzer does not support source-fact extraction for this scope." : state === "unknown" ? "Source-fact coverage is unknown because the source could not be parsed or inspected." : state === "unavailable" ? "Source-fact coverage is unavailable for this model revision." : "No explicit module-to-file containment facts are available for this node in the active scope.";
    return detailSection("Source facts", '<div class="source-facts-state source-facts-' + escapeHTML(state) + '"><span class="state-pill ' + stateClass + '">' + escapeHTML(state) + '</span><p class="muted">' + escapeHTML(copy) + "</p></div>" + (coverage ? '<div class="source-fact-coverage-list">' + coverage + "</div>" : ""));
  }
  const symbolsByFile = {};
  (snapshot.relations || []).forEach(function (relation) {
    if (!relation.from_ref || !relation.to_ref || relation.from_ref.kind !== "file" || relation.to_ref.kind !== "symbol") return;
    if (!symbolsByFile[relation.from_ref.id]) symbolsByFile[relation.from_ref.id] = [];
    symbolsByFile[relation.from_ref.id].push(relation.to_ref.id);
  });
  const symbolsByID = {};
  data.symbols.forEach(function (symbol) { symbolsByID[symbol.id] = symbol; });
  const documentationBySubject = {};
  data.documentation.forEach(function (documentation) {
    if (!documentation.subject_ref || !documentation.subject_ref.id) return;
    if (!documentationBySubject[documentation.subject_ref.id]) documentationBySubject[documentation.subject_ref.id] = [];
    documentationBySubject[documentation.subject_ref.id].push(documentation);
  });
  const files = data.files.slice(0, 24).map(function (file) {
    const fileDocumentation = data.documentation.filter(function (documentation) { return documentation.subject_ref && documentation.subject_ref.kind === "file" && documentation.subject_ref.id === file.id; });
    const fileDocumentationStatus = fileDocumentation.length ? fileDocumentation.map(function (item) { return item.status || "unknown"; }).join(", ") : "not reported";
    const symbolItems = (symbolsByFile[file.id] || []).map(function (id) { return symbolsByID[id]; }).filter(Boolean).slice(0, 8).map(function (symbol) { return sourceFactSymbol(symbol, context, documentationBySubject); }).join("");
    const moreSymbols = Math.max(0, (symbolsByFile[file.id] || []).length - 8);
    return '<li class="source-fact-file"><div class="source-fact-file-heading"><strong>' + escapeHTML(file.path) + '</strong><span>' + escapeHTML((file.size.byte_count || 0) + " bytes · " + (file.size.line_count || 0) + " lines · " + sourceFactHash(file)) + '</span></div><div class="source-fact-file-meta">' + escapeHTML((file.analysis_status || "unknown") + " · " + (file.roles || []).join(", ") + " · docs " + fileDocumentationStatus) + '</div><div class="source-fact-provenance">' + escapeHTML(sourceFactProvenance(file.provenance)) + '</div>' + (symbolItems ? '<ul class="source-fact-symbol-list">' + symbolItems + (moreSymbols ? '<li class="muted">' + moreSymbols + " more symbol(s)</li>" : "") + '</ul>' : '<p class="muted">No named declarations reported.</p>') + '</li>';
  }).join("");
  const moreFiles = Math.max(0, data.files.length - 24);
  return detailSection("Source facts", '<div class="source-facts-state source-facts-' + escapeHTML(state) + '"><span class="state-pill ' + (state === "populated" ? "ok" : "warning") + '">' + escapeHTML(state) + '</span><span>' + escapeHTML(data.files.length + " file(s) · " + data.symbols.length + " symbol(s) · " + data.documentation.length + " documentation record(s)") + '</span></div>' + (coverage ? '<div class="source-fact-coverage-list">' + coverage + "</div>" : "") + '<ul class="source-fact-file-list">' + files + (moreFiles ? '<li class="muted">' + moreFiles + " more file(s)</li>" : "") + '</ul>');
}

function bindDetailActions(context, services) {
  context.elements.detailsContent.querySelectorAll("[data-open-inspection]").forEach(function (element) {
    element.addEventListener("click", function () { services.openInspection("node", element.dataset.openInspection); });
  });
  const drill = context.elements.detailsContent.querySelector("[data-drill-path]");
  if (drill) drill.addEventListener("click", function () { services.navigationTo(drill.dataset.drillPath ? drill.dataset.drillPath.split("/") : []); });
  const filter = context.elements.detailsContent.querySelector("#detail-import-scope");
  if (filter) filter.addEventListener("change", function () { context.state.importScope = filter.value; renderDetails(context, services); });
  context.elements.detailsContent.querySelectorAll("[data-evidence-id]").forEach(function (element) {
    element.addEventListener("click", function () { services.openSource(element.dataset.evidenceId); });
  });
  context.elements.detailsContent.querySelectorAll("[data-source-fact-id]").forEach(function (element) {
    element.addEventListener("click", function () { services.openSourceFact(element.dataset.sourceFactId); });
  });
  context.elements.detailsContent.querySelectorAll("[data-import-kind]").forEach(function (element) {
    element.addEventListener("click", function () { services.selectEntity(element.dataset.importKind, element.dataset.importId); });
  });
}

function sourcePanel(context) {
  const source = context.state.source;
  if (!source) return "";
  if (source.loading) return detailSection("Read-only source", '<p class="muted">Loading source excerpt…</p>');
  if (source.error) return detailSection("Read-only source", '<div class="notice error">' + escapeHTML(source.error) + "</div>");
  const excerpt = source.data;
  const lines = (excerpt.lines || []).map(function (line) {
    return '<span class="source-line"><span class="source-number" aria-hidden="true">' + escapeHTML(line.number) + '</span><span class="source-text">' + escapeHTML(line.text) + "</span></span>";
  }).join("");
  const startLine = excerpt.start && excerpt.start.line;
  const endLine = excerpt.end && excerpt.end.line;
  const range = startLine && endLine
    ? (startLine === endLine ? "Line " + startLine : "Lines " + startLine + "–" + endLine)
    : "File provenance only";
  return detailSection("Read-only source", '<div class="source-meta"><strong>' + escapeHTML(excerpt.path) + '</strong><span>' + escapeHTML(range) + ' · read-only</span></div><pre class="source-excerpt"><code>' + lines + "</code></pre>");
}

function activeScopeDetails(context) {
  const scene = context.state.scene;
  const active = context.state.activeScope || "all";
  if (!context.aggregateEnabled) {
    const languageValue = String(scene.project.language || "unknown").toLowerCase();
    return {
      name: displayProjectRoot(scene.project.root_label),
      language: formatLanguage(languageValue),
      languageValue: languageValue,
      status: scene.status,
      meta: "Local model scope"
    };
  }
  if (active === "all") {
    const scopes = context.state.scopes || [];
    const usable = scopes.filter(function (scope) { return scope.status === "complete" || scope.status === "partial"; }).length;
    return {
      name: "All scopes",
      language: "Multi",
      languageValue: "multi",
      status: scene.aggregate_status || scene.status,
      meta: usable + "/" + scopes.length + " scopes usable"
    };
  }
  const scope = (context.state.scopes || []).find(function (item) { return item.scope_id === active; });
  const analyzer = scope && scope.analyzer ? scope.analyzer : {};
  const languageValue = String(analyzer.language || scene.project.language || "unknown").toLowerCase();
  const summary = scope && scope.summary ? scope.summary : {};
  const moduleCount = summary.module_count == null ? "module count unavailable" : summary.module_count + " module(s)";
  return {
    name: displayProjectRoot(scope && scope.project_root ? scope.project_root : scene.project.root_label),
    language: formatLanguage(languageValue),
    languageValue: languageValue,
    status: scope && scope.status ? scope.status : scene.scope_status || scene.status,
    meta: selectionSourceLabel(scope) + " · " + moduleCount
  };
}

function analyzerIdentityLabel(scope) {
  const analyzer = scope && scope.analyzer ? scope.analyzer : {};
  const id = String(analyzer.id || "").trim();
  if (!id) return "Analyzer identity unavailable";
  const version = String(analyzer.version || "").trim();
  return version ? id + "@" + version : id;
}

function selectionSourceLabel(scope) {
  const source = String(scope && scope.selection_source || "").toLowerCase();
  if (source === "assignment") {
    return scope.assignment_path ? "Configured · " + displayProjectRoot(scope.assignment_path) : "Configured";
  }
  if (source === "cli") return "CLI selection";
  if (source === "automatic") return "Automatic";
  return source || "Selection unavailable";
}

function scopeCacheLabel(scope) {
  if (scope && scope.cache_hit) return "Cache hit";
  const reason = String(scope && scope.invalidation_reason || "").trim();
  return reason ? "Fresh · " + reason.replaceAll("_", " ") : "Fresh analysis";
}

function sourceIdentityLabel(scope) {
  const source = scope && scope.source_scope ? scope.source_scope : {};
  const value = String(source.matched_source_set_fingerprint || "").trim();
  if (!value) return "Source identity unavailable";
  return "Source " + (value.length > 18 ? value.slice(0, 18) + "…" : value);
}

function languageBadgeMarkup(languageValue, language) {
  return '<span class="language-badge ' + classForState(languageValue) + '">' + escapeHTML(String(language || "Unknown").toUpperCase()) + "</span>";
}

function scopeStatusClass(status) {
  const value = String(status || "unknown").toLowerCase();
  if (value === "complete") return "ok";
  if (value === "failed" || value === "cancelled") return "error";
  if (value === "partial") return "warning";
  return "";
}

export function compactSummaryFields(node) {
  const counts = node && node.counts ? node.counts : {};
  return {
    label: node && node.label || "Unnamed item",
    kind: node && node.kind || "node",
    hierarchy: node && node.hierarchy_path ? node.hierarchy_path.slice() : [],
    counts: {
      modules: counts.module_count || 0,
      relationships: counts.relationship_count || 0,
      evidence: counts.evidence_count || 0
    },
    statuses: [node && node.cycle_state, node && node.diagnostic_state, node && node.confidence_state, node && node.identity_state].filter(function (value) {
      return value && value !== "none" && value !== "not_applicable" && value !== "high" && value !== "stable";
    })
  };
}

function renderDetailsContext(context) {
  if (!context.elements.detailsContext || !context.state.scene) return;
  const value = activeScopeDetails(context);
  const status = String(value.status || "unknown").toLowerCase();
  context.elements.detailsContext.innerHTML = '<div class="details-context-label">Current scope</div><div class="details-context-main"><strong>' + escapeHTML(value.name) + '</strong>' + languageBadgeMarkup(value.languageValue, value.language) + '<span class="scope-status-badge ' + scopeStatusClass(status) + '">' + escapeHTML(status) + '</span></div><div class="details-context-meta">' + escapeHTML(value.meta) + "</div>";
}

export function renderDetails(context, services) {
  const state = context.state;
  const scene = state.scene;
  if (!scene) return;
  renderDetailsContext(context);
  if (!state.selected) {
    context.elements.detailsTitle.textContent = "Select an item";
    context.elements.detailsKind.textContent = context.aggregateEnabled ? "All scopes" : "Overview";
    context.elements.detailsContent.innerHTML = '<p class="muted">Select a node or directed relationship to see its compact summary. Use Open inspection for human-oriented source details.</p>' + '<div class="detail-table">' + detailRow("Hierarchy", scene.hierarchy_path.length ? scene.hierarchy_path.join(" / ") : "Top level") + detailRow("Model status", scene.status, scene.status === "complete" ? "high" : "warning") + detailRow("Language", formatLanguage(scene.project.language)) + "</div>";
    bindDetailActions(context, services);
    return;
  }
  if (state.selected.kind === "node") {
    const node = scene.visible_nodes.find(function (item) { return item.id === state.selected.id; });
    if (!node) { state.selected = null; renderDetails(context, services); return; }
    context.elements.detailsTitle.textContent = node.label;
    context.elements.detailsKind.textContent = node.kind + " · " + formatLanguage(nodeLanguageBadge(context, node));
    const drill = node.kind === "group" ? '<button class="button secondary detail-action" type="button" data-drill-path="' + escapeHTML(node.hierarchy_path.join("/")) + '">Open group</button>' : "";
    const summary = compactSummaryFields(node);
    const statusBadges = [];
    if (node.cycle_state && node.cycle_state !== "none") statusBadges.push('<span class="state-pill error">' + escapeHTML(node.cycle_state) + " cycle</span>");
    if (node.diagnostic_state && node.diagnostic_state !== "none") statusBadges.push('<span class="state-pill warning">' + escapeHTML(node.diagnostic_state) + " diagnostics</span>");
    if (node.confidence_state && node.confidence_state !== "not_applicable" && node.confidence_state !== "high") statusBadges.push('<span class="state-pill warning">' + escapeHTML(node.confidence_state) + " confidence</span>");
    if (node.identity_state && node.identity_state !== "stable") statusBadges.push('<span class="state-pill warning">' + escapeHTML(node.identity_state) + " identity</span>");
    const counts = node.counts || {};
    context.elements.detailsContent.innerHTML = '<p class="details-summary-copy">A compact view of the selected item. Open inspection for source structure and evidence.</p><div class="detail-actions"><button class="button detail-action" type="button" data-open-inspection="' + escapeHTML(node.id) + '">Open inspection</button>' + drill + '</div><div class="detail-table">' +
      detailRow("Hierarchy", formatList(summary.hierarchy, "Top level")) + detailRow("Language", nodeLanguageText(context, node), "emphasis") + detailRow("Active scope", activeScopeDetails(context).name) + detailRow("Modules", summary.counts.modules) + detailRow("Visible relationships", summary.counts.relationships) + detailRow("Evidence links", summary.counts.evidence) +
      "</div>" + (statusBadges.length ? '<div class="detail-status-row" aria-label="Non-neutral status">' + statusBadges.join("") + "</div>" : "") + renderQualitySummary(context);
    bindDetailActions(context, services);
    return;
  }
  if (state.selected.kind === "reference-detail") {
    const reference = (scene.reference_details || []).find(function (item) { return item.id === state.selected.id; });
    if (!reference) { state.selected = null; renderDetails(context, services); return; }
    context.elements.detailsTitle.textContent = reference.name;
    context.elements.detailsKind.textContent = "Import detail";
    context.elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Stable ID", reference.id, "emphasis") + detailRow("Reference scope", referenceScopeLabel(reference.scope)) + detailRow("Imported by", formatList(reference.from_visible_ids)) + detailRow("Import count", reference.count) + detailRow("Confidence", reference.confidence_state, reference.confidence_state) + detailRow("Basis", reference.confidence_basis || "Not supplied") + detailRow("Canonical IDs", formatList(reference.relationship_ids)) + "</div>" + detailSection("Source evidence", evidenceLinks(context, reference.evidence_ids)) + sourcePanel(context);
    bindDetailActions(context, services);
    return;
  }
  if (state.selected.kind === "cycle") {
    const cycle = scene.cycle_indicators.find(function (item) { return item.id === state.selected.id; });
    if (!cycle) { state.selected = null; renderDetails(context, services); return; }
    const ids = [];
    scene.visible_relationships.forEach(function (relationship) {
      if (relationship.contributor_relationship_ids.some(function (id) { return cycle.relationship_ids.includes(id); })) ids.push.apply(ids, relationship.evidence_ids || []);
    });
    context.elements.detailsTitle.textContent = "Cycle";
    context.elements.detailsKind.textContent = "Graph health";
    context.elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Stable ID", cycle.id, "emphasis") + detailRow("Modules", moduleNames(context, cycle.module_ids).join(", ")) + detailRow("Relationship IDs", formatList(cycle.relationship_ids)) + detailRow("State", cycle.state, "cycle") + "</div>" + detailSection("Source evidence", evidenceLinks(context, ids)) + sourcePanel(context);
    bindDetailActions(context, services);
    return;
  }
  if (state.selected.kind === "diagnostic") {
    const diagnostic = scene.diagnostic_indicators.find(function (item) { return item.id === state.selected.id; });
    if (!diagnostic) { state.selected = null; renderDetails(context, services); return; }
    context.elements.detailsTitle.textContent = diagnostic.code;
    context.elements.detailsKind.textContent = "Diagnostic";
    context.elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Severity", diagnostic.severity, diagnostic.severity) + detailRow("Message", diagnostic.message, "emphasis") + detailRow("Subject", diagnostic.subject || "Not supplied") + detailRow("Path", diagnostic.path || "Not supplied") + detailRow("Recoverable", diagnostic.recoverable ? "Yes" : "No") + "</div>" + detailSection("Source evidence", evidenceLinks(context, diagnostic.evidence_ids)) + sourcePanel(context);
    bindDetailActions(context, services);
    return;
  }
  if (state.selected.kind === "internal-relationship") {
    const relationship = context.state.model && (context.state.model.relationships || []).find(function (item) { return item.id === state.selected.id; });
    if (!relationship) { state.selected = null; renderDetails(context, services); return; }
    context.elements.detailsTitle.textContent = relationship.type;
    context.elements.detailsKind.textContent = "Internal relation";
    context.elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Canonical ID", relationship.id, "emphasis") + detailRow("Direction", moduleNames(context, [relationship.from_module_id])[0] + " → " + moduleNames(context, [relationship.to_module_id])[0], "emphasis") + detailRow("Confidence", relationship.confidence ? confidenceState(relationship.confidence.score) : "unknown") + detailRow("Evidence", relationship.source_reference_ids.length) + "</div>" + detailSection("Source evidence", evidenceLinks(context, relationship.source_reference_ids)) + sourcePanel(context);
    bindDetailActions(context, services);
    return;
  }
  const relationship = scene.visible_relationships.find(function (item) { return item.id === state.selected.id; });
  if (!relationship) { state.selected = null; renderDetails(context, services); return; }
  const nodesByID = {};
  scene.visible_nodes.forEach(function (node) { nodesByID[node.id] = node; });
  context.elements.detailsTitle.textContent = relationship.type;
  context.elements.detailsKind.textContent = "Directed relation";
  context.elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Stable ID", relationship.id, "emphasis") + detailRow("Direction", (nodesByID[relationship.from_visible_id] || {}).label + " → " + (nodesByID[relationship.to_visible_id] || {}).label, "emphasis") + detailRow("Type", relationship.type) + detailRow("Contributors", relationship.count) + detailRow("Canonical IDs", formatList(relationship.contributor_relationship_ids)) + detailRow("Cycle state", relationship.cycle_state, relationship.cycle_state !== "none" ? "cycle" : "") + detailRow("Confidence", relationship.confidence_state, relationship.confidence_state) + detailRow("Basis", relationship.confidence_basis || "Not supplied") + (relationship.target_scope ? detailRow("Target scope", referenceScopeLabel(relationship.target_scope)) : "") + detailRow("Evidence", relationship.evidence_ids.length) + "</div>" + detailSection("Source evidence", evidenceLinks(context, relationship.evidence_ids)) + sourcePanel(context);
  bindDetailActions(context, services);
}

export async function openSource(context, evidenceID, api, services) {
  const link = (context.state.scene.evidence_links || []).find(function (item) { return item.id === evidenceID; });
  if (!link) return;
  const request = ++context.state.sourceRequest;
  context.state.source = { loading: true, link: link };
  services.renderDetails();
  if (services.renderInspection) services.renderInspection();
  const query = new URLSearchParams({ model_id: api.currentModelID(), evidence_id: evidenceID, path: link.path });
  if (context.aggregateEnabled && context.state.activeScope) query.set("scope", context.state.activeScope);
  if (link.start) query.set("start_line", String(link.start.line));
  if (link.end) query.set("end_line", String(link.end.line));
  try {
    const data = await api.getJSON("/v1/source?" + query.toString());
    if (request !== context.state.sourceRequest || !context.state.scene || data.model_revision !== context.state.scene.model_revision) return;
    context.state.source = { data: data, link: link };
    services.renderDetails();
    if (services.renderInspection) services.renderInspection();
  } catch (error) {
    if (request !== context.state.sourceRequest) return;
    context.state.source = { error: error.message || "The source excerpt could not be loaded.", link: link };
    services.renderDetails();
    if (services.renderInspection) services.renderInspection();
  }
}

export async function openSourceFact(context, entityID, api, services) {
  if (!context.sourceEnabled) return;
  const request = ++context.state.sourceRequest;
  context.state.source = { loading: true, link: { id: entityID, kind: "source fact" } };
  services.renderDetails();
  if (services.renderInspection) services.renderInspection();
  const query = new URLSearchParams({ include_source: "true" });
  if (context.aggregateEnabled && context.state.activeScope) query.set("scope", context.state.activeScope);
  try {
    const endpoint = "/v1/models/" + encodeURIComponent(api.currentModelID()) + "/source-index/evidence/" + encodeURIComponent(entityID) + "?" + query.toString();
    const response = await api.getJSON(endpoint);
    if (request !== context.state.sourceRequest || !context.state.scene) return;
    if (!response.source_context) throw new Error("No bounded source context was returned for this source fact.");
    context.state.source = { data: response.source_context, link: { id: entityID, kind: "source fact" } };
    services.renderDetails();
    if (services.renderInspection) services.renderInspection();
  } catch (error) {
    if (request !== context.state.sourceRequest) return;
    context.state.source = { error: error.message || "The source-fact excerpt could not be loaded.", link: { id: entityID, kind: "source fact" } };
    services.renderDetails();
    if (services.renderInspection) services.renderInspection();
  }
}

export function renderSupportLists(context, services) {
  const scene = context.state.scene;
  context.elements.cycleCount.textContent = scene.cycle_indicators.length;
  context.elements.diagnosticCount.textContent = scene.diagnostic_indicators.length;
  context.elements.cycles.innerHTML = scene.cycle_indicators.length ? scene.cycle_indicators.map(function (cycle) {
    return '<button type="button" class="support-item cycle" data-support-kind="cycle" data-support-id="' + escapeHTML(cycle.id) + '"><strong>' + escapeHTML(cycle.label) + '</strong><br><span>' + escapeHTML(cycle.module_ids.length) + " module(s) · " + escapeHTML(cycle.relationship_ids.length) + " relationship(s)</span></button>";
  }).join("") : '<p class="muted">No cycles are present in this model.</p>';
  context.elements.diagnostics.innerHTML = scene.diagnostic_indicators.length ? scene.diagnostic_indicators.map(function (diagnostic) {
    return '<button type="button" class="support-item ' + classForState(diagnostic.severity) + '" data-support-kind="diagnostic" data-support-id="' + escapeHTML(diagnostic.id) + '"><strong>' + escapeHTML(diagnostic.code) + '</strong><br><span>' + escapeHTML(diagnostic.message) + '</span>' + (diagnostic.path ? '<br><code>' + escapeHTML(diagnostic.path) + "</code>" : "") + "</button>";
  }).join("") : '<p class="muted">No diagnostics are attached to this model.</p>';
  document.querySelectorAll("[data-support-id]").forEach(function (element) {
    element.addEventListener("click", function () { services.selectEntity(element.dataset.supportKind, element.dataset.supportId); });
  });
}
