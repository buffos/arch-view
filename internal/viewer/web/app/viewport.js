import { routePoints } from "../graph_route.js";
import { fallbackLayout } from "./layout.js";
import { sceneLayoutKey } from "./view.js";
import { clampNumber } from "./utils.js";
import { fitViewportTransform, visibleGraphArea } from "./viewport_math.js";
import { applyViewportFit, createViewportState, resetViewportZoom, updateViewportZoom } from "./viewport_runtime.js";

export function viewportKey(scene) {
  return "arch-view:viewport:" + [scene.model_revision || scene.model_id, (scene.hierarchy_path || []).join("/"), scene.reference_visibility].join("|");
}

export function defaultViewport() {
  return createViewportState();
}

export function loadViewport(context, scene) {
  const fallback = defaultViewport();
  try {
    const value = JSON.parse(sessionStorage.getItem(viewportKey(scene)) || "null");
    if (!value || typeof value !== "object") return fallback;
    return {
      zoom: clampNumber(value.zoom, context.constants.minZoom, context.constants.maxZoom, 1),
      panX: clampNumber(value.panX, -context.constants.panLimit, context.constants.panLimit, 0),
      panY: clampNumber(value.panY, -context.constants.panLimit, context.constants.panLimit, 0),
      positions: manualPositionsForScene(value.positions, scene)
    };
  } catch (error) {
    return fallback;
  }
}

export function persistViewport(context) {
  if (!context.state.scene || !context.state.viewport) return;
  try {
    sessionStorage.setItem(viewportKey(context.state.scene), JSON.stringify(context.state.viewport));
  } catch (error) {
    // Session persistence is an enhancement; the active view remains usable.
  }
}

export function restoreScroll(context, scene) {
  const saved = context.state.scrollContexts[viewportKey(scene)];
  window.requestAnimationFrame(function () {
    context.elements.graph.scrollLeft = saved ? saved.left : 0;
    context.elements.graph.scrollTop = saved ? saved.top : 0;
  });
}

function manualPositionsForScene(value, scene) {
  const positions = Object.create(null);
  if (!value || typeof value !== "object" || Array.isArray(value)) return positions;
  const nodeIDs = new Set((scene.visible_nodes || []).map(function (node) { return node.id; }));
  Object.keys(value).forEach(function (nodeID) {
    if (!nodeIDs.has(nodeID)) return;
    const position = value[nodeID];
    if (!position || typeof position !== "object" || Array.isArray(position)) return;
    if (!Number.isFinite(position.x) || !Number.isFinite(position.y)) return;
    positions[nodeID] = { x: position.x, y: position.y };
  });
  return positions;
}

export function currentGraphPositions(context, scene) {
  const fallback = fallbackLayout(context, scene);
  const active = context.state.layout && context.state.layout.key === sceneLayoutKey(scene) ? context.state.layout : fallback;
  const positions = Object.assign({}, fallback.positions, active.positions);
  const manualPositions = context.state.viewport && context.state.viewport.positions ? context.state.viewport.positions : {};
  Object.keys(manualPositions).forEach(function (nodeID) {
    if (positions[nodeID]) positions[nodeID] = Object.assign({}, positions[nodeID], manualPositions[nodeID]);
  });
  return positions;
}

export function layoutBounds(context, scene, layout, positions) {
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  function includePoint(x, y) {
    if (!Number.isFinite(x) || !Number.isFinite(y)) return;
    minX = Math.min(minX, x);
    minY = Math.min(minY, y);
    maxX = Math.max(maxX, x);
    maxY = Math.max(maxY, y);
  }
  function includeBox(box) {
    if (!box) return;
    includePoint(box.x, box.y);
    includePoint(box.x + box.width, box.y + box.height);
  }
  scene.visible_nodes.forEach(function (node) { includeBox(positions[node.id]); });
  const manualPositions = context.state.viewport && context.state.viewport.positions ? context.state.viewport.positions : {};
  const relationshipsByID = {};
  scene.visible_relationships.forEach(function (relationship) { relationshipsByID[relationship.id] = relationship; });
  Object.keys(layout.edges || {}).forEach(function (edgeID) {
    const relationship = relationshipsByID[edgeID];
    if (relationship && (manualPositions[relationship.from_visible_id] || manualPositions[relationship.to_visible_id])) return;
    const points = routePoints(layout.edges[edgeID]);
    if (points) points.forEach(function (point) { includePoint(point.x, point.y); });
  });
  if (!Number.isFinite(minX)) return { minX: 0, minY: 0, maxX: layout.width, maxY: layout.height };
  const padding = 28;
  return { minX: minX - padding, minY: minY - padding, maxX: maxX + padding, maxY: maxY + padding, centerX: (minX + maxX) / 2, centerY: (minY + maxY) / 2 };
}

export function renderViewportControls(context) {
  const viewport = context.state.viewport || defaultViewport();
  context.elements.zoomValue.textContent = Math.round(viewport.zoom * 100) + "%";
  context.elements.resetLayout.disabled = Object.keys(viewport.positions || {}).length === 0;
}

export function changeZoom(context, delta, services) {
	if (!context.state.viewport) context.state.viewport = defaultViewport();
	updateViewportZoom(context.state.viewport, delta, context.constants.minZoom, context.constants.maxZoom);
  persistViewport(context);
  renderViewportControls(context);
  services.renderGraph();
}

export function resetZoom(context, services) {
	if (!context.state.viewport) context.state.viewport = defaultViewport();
	resetViewportZoom(context.state.viewport);
  persistViewport(context);
  renderViewportControls(context);
  services.renderGraph();
}

export function resetLayout(context, services) {
  if (!context.state.viewport) context.state.viewport = defaultViewport();
  if (!Object.keys(context.state.viewport.positions || {}).length) return;
  context.state.viewport.positions = {};
  persistViewport(context);
  renderViewportControls(context);
  services.renderSceneState();
  services.renderGraph();
}

export function isExpandedCanvas() {
  return Boolean(document.fullscreenElement || document.body.classList.contains("canvas-focus") || document.body.classList.contains("canvas-fullscreen"));
}

export function syncCanvasOverflow() {
  document.documentElement.classList.toggle("canvas-focus", document.body.classList.contains("canvas-focus"));
  document.documentElement.classList.toggle("canvas-fullscreen", document.body.classList.contains("canvas-fullscreen"));
}

export function fitViewport(context, services) {
  const scene = context.state.scene;
  if (!scene) return;
  const fallback = fallbackLayout(context, scene);
  const active = context.state.layout && context.state.layout.key === sceneLayoutKey(scene) ? context.state.layout : fallback;
  const available = visibleGraphArea(context.elements.graph.clientWidth, context.elements.graph.clientHeight);
  const availableWidth = available.width;
  const availableHeight = available.height;
  const svg = context.elements.graph.querySelector("svg");
  const renderedWidth = Math.max(1, svg ? svg.clientWidth : availableWidth);
  const renderedHeight = Math.max(1, svg ? svg.clientHeight : availableHeight);
  const positions = currentGraphPositions(context, scene);
  const bounds = layoutBounds(context, scene, active, positions);
  if (!context.state.viewport) context.state.viewport = defaultViewport();
  const fitted = fitViewportTransform({
    availableWidth: availableWidth,
    availableHeight: availableHeight,
    renderedWidth: renderedWidth,
    renderedHeight: renderedHeight,
    layoutWidth: active.width,
    layoutHeight: active.height,
    bounds: bounds,
    minimumZoom: context.constants.minZoom,
    maximumZoom: context.constants.maxZoom,
    panLimit: context.constants.panLimit
  });
	applyViewportFit(context.state.viewport, fitted);
  persistViewport(context);
  renderViewportControls(context);
  services.renderGraph();
}

export function toggleFocusMode(context) {
  const active = isExpandedCanvas() || context.elements.focusToggle.textContent === "Exit canvas";
  if (active) {
    context.state.nativeFullscreen = false;
    if (document.fullscreenElement) document.exitFullscreen().catch(function () {});
    document.body.classList.remove("canvas-focus", "canvas-fullscreen");
    syncCanvasOverflow();
    context.elements.focusToggle.textContent = "Full canvas";
    return;
  }
  document.body.classList.add("canvas-focus");
  syncCanvasOverflow();
  context.elements.focusToggle.textContent = "Exit canvas";
  const sceneCard = document.querySelector(".scene-card");
  if (sceneCard && sceneCard.requestFullscreen) {
    context.state.nativeFullscreen = true;
    sceneCard.requestFullscreen().catch(function () { context.state.nativeFullscreen = false; });
  }
}

export function syncFocusButton(context) {
  if (document.fullscreenElement) {
    document.body.classList.add("canvas-fullscreen");
    syncCanvasOverflow();
    context.elements.focusToggle.textContent = "Exit canvas";
    return;
  }
  document.body.classList.remove("canvas-fullscreen");
  if (context.state.nativeFullscreen) {
    context.state.nativeFullscreen = false;
    document.body.classList.remove("canvas-focus");
  }
  syncCanvasOverflow();
  context.elements.focusToggle.textContent = document.body.classList.contains("canvas-focus") ? "Exit canvas" : "Full canvas";
}
