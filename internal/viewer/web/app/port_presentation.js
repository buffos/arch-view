import { orthogonalRouteBetweenPoints } from "../graph_route.js";
import { escapeHTML } from "./utils.js";

function translatedBounds(bounds, dx, dy) {
  return { ...bounds, x: bounds.x + dx, y: bounds.y + dy };
}

export function geometryPortsForNode(snapshot, nodeID, currentBounds) {
  const node = snapshot?.nodes?.find((item) => item.id === nodeID);
  if (!node || !currentBounds) return [];
  const dx = currentBounds.x - node.bounds.x;
  const dy = currentBounds.y - node.bounds.y;
  return (node.ports || []).map((port) => ({
    ...port,
    position: { x: port.position.x + dx, y: port.position.y + dy },
    bounds: translatedBounds(port.bounds, dx, dy),
    label_bounds: translatedBounds(port.label_bounds, dx, dy)
  }));
}

export function geometryPortMarkup(ports, origin = { x: 0, y: 0 }) {
  return (ports || []).map((port) => {
    const bounds = translatedBounds(port.bounds, -origin.x, -origin.y);
    const label = translatedBounds(port.label_bounds, -origin.x, -origin.y);
    if (![bounds.x, bounds.y, bounds.width, bounds.height, label.x, label.y].every(Number.isFinite)) return "";
    return '<g class="presentation-port-group" aria-hidden="true"><rect class="presentation-port '
      + escapeHTML(port.role) + '" x="' + bounds.x + '" y="' + bounds.y + '" width="' + bounds.width
      + '" height="' + bounds.height + '" rx="2"></rect><text class="presentation-port-label" x="'
      + (label.x + label.width / 2) + '" y="' + (label.y + label.height * 0.8)
      + '" text-anchor="middle">' + escapeHTML(port.label) + "</text></g>";
  }).join("");
}

export function geometryPortRoute(snapshot, geometryEdge, positions, preferredRoute) {
  if (!geometryEdge?.source_port_id || !geometryEdge?.target_port_id) return null;
  const sourcePorts = geometryPortsForNode(snapshot, geometryEdge.source_node_id, positions[geometryEdge.source_node_id]);
  const targetPorts = geometryPortsForNode(snapshot, geometryEdge.target_node_id, positions[geometryEdge.target_node_id]);
  const source = sourcePorts.find((port) => port.id === geometryEdge.source_port_id);
  const target = targetPorts.find((port) => port.id === geometryEdge.target_port_id);
  if (!source || !target) return null;
  const sourceNode = snapshot.nodes.find((node) => node.id === geometryEdge.source_node_id);
  const targetNode = snapshot.nodes.find((node) => node.id === geometryEdge.target_node_id);
  if (!sourceNode || !targetNode || !positions[sourceNode.id] || !positions[targetNode.id]) return null;
  const moved = positions[sourceNode.id].x !== sourceNode.bounds.x || positions[sourceNode.id].y !== sourceNode.bounds.y
    || positions[targetNode.id].x !== targetNode.bounds.x || positions[targetNode.id].y !== targetNode.bounds.y;
  return { route: !moved && preferredRoute ? preferredRoute : orthogonalRouteBetweenPoints(source.position, target.position), preserveEndpoints: true };
}
