import { classForState, confidenceState, escapeHTML, formatList, referenceScopeLabel } from "./utils.js";

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
  const parts = [node.kind, "identity " + (node.identity_state || "stable")];
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
  return scene.visible_nodes.map(function (node) { return { kind: "node", id: node.id, title: node.label, meta: nodeListMeta(node), value: node }; })
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
    return '<button class="list-item ' + (selected ? "selected" : "") + '" type="button" data-list-kind="' + escapeHTML(item.kind) + '" data-list-id="' + escapeHTML(item.id) + '" aria-label="' + escapeHTML(item.title + ". " + item.meta) + '"><span class="list-item-title">' + escapeHTML(item.title) + '</span><span class="list-item-meta">' + escapeHTML(item.meta) + "</span></button>";
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
  return (context.state.model.modules || []).find(function (module) { return module.id === moduleID; }) || null;
}

function moduleNames(context, moduleIDs) {
  return (moduleIDs || []).map(function (moduleID) {
    const module = modelModule(context, moduleID);
    return module ? module.display_name : moduleID;
  });
}

function locationText(link) {
  if (!link) return "";
  const start = link.start ? "L" + link.start.line + ":" + link.start.column : "line unavailable";
  const end = link.end ? "–L" + link.end.line + ":" + link.end.column : "";
  return link.path + " · " + start + end;
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
      const relationship = (context.state.model.relationships || []).find(function (item) { return item.id === relationshipID; });
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

function bindDetailActions(context, services) {
  const drill = context.elements.detailsContent.querySelector("[data-drill-path]");
  if (drill) drill.addEventListener("click", function () { services.navigationTo(drill.dataset.drillPath ? drill.dataset.drillPath.split("/") : []); });
  const filter = context.elements.detailsContent.querySelector("#detail-import-scope");
  if (filter) filter.addEventListener("change", function () { context.state.importScope = filter.value; renderDetails(context, services); });
  context.elements.detailsContent.querySelectorAll("[data-evidence-id]").forEach(function (element) {
    element.addEventListener("click", function () { services.openSource(element.dataset.evidenceId); });
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
  const range = "L" + excerpt.start.line + ":" + excerpt.start.column + "–L" + excerpt.end.line + ":" + excerpt.end.column;
  return detailSection("Read-only source", '<div class="source-meta"><strong>' + escapeHTML(excerpt.path) + '</strong><span>' + escapeHTML(range) + ' · read-only</span></div><pre class="source-excerpt"><code>' + lines + "</code></pre>");
}

export function renderDetails(context, services) {
  const state = context.state;
  const scene = state.scene;
  if (!scene) return;
  if (!state.selected) {
    context.elements.detailsTitle.textContent = "Select an item";
    context.elements.detailsKind.textContent = "Overview";
    context.elements.detailsContent.innerHTML = '<p class="muted">The graphic and list use the same renderer-neutral scene. Select a node or directed relationship to inspect stable IDs, aggregation counts, layers, uncertainty, and evidence.</p>' + '<div class="detail-table">' + detailRow("Hierarchy", scene.hierarchy_path.length ? scene.hierarchy_path.join(" / ") : "Top level") + detailRow("Model status", scene.status, scene.status === "complete" ? "high" : "warning") + detailRow("Language", scene.project.language) + detailRow("Boundary", scene.project.boundary) + detailRow("Revision", scene.model_revision) + "</div>" + sourcePanel(context);
    bindDetailActions(context, services);
    return;
  }
  if (state.selected.kind === "node") {
    const node = scene.visible_nodes.find(function (item) { return item.id === state.selected.id; });
    if (!node) { state.selected = null; renderDetails(context, services); return; }
    context.elements.detailsTitle.textContent = node.label;
    context.elements.detailsKind.textContent = node.kind;
    const drill = node.kind === "group" ? '<button class="button secondary detail-action" type="button" data-drill-path="' + escapeHTML(node.hierarchy_path.join("/")) + '">Open group</button>' : "";
    context.elements.detailsContent.innerHTML = drill + '<div class="detail-table">' +
      detailRow("Stable ID", node.id, "emphasis") + detailRow("Hierarchy", formatList(node.hierarchy_path, "Top level")) + detailRow("Modules", node.counts.module_count) + detailRow("Visible relationships", node.counts.relationship_count) +
      (node.counts.internal_relationship_count ? detailRow("Internal relationships", node.counts.internal_relationship_count + " (collapsed in overview)", "emphasis") : "") +
      (node.internal_relationship_ids && node.internal_relationship_ids.length ? detailRow("Internal IDs", formatList(node.internal_relationship_ids)) : "") + detailRow("Layer", node.layer == null ? formatList((node.layers || []).map(function (item) { return "Layer " + item; }), "Unassigned") : "Layer " + node.layer) +
      detailRow("Cycle state", node.cycle_state, node.cycle_state !== "none" ? "cycle" : "") + detailRow("Diagnostics", node.diagnostic_state, node.diagnostic_state !== "none" ? "warning" : "") + detailRow("Identity", node.identity_state || "stable") +
      detailRow("Relationship confidence", node.confidence_state === "not_applicable" ? "Not aggregated for local node" : node.confidence_state, node.confidence_state === "not_applicable" ? "" : node.confidence_state) + (node.reference_scope ? detailRow("Reference scope", referenceScopeLabel(node.reference_scope)) : "") + detailRow("Evidence", node.counts.evidence_count) + detailRow("Tags", formatList(node.tags)) +
      "</div>" + moduleSection(context, node) + importSection(context, node) + detailSection("Source evidence", evidenceLinks(context, node.evidence_ids)) + sourcePanel(context);
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
  const query = new URLSearchParams({ model_id: api.currentModelID(), evidence_id: evidenceID, path: link.path });
  if (link.start) query.set("start_line", String(link.start.line));
  if (link.end) query.set("end_line", String(link.end.line));
  try {
    const data = await api.getJSON("/v1/source?" + query.toString());
    if (request !== context.state.sourceRequest || !context.state.scene || data.model_revision !== context.state.scene.model_revision) return;
    context.state.source = { data: data, link: link };
    services.renderDetails();
  } catch (error) {
    if (request !== context.state.sourceRequest) return;
    context.state.source = { error: error.message || "The source excerpt could not be loaded.", link: link };
    services.renderDetails();
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
