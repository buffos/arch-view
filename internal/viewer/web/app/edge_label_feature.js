import { addFeatureDiagnostic, failedFeature, featureGeometry } from "./feature_context.js";

const FEATURE_ID = "edge_labels";

function countText(relationship) {
  const count = relationship?.count;
  return Number.isInteger(count) && count > 0 ? String(count) : "";
}

function labelSize(text) {
  return { width: Math.max(18, text.length * 9 + 10), height: 18 };
}

function prepare(context) {
  const relationships = new Map((context.scene?.visible_relationships || []).map((item) => [item.id, item]));
  const placement = context.profile?.options?.["org.eclipse.elk.edgeLabels.placement"];
  context.graph.edges = (context.graph.edges || []).map((edge) => {
    const text = countText(relationships.get(edge.id));
    if (!text) return edge;
    const size = labelSize(text);
    const label = { id: "label::" + edge.id + "::count", text, width: size.width, height: size.height };
    if (placement) label.layoutOptions = { "org.eclipse.elk.edgeLabels.placement": placement };
    return { ...edge, labels: [label] };
  });
  return context;
}

function validBounds(label) {
  return label && [label.x, label.y, label.width, label.height].every(Number.isFinite)
    && label.width >= 0 && label.height >= 0;
}

function normalize(context) {
  const store = featureGeometry(context);
  const relationships = new Map((context.scene?.visible_relationships || []).map((item) => [item.id, item]));
  for (const edge of context.output?.edges || []) {
    const text = countText(relationships.get(edge.id));
    if (!text) continue;
    const expectedID = "label::" + edge.id + "::count";
    const label = (edge.labels || []).find((item) => item.id === expectedID);
    if (!validBounds(label) || label.text !== text) {
      addFeatureDiagnostic(context, "geometry_label_invalid",
        "ELK did not return valid bounds for relationship " + edge.id + "; its existing route label is retained.",
        { feature: FEATURE_ID, relationship_id: edge.id });
      continue;
    }
    store.labelsByEdge[edge.id] = [{
      id: expectedID,
      text,
      kind: "count",
      bounds: { x: label.x, y: label.y, width: label.width, height: label.height },
      position: { x: label.x + label.width / 2, y: label.y + label.height * 0.75 },
      visible: true
    }];
  }
  return context;
}

export const edgeLabelFeature = {
  id: FEATURE_ID,
  prepare,
  normalize,
  validate: (context) => context,
  fallback: (context, error, phase) => failedFeature(context, FEATURE_ID, error, phase)
};

export { countText, labelSize, validBounds };
