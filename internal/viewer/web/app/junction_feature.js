import { addFeatureDiagnostic, failedFeature, featureGeometry } from "./feature_context.js";

const FEATURE_ID = "junctions";
const TOLERANCE = 0.75;

function finitePoint(point) {
  return point && Number.isFinite(point.x) && Number.isFinite(point.y);
}

function distanceToSegment(point, start, end) {
  const dx = end.x - start.x;
  const dy = end.y - start.y;
  if (dx === 0 && dy === 0) return Math.hypot(point.x - start.x, point.y - start.y);
  const ratio = Math.max(0, Math.min(1, ((point.x - start.x) * dx + (point.y - start.y) * dy) / (dx * dx + dy * dy)));
  return Math.hypot(point.x - (start.x + ratio * dx), point.y - (start.y + ratio * dy));
}

function routeContainsPoint(edge, point) {
  for (const section of edge.sections || []) {
    const points = [section.startPoint, ...(section.bendPoints || []), section.endPoint];
    if (!points.every(finitePoint)) continue;
    for (let index = 1; index < points.length; index += 1) {
      if (distanceToSegment(point, points[index - 1], points[index]) <= TOLERANCE) return true;
    }
  }
  return false;
}

function pointKey(point) {
  return Math.round(point.x * 1000) + ":" + Math.round(point.y * 1000);
}

function prepare(context) {
  context.graph.layoutOptions = { ...(context.graph.layoutOptions || {}), "elk.layered.mergeEdges": "true" };
  return context;
}

function normalize(context) {
  const edges = context.output?.edges || [];
  const candidates = new Map();
  for (const edge of edges) {
    for (const point of edge.junctionPoints || []) {
      if (!finitePoint(point)) {
        addFeatureDiagnostic(context, "geometry_junction_invalid",
          "ELK returned a non-finite junction point; the marker was omitted.",
          { feature: FEATURE_ID, relationship_id: edge.id });
        continue;
      }
      candidates.set(pointKey(point), { x: point.x, y: point.y });
    }
  }
  const store = featureGeometry(context);
  for (const point of [...candidates.values()].sort((left, right) => left.x - right.x || left.y - right.y)) {
    const incident = edges.filter((edge) => routeContainsPoint(edge, point)).map((edge) => edge.id).sort();
    if (incident.length < 2) {
      addFeatureDiagnostic(context, "geometry_junction_unshared",
        "ELK returned a junction that is not shared by two visible routes; the marker was omitted.",
        { feature: FEATURE_ID, position: point });
      continue;
    }
    const id = "junction::" + pointKey(point);
    store.junctions.push({ id, position: point, incident_geometry_edge_ids: incident });
    incident.forEach((edgeID) => {
      store.junctionIDsByEdge[edgeID] = [...(store.junctionIDsByEdge[edgeID] || []), id];
    });
  }
  return context;
}

export const junctionFeature = {
  id: FEATURE_ID,
  prepare,
  normalize,
  validate: (context) => context,
  fallback: (context, error, phase) => failedFeature(context, FEATURE_ID, error, phase)
};

export { distanceToSegment, routeContainsPoint };
