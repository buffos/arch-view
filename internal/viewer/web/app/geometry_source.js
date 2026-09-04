export function architectureGeometrySource(scene) {
  return {
    kind: "architecture",
    id: scene?.model_id || "architecture",
    revision: scene?.model_revision || scene?.model_id || "unknown",
    navigation_scope: {
      hierarchy_path: [...(scene?.hierarchy_path || [])],
      reference_visibility: scene?.reference_visibility || "hidden"
    }
  };
}

export function okfGeometrySource(snapshot) {
  return {
    kind: "okf",
    id: snapshot?.source?.bundle_id || "okf",
    revision: snapshot?.projection_revision || snapshot?.source?.source_revision || "unknown",
    navigation_scope: {
      focus_root: snapshot?.navigation?.focus_root || "",
      depth: snapshot?.navigation?.depth || 0,
      full: Boolean(snapshot?.navigation?.full)
    }
  };
}
