export function escapeOKF(value) {
  return String(value == null ? "" : value).replace(/[&<>"']/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[character]));
}

export function groupDiagnostics(diagnostics) {
  const groups = new Map();
  (Array.isArray(diagnostics) ? diagnostics : []).forEach((item) => {
    const value = item || {};
    const key = [value.severity || "info", value.code || "diagnostic", value.message || ""].join("\u0000");
    let group = groups.get(key);
    if (!group) {
      group = { severity: value.severity || "info", code: value.code || "diagnostic", message: value.message || "", count: 0, items: [] };
      groups.set(key, group);
    }
    group.count += 1;
    group.items.push(value);
  });
  return Array.from(groups.values());
}

export function diagnosticMarkup(diagnostics) {
  const values = Array.isArray(diagnostics) ? diagnostics : [];
  if (!values.length) return "";
  const groups = groupDiagnostics(values);
  const total = values.length;
  const summary = total + " finding" + (total === 1 ? "" : "s") + " · " + groups.length + " unique";
  const report = groups.map((group) => {
    const contexts = Array.from(new Set(group.items.map((item) => item.path || item.concept_id || item.profile_id || item.bundle_id || item.operation_id || "").filter(Boolean)));
    const preview = contexts.slice(0, 4);
    const recovery = Array.from(new Set(group.items.map((item) => item.recovery).filter(Boolean)));
    const records = group.items.map(({ severity, category, bundle_id, concept_id, profile_id, operation_id, path, details }) => ({ severity, category, bundle_id, concept_id, profile_id, operation_id, path, details })).filter((item) => Object.values(item).some((value) => value !== undefined));
    const contextMarkup = (preview.length ? "<small class=\"okf-diagnostic-context\">" + escapeOKF(preview.join(" · ")) + (contexts.length > preview.length ? " · …" : "") + "</small>" : "")
      + recovery.map((message) => "<p>" + escapeOKF(message) + "</p>").join("")
      + (records.length ? "<details><summary>Affected sources and diagnostic details</summary><pre>" + escapeOKF(JSON.stringify(records, null, 2)) + "</pre></details>" : "");
    return "<li class=\"okf-diagnostic-group\"><span class=\"okf-diagnostic-count\" aria-label=\"" + group.count + " occurrences\">" + group.count + "×</span><code>" + escapeOKF(group.code) + "</code><span>" + escapeOKF(group.message) + "</span>" + contextMarkup + "</li>";
  }).join("");
  return "<details class=\"okf-diagnostic-report\"><summary><span>View diagnostic report</span><span class=\"muted\">" + escapeOKF(summary) + "</span></summary><ul class=\"okf-diagnostic-list\">" + report + "</ul></details>";
}

export function renderAccessibleItems(container, snapshot, onSelect) {
  if (!container) return;
  const nodes = snapshot && Array.isArray(snapshot.nodes) ? snapshot.nodes : [];
  container.innerHTML = nodes.length ? nodes.map((node) => {
    const fields = Array.isArray(node.presentation_fields) ? node.presentation_fields.slice(0, 3) : [];
    const fieldMarkup = fields.length ? "<span class=\"okf-accessible-fields muted\">" + escapeOKF(fields.map((field) => (field.label || field.source || "value") + ": " + (field.value || "")).join(" · ")) + "</span>" : "";
    return "<button class=\"okf-accessible-item\" type=\"button\" data-okf-concept=\"" + escapeOKF(node.concept_id || node.id) + "\"><span class=\"okf-accessible-label\">" + escapeOKF(node.label || node.title || node.id) + "</span>" + fieldMarkup + "</button>";
  }).join("") : "<p class=\"muted\">No concepts are visible in this projection.</p>";
  container.querySelectorAll("[data-okf-concept]").forEach((element) => element.addEventListener("click", () => onSelect(element.dataset.okfConcept)));
}

export function renderDetail(container, detail) {
  if (!container) return;
  if (!detail) { container.innerHTML = "<p class=\"muted\">Select a concept to inspect its source-backed detail.</p>"; return; }
  const overview = detail.overview || {};
  const unknown = detail.mapped_metadata && Object.keys(detail.mapped_metadata).length ? "<details><summary>Unknown frontmatter</summary><pre>" + escapeOKF(JSON.stringify(detail.mapped_metadata, null, 2)) + "</pre></details>" : "";
  const raw = detail.raw_markdown ? "<details><summary>Raw Markdown</summary><pre>" + escapeOKF(detail.raw_markdown) + "</pre></details>" : "";
  const diagnostics = diagnosticMarkup(detail.diagnostics);
  const facts = detailFacts(detail);
  container.innerHTML = "<div class=\"okf-detail-overview\"><p class=\"eyebrow\">" + escapeOKF(overview.type || "CONCEPT") + "</p><h4>" + escapeOKF(overview.title || detail.concept_id) + "</h4><p class=\"muted\">" + escapeOKF(overview.description || "") + "</p><code>" + escapeOKF(overview.source_path || detail.concept_id) + "</code></div><div class=\"okf-markdown\">" + (detail.rendered_markdown && detail.rendered_markdown.content || "") + "</div>" + facts + unknown + raw + diagnostics;
}

function detailFacts(detail) {
  const state = [
    ["Declared state", detail.declared_state],
    ["Effective state", detail.effective_state]
  ].filter((entry) => entry[1] !== undefined && entry[1] !== "");
  const interpretation = state.length
    ? "<details><summary>State interpretation</summary><dl>" + state.map(([label, value]) => "<dt>" + label + "</dt><dd>" + escapeOKF(value) + "</dd>").join("") + "</dl></details>"
    : "";
  const sections = [
    ["Containment", detail.containment],
    ["Semantic link outcomes", detail.semantic_links],
    ["Source provenance", detail.provenance]
  ].filter((entry) => entry[1] && Object.keys(entry[1]).length);
  return interpretation + sections.map(([label, value]) => "<details><summary>" + label + "</summary><pre>" + escapeOKF(JSON.stringify(value, null, 2)) + "</pre></details>").join("");
}
