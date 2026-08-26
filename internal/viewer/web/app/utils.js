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
