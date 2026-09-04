import { fromELKSections, fromELKSplineSections } from "../graph_route.js";
import { buildELKGraph } from "../layout_request.js";
import { runELKLayout } from "./elk_runtime.js";
import { renderLayoutForm } from "./layout_form.js";
import { sceneLayoutKey } from "./view.js";
import { cloneLayoutProfile, numberOrZero, layoutRequestPayload } from "./utils.js";

export { optionAppliesToAlgorithm, updateLayoutDraftAlgorithm, updateLayoutDraftOption } from "./layout_form.js";

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

function profileEdgeRouting(profile) {
  const options = profile && profile.options ? profile.options : {};
  return String(options["org.eclipse.elk.edgeRouting"] || options["elk.edgeRouting"] || "ORTHOGONAL").toUpperCase();
}

export function adaptELKLayout(scene, result, key, algorithm, profile) {
  const offset = 24;
  const positions = {};
  (result.children || []).forEach(function (child) {
    positions[child.id] = { x: numberOrZero(child.x) + offset, y: numberOrZero(child.y) + offset, width: numberOrZero(child.width) || 190, height: numberOrZero(child.height) || 82 };
  });
  const edges = {};
  const edgeRouting = profileEdgeRouting(profile);
  (result.edges || []).forEach(function (edge) {
    const route = edgeRouting === "SPLINES"
      ? fromELKSplineSections(edge.sections, offset)
      : fromELKSections(edge.sections, offset);
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
  if (!scene || !scene.visible_nodes.length || typeof window.ELK !== "function") return;
  const key = sceneLayoutKey(scene);
  context.state.layoutKey = key;
  const request = ++context.state.layoutRequest;
  try {
    const result = await runELKLayout(buildELKGraph(scene, profile || context.state.layoutProfile, context.state.layoutCatalog), context.workerURL, window.ELK);
    if (request !== context.state.layoutRequest || context.state.scene !== scene) return;
    context.state.layout = adaptELKLayout(scene, result, key, (context.state.layoutProfile && context.state.layoutProfile.algorithm) || "layered", profile || context.state.layoutProfile);
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

export function renderLayoutSettings(context) {
  renderLayoutForm(context, context.state.layoutConfig, context.layoutFormOptions || {});
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
