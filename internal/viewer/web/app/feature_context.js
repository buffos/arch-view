export function featureGeometry(context) {
  if (!context.featureGeometry) {
    context.featureGeometry = {
      labelsByEdge: {}, junctions: [], junctionIDsByEdge: {}, invalidRouteIDs: [],
      portsByNode: {}, portEndpointsByEdge: {}
    };
  }
  return context.featureGeometry;
}

export function addFeatureDiagnostic(context, code, message, details = {}) {
  context.diagnostics = Array.isArray(context.diagnostics) ? context.diagnostics : [];
  context.diagnostics.push({ code, severity: "warning", message, details });
  return context;
}

export function failedFeature(context, id, error, phase) {
  addFeatureDiagnostic(context, "geometry_fallback_applied",
    id + " failed during " + phase + "; ordinary geometry is used.",
    { feature: id, phase, cause: error?.message || String(error) });
  return context;
}
