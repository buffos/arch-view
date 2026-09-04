import { fromELKSplineSections } from "../graph_route.js";
import { addFeatureDiagnostic, failedFeature, featureGeometry } from "./feature_context.js";

const FEATURE_ID = "spline_refinement";

function normalize(context) {
  const store = featureGeometry(context);
  for (const edge of context.output?.edges || []) {
    if (fromELKSplineSections(edge.sections, 0)) continue;
    store.invalidRouteIDs.push(edge.id);
    addFeatureDiagnostic(context, "geometry_route_invalid",
      "ELK returned an invalid spline for relationship " + edge.id + "; deterministic orthogonal geometry is used.",
      { feature: FEATURE_ID, relationship_id: edge.id });
  }
  return context;
}

export const splineFeature = {
  id: FEATURE_ID,
  normalize,
  validate: (context) => context,
  fallback: (context, error, phase) => failedFeature(context, FEATURE_ID, error, phase)
};
