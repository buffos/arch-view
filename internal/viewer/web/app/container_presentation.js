import { escapeHTML } from "./utils.js";

export function geometryNode(snapshot, nodeID) {
  return snapshot?.nodes?.find((node) => node.id === nodeID) || null;
}

export function geometryContainerNode(snapshot, nodeID) {
  const node = geometryNode(snapshot, nodeID);
  return node && node.children_ids?.length ? node : null;
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

export function geometryContainerHeaderBounds(snapshot, nodeID, currentBounds, preferredSize) {
  if (!geometryContainerNode(snapshot, nodeID) || !currentBounds) return currentBounds;
  const width = Math.min(Math.max(1, preferredSize?.width || 190), Math.max(1, currentBounds.width - 24));
  const height = Math.min(Math.max(1, preferredSize?.height || 82), Math.max(1, currentBounds.height - 20));
  return { x: currentBounds.x + 12, y: currentBounds.y + 10, width, height };
}

export function geometryContainerMarkup(snapshot, positions, attributesForNode = () => "") {
  return (snapshot?.nodes || []).filter((node) => node.children_ids?.length).map((node) => {
    const bounds = positions?.[node.id] || node.bounds;
    if (!bounds || ![bounds.x, bounds.y, bounds.width, bounds.height].every(Number.isFinite)) return "";
    return '<rect class="geometry-container-frame" data-geometry-container="' + escapeHTML(node.id)
      + '" aria-hidden="true" x="' + bounds.x + '" y="' + bounds.y
      + '" width="' + bounds.width + '" height="' + bounds.height + '" rx="14"'
      + attributesForNode(node.id) + "></rect>";
  }).join("");
}
