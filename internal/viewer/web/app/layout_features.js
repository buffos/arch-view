// Declarative policy arrives in the same catalog as ELK options. Execution
// stays in focused handlers, registered once at the shared composition root.
export const FEATURE_PHASES = Object.freeze(["negotiate", "prepare", "normalize", "validate", "render"]);

export function featureProblems(profile, catalog) {
  const selected = profile?.features || [];
  const definitions = new Map((catalog?.features || []).map((item) => [item.id, item]));
  const problems = [];
  const seen = new Set();
  for (const id of selected) {
    const feature = definitions.get(id);
    if (!feature) { problems.push("Unknown renderer feature: " + id); continue; }
    if (seen.has(id)) problems.push("Duplicate renderer feature: " + id);
    seen.add(id);
    if (feature.status !== "supported") problems.push(feature.name + " is not implemented in this build.");
    if (!feature.algorithms.includes(profile.algorithm)) problems.push(feature.name + " requires " + feature.algorithms.join(" or ") + ".");
    for (const dependency of feature.prerequisites || []) {
      if (!selected.includes(dependency)) problems.push(feature.name + " requires " + dependency + ".");
    }
    for (const [key, expected] of Object.entries(feature.required_options || {})) {
      if (profile.options?.[key] !== expected) problems.push(feature.name + " requires " + key + " = " + expected + ".");
    }
  }
  return problems;
}

export function createFeatureRegistry(definitions = [], handlers = []) {
  const catalog = new Map();
  const implementations = new Map();
  for (const definition of definitions) {
    if (catalog.has(definition.id)) throw new Error("Duplicate feature: " + definition.id);
    catalog.set(definition.id, definition);
  }
  for (const handler of handlers) {
    if (!catalog.has(handler.id)) throw new Error("Unknown handler: " + handler.id);
    if (implementations.has(handler.id)) throw new Error("Duplicate handler: " + handler.id);
    if (typeof handler.validate !== "function" || typeof handler.fallback !== "function") {
      throw new Error("Incomplete feature handler: " + handler.id);
    }
    implementations.set(handler.id, handler);
  }
  const owners = new Map();
  for (const definition of definitions) {
    for (const field of definition.owns || []) {
      if ([...owners.keys()].some((key) => key === field || key.startsWith(field + ".") || field.startsWith(key + "."))) throw new Error("Conflicting geometry ownership: " + field);
      owners.set(field, definition.id);
    }
    if (definition.status === "supported" && !implementations.has(definition.id)) {
      throw new Error("Supported feature has no handler: " + definition.id);
    }
  }
  const ordered = [];
  const compare = (a, b) => (a.order || 0) - (b.order || 0) || a.id.localeCompare(b.id);
  for (const definition of definitions) {
    for (const id of definition.prerequisites || []) if (!catalog.has(id)) throw new Error("Unknown feature prerequisite: " + id);
  }
  const remaining = new Map(catalog);
  while (remaining.size) {
    const next = [...remaining.values()].filter((item) => (item.prerequisites || []).every((id) => !remaining.has(id))).sort(compare)[0];
    if (!next) throw new Error("Feature dependency cycle");
    ordered.push(next);
    remaining.delete(next.id);
  }
  return {
    negotiate(profile, surface = "browser") {
      const requested = profile.features || [];
      const effective = [];
      const diagnostics = [];
      for (const id of requested) if (!catalog.has(id)) throw new Error("Unknown renderer feature: " + id);
      for (const definition of ordered.filter((item) => requested.includes(item.id))) {
        const available = definition.status === "supported" && definition.surfaces.includes(surface)
          && definition.algorithms.includes(profile.algorithm)
          && (definition.prerequisites || []).every((id) => effective.includes(id))
          && Object.entries(definition.required_options || {}).every(([id, value]) => profile.options?.[id] === value);
        if (available) effective.push(definition.id);
        else diagnostics.push({ code: "renderer_feature_unavailable", message: definition.name + " is unavailable; ordinary geometry is used. Saved preferences are unchanged." });
      }
      return { effective, diagnostics };
    },
    async run(phase, context, effective) {
      if (!FEATURE_PHASES.includes(phase)) throw new Error("Unknown feature phase: " + phase);
      let result = context;
      for (const definition of ordered.filter((item) => effective.includes(item.id))) {
        const handler = implementations.get(definition.id);
        if (!handler) throw new Error("Unavailable feature handler: " + definition.id);
        if (typeof handler[phase] !== "function") continue;
        try { result = await handler[phase](result); }
        catch (error) { result = await handler.fallback(result, error, phase); }
      }
      return result;
    },
    order: ordered.map((item) => item.id)
  };
}

// Later delivery stages register their verified implementations here.
export function sharedFeatureRegistry(catalog) {
  return createFeatureRegistry(catalog?.features || [], []);
}
