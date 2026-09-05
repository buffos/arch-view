import { addFeatureDiagnostic, failedFeature, featureGeometry } from "./feature_context.js";

const FEATURE_ID = "ports";
const PORT_SIZE = 10;
const LABEL_HEIGHT = 12;

export function presentationPortID(nodeID, role) {
  return "port::" + nodeID + "::" + role;
}

function labelID(nodeID, role) {
  return "port-label::" + nodeID + "::" + role;
}

function flowSides(profile) {
  const direction = String(profile?.options?.["org.eclipse.elk.direction"] || "RIGHT").toUpperCase();
  const sides = {
    RIGHT: { in: "WEST", out: "EAST" }, LEFT: { in: "EAST", out: "WEST" },
    DOWN: { in: "NORTH", out: "SOUTH" }, UP: { in: "SOUTH", out: "NORTH" }
  };
  return sides[direction] || sides.RIGHT;
}

function portInput(nodeID, role, side) {
  const text = role;
  return {
    id: presentationPortID(nodeID, role), width: PORT_SIZE, height: PORT_SIZE,
    layoutOptions: { "org.eclipse.elk.port.side": side },
    labels: [{ id: labelID(nodeID, role), text, width: Math.max(14, text.length * 7), height: LABEL_HEIGHT }]
  };
}

function prepare(context) {
  const sides = flowSides(context.profile);
  context.graph.children = (context.graph.children || []).map((node) => ({
    ...node,
    layoutOptions: { ...(node.layoutOptions || {}), "org.eclipse.elk.portConstraints": "FIXED_SIDE" },
    ports: [portInput(node.id, "in", sides.in), portInput(node.id, "out", sides.out)]
  }));
  const relationships = new Map((context.scene?.visible_relationships || []).map((item) => [item.id, item]));
  context.graph.edges = (context.graph.edges || []).map((edge) => {
    const relationship = relationships.get(edge.id);
    if (!relationship) return edge;
    return {
      ...edge,
      sources: [presentationPortID(relationship.from_visible_id, "out")],
      targets: [presentationPortID(relationship.to_visible_id, "in")]
    };
  });
  return context;
}

function finiteBounds(value) {
  return value && [value.x, value.y, value.width, value.height].every(Number.isFinite)
    && value.width > 0 && value.height > 0;
}

function normalizePort(node, port, role, side) {
  const bounds = { x: node.x + port.x, y: node.y + port.y, width: port.width, height: port.height };
  const actualSide = port.layoutOptions?.["org.eclipse.elk.port.side"] || port.layoutOptions?.["elk.port.side"];
  const label = (port.labels || []).find((item) => item.id === labelID(node.id, role));
  const labelBounds = label && { x: bounds.x + label.x, y: bounds.y + label.y, width: label.width, height: label.height };
  if (!finiteBounds(bounds) || actualSide !== side || !finiteBounds(labelBounds) || label.text !== role) return null;
  const center = { x: bounds.x + bounds.width / 2, y: bounds.y + bounds.height / 2 };
  const position = {
    x: side === "WEST" ? bounds.x : side === "EAST" ? bounds.x + bounds.width : center.x,
    y: side === "NORTH" ? bounds.y : side === "SOUTH" ? bounds.y + bounds.height : center.y
  };
  return {
    id: presentationPortID(node.id, role), node_id: node.id, role, side,
    position,
    bounds, label: role, label_bounds: labelBounds
  };
}

function normalize(context) {
  const store = featureGeometry(context);
  const sides = flowSides(context.profile);
  const validPorts = new Set();
  for (const node of context.output?.children || []) {
    const ports = [];
    for (const role of ["in", "out"]) {
      const id = presentationPortID(node.id, role);
      const port = (node.ports || []).find((item) => item.id === id);
      const normalized = port && normalizePort(node, port, role, sides[role]);
      if (!normalized) {
        addFeatureDiagnostic(context, "geometry_port_invalid",
          "ELK returned invalid presentation port geometry for node " + node.id + "; affected edges use ordinary geometry.",
          { feature: FEATURE_ID, node_id: node.id, port_id: id });
        continue;
      }
      ports.push(normalized);
      validPorts.add(id);
    }
    store.portsByNode[node.id] = ports;
  }
  for (const edge of context.output?.edges || []) {
    const source = edge.sources?.[0];
    const target = edge.targets?.[0];
    if (!validPorts.has(source) || !validPorts.has(target)) {
      if (!store.invalidRouteIDs.includes(edge.id)) store.invalidRouteIDs.push(edge.id);
      continue;
    }
    store.portEndpointsByEdge[edge.id] = { source_port_id: source, target_port_id: target };
  }
  return context;
}

export const portFeature = {
  id: FEATURE_ID,
  prepare,
  normalize,
  validate: (context) => context,
  fallback: (context, error, phase) => failedFeature(context, FEATURE_ID, error, phase)
};

export { finiteBounds, flowSides };
