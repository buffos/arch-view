import { escapeHTML } from "./utils.js";

export function geometryLabelMarkup(labels, className = "") {
  return (labels || []).filter((label) => label.visible !== false).map((label) => {
    const position = label.position || {};
    if (!Number.isFinite(position.x) || !Number.isFinite(position.y)) return "";
    return '<text class="edge-label geometry-edge-label ' + escapeHTML(className) + '" data-edge-label-id="'
      + escapeHTML(label.id) + '" x="' + position.x + '" y="' + position.y
      + '" text-anchor="middle" aria-hidden="true">' + escapeHTML(label.text) + "</text>";
  }).join("");
}

export function activeGeometryJunctions(snapshot, manualPositions = {}) {
  const moved = new Set(Object.keys(manualPositions || {}));
  const edges = new Map((snapshot?.edges || []).map((edge) => [edge.id, edge]));
  return (snapshot?.junctions || []).filter((junction) => {
    return (junction.incident_geometry_edge_ids || []).every((edgeID) => {
      const edge = edges.get(edgeID);
      return edge && !moved.has(edge.source_node_id) && !moved.has(edge.target_node_id);
    });
  });
}

export function geometryJunctionMarkup(junctions) {
  return (junctions || []).map((junction) => {
    const point = junction.position || {};
    if (!Number.isFinite(point.x) || !Number.isFinite(point.y)) return "";
    return '<circle class="edge-junction" data-junction-id="' + escapeHTML(junction.id)
      + '" cx="' + point.x + '" cy="' + point.y + '" r="4" aria-hidden="true"></circle>';
  }).join("");
}
