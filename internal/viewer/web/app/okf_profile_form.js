import { escapeOKF } from "./okf_markup.js";
import { readNumericInput } from "./numeric_input.js";
import { shapeChoicesMarkup } from "./okf_shape_choices.js";
import { detailRendererMarkup, readDetailRenderer } from "./okf_detail_form.js";

const OKF_DEFAULT_LAYOUT_ALGORITHM = "mrtree";
const formDrafts = new WeakMap();

export function renderProfileForm(root, profile, effective = profile) {
  root.innerHTML = profileFormMarkup(effective) + detailRendererMarkup(effective.details) + shapeChoicesMarkup(root);
  formDrafts.set(root, {
    original: JSON.parse(JSON.stringify(profile)),
    displayed: readFormControls(root, profile)
  });
}

// Keep omitted inherited settings omitted, including after an edit is reverted.
export function profileFormChanges(original, before, after, path = "") {
  const result = JSON.parse(JSON.stringify(original || {}));
  for (const key of new Set([...Object.keys(before), ...Object.keys(after)])) {
    if (JSON.stringify(before[key]) === JSON.stringify(after[key])) continue;
    if (!Object.hasOwn(after, key)) {
      delete result[key];
    } else {
      const next = after[key];
      const childPath = path ? path + "." + key : key;
      const replacement = ["details.renderer", "state.mapping", "style.tokens", "style.state_tokens", "layout.options", "style.decorations.root", "style.decorations.rollup"].includes(childPath);
      const value = next && typeof next === "object" && !Array.isArray(next) && !replacement
        ? profileFormChanges(result[key], before[key] || {}, next, childPath) : next;
      Object.defineProperty(result, key, { value, writable: true, enumerable: true, configurable: true });
    }
  }
  return result;
}

const repeatLabels = {
  node_fields: "Node presentation fields",
  state_mapping: "State mapping",
  style_tokens: "Style tokens",
  style_state_mapping: "State token mapping",
  rules: "Rule strategies"
};

function text(value) {
  return escapeOKF(value == null ? "" : value);
}

function checked(value) {
  return value ? " checked" : "";
}

function numberValue(value, fallback) {
  return Number.isFinite(Number(value)) ? Number(value) : fallback;
}

function nodeFieldRow(value = {}) {
  return '<div class="okf-repeat-row" data-repeat-row="node_fields"><label><span>Source</span><input data-repeat-value="source" value="' + text(value.source) + '" placeholder="frontmatter.state"></label><label><span>Label</span><input data-repeat-value="label" value="' + text(value.label) + '" placeholder="State"></label><label><span>Max length</span><input data-repeat-value="max_length" type="number" min="0" max="80" value="' + text(value.max_length || "") + '" placeholder="48"></label><button class="button secondary" type="button" data-remove-repeat="node_fields">Remove</button></div>';
}

function mappingRow(value = {}) {
  return '<div class="okf-repeat-row" data-repeat-row="state_mapping"><label><span>Source value</span><input data-repeat-value="source" value="' + text(value.source) + '" placeholder="implemented"></label><label><span>Mapped state</span><input data-repeat-value="target" value="' + text(value.target) + '" placeholder="implemented"></label><button class="button secondary" type="button" data-remove-repeat="state_mapping">Remove</button></div>';
}

function tokenRow(value = {}) {
  return '<div class="okf-repeat-row okf-token-row" data-repeat-row="style_tokens"><label><span>Token ID</span><input data-repeat-value="id" value="' + text(value.id) + '" placeholder="state.implemented"></label><label><span>Fill</span><input data-repeat-value="fill" value="' + text(value.fill) + '" placeholder="#d9f4df"></label><label><span>Stroke</span><input data-repeat-value="stroke" value="' + text(value.stroke) + '" placeholder="#24743b"></label><label><span>Text</span><input data-repeat-value="text" value="' + text(value.text) + '" placeholder="#153b21"></label><label><span>Shape</span><input data-repeat-value="shape" list="okf-shape-choices" value="' + text(value.shape) + '" placeholder="rounded_rectangle"></label><label><span>Emphasis</span><input data-repeat-value="emphasis" value="' + text(value.emphasis) + '" placeholder="Optional"></label><label><span>Stroke width</span><input data-repeat-value="stroke_width" type="number" min="0" step="any" value="' + text(value.stroke_width || "") + '" placeholder="Optional"></label><label><span>Dash array</span><input data-repeat-value="stroke_dasharray" value="' + text(value.stroke_dasharray) + '" placeholder="Optional"></label><button class="button secondary" type="button" data-remove-repeat="style_tokens">Remove</button></div>';
}

function styleMappingRow(value = {}) {
  return '<div class="okf-repeat-row" data-repeat-row="style_state_mapping"><label><span>Effective state</span><input data-repeat-value="source" value="' + text(value.source) + '" placeholder="implemented"></label><label><span>Token ID</span><input data-repeat-value="target" value="' + text(value.target) + '" placeholder="state.implemented"></label><button class="button secondary" type="button" data-remove-repeat="style_state_mapping">Remove</button></div>';
}

function ruleRow(value = {}) {
  return '<div class="okf-repeat-row okf-rule-row" data-repeat-row="rules"><label><span>Strategy ID</span><input data-repeat-value="rule_id" value="' + text(value.rule_id) + '" placeholder="metadata.equals"></label><label><span>Version</span><input data-repeat-value="version" value="' + text(value.version || "1") + '"></label><label><span>Priority</span><input data-repeat-value="priority" type="number" value="' + text(value.priority || "0") + '"></label><label class="okf-check"><span>Enabled</span><input data-repeat-value="enabled" type="checkbox"' + checked(value.enabled) + '></label><label class="full"><span>Parameters (JSON)</span><textarea data-repeat-value="parameters" rows="2" spellcheck="false">' + text(JSON.stringify(value.parameters || {}, null, 2)) + '</textarea></label><button class="button secondary" type="button" data-remove-repeat="rules">Remove</button></div>';
}

function repeatSection(name, values, row) {
  const rows = Array.isArray(values) ? values.map(row).join("") : "";
  return '<section class="okf-form-section"><div class="okf-form-section-heading"><h4>' + repeatLabels[name] + '</h4><button class="button secondary" type="button" data-add-repeat="' + name + '">Add</button></div><div class="okf-repeat-list" data-repeat-list="' + name + '">' + rows + '</div></section>';
}

function decorationMarkup(name, value = {}) {
  return '<fieldset class="okf-decoration"><legend>' + text(name) + ' decoration</legend><div class="okf-form-grid"><label><span>Fill</span><input data-decoration="' + text(name) + '" data-decoration-value="fill" value="' + text(value.fill) + '" placeholder="Optional"></label><label><span>Stroke</span><input data-decoration="' + text(name) + '" data-decoration-value="stroke" value="' + text(value.stroke) + '" placeholder="Optional"></label><label><span>Text</span><input data-decoration="' + text(name) + '" data-decoration-value="text" value="' + text(value.text) + '" placeholder="Optional"></label><label><span>Stroke width</span><input data-decoration="' + text(name) + '" data-decoration-value="stroke_width" type="number" min="0" step="any" value="' + text(value.stroke_width || "") + '" placeholder="Optional"></label><label><span>Dash array</span><input data-decoration="' + text(name) + '" data-decoration-value="stroke_dasharray" value="' + text(value.stroke_dasharray) + '" placeholder="Optional"></label></div></fieldset>';
}

export function profileFormMarkup(profile = {}) {
  const state = profile.state || {};
  const hierarchy = profile.hierarchy || {};
  const relationships = profile.relationships || {};
  const navigation = profile.navigation || {};
  const style = profile.style || {};
  const details = profile.details || {};
  const decorations = style.decorations || {};
  const tokenValues = Object.keys(style.tokens || {}).sort().map((id) => Object.assign({ id }, style.tokens[id])).filter((value) => value.id);
  const mappingValues = Object.keys(state.mapping || {}).sort().map((source) => ({ source, target: state.mapping[source] }));
  const stateTokenValues = Object.keys(style.state_tokens || {}).sort().map((source) => ({ source, target: style.state_tokens[source] }));
  return '<div class="okf-profile-form"><section class="okf-form-section"><h4>Identity and composition</h4><div class="okf-form-grid"><label><span>Profile ID</span><input data-profile="profile_id" value="' + text(profile.profile_id) + '"></label><label><span>Name</span><input data-profile="name" value="' + text(profile.name) + '"></label><label class="full"><span>Base profiles (comma separated)</span><input data-profile="bases" value="' + text((profile.bases || []).join(", ")) + '" placeholder="builtin:neutral"></label></div></section>' + repeatSection("node_fields", profile.node_fields || [], nodeFieldRow) + '<section class="okf-form-section"><h4>Hierarchy and relationships</h4><div class="okf-form-check-grid"><label class="okf-check"><input data-profile="hierarchy.use_explicit" type="checkbox"' + checked(hierarchy.use_explicit) + '> <span>Use explicit hierarchy</span></label><label class="okf-check"><input data-profile="hierarchy.use_filesystem_fallback" type="checkbox"' + checked(hierarchy.use_filesystem_fallback) + '> <span>Use filesystem fallback</span></label><label class="okf-check"><input data-profile="relationships.show_containment" type="checkbox"' + checked(relationships.show_containment) + '> <span>Show containment</span></label><label class="okf-check"><input data-profile="relationships.show_semantic_links" type="checkbox"' + checked(relationships.show_semantic_links) + '> <span>Show semantic links</span></label></div></section><section class="okf-form-section"><h4>State mapping</h4><div class="okf-form-grid"><label><span>State source</span><input data-profile="state.field" value="' + text(state.field) + '" placeholder="frontmatter.state"></label><label class="okf-check"><input data-profile="state.roll_up" type="checkbox"' + checked(state.roll_up) + '> <span>Roll up homogeneous child state</span></label><label class="okf-check"><input data-profile="state.show_declared" type="checkbox"' + checked(state.show_declared) + '> <span>Show declared state in detail</span></label></div>' + repeatSection("state_mapping", mappingValues, mappingRow) + '</section><section class="okf-form-section"><h4>Navigation and detail</h4><div class="okf-form-grid"><label><span>Default depth</span><input data-profile="navigation.default_depth" type="number" min="1" value="' + text(numberValue(navigation.default_depth, 2)) + '"></label><label><span>Maximum nodes</span><input data-profile="navigation.max_nodes" type="number" min="1" value="' + text(numberValue(navigation.max_nodes, 1000)) + '"></label><label><span>Maximum relationships</span><input data-profile="navigation.max_relationships" type="number" min="1" value="' + text(numberValue(navigation.max_relationships, 10000)) + '"></label><label class="okf-check"><input data-profile="details.show_raw_markdown" type="checkbox"' + checked(details.show_raw_markdown) + '> <span>Show raw Markdown</span></label><label class="okf-check"><input data-profile="details.show_unknown_frontmatter" type="checkbox"' + checked(details.show_unknown_frontmatter) + '> <span>Show unknown frontmatter</span></label></div></section><section class="okf-form-section"><h4>Style tokens and structure</h4><div class="okf-form-grid"><label><span>Default token</span><input data-profile="style.default_token" value="' + text(style.default_token) + '" placeholder="state.unknown"></label></div>' + repeatSection("style_state_mapping", stateTokenValues, styleMappingRow) + repeatSection("style_tokens", tokenValues, tokenRow) + decorationMarkup("root", decorations.root) + decorationMarkup("rollup", decorations.rollup) + '</section>' + repeatSection("rules", profile.rules || [], ruleRow) + '<section class="okf-form-section"><div class="okf-form-section-heading"><h4>ELK layout</h4><button class="button secondary" type="button" data-open-layout-settings>Edit layout options</button></div></section></div>';
}

function readRows(root, name, mapper) {
  return Array.from(root.querySelectorAll('[data-repeat-row="' + name + '"]')).map(mapper).filter(Boolean);
}

function setMappingValue(mapping, key, value) {
  if (Object.hasOwn(mapping, key)) throw new Error("Duplicate mapping or token key: " + key);
  Object.defineProperty(mapping, key, { value, writable: true, enumerable: true, configurable: true });
}

function input(root, path) {
  return root.querySelector('[data-profile="' + path + '"]');
}

function stringValue(root, path) {
  const value = input(root, path);
  return value ? value.value.trim() : "";
}

function boolValue(root, path) {
  const value = input(root, path);
  return Boolean(value && value.checked);
}

function integerValue(root, path, fallback) {
  return readNumericInput(input(root, path), fallback, true);
}

function repeatInteger(row, key, fallback) {
  return readNumericInput(row.querySelector('[data-repeat-value="' + key + '"]'), fallback, true);
}

function decorations(root) {
  const result = {};
  root.querySelectorAll("[data-decoration]").forEach((element) => {
    const name = element.dataset.decoration;
    const key = element.dataset.decorationValue;
    if (!result[name]) result[name] = {};
    const value = element.value.trim();
    if (key === "stroke_width") {
      const width = readNumericInput(element, undefined);
      if (width !== undefined) result[name][key] = width;
    } else if (value !== "") result[name][key] = value;
  });
  return result;
}

export function readProfileForm(root, original = {}) {
  const draft = formDrafts.get(root);
  if (!draft) return readFormControls(root, original);
  const result = profileFormChanges(draft.original, draft.displayed, readFormControls(root, draft.original));
  // Layout has its own shared editor and may have changed since form rendering.
  if (Object.hasOwn(original, "layout")) result.layout = JSON.parse(JSON.stringify(original.layout));
  return result;
}

function readFormControls(root, original = {}) {
  const profile = JSON.parse(JSON.stringify(original || {}));
  profile.profile_id = stringValue(root, "profile_id");
  profile.name = stringValue(root, "name");
  profile.bases = stringValue(root, "bases").split(",").map((value) => value.trim()).filter(Boolean);
  profile.node_fields = readRows(root, "node_fields", (row) => ({ source: row.querySelector('[data-repeat-value="source"]').value.trim(), label: row.querySelector('[data-repeat-value="label"]').value.trim(), max_length: repeatInteger(row, "max_length", 0) })).filter((value) => value.source);
  profile.hierarchy = Object.assign({}, profile.hierarchy, { use_explicit: boolValue(root, "hierarchy.use_explicit"), use_filesystem_fallback: boolValue(root, "hierarchy.use_filesystem_fallback") });
  profile.relationships = Object.assign({}, profile.relationships, { show_containment: boolValue(root, "relationships.show_containment"), show_semantic_links: boolValue(root, "relationships.show_semantic_links") });
  profile.state = Object.assign({}, profile.state, { field: stringValue(root, "state.field"), roll_up: boolValue(root, "state.roll_up"), show_declared: boolValue(root, "state.show_declared"), mapping: {} });
  readRows(root, "state_mapping", (row) => ({ source: row.querySelector('[data-repeat-value="source"]').value.trim(), target: row.querySelector('[data-repeat-value="target"]').value.trim() })).filter((value) => value.source && value.target).forEach((value) => { setMappingValue(profile.state.mapping, value.source, value.target); });
  profile.navigation = Object.assign({}, profile.navigation, { default_depth: integerValue(root, "navigation.default_depth", 2), max_nodes: integerValue(root, "navigation.max_nodes", 1000), max_relationships: integerValue(root, "navigation.max_relationships", 10000) });
  profile.details = Object.assign({}, profile.details, { show_raw_markdown: boolValue(root, "details.show_raw_markdown"), show_unknown_frontmatter: boolValue(root, "details.show_unknown_frontmatter") });
  readDetailRenderer(root, profile.details);
  profile.style = Object.assign({}, profile.style, { default_token: stringValue(root, "style.default_token"), tokens: {}, state_tokens: {}, decorations: decorations(root) });
  readRows(root, "style_state_mapping", (row) => ({ source: row.querySelector('[data-repeat-value="source"]').value.trim(), target: row.querySelector('[data-repeat-value="target"]').value.trim() })).filter((value) => value.source && value.target).forEach((value) => { setMappingValue(profile.style.state_tokens, value.source, value.target); });
  readRows(root, "style_tokens", (row) => ({ id: row.querySelector('[data-repeat-value="id"]').value.trim(), fill: row.querySelector('[data-repeat-value="fill"]').value.trim(), stroke: row.querySelector('[data-repeat-value="stroke"]').value.trim(), text: row.querySelector('[data-repeat-value="text"]').value.trim(), shape: row.querySelector('[data-repeat-value="shape"]').value.trim(), emphasis: row.querySelector('[data-repeat-value="emphasis"]').value.trim(), stroke_width: repeatFloat(row, "stroke_width", 0), stroke_dasharray: row.querySelector('[data-repeat-value="stroke_dasharray"]').value.trim() })).filter((value) => value.id).forEach((value) => { setMappingValue(profile.style.tokens, value.id, value); });
  profile.rules = readRows(root, "rules", (row) => {
    let parameters = {};
    const raw = row.querySelector('[data-repeat-value="parameters"]').value.trim();
    if (raw) parameters = JSON.parse(raw);
    return { rule_id: row.querySelector('[data-repeat-value="rule_id"]').value.trim(), version: row.querySelector('[data-repeat-value="version"]').value.trim(), priority: repeatInteger(row, "priority", 0), enabled: Boolean(row.querySelector('[data-repeat-value="enabled"]').checked), parameters };
  }).filter((value) => value.rule_id);
  profile.layout = Object.assign({}, profile.layout, { algorithm: profile.layout && profile.layout.algorithm || OKF_DEFAULT_LAYOUT_ALGORITHM });
  return profile;
}

export function bindProfileForm(root, onChange) {
  if (!root) return;
  root.addEventListener("click", (event) => {
    const add = event.target.closest("[data-add-repeat]");
    const remove = event.target.closest("[data-remove-repeat]");
    if (add) {
      const list = root.querySelector('[data-repeat-list="' + add.dataset.addRepeat + '"]');
      if (list) list.insertAdjacentHTML("beforeend", emptyRow(add.dataset.addRepeat));
      if (onChange) onChange();
    }
    if (remove) {
      remove.closest("[data-repeat-row]")?.remove();
      if (onChange) onChange();
    }
    if (event.target.closest("[data-open-layout-settings]") && onChange) onChange("layout");
  });
  root.addEventListener("input", () => onChange && onChange());
  root.addEventListener("change", (event) => onChange && onChange(event.target.dataset.profile === "bases" ? "bases" : undefined));
}

function emptyRow(name) {
  if (name === "node_fields") return nodeFieldRow();
  if (name === "state_mapping") return mappingRow();
  if (name === "style_state_mapping") return styleMappingRow();
  if (name === "style_tokens") return tokenRow();
  return ruleRow();
}

function repeatFloat(row, key, fallback) {
  return readNumericInput(row.querySelector('[data-repeat-value="' + key + '"]'), fallback);
}
