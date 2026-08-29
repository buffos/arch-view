export function escapeHTML(value) {
  return String(value == null ? "" : value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function truncate(value, length) {
  const text = String(value || "");
  return text.length > length ? text.slice(0, length - 1) + "…" : text;
}

export function classForState(value) {
  return String(value || "none").replace(/[^a-z0-9_-]/gi, "-");
}

export function formatList(values, empty) {
  if (!values || values.length === 0) return empty || "None";
  return values.join(", ");
}

export function formatLanguage(value) {
  const text = String(value || "").trim().toLowerCase();
  if (!text) return "Unknown";
  if (text === "typescript") return "TypeScript";
  if (text === "multi") return "Multi";
  return text.charAt(0).toUpperCase() + text.slice(1);
}

export function displayProjectRoot(value) {
  const text = String(value || "").trim();
  return !text || text === "." ? "Repository root" : text;
}

function moduleForNode(context, moduleID) {
  const modules = context && context.state && context.state.model && context.state.model.modules
    ? context.state.model.modules
    : [];
  const exact = modules.find(function (module) { return module.id === moduleID; });
  if (exact) return exact;
  const activeScope = context && context.state ? context.state.activeScope : "";
  if (!activeScope || activeScope === "all" || !moduleID) return null;
  const prefix = activeScope + "::";
  return modules.find(function (module) {
    if (!module.id || !module.id.startsWith(prefix)) return false;
    try {
      return decodeURIComponent(module.id.slice(prefix.length)) === moduleID;
    } catch (_) {
      return false;
    }
  }) || null;
}

export function nodeLanguages(context, node) {
  const languages = [];
  (node && node.module_ids || []).forEach(function (moduleID) {
    const module = moduleForNode(context, moduleID);
    if (!module || !module.language) return;
    const language = String(module.language).trim().toLowerCase();
    if (language && !languages.includes(language)) languages.push(language);
  });
  if (languages.length) return languages.sort();
  if (node && node.kind === "reference") return ["reference"];
  const sceneLanguage = context && context.state && context.state.scene && context.state.scene.project
    ? String(context.state.scene.project.language || "").trim().toLowerCase()
    : "";
  return sceneLanguage ? [sceneLanguage] : [];
}

export function nodeLanguageBadge(context, node) {
  const languages = nodeLanguages(context, node);
  return languages.length > 1 ? "multi" : (languages[0] || "unknown");
}

export function nodeLanguageText(context, node) {
  const languages = nodeLanguages(context, node);
  if (!languages.length) return "Unknown";
  return languages.length > 1
    ? languages.map(formatLanguage).join(" + ")
    : formatLanguage(languages[0]);
}

export function clampNumber(value, minimum, maximum, fallback) {
  return typeof value === "number" && Number.isFinite(value) ? Math.max(minimum, Math.min(maximum, value)) : fallback;
}

export function numberOrZero(value) {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

export function samePath(left, right) {
  const a = left || [];
  const b = right || [];
  return a.length === b.length && a.every(function (segment, index) { return segment === b[index]; });
}

export function pathKey(pathValue) {
  return (pathValue || []).join("/");
}

export function cloneLayoutProfile(profile) {
  const value = profile || {};
  return { algorithm: value.algorithm || "layered", options: Object.assign({}, value.options || {}) };
}

export function layoutRequestPayload(profile) {
  return { schema_version: "arch-view.config/v1", layout: cloneLayoutProfile(profile) };
}

export function formatOptionValue(value) {
  if (value == null) return "—";
  if (typeof value === "string") return value;
  return JSON.stringify(value);
}

export function optionBoundsText(option) {
  const bounds = [];
  if (option.minimum != null) bounds.push((option.minimum_exclusive ? "> " : "≥ ") + formatOptionValue(option.minimum));
  if (option.maximum != null) bounds.push((option.maximum_exclusive ? "< " : "≤ ") + formatOptionValue(option.maximum));
  return bounds.length ? " · bounds: " + bounds.join(", ") : "";
}

export function referenceVisibilityLabel(value) {
  return value === "expanded" ? "Expanded" : value === "aggregated" ? "Aggregated" : "Local-first";
}

export function referenceScopeLabel(value) {
  return String(value || "reference").replaceAll("_", " ");
}

export function confidenceState(score) {
  if (score == null) return "unknown";
  if (score >= 0.85) return "high";
  if (score >= 0.55) return "medium";
  return "low";
}
