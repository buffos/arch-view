import { classForState, escapeHTML, formatList, referenceScopeLabel, referenceVisibilityLabel } from "./utils.js";

export function showError(context, message) {
  context.elements.errorBanner.textContent = message;
  context.elements.errorBanner.hidden = false;
}

export function hideError(context) {
  context.elements.errorBanner.hidden = true;
  context.elements.errorBanner.textContent = "";
}

export function sceneLayoutKey(scene) {
  const nodes = scene.visible_nodes.map(function (node) { return node.id; }).sort().join(",");
  const relationships = scene.visible_relationships.map(function (relationship) {
    return relationship.id + ":" + relationship.from_visible_id + ":" + relationship.to_visible_id;
  }).sort().join(",");
  return [scene.model_revision || scene.model_id, (scene.hierarchy_path || []).join("/"), scene.reference_visibility, nodes, relationships].join("|");
}

export function renderAll(context, services) {
  const scene = context.state.scene;
  context.elements.projectLabel.textContent = scene.project.root_label + " · " + scene.project.language;
  context.elements.modelStatus.textContent = scene.status;
  context.elements.viewDescription.textContent = scene.hierarchy_path.length === 0
    ? "A local-first semantic overview. Non-local references remain available through the boundary summary and imports list."
    : "Viewing hierarchy: " + scene.hierarchy_path.join(" / ");
  renderSummary(context);
  renderSceneState(context);
  renderBreadcrumbs(context, services.navigationTo);
  renderLayerLegend(context);
  renderReferenceSummary(context);
  services.renderViewportControls();
  services.renderGraph();
  services.renderAccessibleList();
  services.renderSupportLists();
  services.renderDetails();
}

export function renderSummary(context) {
  const summary = context.state.scene.summary;
  const values = [
    [summary.visible_node_count, "visible nodes"],
    [summary.visible_relationship_count, "directed relations"],
    [summary.module_count, "canonical modules"],
    [summary.cycle_count, "cycles"],
    [summary.diagnostic_count, "diagnostics"],
    [summary.evidence_count, "evidence links"]
  ];
  context.elements.summary.innerHTML = values.map(function (item) {
    return '<div class="summary-item"><div class="summary-value">' + escapeHTML(item[0]) + '</div><p class="summary-label">' + escapeHTML(item[1]) + "</p></div>";
  }).join("");
}

export function renderSceneState(context) {
  const state = context.state;
  const scene = state.scene;
  const pills = [];
  pills.push('<span class="state-pill ' + (scene.status === "complete" ? "ok" : "warning") + '">' + escapeHTML(scene.status) + " model</span>");
  if (scene.cycle_indicators.length) pills.push('<span class="state-pill error">' + escapeHTML(scene.cycle_indicators.length) + " cycle(s)</span>");
  if (scene.diagnostic_indicators.length) pills.push('<span class="state-pill warning">' + escapeHTML(scene.diagnostic_indicators.length) + " diagnostic(s)</span>");
  pills.push('<span class="state-pill">' + escapeHTML(referenceVisibilityLabel(scene.reference_visibility)) + " references</span>");
  if (state.layout && state.layout.key === sceneLayoutKey(scene)) {
    pills.push('<span class="state-pill ok">ELK ' + escapeHTML((state.layoutProfile && state.layoutProfile.algorithm) || "layered") + " layout</span>");
  } else if (state.layoutError) {
    pills.push('<span class="state-pill warning">deterministic fallback layout</span>');
  } else if (context.embeddedExport) {
    pills.push('<span class="state-pill ok">ELK ' + escapeHTML((state.layoutProfile && state.layoutProfile.algorithm) || "layered") + " layout loading</span>");
  }
  context.elements.sceneState.innerHTML = pills.join("");
}

export function renderBreadcrumbs(context, navigate) {
  const scene = context.state.scene;
  context.elements.backButton.disabled = context.state.history.length === 0;
  const top = scene.hierarchy_path.length === 0
    ? '<span class="breadcrumb current" aria-current="page">Top level</span>'
    : '<button class="breadcrumb" type="button" data-breadcrumb-path="">Top level</button>';
  const crumbs = [top];
  scene.hierarchy_path.forEach(function (segment, index) {
    const current = index === scene.hierarchy_path.length - 1;
    crumbs.push('<span class="breadcrumb-separator" aria-hidden="true">/</span>');
    crumbs.push(current
      ? '<span class="breadcrumb current" aria-current="page">' + escapeHTML(segment) + "</span>"
      : '<button class="breadcrumb" type="button" data-breadcrumb-path="' + escapeHTML(scene.hierarchy_path.slice(0, index + 1).join("/")) + '">' + escapeHTML(segment) + "</button>");
  });
  context.elements.breadcrumbs.innerHTML = crumbs.join("");
  context.elements.breadcrumbs.querySelectorAll("[data-breadcrumb-path]").forEach(function (element) {
    element.addEventListener("click", function () {
      navigate(element.dataset.breadcrumbPath ? element.dataset.breadcrumbPath.split("/") : []);
    });
  });
}

export function renderLayerLegend(context) {
  context.elements.layerLegend.innerHTML = context.state.scene.layer_labels.map(function (layer) {
    return '<span class="layer-pill">' + escapeHTML(layer.label) + " · " + escapeHTML(layer.module_ids.length) + " module(s)</span>";
  }).join("");
}

export function renderReferenceSummary(context) {
  const summary = context.state.scene.reference_summary;
  if (!summary || !summary.total) {
    context.elements.referenceSummary.innerHTML = '<span class="reference-pill">No non-local references in this projection</span>';
    return;
  }
  const scopePills = (summary.by_scope || []).map(function (scope) {
    const visible = scope.visible_node_count ? " · " + scope.visible_node_count + " shown" : "";
    return '<span class="reference-pill"><strong>' + escapeHTML(scope.reference_count) + "</strong> " + escapeHTML(referenceScopeLabel(scope.scope)) + escapeHTML(visible) + "</span>";
  }).join("");
  const policyText = summary.hidden_count
    ? summary.hidden_count + " hidden in overview"
    : summary.aggregated_count
      ? summary.aggregated_count + " boundary node(s)"
      : summary.expanded_count + " shown individually";
  context.elements.referenceSummary.innerHTML = '<span class="reference-pill"><strong>' + escapeHTML(summary.relationship_count) + "</strong> import relation(s) · " + escapeHTML(policyText) + "</span>" + scopePills;
}

export { classForState, formatList, referenceScopeLabel, referenceVisibilityLabel };
