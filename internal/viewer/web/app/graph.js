import { edgeGeometry } from "../graph_route.js";
import { fallbackLayout } from "./layout.js";
import { sceneLayoutKey } from "./view.js";
import { defaultViewport, persistViewport, renderViewportControls } from "./viewport.js";
import { classForState, escapeHTML, referenceScopeLabel, truncate } from "./utils.js";

export function renderGraph(context, services) {
  const state = context.state;
  if (!state.scene) return;
  const scene = state.scene;
  const nodes = scene.visible_nodes;
  const nodesByID = {};
  nodes.forEach(function (node) { nodesByID[node.id] = node; });
  const fallback = fallbackLayout(context, scene);
  const activeLayout = state.layout && state.layout.key === sceneLayoutKey(scene) ? state.layout : fallback;
  const positions = Object.assign({}, fallback.positions, activeLayout.positions);
  const manualPositions = state.viewport && state.viewport.positions ? state.viewport.positions : {};
  Object.keys(manualPositions).forEach(function (nodeID) {
    if (!positions[nodeID]) return;
    positions[nodeID] = Object.assign({}, positions[nodeID], manualPositions[nodeID]);
  });
  const viewport = state.viewport || defaultViewport();
  const transform = "translate(" + viewport.panX + " " + viewport.panY + ") scale(" + viewport.zoom + ")";

  const edgeMarkup = scene.visible_relationships.map(function (relationship) {
    const from = positions[relationship.from_visible_id];
    const to = positions[relationship.to_visible_id];
    if (!from || !to) return "";
    const selected = state.selected && state.selected.kind === "relationship" && state.selected.id === relationship.id;
    const matches = relationshipMatches(state, relationship, nodesByID);
    const className = "edge-line " + classForState(relationship.cycle_state) + (selected ? " selected" : "") + (!matches ? " dimmed" : "");
    const hasManualEndpoint = manualPositions[relationship.from_visible_id] || manualPositions[relationship.to_visible_id];
    const geometry = edgeGeometry(relationship, from, to, hasManualEndpoint ? null : activeLayout.edges[relationship.id]);
    return '<g class="edge-group" data-edge-id="' + escapeHTML(relationship.id) + '" tabindex="0" role="button" aria-label="' + escapeHTML(relationship.accessible_label) + '">' +
      '<path class="edge-hit" d="' + geometry.path + '"></path><path class="' + className + '" d="' + geometry.path + '" marker-end="url(#arrow)"></path>' +
      '<text class="edge-label ' + classForState(relationship.cycle_state) + (!matches ? " dimmed" : "") + '" x="' + geometry.labelX + '" y="' + geometry.labelY + '" text-anchor="middle">' + escapeHTML(relationship.count) + "</text></g>";
  }).join("");

  const nodeMarkup = nodes.map(function (node) {
    const position = positions[node.id];
    if (!position) return "";
    const selected = state.selected && state.selected.kind === "node" && state.selected.id === node.id;
    const matches = nodeMatches(state, node);
    const diagnosticClass = node.diagnostic_state === "none" ? "" : " " + classForState(node.diagnostic_state);
    const className = "node-shape " + classForState(node.kind) + " " + classForState(node.cycle_state) + diagnosticClass + (selected ? " selected" : "") + (!matches ? " dimmed" : "");
    const layer = node.layer == null ? (node.layers && node.layers.length ? "L" + node.layers.join(", L") : "—") : "L" + node.layer;
    const scope = node.reference_scope ? " · " + referenceScopeLabel(node.reference_scope) : "";
    const subtitle = node.kind + scope + " · " + layer + (node.counts.module_count > 1 ? " · " + node.counts.module_count + " modules" : "");
    const status = nodeStatusText(node);
    return '<g data-node-id="' + escapeHTML(node.id) + '" data-node-kind="' + escapeHTML(node.kind) + '" tabindex="0" role="button" aria-label="' + escapeHTML(node.accessible_label) + '">' +
      '<title>' + escapeHTML(node.accessible_label) + "</title>" +
      '<rect class="' + className + '" x="' + position.x + '" y="' + position.y + '" width="' + position.width + '" height="' + position.height + '" rx="12"></rect>' +
      '<rect class="node-hitzone" x="' + position.x + '" y="' + position.y + '" width="' + position.width + '" height="' + position.height + '" rx="12"></rect>' +
      '<text class="node-label ' + (!matches ? "dimmed" : "") + '" x="' + (position.x + 14) + '" y="' + (position.y + 30) + '">' + escapeHTML(truncate(node.label, 25)) + "</text>" +
      '<text class="node-subtitle" x="' + (position.x + 14) + '" y="' + (position.y + 51) + '">' + escapeHTML(truncate(subtitle, 29)) + "</text>" +
      '<text class="node-subtitle" x="' + (position.x + 14) + '" y="' + (position.y + 68) + '">' + escapeHTML(truncate(status, 29)) + "</text></g>";
  }).join("");

  const empty = nodes.length === 0 ? '<p class="muted">No visible nodes in this projection.</p>' : "";
  context.elements.graph.innerHTML = empty + '<svg viewBox="0 0 ' + activeLayout.width + ' ' + activeLayout.height + '" role="img" aria-labelledby="graph-title graph-desc" xmlns="http://www.w3.org/2000/svg"><title id="graph-title">Architecture graph</title><desc id="graph-desc">' + escapeHTML(scene.accessibility.reading_order.length + " semantic items in the current scene") + '</desc><defs><marker id="arrow" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto"><path d="M 0 0 L 9 4.5 L 0 9 z" fill="#7483a9"></path></marker></defs><g class="viewport-content" transform="' + transform + '"><g class="edges">' + edgeMarkup + '</g><g class="nodes">' + nodeMarkup + "</g></g></svg>";
  const svg = context.elements.graph.querySelector("svg");
  if (!svg) return;
  context.elements.graph.insertAdjacentHTML("beforeend", '<span class="viewport-hint graph-viewport-hint">Drag the canvas to pan; Shift-drag a node to adjust this session.</span>');
  bindGraphInteractions(context, svg, positions, services);
}

function bindGraphInteractions(context, svg, positions, services) {
  context.elements.graph.querySelectorAll("[data-node-id]").forEach(function (element) {
    const nodeID = element.dataset.nodeId;
    const select = function (preserveDoubleClick) {
      if (Date.now() < context.state.suppressClickUntil) return;
      services.selectEntity("node", nodeID, preserveDoubleClick);
    };
    const navigate = function () {
      const node = context.state.scene.visible_nodes.find(function (item) { return item.id === nodeID; });
      if (node && node.kind === "group") services.navigationTo(node.hierarchy_path);
    };
    const handleClick = function (event) {
      const now = Date.now();
      if (now < context.state.suppressClickUntil) return;
      const previous = context.state.lastNodeClick;
      const dx = previous && event ? event.clientX - previous.x : Infinity;
      const dy = previous && event ? event.clientY - previous.y : Infinity;
      const isDoubleClick = previous && previous.nodeID === nodeID && (event.detail >= 2 || (now - previous.time <= context.constants.nodeDoubleClickWindow && Math.sqrt(dx * dx + dy * dy) <= context.constants.nodeDoubleClickDistance));
      if (isDoubleClick) {
        context.state.lastNodeClick = null;
        navigate();
        return;
      }
      context.state.lastNodeClick = { nodeID: nodeID, time: now, x: event.clientX, y: event.clientY };
      select(true);
    };
    element.addEventListener("click", handleClick);
    element.addEventListener("keydown", function (event) {
      if (event.key === "Enter" || event.key === " ") { event.preventDefault(); select(false); }
      if (event.key === "ArrowRight") {
        const node = context.state.scene.visible_nodes.find(function (item) { return item.id === nodeID; });
        if (node && node.kind === "group") services.navigationTo(node.hierarchy_path);
      }
    });
    element.addEventListener("pointerdown", function (event) {
      if (event.shiftKey) beginNodeDrag(context, event, svg, nodeID, positions[nodeID], services);
    });
  });
  context.elements.graph.querySelectorAll("[data-edge-id]").forEach(function (element) {
    const select = function () {
      if (Date.now() < context.state.suppressClickUntil) return;
      services.selectEntity("relationship", element.dataset.edgeId);
    };
    element.addEventListener("click", select);
    element.addEventListener("keydown", function (event) {
      if (event.key === "Enter" || event.key === " ") { event.preventDefault(); select(); }
    });
  });
  svg.addEventListener("pointerdown", function (event) {
    if (event.button !== 0 || event.target.closest("[data-node-id], [data-edge-id]")) return;
    beginPan(context, event, svg, services);
  });
  svg.addEventListener("wheel", function (event) {
    event.preventDefault();
    services.changeZoom(event.deltaY < 0 ? 0.08 : -0.08);
  }, { passive: false });
}

function graphBaseScale(context, svg) {
  if (!context.state.scene || !svg) return 1;
  const fallback = fallbackLayout(context, context.state.scene);
  const active = context.state.layout && context.state.layout.key === sceneLayoutKey(context.state.scene) ? context.state.layout : fallback;
  const width = Math.max(1, svg.clientWidth);
  const height = Math.max(1, svg.clientHeight);
  return Math.max(0.01, Math.min(width / Math.max(1, active.width), height / Math.max(1, active.height)));
}

function beginPan(context, event, svg, services) {
  if (!context.state.viewport) context.state.viewport = defaultViewport();
  const start = { x: event.clientX, y: event.clientY, panX: context.state.viewport.panX, panY: context.state.viewport.panY };
  const baseScale = graphBaseScale(context, svg);
  event.preventDefault();
  function move(pointerEvent) {
    context.state.viewport.panX = clamp(context, start.panX + ((pointerEvent.clientX - start.x) / baseScale) * context.constants.panSpeed, -context.constants.panLimit, context.constants.panLimit, start.panX);
    context.state.viewport.panY = clamp(context, start.panY + ((pointerEvent.clientY - start.y) / baseScale) * context.constants.panSpeed, -context.constants.panLimit, context.constants.panLimit, start.panY);
    persistViewport(context);
    services.renderGraph();
  }
  function stop() {
    document.removeEventListener("pointermove", move);
    document.removeEventListener("pointerup", stop);
    document.removeEventListener("pointercancel", stop);
  }
  document.addEventListener("pointermove", move);
  document.addEventListener("pointerup", stop);
  document.addEventListener("pointercancel", stop);
}

function beginNodeDrag(context, event, svg, nodeID, position, services) {
  if (event.button !== 0 || !position) return;
  if (!context.state.viewport) context.state.viewport = defaultViewport();
  const start = { x: event.clientX, y: event.clientY, nodeX: position.x, nodeY: position.y };
  const baseScale = graphBaseScale(context, svg);
  const drag = { nodeID: nodeID };
  context.state.draggingNode = drag;
  let moved = false;
  event.preventDefault();
  event.stopPropagation();
  function move(pointerEvent) {
    const dx = (pointerEvent.clientX - start.x) / (baseScale * context.state.viewport.zoom);
    const dy = (pointerEvent.clientY - start.y) / (baseScale * context.state.viewport.zoom);
    if (Math.abs(dx) + Math.abs(dy) > 3) moved = true;
    context.state.viewport.positions[nodeID] = { x: start.nodeX + dx, y: start.nodeY + dy };
    persistViewport(context);
    if (!context.state.dragFrame) {
      context.state.dragFrame = window.requestAnimationFrame(function () {
        context.state.dragFrame = 0;
        if (context.state.draggingNode === drag) {
          renderViewportControls(context);
          services.renderGraph();
        }
      });
    }
  }
  function stop() {
    document.removeEventListener("pointermove", move);
    document.removeEventListener("pointerup", stop);
    document.removeEventListener("pointercancel", stop);
    if (context.state.dragFrame) window.cancelAnimationFrame(context.state.dragFrame);
    context.state.dragFrame = 0;
    if (context.state.draggingNode === drag) context.state.draggingNode = null;
    renderViewportControls(context);
    services.renderGraph();
    if (moved) {
      context.state.lastNodeClick = null;
      context.state.suppressClickUntil = Date.now() + 180;
    }
  }
  document.addEventListener("pointermove", move);
  document.addEventListener("pointerup", stop);
  document.addEventListener("pointercancel", stop);
}

function clamp(context, value, minimum, maximum, fallback) {
  return typeof value === "number" && Number.isFinite(value) ? Math.max(minimum, Math.min(maximum, value)) : fallback;
}

function nodeMatches(state, node) {
  if (!state.query) return true;
  return [node.label, node.id, node.kind, node.reference_scope, (node.hierarchy_path || []).join(" ")].join(" ").toLowerCase().includes(state.query);
}

function relationshipMatches(state, relationship, nodesByID) {
  if (!state.query) return true;
  const from = nodesByID[relationship.from_visible_id];
  const to = nodesByID[relationship.to_visible_id];
  return [relationship.id, relationship.type, relationship.target_scope, from && from.label, to && to.label].join(" ").toLowerCase().includes(state.query);
}

function nodeStatusText(node) {
  if (node.counts.internal_relationship_count > 0) return node.counts.internal_relationship_count + " internal relationship" + (node.counts.internal_relationship_count === 1 ? "" : "s");
  const identity = node.identity_state || "stable";
  if (node.confidence_state === "not_applicable") return (node.cycle_state !== "none" ? node.cycle_state + " · " : "") + "identity " + identity;
  return (node.cycle_state !== "none" ? node.cycle_state + " · " : "") + node.confidence_state + " confidence";
}
