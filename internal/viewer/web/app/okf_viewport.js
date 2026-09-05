import { clampNumber } from "./utils.js";
import { geometryContainerBounds } from "./container_presentation.js";
import { fitViewportTransform, visibleGraphArea } from "./viewport_math.js";
import { applyViewportFit, createViewportState, resetViewportZoom, updateViewportZoom, viewportTransform } from "./viewport_runtime.js";

const OKF_MIN_ZOOM = 0.35;
const OKF_MAX_ZOOM = 40000;
const OKF_PAN_LIMIT = 100000;

export function ensureOKFViewport(state, snapshot) {
  const key = viewportKey(snapshot);
  if (state.viewportKey === key && state.viewport) return false;
  state.viewportKey = key;
  state.layout = null;
  const saved = readViewport(key, snapshot);
  state.viewport = saved || createViewportState();
  state.viewportInitialized = Boolean(saved);
  return !saved;
}

export function persistOKFViewport(state) {
  if (!state.viewport || !state.viewportKey) return;
  try {
    sessionStorage.setItem(state.viewportKey, JSON.stringify({
      zoom: state.viewport.zoom,
      panX: state.viewport.panX,
      panY: state.viewport.panY,
      positions: state.viewport.positions || {}
    }));
  } catch (error) {
    // A full session store must not be required for the active graph to work.
  }
}

export function renderOKFViewportControls(state, elements) {
  if (elements.zoomValue && state.viewport) elements.zoomValue.textContent = Math.round(state.viewport.zoom * 100) + "%";
  if (elements.resetLayout) elements.resetLayout.disabled = !state.viewport || !Object.keys(state.viewport.positions || {}).length;
}

export function changeOKFZoom(state, elements, delta) {
  if (!state.viewport) state.viewport = createViewportState();
  updateViewportZoom(state.viewport, delta, OKF_MIN_ZOOM, OKF_MAX_ZOOM);
  state.viewportInitialized = true;
  persistOKFViewport(state);
  renderOKFViewportControls(state, elements);
  applyOKFViewport(elements, state);
}

export function resetOKFZoom(state, elements) {
  if (!state.viewport) state.viewport = createViewportState();
  resetViewportZoom(state.viewport);
  state.viewportInitialized = true;
  persistOKFViewport(state);
  renderOKFViewportControls(state, elements);
  applyOKFViewport(elements, state);
}

export function fitOKFViewport(state, elements) {
  if (!state.viewport || !state.layout || !elements.graph) return;
  const svg = elements.graph.querySelector("svg");
  if (!svg) return;
  const available = visibleGraphArea(elements.graph.clientWidth, elements.graph.clientHeight);
  const renderedWidth = Math.max(1, svg.clientWidth || available.width);
  const renderedHeight = Math.max(1, svg.clientHeight || available.height);
  const bounds = layoutBounds(state.layout, state.viewport.positions);
  const fitted = fitViewportTransform({
    availableWidth: available.width,
    availableHeight: available.height,
    renderedWidth: renderedWidth,
    renderedHeight: renderedHeight,
    layoutWidth: state.layout.width,
    layoutHeight: state.layout.height,
    bounds: bounds,
    minimumZoom: OKF_MIN_ZOOM,
    maximumZoom: OKF_MAX_ZOOM,
    panLimit: OKF_PAN_LIMIT
  });
  applyViewportFit(state.viewport, fitted);
  state.viewportInitialized = true;
  persistOKFViewport(state);
  renderOKFViewportControls(state, elements);
  applyOKFViewport(elements, state);
}

export function applyOKFViewport(elements, state) {
  const content = elements.graph && elements.graph.querySelector(".okf-viewport-content");
  if (content) content.setAttribute("transform", viewportTransform(state.viewport));
}

function viewportKey(snapshot) {
  const source = snapshot && snapshot.source ? snapshot.source : {};
  const profile = snapshot && snapshot.profile ? snapshot.profile : {};
  const navigation = snapshot && snapshot.navigation ? snapshot.navigation : {};
  return "arch-view:okf:viewport:" + [source.bundle_id, source.source_revision, profile.profile_id, profile.profile_revision, navigation.focus_root, navigation.depth, navigation.full].join("|");
}

function readViewport(key, snapshot) {
  try {
    const value = JSON.parse(sessionStorage.getItem(key) || "null");
    if (!value || typeof value !== "object") return null;
    const nodeIDs = new Set((snapshot && snapshot.nodes || []).map((node) => node.id || node.concept_id));
    const positions = {};
    Object.keys(value.positions || {}).forEach((id) => {
      const position = value.positions[id];
      if (!nodeIDs.has(id) || !position || !Number.isFinite(position.x) || !Number.isFinite(position.y)) return;
      positions[id] = { x: position.x, y: position.y };
    });
    return {
      zoom: clampNumber(value.zoom, OKF_MIN_ZOOM, OKF_MAX_ZOOM, 1),
      panX: clampNumber(value.panX, -OKF_PAN_LIMIT, OKF_PAN_LIMIT, 0),
      panY: clampNumber(value.panY, -OKF_PAN_LIMIT, OKF_PAN_LIMIT, 0),
      positions
    };
  } catch (error) {
    return null;
  }
}

function layoutBounds(layout, manualPositions = {}) {
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  const positions = {};
  Object.keys(layout.positions || {}).forEach(function (id) {
    positions[id] = Object.assign({}, layout.positions[id], manualPositions[id] || {});
    const position = positions[id];
    if (!position) return;
    minX = Math.min(minX, position.x);
    minY = Math.min(minY, position.y);
    maxX = Math.max(maxX, position.x + position.width);
    maxY = Math.max(maxY, position.y + position.height);
  });
  Object.values(geometryContainerBounds(layout.geometry, positions)).forEach(function (bounds) {
    minX = Math.min(minX, bounds.x);
    minY = Math.min(minY, bounds.y);
    maxX = Math.max(maxX, bounds.x + bounds.width);
    maxY = Math.max(maxY, bounds.y + bounds.height);
  });
  if (!Number.isFinite(minX)) return { minX: 0, minY: 0, maxX: layout.width, maxY: layout.height, centerX: layout.width / 2, centerY: layout.height / 2 };
  const padding = 28;
  return { minX: minX - padding, minY: minY - padding, maxX: maxX + padding, maxY: maxY + padding, centerX: (minX + maxX) / 2, centerY: (minY + maxY) / 2 };
}
