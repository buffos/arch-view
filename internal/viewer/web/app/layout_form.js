import { escapeHTML, formatOptionValue, optionBoundsText } from "./utils.js";
import { renderOptionControl, updateOptionValue } from "./layout_option_controls.js";
import { featureProblems } from "./layout_features.js";
import { validPadding } from "./layout_value.js";

export function optionAppliesToAlgorithm(option, algorithm) {
  const algorithms = option && Array.isArray(option.algorithms) ? option.algorithms : [];
  return !algorithms.length || algorithms.includes("all") || algorithms.includes(algorithm);
}

function optionDefaultText(option) {
  return option.default === "engine default" ? "engine default" : formatOptionValue(option.default);
}

function optionDisplayValue(option, draft) {
  if (draft && draft.options && Object.prototype.hasOwnProperty.call(draft.options, option.id)) return formatOptionValue(draft.options[option.id]);
  return "";
}

export function optionAvailability(option, draft) {
  if (!optionAppliesToAlgorithm(option, draft.algorithm)) return "Not applicable to this algorithm";
  const missing = (option.required_features || []).filter((id) => !(draft.features || []).includes(id));
  if (missing.length) return "Requires an enabled feature: " + missing.join(", ");
  if (!option.editable || option.renderer_support !== "supported") return "Not implemented";
  return "";
}

function optionInput(option, draft) {
  const reason = optionAvailability(option, draft);
  return reason ? '<span class="layout-option-state">' + escapeHTML(reason) + "</span>" : renderOptionControl(option, draft);
}

export function layoutDraftProblems(draft, catalog) {
  const problems = featureProblems(draft, catalog);
  for (const [id, value] of Object.entries(draft?.options || {})) {
    const option = catalog?.options?.find((item) => item.id === id);
    if (!option) { problems.push("Unknown layout option: " + id); continue; }
    const reason = optionAvailability(option, draft);
    if (reason) problems.push(option.name + ": " + reason);
    if (option.control === "padding" && !validPadding(value)) problems.push("Padding requires four numbers between 0 and 10000.");
    if (["INT", "DOUBLE"].includes(option.type) && (!Number.isFinite(value) || option.type === "INT" && !Number.isInteger(value)
      || option.minimum != null && (value < option.minimum || option.minimum_exclusive && value === option.minimum)
      || option.maximum != null && (value > option.maximum || option.maximum_exclusive && value === option.maximum))) problems.push(option.name + " is outside its supported range.");
    if (option.allowed_values?.length && !option.allowed_values.includes(value)) problems.push(option.name + " has an unsupported value.");
  }
  return problems;
}

function featureControls(state, expanded) {
  return '<label><input type="checkbox" data-layout-catalog-filter' + (state.layoutShowCatalog ? " checked" : "") + '> Show complete catalog</label>'
    + '<details data-layout-feature-section' + (expanded ? " open" : "") + '><summary>Advanced renderer features</summary>'
    + (state.layoutCatalog.features || []).map((feature) => {
      const selected = (state.layoutDraft.features || []).includes(feature.id);
      return '<label class="layout-option-state"><input type="checkbox" data-layout-feature="' + escapeHTML(feature.id) + '"'
        + (selected ? " checked" : "") + (!selected && feature.status !== "supported" ? " disabled" : "") + "> "
        + escapeHTML(feature.name) + (feature.status === "supported" ? "" : " · Not implemented, delivery stage " + feature.stage)
        + (feature.support_note ? '<span class="layout-feature-note">' + escapeHTML(feature.support_note) + "</span>" : "") + "</label>";
    }).join("") + "</details>";
}

function originLabel(config, options) {
  if (options.originLabel) return options.originLabel;
  if (!config) return "Loading layout profile…";
  const origin = config.origin || "default";
  const labels = { default: "Built-in defaults", project: "Project .archview.json", ancestor: "Ancestor .archview.json", custom: "Custom .archview.json", session: "Current session (unsaved)" };
  const label = labels[origin] || origin;
  return config.active_path ? label + " · " + config.active_path : label;
}

function renderDiagnostic(elements, config) {
  const diagnostics = config && config.diagnostics ? config.diagnostics : [];
  if (!elements.layoutSettingsDiagnostic) return;
  if (!diagnostics.length) {
    elements.layoutSettingsDiagnostic.hidden = true;
    elements.layoutSettingsDiagnostic.textContent = "";
    return;
  }
  elements.layoutSettingsDiagnostic.hidden = false;
  elements.layoutSettingsDiagnostic.className = "layout-settings-diagnostic " + (diagnostics.some(function (item) { return item.severity === "error"; }) ? "error" : "info");
  elements.layoutSettingsDiagnostic.innerHTML = diagnostics.map(function (item) {
    return "<strong>" + escapeHTML(item.code || "diagnostic") + "</strong> " + escapeHTML(item.message || "") + (item.path ? " <code>" + escapeHTML(item.path) + "</code>" : "");
  }).join("<br>");
}

export function renderLayoutForm(context, config, options = {}) {
  const elements = context.elements;
  const state = context.state;
  if (!elements.layoutSettingsDialog) return;
  const featureSectionExpanded = Boolean(elements.layoutOptionsList.querySelector?.("[data-layout-feature-section]")?.open);
  elements.layoutSettingsOrigin.textContent = originLabel(config || state.layoutConfig, options);
  renderDiagnostic(elements, config || state.layoutConfig);
  elements.layoutSettingsStatus.textContent = options.status || state.layoutMessage || (config && config.status === "invalid" ? "Safe defaults are active until the profile is corrected." : "");
  elements.layoutSettingsStatus.className = "muted" + (state.layoutMessageError || (config && config.status === "invalid") ? " layout-settings-status-error" : "");
  if (!state.layoutCatalog || !state.layoutCatalog.algorithms || !state.layoutCatalog.options || !state.layoutDraft) {
    elements.layoutAlgorithm.innerHTML = "<option>Loading…</option>";
    elements.layoutAlgorithm.disabled = true;
    elements.layoutOptionsList.innerHTML = '<p class="muted">Loading the pinned ELK catalog…</p>';
    elements.layoutSettingsApply.disabled = true;
    if (elements.layoutSettingsSave) elements.layoutSettingsSave.disabled = true;
    if (elements.layoutSettingsSaveAs) elements.layoutSettingsSaveAs.disabled = true;
    return;
  }
  elements.layoutAlgorithm.disabled = false;
  elements.layoutAlgorithm.innerHTML = state.layoutCatalog.algorithms.map(function (algorithm) {
    return '<option value="' + escapeHTML(algorithm.id) + '"' + (state.layoutDraft.algorithm === algorithm.id ? " selected" : "") + ">" + escapeHTML(algorithm.name) + "</option>";
  }).join("");
  const selectedAlgorithm = state.layoutCatalog.algorithms.find(function (algorithm) { return algorithm.id === state.layoutDraft.algorithm; });
  elements.layoutAlgorithmHelp.textContent = selectedAlgorithm ? selectedAlgorithm.description + (selectedAlgorithm.category ? " Category: " + selectedAlgorithm.category + "." : "") : "The selected algorithm is not available in the pinned catalog.";
  const query = state.layoutOptionSearch || "";
  const optionsList = state.layoutCatalog.options.slice().sort(function (left, right) {
    return String(left.group || "").localeCompare(String(right.group || "")) || String(left.name).localeCompare(String(right.name));
  }).filter(function (option) {
    return (state.layoutShowCatalog || !optionAvailability(option, state.layoutDraft) || Object.hasOwn(state.layoutDraft.options || {}, option.id))
      && (!query || [option.id, option.name, option.group, option.description].join(" ").toLowerCase().includes(query));
  });
  if (!optionsList.length) {
    elements.layoutOptionsList.innerHTML = '<p class="muted">No ELK options match this search.</p>';
  } else {
    let lastGroup = null;
    const markup = [];
    optionsList.forEach(function (option) {
      const group = option.group || "General";
      const applicable = optionAppliesToAlgorithm(option, state.layoutDraft.algorithm);
      const current = optionDisplayValue(option, state.layoutDraft);
      const currentLabel = current || optionDefaultText(option);
      const supportClass = !applicable ? "not-applicable" : option.editable && option.renderer_support === "supported" ? "editable" : "catalog-only";
      if (group !== lastGroup) {
        markup.push('<h4 class="layout-option-group">' + escapeHTML(group) + "</h4>");
        lastGroup = group;
      }
      const reset = Object.hasOwn(state.layoutDraft.options || {}, option.id) ? '<label><input type="checkbox" data-layout-clear="' + escapeHTML(option.id) + '"> Reset to engine default</label>' : "";
      markup.push('<article class="layout-option ' + supportClass + '"><div class="layout-option-copy"><div class="layout-option-title"><strong>' + escapeHTML(option.name) + '</strong><code>' + escapeHTML(option.id) + '</code></div><div class="layout-option-meta">' + escapeHTML(group) + " · " + escapeHTML(option.type) + " · default: " + escapeHTML(optionDefaultText(option)) + " · current: " + escapeHTML(currentLabel) + escapeHTML(optionBoundsText(option)) + '</div><p>' + escapeHTML(option.description) + '</p><p>' + escapeHTML(option.support_note || "") + '</p></div><div class="layout-option-control">' + optionInput(option, state.layoutDraft) + reset + '</div></article>');
    });
    elements.layoutOptionsList.innerHTML = markup.join("");
  }
  elements.layoutOptionsList.innerHTML = featureControls(state, featureSectionExpanded) + elements.layoutOptionsList.innerHTML;
  const problems = layoutDraftProblems(state.layoutDraft, state.layoutCatalog);
  if (problems.length) {
    elements.layoutSettingsStatus.textContent = problems.join(" ");
    elements.layoutSettingsStatus.className = "layout-settings-status-error";
  }
  elements.layoutSettingsApply.disabled = Boolean(options.disableApply) || problems.length > 0;
  if (elements.layoutSettingsSave) elements.layoutSettingsSave.hidden = options.showPersistence === false;
  if (elements.layoutSettingsSaveAs) elements.layoutSettingsSaveAs.hidden = options.showPersistence === false;
  if (elements.layoutResetDefaults) elements.layoutResetDefaults.hidden = options.showPersistence === false;
  if (elements.layoutSaveAsDirectory) {
    const saveAs = elements.layoutSaveAsDirectory.closest(".layout-save-as");
    if (saveAs) saveAs.hidden = options.showPersistence === false;
  }
  if (elements.layoutSettingsSave) elements.layoutSettingsSave.disabled = problems.length > 0 || options.showPersistence === false || !config || !config.can_save;
  if (elements.layoutSettingsSaveAs) elements.layoutSettingsSaveAs.disabled = problems.length > 0 || options.showPersistence === false || !config || !config.can_save_as;
}

export function updateLayoutDraftAlgorithm(context, event) {
  const state = context.state;
  if (!state.layoutDraft) state.layoutDraft = { algorithm: "layered", options: {} };
  state.layoutDraft.algorithm = event.target.value;
  state.layoutMessage = "Unsaved settings changes";
  state.layoutMessageError = false;
  renderLayoutForm(context, context.state.layoutConfig, context.layoutFormOptions || {});
}

export function updateLayoutDraftOption(context, event) {
  const state = context.state;
  const control = event.target.dataset || {};
  if (Object.hasOwn(control, "layoutCatalogFilter")) {
    state.layoutShowCatalog = event.target.checked;
    renderLayoutForm(context, state.layoutConfig, context.layoutFormOptions || {}); return;
  }
  if (control.layoutFeature) {
    const selected = new Set(state.layoutDraft.features || []);
    if (event.target.checked) selected.add(control.layoutFeature); else selected.delete(control.layoutFeature);
    state.layoutDraft.features = [...selected].sort();
    renderLayoutForm(context, state.layoutConfig, context.layoutFormOptions || {}); return;
  }
  if (control.layoutClear) {
    delete state.layoutDraft.options[control.layoutClear];
    renderLayoutForm(context, state.layoutConfig, context.layoutFormOptions || {}); return;
  }
  const input = event.target.closest("[data-layout-option-id]");
  if (!input || !state.layoutDraft || !state.layoutCatalog) return;
  const option = (state.layoutCatalog.options || []).find(function (item) { return item.id === input.dataset.layoutOptionId; });
  if (!option) return;
  updateOptionValue(option, input, state.layoutDraft);
  state.layoutMessage = "Unsaved settings changes";
  state.layoutMessageError = false;
  context.elements.layoutSettingsStatus.textContent = state.layoutMessage;
  context.elements.layoutSettingsStatus.className = "muted";
  if (event.type === "change") renderLayoutForm(context, state.layoutConfig, context.layoutFormOptions || {});
}
