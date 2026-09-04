import { edgeGeometry, straightRoute } from "../graph_route.js";
import { nodeShapeMarkup, shapeContentBox } from "./node_shape.js";
import { nodeDimensions } from "../layout_request.js";
import { escapeOKF } from "./okf_markup.js";
import { cloneOKFLayoutProfile } from "./okf_layout.js";
import { adaptELKLayout } from "./layout.js";
import { runFeatureLayout } from "./elk_runtime.js";
import { activeGeometryJunctions, geometryJunctionMarkup, geometryLabelMarkup } from "./edge_presentation.js";
import { geometryEdge } from "./geometry_snapshot.js";
import { okfGeometrySource } from "./geometry_source.js";
import { bindViewportGestures, createViewportState, viewportTransform } from "./viewport_runtime.js";

const DEFAULT_PAN_LIMIT = 100000;
const renderedLayouts = new WeakMap();

export async function renderOKFGraph(container, snapshot, selectedID, handlers, workerURL) {
  if (!container) return null;
  const options = handlers || {};
  const nodes = snapshot && Array.isArray(snapshot.nodes) ? snapshot.nodes : [];
  const allRelationships = snapshot && Array.isArray(snapshot.relationships) ? snapshot.relationships : [];
  const relationships = allRelationships.filter((relationship) => relationship.kind !== "semantic_link");
  const profile = cloneOKFLayoutProfile(options.layoutProfile || snapshot?.profile?.layout);
  const sceneKey = JSON.stringify([nodes, relationships]);
  const previous = renderedLayouts.get(container);
  let layout = fallbackLayout(nodes);
  if (nodes.length && typeof globalThis.ELK === "function") {
    try {
      const scene = {
        visible_nodes: nodes.map((node) => Object.assign({ id: node.id || node.concept_id }, node)),
        visible_relationships: relationships.map((relationship) => Object.assign({ id: relationship.id, from_visible_id: relationship.from, to_visible_id: relationship.to }, relationship))
      };
      const result = await runFeatureLayout(scene, profile, options.layoutCatalog, workerURL, globalThis.ELK, okfGeometrySource(snapshot));
      layout = adaptELKLayout(scene, result, "okf:" + (snapshot.projection_revision || "projection"), profile.algorithm, profile);
    } catch (error) {
      if (options.isCurrent && !options.isCurrent()) return null;
      const retained = previous && previous.sceneKey === sceneKey;
      if (retained) layout = previous.layout;
      if (options.onLayoutError) options.onLayoutError(error, { retained: Boolean(retained) });
    }
  }
  if (options.isCurrent && !options.isCurrent()) return null;
  renderedLayouts.set(container, { sceneKey, layout });
  const viewport = options.getViewport ? options.getViewport() : createViewportState();
  const positions = withManualPositions(layout.positions, viewport && viewport.positions, nodes);
  container.classList.add("okf-graph-wrap");
  const edges = relationships.map((relationship) => edgeMarkup(relationship, positions, layout, viewport)).join("");
  const junctions = geometryJunctionMarkup(activeGeometryJunctions(layout.geometry, viewport?.positions));
  const nodeMarkup = nodes.map((node) => nodeMarkupFor(node, positions[node.id || node.concept_id], selectedID)).join("");
  container.innerHTML = nodes.length
    ? "<svg class=\"okf-graph-svg\" viewBox=\"0 0 " + layout.width + " " + layout.height + "\" preserveAspectRatio=\"xMidYMid meet\" role=\"img\" aria-label=\"OKF knowledge graph\"><defs><marker id=\"okf-arrow\" markerWidth=\"9\" markerHeight=\"9\" refX=\"8\" refY=\"4.5\" orient=\"auto\"><path d=\"M 0 0 L 9 4.5 L 0 9 z\" fill=\"#7483a9\"></path></marker></defs><g class=\"okf-viewport-content\" transform=\"" + viewportTransform(viewport) + "\"><g class=\"okf-edges\">" + edges + junctions + "</g><g class=\"okf-nodes\">" + nodeMarkup + "</g></g></svg>"
    : "<p class=\"muted\">No concepts are visible in the selected OKF projection.</p>";
  if (options.onLayoutReady) options.onLayoutReady(layout);
  const svg = container.querySelector("svg");
  if (!svg) return layout;
  bindNodeInteractions(container, options, positions, allRelationships, layout);
  bindViewportGestures(svg, {
    getViewport: options.getViewport || (() => createViewportState()),
    getBaseScale: () => graphBaseScale(svg, layout),
    panSpeed: Number.isFinite(options.panSpeed) ? options.panSpeed : 1,
    panLimit: Number.isFinite(options.panLimit) ? options.panLimit : DEFAULT_PAN_LIMIT,
    isPanTarget: (target) => !(target && typeof target.closest === "function" && target.closest("[data-okf-node], [data-okf-edge]")),
    onChange: options.onViewportChange,
    onRender: () => {
      const current = options.getViewport ? options.getViewport() : viewport;
      applyOKFViewport(container, current);
      if (options.onViewportRender) options.onViewportRender(current);
    },
    onZoom: options.onZoom
  });
  return layout;
}

export function updateOKFSelection(container, selectedID) {
  if (!container) return;
  container.querySelectorAll("[data-okf-node]").forEach((element) => {
    const selected = element.dataset.okfNode === selectedID;
    element.setAttribute("aria-pressed", String(selected));
    element.querySelector(".okf-node-shape")?.classList.toggle("selected", selected);
  });
}

export function updateOKFSemanticLinks(container, snapshot, selectedID, layout, viewport) {
  const edgeLayer = container && container.querySelector(".okf-edges");
  if (!edgeLayer) return;
  const nodes = snapshot && Array.isArray(snapshot.nodes) ? snapshot.nodes : [];
  const relationships = snapshot && Array.isArray(snapshot.relationships) ? snapshot.relationships : [];
  const positions = withManualPositions(layout && layout.positions, viewport && viewport.positions, nodes);
  const containment = relationships.filter((relationship) => relationship.kind !== "semantic_link");
  const selectedSemantic = selectedID
    ? relationships.filter((relationship) => relationship.kind === "semantic_link" && ((relationship.from_visible_id || relationship.from) === selectedID || (relationship.to_visible_id || relationship.to) === selectedID))
    : [];
  const edges = containment.concat(selectedSemantic).map((relationship) => {
    return edgeMarkup(relationship, positions, layout || {}, viewport);
  }).join("");
  edgeLayer.innerHTML = edges + geometryJunctionMarkup(activeGeometryJunctions(layout?.geometry, viewport?.positions));
}

function withManualPositions(base, manual, nodes) {
  const positions = Object.assign({}, base || {});
  nodes.forEach((node) => {
    const id = node.id || node.concept_id;
    if (positions[id]) positions[id] = { ...positions[id], shape: node.shape_definition || node.shape || node.presentation_style?.shape };
  });
  const ids = new Set(nodes.map((node) => node.id || node.concept_id));
  Object.keys(manual || {}).forEach((id) => {
    if (!ids.has(id) || !positions[id] || !Number.isFinite(manual[id]?.x) || !Number.isFinite(manual[id]?.y)) return;
    positions[id] = Object.assign({}, positions[id], { x: manual[id].x, y: manual[id].y });
  });
  return positions;
}

function edgeMarkup(relationship, positions, layout, viewport) {
  const fromID = relationship.from_visible_id || relationship.from;
  const toID = relationship.to_visible_id || relationship.to;
  const from = positions[fromID];
  const to = positions[toID];
  if (!from || !to) return "";
  const edgeClass = relationship.kind === "semantic_link" ? "semantic-link" : "containment";
  const routedRelationship = Object.assign({}, relationship, { from_visible_id: fromID, to_visible_id: toID });
  const geometry = edgeGeometry(routedRelationship, from, to, routeForEdge(relationship, layout, viewport, fromID, toID, from, to));
  if (!geometry) return "";
  const arrow = relationship.kind === "semantic_link" ? "" : " marker-end=\"url(#okf-arrow)\"";
  const hasManualEndpoint = viewport?.positions?.[fromID] || viewport?.positions?.[toID];
  const advanced = relationship.kind !== "semantic_link" && !hasManualEndpoint ? geometryEdge(layout?.geometry, relationship.id) : null;
  const labels = advanced?.labels?.length ? geometryLabelMarkup(advanced.labels) : "";
  return "<g class=\"okf-edge-group\"><path class=\"okf-edge-line " + edgeClass + "\" data-okf-edge=\"" + escapeOKF(relationship.id) + "\" d=\"" + geometry.path + "\"" + arrow + "><title>" + escapeOKF(relationship.accessible_label || relationship.kind || "relationship") + "</title></path>" + labels + "</g>";
}

function nodeMarkupFor(node, position, selectedID) {
  if (!position) return "";
  const id = node.id || node.concept_id;
  const label = String(node.label || node.title || id);
  const fields = Array.isArray(node.presentation_fields) ? node.presentation_fields.slice(0, 3) : [];
  const fieldText = fields.map((field) => String(field.label || field.source || "value") + ": " + String(field.value || "")).join("; ");
  const accessibleLabel = fieldText ? label + "; " + fieldText : label;
  const selected = id === selectedID;
  const style = node.presentation_style || {};
  const shape = shapeMarkup(node.shape_definition || node.shape || style.shape, position, selected, style);
  const content = shapeContentBox(node.shape_definition || node.shape || style.shape, { x: 0, y: 0, width: position.width, height: position.height });
  const availableWidth = Math.max(24, content.width - 28);
  const fieldMarkup = fields.map((field, index) => {
    const value = limitText((field.label || field.source || "value") + ": " + (field.value || ""), 48);
    return "<text class=\"okf-node-field\" x=\"14\" y=\"" + (49 + index * 17) + "\"" + textStyle(style) + fitTextAttributes(value, availableWidth, 11, false) + ">" + escapeOKF(value) + "</text>";
  }).join("");
  const labelValue = limitText(label, 48);
  return "<g class=\"okf-node\" data-okf-node=\"" + escapeOKF(id) + "\" data-okf-token=\"" + escapeOKF(node.presentation_token || "") + "\" data-okf-root=\"" + Boolean(node.is_root) + "\" data-okf-rollup=\"" + Boolean(node.is_rollup) + "\" transform=\"translate(" + position.x + " " + position.y + ")\" tabindex=\"0\" role=\"button\" aria-pressed=\"" + selected + "\" aria-label=\"" + escapeOKF(accessibleLabel) + "\"><title>" + escapeOKF(accessibleLabel) + "</title>" + shape + '<g transform="translate(' + content.x + ' ' + content.y + ')">' + "<text class=\"okf-node-label\" x=\"14\" y=\"30\"" + textStyle(style, true) + fitTextAttributes(labelValue, availableWidth, 14, true) + ">" + escapeOKF(labelValue) + "</text>" + fieldMarkup + "</g></g>";
}

function shapeMarkup(shape, position, selected, style) {
  const attributes = " class=\"okf-node-shape" + (selected ? " selected" : "") + "\"" + presentationStyle(style);
  return nodeShapeMarkup(shape, { x: 0, y: 0, width: position.width, height: position.height }, attributes);
}

function bindNodeInteractions(container, handlers, positions, relationships, layout) {
  let suppressClickUntil = 0;
  container.querySelectorAll("[data-okf-node]").forEach((element) => {
    const id = element.dataset.okfNode;
    element.addEventListener("click", (event) => {
      if (Date.now() < suppressClickUntil || event.detail >= 2) return;
      if (handlers.onSelect) handlers.onSelect(id);
    });
    element.addEventListener("dblclick", () => handlers.onFocus && handlers.onFocus(id));
    element.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        if (handlers.onSelect) handlers.onSelect(id);
      }
      if (event.key === "ArrowRight" && handlers.onFocus) handlers.onFocus(id);
    });
    element.addEventListener("pointerdown", (event) => {
      if (event.shiftKey) beginNodeDrag(event, id, handlers, positions, relationships, layout, container, () => { suppressClickUntil = Date.now() + 180; });
    });
  });
}

function beginNodeDrag(event, id, handlers, positions, relationships, layout, container, suppress) {
  if (event.button !== 0 || !positions[id]) return;
  const viewport = handlers.getViewport ? handlers.getViewport() : createViewportState();
  const start = { x: event.clientX, y: event.clientY, nodeX: positions[id].x, nodeY: positions[id].y };
  const baseScale = graphBaseScale(container.querySelector("svg"), layout);
  let moved = false;
  event.preventDefault();
  event.stopPropagation();
  const move = (pointerEvent) => {
    const scale = Math.max(0.01, baseScale * (viewport.zoom || 1));
    const x = start.nodeX + (pointerEvent.clientX - start.x) / scale;
    const y = start.nodeY + (pointerEvent.clientY - start.y) / scale;
    moved = moved || Math.abs(x - start.nodeX) + Math.abs(y - start.nodeY) > 3;
    positions[id] = Object.assign({}, positions[id], { x, y });
    if (!viewport.positions) viewport.positions = {};
    viewport.positions[id] = { x, y };
    if (handlers.onViewportChange) handlers.onViewportChange(viewport);
    renderPositionUpdates(container, positions, relationships, layout, viewport);
    if (handlers.onViewportRender) handlers.onViewportRender();
  };
  const stop = () => {
    document.removeEventListener("pointermove", move);
    document.removeEventListener("pointerup", stop);
    document.removeEventListener("pointercancel", stop);
    if (moved) suppress();
    if (handlers.onViewportChange) handlers.onViewportChange(viewport);
  };
  document.addEventListener("pointermove", move);
  document.addEventListener("pointerup", stop);
  document.addEventListener("pointercancel", stop);
}

function renderPositionUpdates(container, positions, relationships, layout, viewport) {
  container.querySelectorAll("[data-okf-node]").forEach((element) => {
    const position = positions[element.dataset.okfNode];
    if (position) element.setAttribute("transform", "translate(" + position.x + " " + position.y + ")");
  });
  relationships.forEach((relationship) => {
    const element = Array.from(container.querySelectorAll("[data-okf-edge]")).find((candidate) => candidate.dataset.okfEdge === relationship.id);
    const fromID = relationship.from_visible_id || relationship.from;
    const toID = relationship.to_visible_id || relationship.to;
    const from = positions[fromID];
    const to = positions[toID];
    if (!element || !from || !to) return;
    const routedRelationship = Object.assign({}, relationship, { from_visible_id: fromID, to_visible_id: toID });
    const geometry = edgeGeometry(routedRelationship, from, to, routeForEdge(relationship, layout, viewport, fromID, toID, from, to));
    if (geometry) element.setAttribute("d", geometry.path);
  });
}

function routeForEdge(relationship, layout, viewport, fromID, toID, from, to) {
  const hasManualEndpoint = viewport && viewport.positions && (viewport.positions[fromID] || viewport.positions[toID]);
  if (relationship.kind === "semantic_link") return straightRoute(from, to);
  if (hasManualEndpoint) return null;
  return layout && layout.edges && layout.edges[relationship.id];
}

function applyOKFViewport(container, viewport) {
  const content = container.querySelector(".okf-viewport-content");
  if (content) content.setAttribute("transform", viewportTransform(viewport));
}

function graphBaseScale(svg, layout) {
  if (!svg) return 1;
  return Math.max(0.01, Math.min(Math.max(1, svg.clientWidth || 1) / Math.max(1, layout.width), Math.max(1, svg.clientHeight || 1) / Math.max(1, layout.height)));
}

function fallbackLayout(nodes) {
  const columns = Math.max(1, Math.ceil(Math.sqrt(nodes.length || 1)));
  const rows = Math.ceil((nodes.length || 1) / columns);
  const rowHeights = Array.from({ length: rows }, () => 0);
  const columnWidths = Array.from({ length: columns }, () => 210);
  nodes.forEach((node, index) => { rowHeights[Math.floor(index / columns)] = Math.max(rowHeights[Math.floor(index / columns)], nodeDimensions(node).height); });
  nodes.forEach((node, index) => { columnWidths[index % columns] = Math.max(columnWidths[index % columns], nodeDimensions(node).width); });
  const columnOffsets = [];
  let horizontalOffset = 36;
  columnWidths.forEach((width) => { columnOffsets.push(horizontalOffset); horizontalOffset += width + 60; });
  const rowOffsets = [];
  let offset = 36;
  rowHeights.forEach((height) => { rowOffsets.push(offset); offset += height + 42; });
  const positions = {};
  nodes.forEach((node, index) => {
    const id = node.id || node.concept_id;
    const dimensions = nodeDimensions(node);
    positions[id] = { x: columnOffsets[index % columns], y: rowOffsets[Math.floor(index / columns)], width: dimensions.width, height: dimensions.height };
  });
  return { width: Math.max(760, horizontalOffset), height: Math.max(430, offset), positions };
}

function presentationStyle(style) {
  const declarations = [];
  const fill = safeSVGColor(style.fill);
  const stroke = safeSVGColor(style.stroke);
  const width = safeStrokeWidth(style.stroke_width);
  const dash = safeDashArray(style.stroke_dasharray);
  if (fill) declarations.push("fill:" + fill);
  if (stroke) declarations.push("stroke:" + stroke);
  if (width) declarations.push("stroke-width:" + width);
  if (dash) declarations.push("stroke-dasharray:" + dash);
  return declarations.length ? " style=\"" + escapeOKF(declarations.join(";")) + "\"" : "";
}

function textStyle(style, label) {
  const declarations = [];
  const color = safeSVGColor(style.text);
  const weight = label ? safeFontWeight(style.emphasis) || "800" : "";
  if (color) declarations.push("fill:" + color);
  if (weight) declarations.push("font-weight:" + weight);
  return declarations.length ? " style=\"" + escapeOKF(declarations.join(";")) + "\"" : "";
}

function fitTextAttributes(value, availableWidth, fontSize, bold) {
  const estimatedWidth = String(value || "").length * fontSize * (bold ? 0.62 : 0.55);
  if (estimatedWidth <= availableWidth) return "";
  return " textLength=\"" + availableWidth + "\" lengthAdjust=\"spacingAndGlyphs\"";
}

function safeFontWeight(value) {
  switch (String(value || "").trim().toLowerCase()) {
    case "normal": return "400";
    case "medium": return "600";
    case "bold":
    case "strong":
    case "high": return "800";
    default: return "";
  }
}

function safeStrokeWidth(value) {
  const number = Number(value);
  return Number.isFinite(number) && number > 0 && number <= 24 ? String(number) : "";
}

function safeDashArray(value) {
  const text = String(value || "").trim();
  return /^[0-9 .,-]+$/.test(text) && text ? text : "";
}

function safeSVGColor(value) {
  const text = String(value || "").trim();
  if (/^#[0-9a-f]{3,8}$/i.test(text)) return text;
  if (/^(?:rgb|rgba|hsl|hsla)\([\d\s.,%+\-]+\)$/i.test(text)) return text;
  if (/^[a-z]+$/i.test(text)) return text;
  return "";
}

function limitText(value, length) {
  const text = String(value || "");
  return text.length > length ? text.slice(0, length - 1) + "…" : text;
}
