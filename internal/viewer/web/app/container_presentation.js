import { escapeHTML } from "./utils.js";

export function geometryNode(snapshot, nodeID) {
  return snapshot?.nodes?.find((node) => node.id === nodeID) || null;
}

export function geometryContainerNode(snapshot, nodeID) {
  const node = geometryNode(snapshot, nodeID);
  return node && node.children_ids?.length ? node : null;
}

export function geometryContainer(snapshot, nodeID) {
  return snapshot?.containers?.find((container) => container.semantic_node_id === nodeID) || null;
}

export function geometryMoveIDs(snapshot, nodeID) {
  const byID = new Map((snapshot?.nodes || []).map((node) => [node.id, node]));
  const result = [];
  const visit = (id) => {
    if (!byID.has(id) || result.includes(id)) return;
    result.push(id);
    for (const child of byID.get(id).children_ids || []) visit(child);
  };
  visit(nodeID);
  return result;
}

export function geometryMoveStart(snapshot, nodeID, positions) {
  const requested = geometryMoveIDs(snapshot, nodeID);
  const ids = (requested.length ? requested : [nodeID]).filter((id) => positions?.[id]);
  return { ids, positions: Object.fromEntries(ids.map((id) => [id, { x: positions[id].x, y: positions[id].y }])) };
}

export function applyGeometryMove(move, positions, manualPositions, dx, dy) {
  for (const id of move?.ids || []) {
    const start = move.positions[id];
    const next = { x: start.x + dx, y: start.y + dy };
    positions[id] = { ...positions[id], ...next };
    manualPositions[id] = next;
  }
}

function includeBox(bounds, box, padding = 12) {
  if (!box) return bounds;
  return {
    x: Math.min(bounds.x, box.x - padding),
    y: Math.min(bounds.y, box.y - padding),
    width: Math.max(bounds.x + bounds.width, box.x + box.width + padding) - Math.min(bounds.x, box.x - padding),
    height: Math.max(bounds.y + bounds.height, box.y + box.height + padding) - Math.min(bounds.y, box.y - padding)
  };
}

export function geometryContainerBounds(snapshot, positions) {
  const result = {};
  const calculate = (nodeID) => {
    if (result[nodeID]) return result[nodeID];
    const container = geometryContainer(snapshot, nodeID);
    const node = geometryNode(snapshot, nodeID);
    if (!container || !node) return null;
    const current = positions?.[nodeID] || node.bounds;
    const dx = current.x - node.bounds.x;
    const dy = current.y - node.bounds.y;
    let bounds = { ...container.bounds, x: container.bounds.x + dx, y: container.bounds.y + dy };
    for (const childID of node.children_ids || []) {
      const child = geometryContainer(snapshot, childID)
        ? calculate(childID)
        : positions?.[childID] || geometryNode(snapshot, childID)?.bounds;
      bounds = includeBox(bounds, child);
    }
    result[nodeID] = bounds;
    return bounds;
  };
  for (const container of snapshot?.containers || []) calculate(container.semantic_node_id);
  return result;
}

export function geometryContainerMarkup(snapshot, positions, attributesForNode = () => "") {
  const calculated = geometryContainerBounds(snapshot, positions);
  return (snapshot?.containers || []).map((container) => {
    const nodeID = container.semantic_node_id;
    const bounds = calculated[nodeID];
    if (!bounds || ![bounds.x, bounds.y, bounds.width, bounds.height].every(Number.isFinite)) return "";
    return '<rect class="geometry-container-frame" data-geometry-container="' + escapeHTML(nodeID)
      + '" aria-hidden="true" x="' + bounds.x + '" y="' + bounds.y
      + '" width="' + bounds.width + '" height="' + bounds.height + '" rx="14"'
      + attributesForNode(nodeID) + "></rect>";
  }).join("");
}
