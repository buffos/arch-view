import { fromELKSections } from "../graph_route.js";
import { buildELKGraph } from "../layout_request.js";
import { sceneLayoutKey } from "./view.js";
import { cloneLayoutProfile, escapeHTML, formatOptionValue, numberOrZero, optionBoundsText, layoutRequestPayload } from "./utils.js";

export function optionAppliesToAlgorithm(option, algorithm) {
  const algorithms = option && Array.isArray(option.algorithms) ? option.algorithms : [];
  return !algorithms.length || algorithms.includes("all") || algorithms.includes(algorithm);
}

export function fallbackLayout(context, scene) {
  if (context.embeddedExport && context.embeddedExport.layouts) {
    const exported = context.embeddedExport.layouts[sceneLayoutKey(scene)];
    if (exported) return exported;
  }
  const nodes = scene.visible_nodes;
  const byLayer = {};
  nodes.forEach(function (node) {
    const layer = node.layer == null ? ((node.layers && node.layers[0]) || 0) : node.layer;
    if (!byLayer[layer]) byLayer[layer] = [];
    byLayer[layer].push(node);
  });
  const layers = Object.keys(byLayer).sort(function (left, right) { return Number(left) - Number(right); });
  const nodeWidth = 190;
  const nodeHeight = 82;
  const columnGap = 84;
  const rowGap = 35;
  const maxRows = Math.max.apply(null, layers.map(function (layer) { return byLayer[layer].length; }).concat([1]));
  const width = Math.max(760, layers.length * (nodeWidth + columnGap) + 80);
  const height = Math.max(430, maxRows * (nodeHeight + rowGap) + 100);
  const positions = {};
  layers.forEach(function (layer, layerIndex) {
    byLayer[layer].sort(function (left, right) {
      if (left.label === right.label) return left.id < right.id ? -1 : left.id > right.id ? 1 : 0;
      return left.label < right.label ? -1 : 1;
    });
    byLayer[layer].forEach(function (node, rowIndex) {
      positions[node.id] = { x: 40 + layerIndex * (nodeWidth + columnGap), y: 42 + rowIndex * (nodeHeight + rowGap), width: nodeWidth, height: nodeHeight };
    });
  });
  return { engine: "fallback", key: sceneLayoutKey(scene), width: width, height: height, positions: positions, edges: {} };
}

export function adaptELKLayout(scene, result, key, algorithm) {
  const offset = 24;
  const positions = {};
  (result.children || []).forEach(function (child) {
    positions[child.id] = { x: numberOrZero(child.x) + offset, y: numberOrZero(child.y) + offset, width: numberOrZero(child.width) || 190, height: numberOrZero(child.height) || 82 };
  });
  const edges = {};
  (result.edges || []).forEach(function (edge) {
    const route = fromELKSections(edge.sections, offset);
    if (route) edges[edge.id] = route;
  });
  const fallback = fallbackLayout({ embeddedExport: null }, scene);
  return {
    engine: "elk." + (algorithm || "layered"),
    key: key,
    width: Math.max(760, numberOrZero(result.width) + offset * 2, fallback.width),
    height: Math.max(430, numberOrZero(result.height) + offset * 2, fallback.height),
    positions: positions,
    edges: edges
  };
}

export async function prepareLayout(context, scene, profile, services) {
  if (context.embeddedExport || !context.workerURL || !scene || !scene.visible_nodes.length || typeof window.ELK !== "function") return;
  const key = sceneLayoutKey(scene);
  context.state.layoutKey = key;
  const request = ++context.state.layoutRequest;
  let elk;
  try {
    elk = new window.ELK({ workerUrl: context.workerURL });
    const result = await elk.layout(buildELKGraph(scene, profile || context.state.layoutProfile, context.state.layoutCatalog));
    if (request !== context.state.layoutRequest || context.state.scene !== scene) return;
    context.state.layout = adaptELKLayout(scene, result, key, (context.state.layoutProfile && context.state.layoutProfile.algorithm) || "layered");
    context.state.layoutError = false;
    services.renderSceneState();
    services.renderViewportControls();
    services.renderGraph();
  } catch (error) {
    if (request !== context.state.layoutRequest || context.state.scene !== scene) return;
    context.state.layout = null;
    context.state.layoutError = true;
    context.state.layoutMessage = "ELK layout failed; deterministic fallback is active.";
    context.state.layoutMessageError = true;
    services.renderSceneState();
    services.renderGraph();
    if (context.state.layoutSettingsOpen) services.renderLayoutSettings();
  } finally {
    if (elk && typeof elk.terminateWorker === "function") {
      try {
        const termination = elk.terminateWorker();
        if (termination && typeof termination.catch === "function") termination.catch(function () {});
      } catch (error) {
        // The bundled ELK build can fall back to a non-worker adapter.
      }
    }
  }
}

export async function loadLayoutConfig(context, api, services) {
  if (context.embeddedExport) return;
  const request = ++context.state.layoutConfigRequest;
  try {
    const values = await Promise.all([api.getJSON("/v1/layout/options"), api.getJSON("/v1/layout/config")]);
    if (request !== context.state.layoutConfigRequest) return;
    context.state.layoutCatalog = values[0];
    context.state.layoutConfig = values[1];
    context.state.layoutProfile = cloneLayoutProfile(values[1].layout);
    if (!context.state.layoutSettingsOpen) context.state.layoutDraft = cloneLayoutProfile(context.state.layoutProfile);
    if (context.state.layoutSettingsOpen) renderLayoutSettings(context);
    if (context.state.scene) {
      context.state.layout = null;
      context.state.layoutRequest += 1;
      services.renderSceneState();
      services.renderGraph();
      void prepareLayout(context, context.state.scene, context.state.layoutProfile, services);
    }
  } catch (error) {
    if (request !== context.state.layoutConfigRequest) return;
    context.state.layoutConfig = { layout: cloneLayoutProfile(context.state.layoutProfile), origin: "session", status: "invalid", can_save: false, can_save_as: false, diagnostics: [{ code: "config_unavailable", severity: "error", message: error.message || "Layout settings could not be loaded." }] };
    context.state.layoutMessage = error.message || "Layout settings could not be loaded.";
    context.state.layoutMessageError = true;
    renderLayoutSettings(context);
  }
}

function optionDefaultText(option) {
  return option.default === "engine default" ? "engine default" : formatOptionValue(option.default);
}

function optionDisplayValue(option, draft) {
  if (draft && draft.options && Object.prototype.hasOwnProperty.call(draft.options, option.id)) return String(draft.options[option.id]);
  return "";
}

function layoutOptionInput(option, draft, applicable) {
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

function layoutOriginLabel(config) {
  if (!config) return "Loading layout profile…";
  const origin = config.origin || "default";
  const labels = { default: "Built-in defaults", project: "Project .archview.json", ancestor: "Ancestor .archview.json", custom: "Custom .archview.json", session: "Current session (unsaved)" };
  const label = labels[origin] || origin;
  return config.active_path ? label + " · " + config.active_path : label;
}

function renderLayoutSettingsDiagnostic(context, config) {
  const diagnostics = config && config.diagnostics ? config.diagnostics : [];
  if (!diagnostics.length) {
    context.elements.layoutSettingsDiagnostic.hidden = true;
    context.elements.layoutSettingsDiagnostic.textContent = "";
    return;
  }
  context.elements.layoutSettingsDiagnostic.hidden = false;
  context.elements.layoutSettingsDiagnostic.className = "layout-settings-diagnostic " + (diagnostics.some(function (item) { return item.severity === "error"; }) ? "error" : "info");
  context.elements.layoutSettingsDiagnostic.innerHTML = diagnostics.map(function (item) {
    return "<strong>" + escapeHTML(item.code || "diagnostic") + "</strong> " + escapeHTML(item.message || "") + (item.path ? " <code>" + escapeHTML(item.path) + "</code>" : "");
  }).join("<br>");
}

export function renderLayoutSettings(context) {
  const elements = context.elements;
  const state = context.state;
  if (!elements.layoutSettingsDialog) return;
  const config = state.layoutConfig;
  const catalog = state.layoutCatalog;
  elements.layoutSettingsOrigin.textContent = layoutOriginLabel(config);
  renderLayoutSettingsDiagnostic(context, config);
  elements.layoutSettingsStatus.textContent = state.layoutMessage || (config && config.status === "invalid" ? "Safe defaults are active until the profile is corrected." : "");
  elements.layoutSettingsStatus.className = "muted" + (state.layoutMessageError || (config && config.status === "invalid") ? " layout-settings-status-error" : "");
  if (!catalog || !catalog.algorithms || !catalog.options || !state.layoutDraft) {
    elements.layoutAlgorithm.innerHTML = "<option>Loading…</option>";
    elements.layoutAlgorithm.disabled = true;
    elements.layoutOptionsList.innerHTML = '<p class="muted">Loading the pinned ELK catalog…</p>';
    elements.layoutSettingsApply.disabled = true;
    elements.layoutSettingsSave.disabled = true;
    elements.layoutSettingsSaveAs.disabled = true;
    return;
  }
  elements.layoutAlgorithm.disabled = false;
  elements.layoutAlgorithm.innerHTML = catalog.algorithms.map(function (algorithm) {
    return '<option value="' + escapeHTML(algorithm.id) + '"' + (state.layoutDraft.algorithm === algorithm.id ? " selected" : "") + ">" + escapeHTML(algorithm.name) + "</option>";
  }).join("");
  const selectedAlgorithm = catalog.algorithms.find(function (algorithm) { return algorithm.id === state.layoutDraft.algorithm; });
  elements.layoutAlgorithmHelp.textContent = selectedAlgorithm ? selectedAlgorithm.description + (selectedAlgorithm.category ? " Category: " + selectedAlgorithm.category + "." : "") : "The selected algorithm is not available in the pinned catalog.";
  const query = state.layoutOptionSearch || "";
  const options = catalog.options.slice().sort(function (left, right) {
    const group = String(left.group || "").localeCompare(String(right.group || ""));
    return group || String(left.name).localeCompare(String(right.name));
  }).filter(function (option) {
    return !query || [option.id, option.name, option.group, option.description].join(" ").toLowerCase().includes(query);
  });
  if (!options.length) {
    elements.layoutOptionsList.innerHTML = '<p class="muted">No ELK options match this search.</p>';
  } else {
    let lastGroup = null;
    const markup = [];
    options.forEach(function (option) {
      const group = option.group || "General";
      const applicable = optionAppliesToAlgorithm(option, state.layoutDraft.algorithm);
      const current = optionDisplayValue(option, state.layoutDraft);
      const currentLabel = current || optionDefaultText(option);
      const supportClass = !applicable ? "not-applicable" : option.editable && option.renderer_support === "supported" ? "editable" : "catalog-only";
      if (group !== lastGroup) {
        markup.push('<h4 class="layout-option-group">' + escapeHTML(group) + "</h4>");
        lastGroup = group;
      }
      markup.push('<article class="layout-option ' + supportClass + '"><div class="layout-option-copy"><div class="layout-option-title"><strong>' + escapeHTML(option.name) + '</strong><code>' + escapeHTML(option.id) + '</code></div><div class="layout-option-meta">' + escapeHTML(group) + " · " + escapeHTML(option.type) + " · default: " + escapeHTML(optionDefaultText(option)) + " · current: " + escapeHTML(currentLabel) + escapeHTML(optionBoundsText(option)) + '</div><p>' + escapeHTML(option.description) + '</p></div><div class="layout-option-control">' + layoutOptionInput(option, state.layoutDraft, applicable) + '</div></article>');
    });
    elements.layoutOptionsList.innerHTML = markup.join("");
  }
  elements.layoutSettingsApply.disabled = false;
  elements.layoutSettingsSave.disabled = !config || !config.can_save;
  elements.layoutSettingsSaveAs.disabled = !config || !config.can_save_as;
}

export function openLayoutSettings(context) {
  if (context.embeddedExport) return;
  context.state.layoutSettingsOpen = true;
  context.state.layoutMessage = "";
  context.state.layoutMessageError = false;
  context.state.layoutDraft = cloneLayoutProfile(context.state.layoutProfile);
  renderLayoutSettings(context);
  if (context.elements.layoutSettingsDialog.showModal) context.elements.layoutSettingsDialog.showModal();
  else context.elements.layoutSettingsDialog.setAttribute("open", "");
}

export function closeLayoutSettings(context) {
  context.state.layoutSettingsOpen = false;
  context.state.layoutMessage = "";
  context.state.layoutMessageError = false;
  if (context.elements.layoutSettingsDialog.close) context.elements.layoutSettingsDialog.close();
  else context.elements.layoutSettingsDialog.removeAttribute("open");
}

export function updateLayoutDraftAlgorithm(context, event) {
  const state = context.state;
  if (!state.layoutDraft) state.layoutDraft = cloneLayoutProfile(state.layoutProfile);
  state.layoutDraft.algorithm = event.target.value;
  if (state.layoutCatalog && state.layoutDraft.options) {
    Object.keys(state.layoutDraft.options).forEach(function (optionID) {
      const option = state.layoutCatalog.options.find(function (item) { return item.id === optionID; });
      if (option && !optionAppliesToAlgorithm(option, state.layoutDraft.algorithm)) delete state.layoutDraft.options[optionID];
    });
  }
  state.layoutMessage = "Unsaved settings changes";
  state.layoutMessageError = false;
  renderLayoutSettings(context);
}

export function updateLayoutDraftOption(context, event) {
  const input = event.target.closest("[data-layout-option-id]");
  const state = context.state;
  if (!input || !state.layoutDraft) return;
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

function clearManualLayoutPositions(context, services) {
  if (!context.state.viewport) context.state.viewport = { zoom: 1, panX: 0, panY: 0, positions: {} };
  context.state.viewport.positions = {};
  services.persistViewport();
}

async function updateActiveProfile(context, response, message, services) {
  context.state.layoutConfig = response;
  context.state.layoutProfile = cloneLayoutProfile(response.layout);
  context.state.layoutDraft = cloneLayoutProfile(context.state.layoutProfile);
  clearManualLayoutPositions(context, services);
  context.state.layout = null;
  context.state.layoutError = false;
  context.state.layoutRequest += 1;
  context.state.layoutMessage = message;
  context.state.layoutMessageError = false;
  services.renderAll();
  renderLayoutSettings(context);
  void prepareLayout(context, context.state.scene, context.state.layoutProfile, services);
}

export async function applyLayoutProfile(context, api, services) {
  if (!context.state.layoutDraft || context.embeddedExport) return;
  context.state.layoutMessage = "Applying layout…";
  context.state.layoutMessageError = false;
  renderLayoutSettings(context);
  try {
    const response = await api.postJSON("/v1/layout/apply", layoutRequestPayload(context.state.layoutDraft));
    await updateActiveProfile(context, response, "Applied to the current session", services);
  } catch (error) {
    context.state.layoutMessage = error.message || "Layout settings could not be applied.";
    context.state.layoutMessageError = true;
    renderLayoutSettings(context);
  }
}

export async function resetLayoutProfile(context, api, services) {
  if (context.embeddedExport) return;
  context.state.layoutMessage = "Restoring built-in defaults…";
  context.state.layoutMessageError = false;
  renderLayoutSettings(context);
  try {
    const response = await api.postJSON("/v1/layout/reset", {});
    await updateActiveProfile(context, response, "Built-in defaults restored for this session", services);
  } catch (error) {
    context.state.layoutMessage = error.message || "Layout defaults could not be restored.";
    context.state.layoutMessageError = true;
    renderLayoutSettings(context);
  }
}

export async function saveLayoutProfile(context, api, services) {
  if (!context.state.layoutDraft || context.embeddedExport) return;
  context.state.layoutMessage = "Saving active profile…";
  context.state.layoutMessageError = false;
  renderLayoutSettings(context);
  try {
    const response = await api.putJSON("/v1/layout/config", layoutRequestPayload(context.state.layoutDraft));
    await updateActiveProfile(context, response, "Saved active .archview.json", services);
  } catch (error) {
    context.state.layoutMessage = error.message || "The active layout profile could not be saved.";
    context.state.layoutMessageError = true;
    renderLayoutSettings(context);
  }
}

export async function saveLayoutProfileAs(context, api, services) {
  if (!context.state.layoutDraft || context.embeddedExport) return;
  const destination = context.elements.layoutSaveAsDirectory.value.trim();
  const confirm = context.elements.layoutSaveAsConfirm.checked;
  if (!destination || !confirm) {
    context.state.layoutMessage = "Enter a directory and confirm Save As before writing .archview.json.";
    context.state.layoutMessageError = true;
    renderLayoutSettings(context);
    return;
  }
  context.state.layoutMessage = "Saving a custom profile…";
  context.state.layoutMessageError = false;
  renderLayoutSettings(context);
  try {
    const response = await api.putJSON("/v1/layout/config/save-as", { schema_version: "arch-view.config/v1", layout: cloneLayoutProfile(context.state.layoutDraft), destination_dir: destination, confirm: confirm });
    await updateActiveProfile(context, response, "Saved custom .archview.json", services);
  } catch (error) {
    context.state.layoutMessage = error.message || "The custom layout profile could not be saved.";
    context.state.layoutMessageError = true;
    renderLayoutSettings(context);
  }
}
