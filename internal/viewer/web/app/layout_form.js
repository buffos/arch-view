import { escapeHTML, formatOptionValue, optionBoundsText } from "./utils.js";

export function optionAppliesToAlgorithm(option, algorithm) {
  const algorithms = option && Array.isArray(option.algorithms) ? option.algorithms : [];
  return !algorithms.length || algorithms.includes("all") || algorithms.includes(algorithm);
}

function optionDefaultText(option) {
  return option.default === "engine default" ? "engine default" : formatOptionValue(option.default);
}

function optionDisplayValue(option, draft) {
  if (draft && draft.options && Object.prototype.hasOwnProperty.call(draft.options, option.id)) return String(draft.options[option.id]);
  return "";
}

function optionInput(option, draft, applicable) {
  if (!applicable) return '<span class="layout-option-state">Not applicable to this algorithm</span>';
  if (!option.editable || option.renderer_support !== "supported") return '<span class="layout-option-state">Cataloged · unsupported by this renderer</span>';
  const value = optionDisplayValue(option, draft);
  const id = "layout-option-" + option.id.replace(/[^a-z0-9_-]/gi, "-");
  const attributes = 'data-layout-option-id="' + escapeHTML(option.id) + '" aria-label="' + escapeHTML(option.name) + '"';
  if (option.allowed_values && option.allowed_values.length) {
    const choices = ['<option value="">Engine default</option>'].concat(option.allowed_values.map(function (choice) {
      const text = String(choice);
      return '<option value="' + escapeHTML(text) + '"' + (value === text ? " selected" : "") + ">" + escapeHTML(text) + "</option>";
    }));
    return '<select id="' + id + '" class="layout-option-input" ' + attributes + ">" + choices.join("") + "</select>";
  }
  if (option.type === "BOOLEAN") return '<select id="' + id + '" class="layout-option-input" ' + attributes + '><option value="">Engine default</option><option value="true"' + (value === "true" ? " selected" : "") + '>true</option><option value="false"' + (value === "false" ? " selected" : "") + '>false</option></select>';
  if (option.type === "INT" || option.type === "DOUBLE") {
    const step = option.type === "INT" ? "1" : "any";
    const min = option.minimum == null ? "" : ' min="' + escapeHTML(option.minimum) + '"';
    const max = option.maximum == null ? "" : ' max="' + escapeHTML(option.maximum) + '"';
    return '<input id="' + id + '" class="layout-option-input" type="number" step="' + step + '" value="' + escapeHTML(value) + '" placeholder="Engine default"' + min + max + " " + attributes + ">";
  }
  return '<input id="' + id + '" class="layout-option-input" type="text" value="' + escapeHTML(value) + '" placeholder="Engine default" ' + attributes + ">";
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
    return !query || [option.id, option.name, option.group, option.description].join(" ").toLowerCase().includes(query);
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
      markup.push('<article class="layout-option ' + supportClass + '"><div class="layout-option-copy"><div class="layout-option-title"><strong>' + escapeHTML(option.name) + '</strong><code>' + escapeHTML(option.id) + '</code></div><div class="layout-option-meta">' + escapeHTML(group) + " · " + escapeHTML(option.type) + " · default: " + escapeHTML(optionDefaultText(option)) + " · current: " + escapeHTML(currentLabel) + escapeHTML(optionBoundsText(option)) + '</div><p>' + escapeHTML(option.description) + '</p></div><div class="layout-option-control">' + optionInput(option, state.layoutDraft, applicable) + '</div></article>');
    });
    elements.layoutOptionsList.innerHTML = markup.join("");
  }
  elements.layoutSettingsApply.disabled = Boolean(options.disableApply);
  if (elements.layoutSettingsSave) elements.layoutSettingsSave.hidden = options.showPersistence === false;
  if (elements.layoutSettingsSaveAs) elements.layoutSettingsSaveAs.hidden = options.showPersistence === false;
  if (elements.layoutResetDefaults) elements.layoutResetDefaults.hidden = options.showPersistence === false;
  if (elements.layoutSaveAsDirectory) {
    const saveAs = elements.layoutSaveAsDirectory.closest(".layout-save-as");
    if (saveAs) saveAs.hidden = options.showPersistence === false;
  }
  if (elements.layoutSettingsSave) elements.layoutSettingsSave.disabled = options.showPersistence === false || !config || !config.can_save;
  if (elements.layoutSettingsSaveAs) elements.layoutSettingsSaveAs.disabled = options.showPersistence === false || !config || !config.can_save_as;
}

export function updateLayoutDraftAlgorithm(context, event) {
  const state = context.state;
  if (!state.layoutDraft) state.layoutDraft = { algorithm: "layered", options: {} };
  state.layoutDraft.algorithm = event.target.value;
  if (state.layoutCatalog && state.layoutDraft.options) {
    Object.keys(state.layoutDraft.options).forEach(function (optionID) {
      const option = state.layoutCatalog.options.find(function (item) { return item.id === optionID; });
      if (option && !optionAppliesToAlgorithm(option, state.layoutDraft.algorithm)) delete state.layoutDraft.options[optionID];
    });
  }
  state.layoutMessage = "Unsaved settings changes";
  state.layoutMessageError = false;
  renderLayoutForm(context, context.state.layoutConfig, context.layoutFormOptions || {});
}

export function updateLayoutDraftOption(context, event) {
  const input = event.target.closest("[data-layout-option-id]");
  const state = context.state;
  if (!input || !state.layoutDraft || !state.layoutCatalog) return;
  const option = (state.layoutCatalog.options || []).find(function (item) { return item.id === input.dataset.layoutOptionId; });
  if (!option) return;
  const raw = input.value;
  if (raw === "") delete state.layoutDraft.options[option.id];
  else if (option.type === "BOOLEAN") state.layoutDraft.options[option.id] = raw === "true";
  else if (option.type === "INT") state.layoutDraft.options[option.id] = Number.parseInt(raw, 10);
  else if (option.type === "DOUBLE") state.layoutDraft.options[option.id] = Number.parseFloat(raw);
  else state.layoutDraft.options[option.id] = raw;
  state.layoutMessage = "Unsaved settings changes";
  state.layoutMessageError = false;
  context.elements.layoutSettingsStatus.textContent = state.layoutMessage;
  context.elements.layoutSettingsStatus.className = "muted";
}
