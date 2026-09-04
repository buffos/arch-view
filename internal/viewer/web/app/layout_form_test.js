const { test } = require("node:test");
const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");
const moduleURL = (name) => pathToFileURL(path.join(__dirname, name)).href;
const option = (id, extra = {}) => ({ id, name: id, type: "DOUBLE", targets: ["PARENTS"], algorithms: ["layered"], editable: true, renderer_support: "supported", ...extra });
function context() {
  const elements = Object.fromEntries(["layoutSettingsDialog","layoutSettingsOrigin","layoutSettingsStatus","layoutAlgorithm","layoutAlgorithmHelp","layoutOptionsList","layoutSettingsApply"].map((id) => [id, {}]));
  return { elements, state: { layoutDraft: { algorithm: "layered", options: {} }, layoutCatalog: {
    algorithms: [{ id: "layered", name: "Layered" }, { id: "mrtree", name: "Mr. Tree" }],
    options: [option("usable"), option("other", { algorithms: ["mrtree"] }), option("future", { editable: false }), option("gated", { required_features: ["ports"] })]
  } } };
}
test("SC-AER-008 shared form shows usable options and complete catalog on demand", async () => {
  const { renderLayoutForm, updateLayoutDraftOption } = await import(moduleURL("layout_form.js"));
  const value = context();
  renderLayoutForm(value);
  assert.match(value.elements.layoutOptionsList.innerHTML, /usable/);
  assert.doesNotMatch(value.elements.layoutOptionsList.innerHTML, /Not applicable|Requires an enabled feature|<strong>future/);
  updateLayoutDraftOption(value, { target: { dataset: { layoutCatalogFilter: "" }, checked: true } });
  assert.match(value.elements.layoutOptionsList.innerHTML, /Not applicable to this algorithm/);
  assert.match(value.elements.layoutOptionsList.innerHTML, /Not implemented/);
  assert.match(value.elements.layoutOptionsList.innerHTML, /Requires an enabled feature: ports/);
});
test("SC-AER-008 algorithm changes keep preferences and block conflicting Apply", async () => {
  const { updateLayoutDraftAlgorithm } = await import(moduleURL("layout_form.js"));
  const value = context(); value.state.layoutDraft.options.usable = 20;
  value.state.layoutDraft.features = [];
  updateLayoutDraftAlgorithm(value, { target: { value: "mrtree" } });
  assert.equal(value.state.layoutDraft.options.usable, 20);
  assert.deepEqual(value.state.layoutDraft.features, []);
  assert.equal(value.elements.layoutSettingsApply.disabled, true);
  assert.match(value.elements.layoutSettingsStatus.textContent, /Not applicable/);
});
test("SC-AER-001 clone and request serialization preserve inheritance and drafts", async () => {
  const { cloneLayoutProfile, layoutRequestPayload } = await import(moduleURL("utils.js"));
  const { cloneOKFLayoutProfile } = await import(moduleURL("okf_layout.js"));
  for (const clone of [cloneLayoutProfile, cloneOKFLayoutProfile]) {
    assert.equal(Object.hasOwn(clone({}), "features"), false);
    assert.deepEqual(clone({ features: [] }).features, []);
    const original = { features: ["ports"], options: { padding: { left: 5 } } };
    const copy = clone(original); copy.features.push("edge_labels"); copy.options.padding.left = 90;
    assert.deepEqual(original, { features: ["ports"], options: { padding: { left: 5 } } });
    assert.equal(layoutRequestPayload(original).schema_version, "arch-view.config/v2");
  }
});
test("SC-AER-008 padding is four numeric fields with validated request encoding", async () => {
  const { renderOptionControl, updateOptionValue } = await import(moduleURL("layout_option_controls.js"));
  const { serializeLayoutValue } = await import(moduleURL("layout_value.js"));
  const padding = option("org.eclipse.elk.padding", { type: "OBJECT", control: "padding" });
  const draft = { options: {} };
  const html = renderOptionControl(padding, draft);
  assert.equal((html.match(/type="number"/g) || []).length, 4);
  updateOptionValue(padding, { value: "12", dataset: { layoutPadding: "top" } }, draft);
  assert.equal(serializeLayoutValue(padding, draft.options[padding.id]), "[top=12,right=0,bottom=0,left=0]");
  assert.throws(() => serializeLayoutValue(padding, { top: NaN }), /four finite/);
});
