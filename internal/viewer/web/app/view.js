import { classForState, displayProjectRoot, escapeHTML, formatLanguage, formatList, nodeLanguageBadge, nodeLanguageText, referenceScopeLabel, referenceVisibilityLabel } from "./utils.js";

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
  const activeScope = activeScopeDescriptor(context);
  const projectLabel = context.aggregateEnabled && activeScope && activeScope.value !== "all"
    ? scene.project.root_label + " · " + activeScope.root + " · " + activeScope.language
    : scene.project.root_label + " · " + formatLanguage(scene.project.language) + (context.aggregateEnabled ? " · All scopes" : "");
  context.elements.projectLabel.textContent = projectLabel;
  context.elements.modelStatus.textContent = scene.aggregate_status || scene.status;
  context.elements.viewDescription.textContent = scene.hierarchy_path.length === 0
    ? "A local-first semantic overview. Non-local references remain available through the boundary summary and imports list."
    : "Viewing hierarchy: " + scene.hierarchy_path.join(" / ");
  renderSummary(context);
  renderScopeSelector(context);
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

export function renderScopeSelector(context) {
  const control = context.elements.scopeSelectorControl;
  const selector = context.elements.scopeSelector;
  const button = context.elements.scopeSelectorButton;
  const menu = context.elements.scopeSelectorMenu;
  if (!control || !selector || !button || !menu) return;
  if (!context.aggregateEnabled || context.embeddedExport) {
    closeScopePicker(context);
    control.hidden = true;
    return;
  }
  control.hidden = false;
  const scopes = (context.state.scopes || []).slice().sort(function (left, right) {
    return String(left.scope_id || "").localeCompare(String(right.scope_id || ""));
  });
  const descriptors = [aggregateScopeDescriptor(context, scopes)].concat(scopes.map(function (scope) {
    return scopeDescriptor(scope);
  }));
  const activeValue = context.state.activeScope || "all";
  selector.innerHTML = descriptors.map(function (descriptor) {
    return '<option value="' + escapeHTML(descriptor.value) + '">' + escapeHTML(descriptor.optionLabel) + "</option>";
  }).join("");
  selector.value = activeValue;
  if (!descriptors.some(function (descriptor) { return descriptor.value === selector.value; })) {
    context.state.activeScope = "all";
    selector.value = "all";
  }
  const selected = descriptors.find(function (descriptor) { return descriptor.value === selector.value; }) || descriptors[0];
  button.innerHTML = scopeButtonMarkup(selected);
  button.setAttribute("aria-label", "Analysis scope: " + selected.accessibleLabel);
  menu.innerHTML = descriptors.map(function (descriptor) {
    const selectedOption = descriptor.value === selector.value;
    return '<div class="scope-picker-option" role="option" tabindex="-1" data-scope-value="' + escapeHTML(descriptor.value) + '" aria-selected="' + String(selectedOption) + '" aria-label="' + escapeHTML(descriptor.accessibleLabel) + '">' +
      '<div class="scope-option-main"><span class="scope-option-title">' + escapeHTML(descriptor.root) + '</span>' + languageBadgeMarkup(descriptor.languageValue, descriptor.language) + '</div>' +
      '<div class="scope-option-meta">' + scopeStatusMarkup(descriptor.status, descriptor.statusClass) + '<span>' + escapeHTML(descriptor.countText) + '</span></div>' +
      "</div>";
  }).join("");
  closeScopePicker(context);
  bindScopePicker(context);
}

function aggregateScopeDescriptor(context, scopes) {
  const status = context.state.scene && (context.state.scene.aggregate_status || context.state.scene.status)
    ? (context.state.scene.aggregate_status || context.state.scene.status)
    : "ready";
  const usable = scopes.filter(function (scope) { return scope.status === "complete" || scope.status === "partial"; }).length;
  const countText = usable + "/" + scopes.length + " usable";
  return {
    value: "all",
    root: "All scopes",
    language: "Multi",
    languageValue: "multi",
    status: status,
    statusClass: scopeStatusClass(status),
    countText: countText,
    optionLabel: "All scopes · " + status + " · " + countText,
    accessibleLabel: "All scopes, " + status + ", " + countText
  };
}

function scopeDescriptor(scope) {
  const analyzer = scope && scope.analyzer ? scope.analyzer : {};
  const languageValue = String(analyzer.language || analyzer.id || "analyzer").trim().toLowerCase();
  const language = formatLanguage(languageValue);
  const root = displayProjectRoot(scope && scope.project_root);
  const status = String(scope && scope.status || "unknown").toLowerCase();
  const summary = scope && scope.summary ? scope.summary : {};
  const moduleCount = summary.module_count == null ? "module count unavailable" : summary.module_count + " module(s)";
  return {
    value: String(scope && scope.scope_id || ""),
    root: root,
    language: language,
    languageValue: languageValue,
    status: status,
    statusClass: scopeStatusClass(status),
    countText: moduleCount,
    optionLabel: root + " · " + language + " · " + status + " · " + moduleCount,
    accessibleLabel: root + ", " + language + ", " + status + ", " + moduleCount
  };
}

function activeScopeDescriptor(context) {
  const active = context.state.activeScope || "all";
  const scopes = context.state.scopes || [];
  if (active === "all") return aggregateScopeDescriptor(context, scopes);
  const scope = scopes.find(function (item) { return item.scope_id === active; });
  return scope ? scopeDescriptor(scope) : aggregateScopeDescriptor(context, scopes);
}

function scopeStatusClass(status) {
  const value = String(status || "unknown").toLowerCase();
  if (value === "complete") return "ok";
  if (value === "failed" || value === "cancelled") return "error";
  if (value === "partial") return "warning";
  return "";
}

function languageBadgeMarkup(languageValue, language) {
  return '<span class="language-badge ' + classForState(languageValue) + '">' + escapeHTML(String(language || "Unknown").toUpperCase()) + "</span>";
}

function scopeStatusMarkup(status, extraClass) {
  return '<span class="scope-status-badge ' + (extraClass || "") + '">' + escapeHTML(String(status || "unknown")) + "</span>";
}

function scopeButtonMarkup(descriptor) {
  return '<span class="scope-picker-button-content"><span class="scope-picker-button-primary"><span>' + escapeHTML(descriptor.root) + '</span>' + languageBadgeMarkup(descriptor.languageValue, descriptor.language) + '</span><span class="scope-picker-button-secondary">' + scopeStatusMarkup(descriptor.status, descriptor.statusClass) + ' <span>' + escapeHTML(descriptor.countText) + '</span></span></span><span class="scope-picker-chevron" aria-hidden="true">⌄</span>';
}

function bindScopePicker(context) {
  if (context.state.scopePickerBound) return;
  const button = context.elements.scopeSelectorButton;
  const menu = context.elements.scopeSelectorMenu;
  const control = context.elements.scopeSelectorControl;
  button.addEventListener("click", function () {
    if (menu.hidden) openScopePicker(context);
    else closeScopePicker(context);
  });
  button.addEventListener("keydown", function (event) {
    if (event.key === "ArrowDown" || event.key === "ArrowUp" || event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      openScopePicker(context);
      if (event.key === "ArrowUp") moveScopePickerFocus(context, -1);
    }
  });
  menu.addEventListener("click", function (event) {
    const option = event.target.closest("[data-scope-value]");
    if (option) chooseScope(context, option.dataset.scopeValue);
  });
  menu.addEventListener("keydown", function (event) {
    const option = event.target.closest("[data-scope-value]");
    if (!option) return;
    if (event.key === "ArrowDown" || event.key === "ArrowRight") {
      event.preventDefault();
      moveScopePickerFocus(context, 1);
    } else if (event.key === "ArrowUp" || event.key === "ArrowLeft") {
      event.preventDefault();
      moveScopePickerFocus(context, -1);
    } else if (event.key === "Home") {
      event.preventDefault();
      focusScopeOption(context, 0);
    } else if (event.key === "End") {
      event.preventDefault();
      focusScopeOption(context, menu.querySelectorAll("[data-scope-value]").length - 1);
    } else if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      chooseScope(context, option.dataset.scopeValue);
    } else if (event.key === "Escape") {
      event.preventDefault();
      closeScopePicker(context, true);
    }
  });
  document.addEventListener("click", function (event) {
    if (!control.contains(event.target)) closeScopePicker(context);
  });
  context.state.scopePickerBound = true;
}

function openScopePicker(context) {
  const button = context.elements.scopeSelectorButton;
  const menu = context.elements.scopeSelectorMenu;
  menu.hidden = false;
  button.setAttribute("aria-expanded", "true");
  const selected = menu.querySelector('[aria-selected="true"]');
  if (selected) selected.focus();
}

function closeScopePicker(context, restoreFocus) {
  const button = context.elements.scopeSelectorButton;
  const menu = context.elements.scopeSelectorMenu;
  if (!button || !menu) return;
  menu.hidden = true;
  button.setAttribute("aria-expanded", "false");
  if (restoreFocus) button.focus();
}

function focusScopeOption(context, index) {
  const options = context.elements.scopeSelectorMenu.querySelectorAll("[data-scope-value]");
  if (!options.length) return;
  const bounded = Math.max(0, Math.min(options.length - 1, index));
  options[bounded].focus();
}

function moveScopePickerFocus(context, delta) {
  const options = Array.from(context.elements.scopeSelectorMenu.querySelectorAll("[data-scope-value]"));
  if (!options.length) return;
  const current = options.indexOf(document.activeElement);
  const next = current < 0 ? 0 : (current + delta + options.length) % options.length;
  options[next].focus();
}

function chooseScope(context, value) {
  if (!value) return;
  const selector = context.elements.scopeSelector;
  const changed = context.state.activeScope !== value;
  context.state.activeScope = value;
  selector.value = value;
  closeScopePicker(context);
  renderScopeSelector(context);
  if (changed) selector.dispatchEvent(new Event("change", { bubbles: true }));
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
  if (scene.aggregate_status && scene.aggregate_status !== scene.status) pills.push('<span class="state-pill warning">aggregate ' + escapeHTML(scene.aggregate_status) + '</span>');
  if (scene.scope_status && scene.scope_status !== "aggregate") pills.push('<span class="state-pill ' + (scene.scope_status === "complete" ? "ok" : "warning") + '">scope ' + escapeHTML(scene.scope_status) + '</span>');
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
