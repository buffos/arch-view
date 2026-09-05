import { edgeGeometry } from "../graph_route.js";
import { applyGeometryMove, geometryContainerMarkup, geometryMoveStart } from "./container_presentation.js";
import { activeGeometryJunctions, geometryJunctionMarkup, geometryLabelMarkup } from "./edge_presentation.js";
import { geometryEdge } from "./geometry_snapshot.js";
import { nodeShapeMarkup } from "./node_shape.js";
import { geometryPortMarkup, geometryPortRoute, geometryPortsForNode } from "./port_presentation.js";
import { fallbackLayout } from "./layout.js";
import { sceneLayoutKey } from "./view.js";
import { defaultViewport, persistViewport, renderViewportControls } from "./viewport.js";
import { applyViewportTransform, bindViewportGestures } from "./viewport_runtime.js";
import { classForState, escapeHTML, formatLanguage, nodeLanguageBadge, nodeLanguageText, referenceScopeLabel, truncate } from "./utils.js";

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
    const snapshotEdge = geometryEdge(activeLayout.geometry, relationship.id);
    const portRoute = geometryPortRoute(activeLayout.geometry, snapshotEdge, positions, hasManualEndpoint ? null : activeLayout.edges[relationship.id]);
    const geometry = edgeGeometry(relationship, from, to, portRoute?.route || (hasManualEndpoint ? null : activeLayout.edges[relationship.id]), portRoute?.preserveEndpoints);
    const advanced = !hasManualEndpoint && snapshotEdge;
    const label = advanced?.labels?.length
      ? geometryLabelMarkup(advanced.labels, classForState(relationship.cycle_state) + (!matches ? " dimmed" : ""))
      : '<text class="edge-label ' + classForState(relationship.cycle_state) + (!matches ? " dimmed" : "") + '" x="' + geometry.labelX + '" y="' + geometry.labelY + '" text-anchor="middle">' + escapeHTML(relationship.count) + "</text>";
    return '<g class="edge-group" data-edge-id="' + escapeHTML(relationship.id) + '" tabindex="0" role="button" aria-label="' + escapeHTML(relationship.accessible_label) + '">' +
      '<path class="edge-hit" d="' + geometry.path + '"></path><path class="' + className + '" d="' + geometry.path + '" marker-end="url(#arrow)"></path>' +
      label + "</g>";
  }).join("");
  const junctionMarkup = geometryJunctionMarkup(activeGeometryJunctions(activeLayout.geometry, manualPositions));
  const containerMarkup = geometryContainerMarkup(activeLayout.geometry, positions);

  const nodeMarkup = nodes.map(function (node) {
    const position = positions[node.id];
    if (!position) return "";
    const renderPosition = position;
    const selected = state.selected && state.selected.kind === "node" && state.selected.id === node.id;
    const matches = nodeMatches(state, node);
    const diagnosticClass = node.diagnostic_state === "none" ? "" : " " + classForState(node.diagnostic_state);
    const className = "node-shape " + classForState(node.kind) + " " + classForState(node.cycle_state) + diagnosticClass + (selected ? " selected" : "") + (!matches ? " dimmed" : "");
    const layer = node.layer == null ? (node.layers && node.layers.length ? "L" + node.layers.join(", L") : "—") : "L" + node.layer;
    const scope = node.reference_scope ? " · " + referenceScopeLabel(node.reference_scope) : "";
    const languageValue = nodeLanguageBadge(context, node);
    const language = nodeLanguageText(context, node);
    const subtitle = language + " · " + node.kind + scope + " · " + layer + (node.counts.module_count > 1 ? " · " + node.counts.module_count + " modules" : "");
    const status = nodeStatusText(node);
    const accessibleLabel = node.accessible_label + "; " + language + " language";
    const badgeText = String(languageValue === "multi" ? "MULTI" : formatLanguage(languageValue)).toUpperCase();
    const badgeWidth = Math.max(48, Math.min(renderPosition.width - 24, 18 + badgeText.length * 6.2));
    const badgeX = renderPosition.x + renderPosition.width - badgeWidth - 12;
    const badgeClass = "node-language-badge " + classForState(languageValue) + (!matches ? " dimmed" : "");
    const ports = geometryPortMarkup(geometryPortsForNode(activeLayout.geometry, node.id, position));
    return '<g data-node-id="' + escapeHTML(node.id) + '" data-node-kind="' + escapeHTML(node.kind) + '" data-node-language="' + escapeHTML(languageValue) + '" tabindex="0" role="button" aria-label="' + escapeHTML(accessibleLabel) + '">' +
      '<title>' + escapeHTML(accessibleLabel) + "</title>" +
      nodeShapeMarkup("rounded_rectangle", renderPosition, ' class="' + className + '"') +
      nodeShapeMarkup("rounded_rectangle", renderPosition, ' class="node-hitzone"') +
      '<rect class="' + badgeClass + '" x="' + badgeX + '" y="' + (renderPosition.y + 10) + '" width="' + badgeWidth + '" height="18" rx="9"></rect>' +
      '<text class="node-language-badge-label ' + (!matches ? "dimmed" : "") + '" x="' + (badgeX + badgeWidth / 2) + '" y="' + (renderPosition.y + 22.5) + '" text-anchor="middle">' + escapeHTML(badgeText) + "</text>" +
      '<text class="node-label ' + (!matches ? "dimmed" : "") + '" x="' + (renderPosition.x + 14) + '" y="' + (renderPosition.y + 30) + '">' + escapeHTML(truncate(node.label, 25)) + "</text>" +
      '<text class="node-subtitle" x="' + (renderPosition.x + 14) + '" y="' + (renderPosition.y + 51) + '">' + escapeHTML(truncate(subtitle, 29)) + "</text>" +
      '<text class="node-subtitle" x="' + (renderPosition.x + 14) + '" y="' + (renderPosition.y + 68) + '">' + escapeHTML(truncate(status, 29)) + "</text>" + ports + "</g>";
  }).join("");

  const empty = nodes.length === 0 ? '<p class="muted">No visible nodes in this projection.</p>' : "";
  context.elements.graph.innerHTML = empty + '<svg viewBox="0 0 ' + activeLayout.width + ' ' + activeLayout.height + '" role="img" aria-labelledby="graph-title graph-desc" xmlns="http://www.w3.org/2000/svg"><title id="graph-title">Architecture graph</title><desc id="graph-desc">' + escapeHTML(scene.accessibility.reading_order.length + " semantic items in the current scene") + '</desc><defs><marker id="arrow" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto"><path d="M 0 0 L 9 4.5 L 0 9 z" fill="#7483a9"></path></marker></defs><g class="viewport-content" transform="' + transform + '"><g class="geometry-containers">' + containerMarkup + '</g><g class="edges">' + edgeMarkup + junctionMarkup + '</g><g class="nodes">' + nodeMarkup + "</g></g></svg>";
  const svg = context.elements.graph.querySelector("svg");
  if (!svg) return;
  context.elements.graph.insertAdjacentHTML("beforeend", '<span class="viewport-hint graph-viewport-hint">Drag the canvas to pan; Shift-drag a node to adjust this session.</span>');
  bindGraphInteractions(context, svg, positions, activeLayout.geometry, services);
}

function bindGraphInteractions(context, svg, positions, geometrySnapshot, services) {
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
      if (event.shiftKey) beginNodeDrag(context, event, svg, nodeID, positions, geometrySnapshot, services);
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
  bindViewportGestures(svg, {
    getViewport: function () {
      if (!context.state.viewport) context.state.viewport = defaultViewport();
      return context.state.viewport;
    },
    getBaseScale: function () { return graphBaseScale(context, svg); },
    panSpeed: context.constants.panSpeed,
    panLimit: context.constants.panLimit,
    isPanTarget: function (target) { return !(target && typeof target.closest === "function" && target.closest("[data-node-id], [data-edge-id]")); },
    onChange: function () { persistViewport(context); },
    onRender: function () { applyViewportTransform(context.elements.graph, ".viewport-content", context.state.viewport); },
    onZoom: function (delta) { services.changeZoom(delta); }
  });
}

function graphBaseScale(context, svg) {
  if (!context.state.scene || !svg) return 1;
  const fallback = fallbackLayout(context, context.state.scene);
  const active = context.state.layout && context.state.layout.key === sceneLayoutKey(context.state.scene) ? context.state.layout : fallback;
  const width = Math.max(1, svg.clientWidth);
  const height = Math.max(1, svg.clientHeight);
  return Math.max(0.01, Math.min(width / Math.max(1, active.width), height / Math.max(1, active.height)));
}

function beginNodeDrag(context, event, svg, nodeID, positions, geometrySnapshot, services) {
  const position = positions[nodeID];
  if (event.button !== 0 || !position) return;
  if (!context.state.viewport) context.state.viewport = defaultViewport();
  const start = { x: event.clientX, y: event.clientY, nodeX: position.x, nodeY: position.y };
  const baseScale = graphBaseScale(context, svg);
  const drag = { nodeID: nodeID };
  const moveSet = geometryMoveStart(geometrySnapshot, nodeID, positions);
  context.state.draggingNode = drag;
  let moved = false;
  event.preventDefault();
  event.stopPropagation();
  function move(pointerEvent) {
    const dx = (pointerEvent.clientX - start.x) / (baseScale * context.state.viewport.zoom);
    const dy = (pointerEvent.clientY - start.y) / (baseScale * context.state.viewport.zoom);
    if (Math.abs(dx) + Math.abs(dy) > 3) moved = true;
    applyGeometryMove(moveSet, positions, context.state.viewport.positions, dx, dy);
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
