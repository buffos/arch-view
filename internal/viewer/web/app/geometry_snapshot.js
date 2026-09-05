import { fromELKSections, fromELKSplineSections, orthogonalRoute, orthogonalRouteBetweenPoints, selfLoopRoute } from "../graph_route.js";

function finite(value, fallback = 0) {
  return Number.isFinite(value) ? value : fallback;
}

function nodePositions(output, offset) {
  const positions = {};
  for (const node of output?.children || []) {
    positions[node.id] = {
      x: finite(node.x) + offset,
      y: finite(node.y) + offset,
      width: finite(node.width, 190) || 190,
      height: finite(node.height, 82) || 82
    };
  }
  return positions;
}

function routeFor(edge, relationship, positions, profile, featureGeometry, offset) {
  const from = positions[relationship.from_visible_id];
  const to = positions[relationship.to_visible_id];
  if (!from || !to) return null;
  const spline = String(profile?.options?.["org.eclipse.elk.edgeRouting"] || profile?.options?.["elk.edgeRouting"] || "ORTHOGONAL").toUpperCase() === "SPLINES";
  const invalid = (featureGeometry.invalidRouteIDs || []).includes(edge?.id);
  if (!invalid && edge) {
    const route = spline ? fromELKSplineSections(edge.sections, offset) : fromELKSections(edge.sections, offset);
    if (route) return route;
  }
  const endpoints = featureGeometry.portEndpointsByEdge?.[relationship.id];
  if (endpoints) {
    const source = (featureGeometry.portsByNode?.[relationship.from_visible_id] || []).find((port) => port.id === endpoints.source_port_id);
    const target = (featureGeometry.portsByNode?.[relationship.to_visible_id] || []).find((port) => port.id === endpoints.target_port_id);
    if (source && target) {
      return orthogonalRouteBetweenPoints(
        { x: source.position.x + offset, y: source.position.y + offset },
        { x: target.position.x + offset, y: target.position.y + offset }
      );
    }
  }
  if (relationship.from_visible_id === relationship.to_visible_id) return selfLoopRoute(from);
  return orthogonalRoute(from, to);
}

function labelsFor(edgeID, featureGeometry, offset) {
  return (featureGeometry.labelsByEdge?.[edgeID] || []).map((label) => ({
    ...label,
    bounds: { ...label.bounds, x: label.bounds.x + offset, y: label.bounds.y + offset },
    position: { x: label.position.x + offset, y: label.position.y + offset }
  }));
}

function junctionsFor(featureGeometry, offset) {
  return (featureGeometry.junctions || []).map((junction) => ({
    ...junction,
    position: { x: junction.position.x + offset, y: junction.position.y + offset },
    incident_geometry_edge_ids: [...junction.incident_geometry_edge_ids]
  }));
}

function portsFor(nodeID, featureGeometry, offset) {
  return (featureGeometry.portsByNode?.[nodeID] || []).map((port) => ({
    ...port,
    position: { x: port.position.x + offset, y: port.position.y + offset },
    bounds: { ...port.bounds, x: port.bounds.x + offset, y: port.bounds.y + offset },
    label_bounds: { ...port.label_bounds, x: port.label_bounds.x + offset, y: port.label_bounds.y + offset }
  }));
}

export function normalizeGeometrySnapshot(scene, output, profile, source, offset = 24) {
  const positions = nodePositions(output, offset);
  const outputEdges = new Map((output?.edges || []).map((edge) => [edge.id, edge]));
  const featureGeometry = output?.featureGeometry || { labelsByEdge: {}, junctions: [], junctionIDsByEdge: {}, invalidRouteIDs: [], portsByNode: {}, portEndpointsByEdge: {}, parentByNode: {}, childrenByNode: {} };
  const geometryEdges = [];
  const routes = {};
  for (const relationship of scene?.visible_relationships || []) {
    if (!positions[relationship.from_visible_id] || !positions[relationship.to_visible_id]) continue;
    const route = routeFor(outputEdges.get(relationship.id), relationship, positions, profile, featureGeometry, offset);
    if (!route) continue;
    routes[relationship.id] = route;
    const endpoints = featureGeometry.portEndpointsByEdge?.[relationship.id] || {};
    geometryEdges.push({
      id: relationship.id,
      semantic_relationship_id: relationship.id,
      source_node_id: relationship.from_visible_id,
      target_node_id: relationship.to_visible_id,
      ...(endpoints.source_port_id ? { source_port_id: endpoints.source_port_id } : {}),
      ...(endpoints.target_port_id ? { target_port_id: endpoints.target_port_id } : {}),
      labels: labelsFor(relationship.id, featureGeometry, offset),
      junctions: [...(featureGeometry.junctionIDsByEdge?.[relationship.id] || [])],
      route
    });
  }
  const nodes = (scene?.visible_nodes || []).filter((node) => positions[node.id]).map((node) => ({
    id: node.id,
    semantic_node_id: node.id,
    bounds: { ...positions[node.id] },
    parent_id: featureGeometry.parentByNode?.[node.id] || null,
    children_ids: [...(featureGeometry.childrenByNode?.[node.id] || [])],
    ports: portsFor(node.id, featureGeometry, offset)
  }));
  return {
    schema_version: "arch-view.geometry/v1",
    source,
    nodes,
    edges: geometryEdges,
    junctions: junctionsFor(featureGeometry, offset),
    diagnostics: [...(output?.featureDiagnostics || [])],
    provenance: {
      engine: "elkjs",
      algorithm: profile?.algorithm || "layered",
      features: [...(output?.effectiveFeatures || [])]
    },
    positions,
    routes
  };
}

export function geometryEdge(snapshot, relationshipID) {
  return snapshot?.edges?.find((edge) => edge.id === relationshipID) || null;
}
