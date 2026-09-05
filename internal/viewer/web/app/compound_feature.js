import { addFeatureDiagnostic, failedFeature, featureGeometry } from "./feature_context.js";
import { nestELKNodes, presentationContainerID, visibleHierarchy } from "./compound_hierarchy.js";

const FEATURE_ID = "compound";
const BOUNDS_TOLERANCE = 0.75;

function finiteBounds(node) {
  return node && [node.x, node.y, node.width, node.height].every(Number.isFinite)
    && node.width > 0 && node.height > 0;
}

function translatedPoint(point, origin) {
  return point && { x: point.x + origin.x, y: point.y + origin.y };
}

function translatedEdge(edge, origin) {
  const translateSection = (section) => ({
    ...section,
    startPoint: translatedPoint(section.startPoint, origin),
    endPoint: translatedPoint(section.endPoint, origin),
    bendPoints: (section.bendPoints || []).map((point) => translatedPoint(point, origin))
  });
  return {
    ...edge,
    sections: (edge.sections || []).map(translateSection),
    labels: (edge.labels || []).map((label) => ({ ...label, x: label.x + origin.x, y: label.y + origin.y })),
    junctionPoints: (edge.junctionPoints || []).map((point) => translatedPoint(point, origin))
  };
}

function flattenOutput(output, hierarchy) {
  const flat = [];
  const origins = new Map([[output.id, { x: 0, y: 0 }]]);
  const actualParents = {};
  const containerBoundsByNode = {};
  const containerOwners = new Map(hierarchy.containerNodeIDs.map((id) => [presentationContainerID(id), id]));
  let valid = true;
  const visit = (nodes, parentID, parentOrigin, parentBounds) => {
    for (const node of nodes || []) {
      if (!finiteBounds(node)) { valid = false; continue; }
      const absolute = { ...node, x: parentOrigin.x + node.x, y: parentOrigin.y + node.y };
      if (parentBounds && (node.x < -BOUNDS_TOLERANCE || node.y < -BOUNDS_TOLERANCE
        || node.x + node.width > parentBounds.width + BOUNDS_TOLERANCE
        || node.y + node.height > parentBounds.height + BOUNDS_TOLERANCE)) valid = false;
      if (origins.has(node.id)) valid = false;
      origins.set(node.id, { x: absolute.x, y: absolute.y });
      actualParents[node.id] = parentID;
      const ownerID = containerOwners.get(node.id);
      if (ownerID) containerBoundsByNode[ownerID] = {
        x: absolute.x, y: absolute.y, width: absolute.width, height: absolute.height
      };
      else flat.push({ ...absolute, children: undefined });
      visit(node.children, node.id, { x: absolute.x, y: absolute.y }, node);
    }
  };
  visit(output.children, null, { x: 0, y: 0 }, null);
  const semanticIDs = Object.keys(hierarchy.parentByNode).sort();
  const expectedIDs = [...semanticIDs, ...containerOwners.keys()].sort();
  const actualIDs = Object.keys(actualParents).sort();
  if (JSON.stringify(expectedIDs) !== JSON.stringify(actualIDs)) valid = false;
  for (const id of semanticIDs) {
    const semanticParent = hierarchy.parentByNode[id];
    const expectedParent = containerOwners.has(presentationContainerID(id))
      ? presentationContainerID(id)
      : semanticParent && containerOwners.has(presentationContainerID(semanticParent))
        ? presentationContainerID(semanticParent) : null;
    if ((actualParents[id] || null) !== expectedParent) valid = false;
  }
  for (const ownerID of hierarchy.containerNodeIDs) {
    const semanticParent = hierarchy.parentByNode[ownerID];
    const expectedParent = semanticParent && containerOwners.has(presentationContainerID(semanticParent))
      ? presentationContainerID(semanticParent) : null;
    if ((actualParents[presentationContainerID(ownerID)] || null) !== expectedParent) valid = false;
  }
  const edges = [];
  const collectEdges = (owner, ownerOrigin) => {
    for (const edge of owner.edges || []) {
      const origin = origins.get(edge.container) || ownerOrigin;
      edges.push(translatedEdge(edge, origin));
    }
    for (const child of owner.children || []) collectEdges(child, origins.get(child.id) || ownerOrigin);
  };
  collectEdges(output, { x: 0, y: 0 });
  return { valid, nodes: flat, edges, containerBoundsByNode };
}

function deterministicFlatOutput(context) {
  const nodes = context.compoundInput.flatNodes;
  const columns = Math.max(1, Math.ceil(Math.sqrt(nodes.length || 1)));
  const placed = nodes.map((node, index) => ({
    ...node,
    children: undefined,
    x: 30 + (index % columns) * 240,
    y: 30 + Math.floor(index / columns) * 120
  }));
  return {
    ...context.output,
    width: Math.max(760, columns * 240 + 60),
    height: Math.max(430, Math.ceil(nodes.length / columns) * 120 + 60),
    children: placed,
    edges: (context.graph.edges || []).map((edge) => ({ ...edge, sections: [], labels: [], junctionPoints: [] }))
  };
}

function prepare(context) {
  const hierarchy = visibleHierarchy(context.scene?.visible_nodes || []);
  context.compoundInput = { hierarchy, flatNodes: (context.graph.children || []).map((node) => ({ ...node })) };
  if (!hierarchy.containerNodeIDs.length) return context;
  const nested = nestELKNodes(context.graph.children, hierarchy, context.graph.layoutOptions);
  context.graph = {
    ...context.graph,
    layoutOptions: { ...(context.graph.layoutOptions || {}), "elk.hierarchyHandling": "INCLUDE_CHILDREN" },
    children: nested
  };
  return context;
}

function normalize(context) {
  const hierarchy = context.compoundInput?.hierarchy;
  if (!hierarchy?.containerNodeIDs.length) return context;
  const flattened = flattenOutput(context.output, hierarchy);
  if (!flattened.valid) {
    context.output = deterministicFlatOutput(context);
    addFeatureDiagnostic(context, "geometry_hierarchy_invalid",
      "ELK returned invalid compound hierarchy; deterministic flat geometry is used.", { feature: FEATURE_ID });
    return context;
  }
  context.output = { ...context.output, children: flattened.nodes, edges: flattened.edges };
  const store = featureGeometry(context);
  store.parentByNode = { ...hierarchy.parentByNode };
  store.childrenByNode = Object.fromEntries(Object.entries(hierarchy.childrenByNode).map(([id, children]) => [id, [...children]]));
  store.containerNodeIDs = [...hierarchy.containerNodeIDs];
  store.containerBoundsByNode = Object.fromEntries(Object.entries(flattened.containerBoundsByNode).map(([id, bounds]) => [id, { ...bounds }]));
  return context;
}

function fallback(context, error, phase) {
  addFeatureDiagnostic(context, "geometry_hierarchy_invalid",
    "Compound hierarchy failed during " + phase + "; ordinary flat geometry is used.",
    { feature: FEATURE_ID, phase, cause: error?.message || String(error) });
  return failedFeature(context, FEATURE_ID, error, phase);
}

export const compoundFeature = { id: FEATURE_ID, prepare, normalize, validate: (context) => context, fallback };

export { finiteBounds, flattenOutput, translatedEdge };
