import { createOKFAPI } from "./okf_api.js";
import { downloadCurrentSVG } from "./export.js";
import { confirmAction } from "./confirm_action.js";
import { loadShapeChoices } from "./okf_shape_choices.js";
import { renderHiddenConcepts } from "./okf_hidden_concepts.js";
import { changeProfileConfiguration, renameEditorProfile, deleteEditorProfile, loadConfigurationCatalogs } from "./okf_profile_lifecycle.js";
import { saveEditorProfile } from "./okf_profile_save.js";
import { readNumericInput } from "./numeric_input.js";
import { selectOKFConcept } from "./okf_detail.js";
import { clearOKFProjection } from "./okf_projection_reset.js";
import { refreshEditorPreview, updateEditorFromJSON } from "./okf_profile_preview.js";
import { createOKFState, beginRequest, isCurrent, profileByID, refreshCatalogState, catalogDiagnostics } from "./okf_state.js";
import { diagnosticMarkup, escapeOKF, renderAccessibleItems, renderDetail } from "./okf_markup.js";
import { renderOKFGraph, updateOKFSelection, updateOKFSemanticLinks } from "./okf_graph.js";
import { changeOKFZoom, ensureOKFViewport, fitOKFViewport, persistOKFViewport, renderOKFViewportControls, resetOKFZoom } from "./okf_viewport.js";
import { syncFocusButton, toggleFocusMode } from "./viewport.js";
import { configureModeNavigation } from "./mode_navigation.js";
import { bindProfileForm, renderProfileForm, readProfileForm } from "./okf_profile_form.js";
import { cloneOKFLayoutProfile, OKF_DEFAULT_LAYOUT_ALGORITHM } from "./okf_layout.js";
import { renderLayoutForm, updateLayoutDraftAlgorithm, updateLayoutDraftOption, layoutDraftProblems } from "./layout_form.js";
import { sharedFeatureRegistry } from "./layout_features.js";

export function bootstrapOKF() {
  const sessionMeta = document.querySelector('meta[name="okf-session-id"]');
  const workerMeta = document.querySelector('meta[name="worker-url"]');
  const state = createOKFState(sessionMeta && sessionMeta.content);
  state.workerURL = workerMeta && workerMeta.content || "";
  const api = createOKFAPI();
  const elements = createOKFShell();
  elements.hiddenConcepts = document.getElementById("okf-hidden-concepts");
  void loadShapeChoices(elements.editorForm, api);
  configureModeNavigation("okf", false);
  document.body.classList.add("okf-mode");
  document.title = "OKF Knowledge View · Arch View";
  document.getElementById("project-label")?.replaceChildren(document.createTextNode("OKF bundle session"));
  document.getElementById("model-status")?.replaceChildren(document.createTextNode("Read-only"));
  document.getElementById("view-title")?.replaceChildren(document.createTextNode("OKF knowledge view"));
  document.getElementById("details-title")?.replaceChildren(document.createTextNode("Select a concept"));
  document.querySelector(".scene-heading h3")?.replaceChildren(document.createTextNode("Knowledge graph"));
  document.querySelector(".scene-help")?.replaceChildren(document.createTextNode("Legend: solid edges are explicit/filesystem containment; dashed edges are Markdown semantic links. Click a concept to inspect it without relayout; double-click to focus its subtree. Drag the canvas to pan; use Fit or the zoom controls to see the full graph."));

  const services = {
    load: () => load(state, api, elements, workerMeta && workerMeta.content, services),
    selectBundle: () => selectBundle(state, api, elements, services),
    selectProfile: () => selectProfile(state, api, elements, services),
    navigate: () => navigate(state, api, elements, services),
    focus: (conceptID) => focus(state, api, elements, services, conceptID),
    select: (conceptID) => selectOKFConcept(state, api, elements, conceptID, (error) => setStatus(elements, error.message, error.diagnostics || [], "error")),
    back: () => back(state, api, elements, services),
    edit: () => { openEditor(state, elements); return services.previewProfile(); },
    previewProfile: () => refreshEditorPreview(state, api, elements),
    renderHiddenConcepts: () => renderHiddenConcepts(document.getElementById("okf-hidden-concepts"), state, api, services.focus),
    bind: () => bindProfile(state, api, elements),
    rename: () => renameProfile(state, api, elements, services),
    remove: () => deleteProfile(state, api, elements, services),
    saveProfile: () => saveProfile(state, api, elements, services, false),
    saveAs: () => saveProfile(state, api, elements, services, true),
    applyLayout: () => applyOKFLayout(state, elements, services),
    saveLayout: () => saveOKFLayout(state, api, elements, services),
    saveLayoutAs: () => { openEditorForSaveAs(state, elements); return services.previewProfile(); }
  };
  bindEvents(state, elements, services);
  document.getElementById("download-svg")?.addEventListener("click", () => downloadCurrentSVG({ elements, exportLabel: "okf-" + (state.bundleID || "knowledge") }));
  void services.load();
  return { state, api, elements };
}

function createOKFShell() {
  const graphView = document.getElementById("graph-view");
  const shell = document.createElement("section");
  shell.className = "okf-shell";
  shell.setAttribute("aria-labelledby", "okf-view-title");
  shell.innerHTML = "<div class=\"okf-toolbar\"><label>Bundle<select id=\"okf-bundle\" aria-label=\"OKF bundle\"><option>Loading bundles…</option></select></label><label>Profile<select id=\"okf-profile\" aria-label=\"OKF view profile\"><option>Loading profiles…</option></select></label><label class=\"okf-depth-control\">Depth<input id=\"okf-depth\" type=\"number\" min=\"1\" value=\"2\" aria-label=\"Containment depth\"></label><label class=\"okf-full-control\"><span>Full</span><input id=\"okf-full\" type=\"checkbox\" aria-label=\"Show full containment scope\"></label><button id=\"okf-refresh\" class=\"button secondary\" type=\"button\">Refresh</button><button id=\"okf-edit\" class=\"button secondary\" type=\"button\">Edit profile</button></div><div id=\"okf-status\" class=\"okf-status-card\" data-tone=\"info\" role=\"status\" aria-live=\"polite\"><div class=\"okf-status-heading\"><strong id=\"okf-view-title\">OKF knowledge view</strong><span id=\"okf-counts\" class=\"muted\"></span></div><div id=\"okf-diagnostics\"></div></div><details class=\"okf-list-disclosure\"><summary><span>Concept list</span><span id=\"okf-accessible-count\" class=\"muted\"></span></summary><div id=\"okf-accessible\" class=\"okf-accessible-list\" aria-label=\"OKF concepts\"></div></details><div id=\"okf-hidden-concepts\"></div><dialog id=\"okf-profile-editor\"><div class=\"okf-editor-shell\"><div><p class=\"eyebrow\">PROJECT PROFILE</p><h3>Edit OKF presentation profile</h3><p id=\"okf-editor-status\" class=\"muted\">Built-in profiles are immutable; Save As creates a project-local copy.</p><p class=\"muted\">The form controls common presentation choices. Advanced JSON remains available for extensions and uncommon profile fields.</p></div><div id=\"okf-profile-form\"></div><details class=\"okf-advanced-json\"><summary>Advanced JSON</summary><p class=\"muted\">Unknown top-level fields are preserved when this profile is saved.</p><textarea id=\"okf-profile-json\" aria-label=\"Advanced profile JSON\" spellcheck=\"false\"></textarea></details><div class=\"okf-editor-actions\"><button id=\"okf-editor-close\" class=\"button secondary\" type=\"button\">Cancel</button><button id=\"okf-editor-bind\" class=\"button secondary\" type=\"button\">Bind bundle</button><button id=\"okf-editor-rename\" class=\"button secondary\" type=\"button\">Rename</button><button id=\"okf-editor-delete\" class=\"button secondary\" type=\"button\">Delete</button><button id=\"okf-editor-save-as\" class=\"button secondary\" type=\"button\">Save As</button><button id=\"okf-editor-save\" class=\"button\" type=\"button\">Save</button></div></div></dialog>";
  graphView?.prepend(shell);
  return {
    shell,
    bundle: shell.querySelector("#okf-bundle"), profile: shell.querySelector("#okf-profile"), depth: shell.querySelector("#okf-depth"), full: shell.querySelector("#okf-full"), refresh: shell.querySelector("#okf-refresh"), edit: shell.querySelector("#okf-edit"), status: shell.querySelector("#okf-status"), diagnostics: shell.querySelector("#okf-diagnostics"), counts: shell.querySelector("#okf-counts"), accessible: shell.querySelector("#okf-accessible"), accessibleCount: shell.querySelector("#okf-accessible-count"), editor: shell.querySelector("#okf-profile-editor"), editorForm: shell.querySelector("#okf-profile-form"), editorJSON: shell.querySelector("#okf-profile-json"), editorStatus: shell.querySelector("#okf-editor-status"), editorClose: shell.querySelector("#okf-editor-close"), editorSave: shell.querySelector("#okf-editor-save"), editorSaveAs: shell.querySelector("#okf-editor-save-as"), editorBind: shell.querySelector("#okf-editor-bind"), editorRename: shell.querySelector("#okf-editor-rename"), editorDelete: shell.querySelector("#okf-editor-delete"), zoomOut: document.getElementById("zoom-out"), zoomValue: document.getElementById("zoom-value"), zoomIn: document.getElementById("zoom-in"), resetZoom: document.getElementById("reset-zoom"), fitViewport: document.getElementById("fit-viewport"), resetLayout: document.getElementById("reset-layout"), layoutSettingsButton: document.getElementById("layout-settings-button"), focusToggle: document.getElementById("focus-toggle"), graph: document.getElementById("graph-wrap"), details: document.getElementById("details-content"), detailsTitle: document.getElementById("details-title"), back: document.getElementById("back-button"), breadcrumbs: document.getElementById("breadcrumbs"), summary: document.getElementById("summary"), layoutSettingsDialog: document.getElementById("layout-settings-dialog"), layoutSettingsClose: document.getElementById("layout-settings-close"), layoutSettingsOrigin: document.getElementById("layout-settings-origin"), layoutSettingsDiagnostic: document.getElementById("layout-settings-diagnostic"), layoutAlgorithm: document.getElementById("layout-algorithm"), layoutAlgorithmHelp: document.getElementById("layout-algorithm-help"), layoutOptionSearch: document.getElementById("layout-option-search"), layoutOptionsList: document.getElementById("layout-options-list"), layoutSaveAsDirectory: document.getElementById("layout-save-as-directory"), layoutSaveAsConfirm: document.getElementById("layout-save-as-confirm"), layoutSettingsStatus: document.getElementById("layout-settings-status"), layoutResetDefaults: document.getElementById("layout-reset-defaults"), layoutSettingsApply: document.getElementById("layout-settings-apply"), layoutSettingsSave: document.getElementById("layout-settings-save"), layoutSettingsSaveAs: document.getElementById("layout-settings-save-as")
  };
}

function bindEvents(state, elements, services) {
  elements.bundle.addEventListener("change", services.selectBundle);
  elements.profile.addEventListener("change", services.selectProfile);
  elements.depth.addEventListener("change", services.navigate);
  elements.full.addEventListener("change", services.navigate);
  elements.refresh.addEventListener("click", services.load);
  elements.edit.addEventListener("click", services.edit);
  elements.back?.addEventListener("click", services.back);
  elements.editorClose.addEventListener("click", () => closeEditor(state, elements));
  elements.editor.addEventListener("cancel", () => { state.editorProfile = null; state.editorAdvancedDirty = false; });
  elements.editorSave.addEventListener("click", services.saveProfile);
  elements.editorSaveAs.addEventListener("click", services.saveAs);
  elements.editorBind.addEventListener("click", services.bind);
  elements.editorRename.addEventListener("click", services.rename);
  elements.editorDelete.addEventListener("click", services.remove);
  bindProfileForm(elements.editorForm, function (kind) {
    state.editorAdvancedDirty = false;
    const valid = syncEditorProfileFromForm(state, elements);
    if (valid && kind === "bases") void services.previewProfile();
    if (kind === "layout") openOKFLayoutSettings(state, elements);
  });
  elements.editorJSON.addEventListener("input", function () {
    void updateEditorFromJSON(state, elements, services.previewProfile);
  });
  elements.zoomOut?.addEventListener("click", () => changeOKFZoom(state, elements, -0.12));
  elements.zoomIn?.addEventListener("click", () => changeOKFZoom(state, elements, 0.12));
  elements.resetZoom?.addEventListener("click", () => resetOKFZoom(state, elements));
  elements.fitViewport?.addEventListener("click", () => fitOKFViewport(state, elements));
  elements.resetLayout?.addEventListener("click", () => resetOKFLayout(state, elements, services));
  elements.layoutSettingsButton?.addEventListener("click", () => openOKFLayoutSettings(state, elements));
  elements.layoutSettingsClose?.addEventListener("click", () => closeOKFLayoutSettings(state, elements));
  elements.layoutSettingsDialog?.addEventListener("cancel", (event) => { event.preventDefault(); closeOKFLayoutSettings(state, elements); });
  elements.layoutAlgorithm?.addEventListener("change", (event) => updateOKFLayoutDraft(state, elements, updateLayoutDraftAlgorithm, event));
  elements.layoutOptionSearch?.addEventListener("input", (event) => {
    state.layoutOptionSearch = event.target.value.trim().toLowerCase();
    renderOKFLayoutForm(state, elements);
  });
  elements.layoutOptionsList?.addEventListener("input", (event) => updateOKFLayoutDraft(state, elements, updateLayoutDraftOption, event));
  elements.layoutOptionsList?.addEventListener("change", (event) => updateOKFLayoutDraft(state, elements, updateLayoutDraftOption, event));
  elements.layoutResetDefaults?.addEventListener("click", () => resetOKFLayoutDraft(state, elements));
  elements.layoutSettingsApply?.addEventListener("click", services.applyLayout);
  elements.layoutSettingsSave?.addEventListener("click", services.saveLayout);
  elements.layoutSettingsSaveAs?.addEventListener("click", services.saveLayoutAs);
  elements.focusToggle?.addEventListener("click", () => toggleFocusMode({ state, elements }));
  document.addEventListener("fullscreenchange", () => syncFocusButton({ state, elements }));
  syncFocusButton({ state, elements });
  elements.details?.addEventListener("click", (event) => {
    const link = event.target.closest("a[href^=\"#okf-concept=\"]");
    if (!link) return;
    event.preventDefault();
    services.select(decodeURIComponent(link.getAttribute("href").slice("#okf-concept=".length)));
  });
  state.busy = false;
}

async function load(state, api, elements, workerURL, services) {
  const request = beginRequest(state);
  setStatus(elements, "Discovering independent OKF bundles…", [], "info");
  try {
    const catalog = await refreshCatalogState(state, api, request);
    if (!catalog || !isCurrent(state, request)) return;
    const diagnostics = catalogDiagnostics(catalog);
    state.configurationRevision = state.profiles.configuration_revision || state.configurationRevision;
    renderSelectors(state, elements);
    if (!catalog.default_bundle_id) {
      clearOKFProjection(state, elements);
      setStatus(elements, "No valid OKF bundle is available.", diagnostics, "warning");
      return;
    }
    const selectedBundle = (catalog.bundles || []).find((bundle) => bundle.bundle_id === state.bundleID);
    if (state.bundleID && !selectedBundle?.selectable) {
      clearOKFProjection(state, elements);
      setStatus(elements, "The selected bundle is unavailable. Choose another bundle or repair its source and refresh.", diagnostics, "warning");
      return;
    }
    const session = await api.selectBundle(state.sessionID, elements.bundle.value || catalog.default_bundle_id);
    if (!isCurrent(state, request)) return;
    clearSelection(state, elements);
    applySession(state, session);
    await renderState(state, elements, workerURL, services, [], request);
  } catch (error) {
    if (!isCurrent(state, request)) return;
    state.error = error.message || "OKF discovery failed.";
    setStatus(elements, state.error, error.diagnostics || [], "error");
  }
}

function renderSelectors(state, elements) {
  const bundles = state.catalog && Array.isArray(state.catalog.bundles) ? state.catalog.bundles : [];
  elements.bundle.innerHTML = bundles.map((value) => "<option value=\"" + escapeOKF(value.bundle_id) + "\"" + (value.bundle_id === (state.bundleID || state.catalog.default_bundle_id) ? " selected" : "") + (value.selectable ? "" : " disabled") + ">" + escapeOKF(value.relative_path || value.bundle_id) + (value.status === "valid" ? "" : " · " + escapeOKF(value.status)) + "</option>").join("") || "<option value=\"\">No bundles</option>";
  const profiles = state.profiles && Array.isArray(state.profiles.profiles) ? state.profiles.profiles : [];
  elements.profile.innerHTML = profiles.map((value) => "<option value=\"" + escapeOKF(value.profile_id) + "\"" + (value.profile_id === state.profileID ? " selected" : "") + ">" + escapeOKF(value.name || value.profile_id) + (value.immutable ? " · built-in" : "") + "</option>").join("") || "<option value=\"builtin:neutral\">Neutral</option>";
  elements.depth.value = String(state.depth || 2);
  elements.full.checked = Boolean(state.full);
}

async function selectBundle(state, api, elements, services) {
  if (!elements.bundle.value) return;
  const request = beginRequest(state);
  try { const value = await api.selectBundle(state.sessionID, elements.bundle.value); if (!isCurrent(state, request)) return; state.layoutOverride = null; applySession(state, value); clearSelection(state, elements); await renderState(state, elements, state.workerURL, services, [], request); }
  catch (error) { if (isCurrent(state, request)) setStatus(elements, error.message, error.diagnostics || [], "error"); }
}

async function selectProfile(state, api, elements, services) {
  if (!elements.profile.value) return;
  const request = beginRequest(state);
  try { const value = await api.selectProfile(state.sessionID, elements.profile.value); if (!isCurrent(state, request)) return; state.layoutOverride = null; applySession(state, value); clearSelection(state, elements); await renderState(state, elements, state.workerURL, services, [], request); }
  catch (error) { if (isCurrent(state, request)) setStatus(elements, error.message, error.diagnostics || [], "error"); }
}

async function navigate(state, api, elements, services) {
  let depth;
  try { depth = readNumericInput(elements.depth, 2, true); }
  catch (error) { setStatus(elements, error.message, [], "error"); return; }
  const request = beginRequest(state);
  try { const value = await api.setDepth(state.sessionID, depth, elements.full.checked); if (!isCurrent(state, request)) return; applySession(state, value); clearSelection(state, elements); await renderState(state, elements, state.workerURL, services, [], request); }
  catch (error) { if (isCurrent(state, request)) setStatus(elements, error.message, error.diagnostics || [], "error"); }
}

async function focus(state, api, elements, services, conceptID) {
  const request = beginRequest(state);
  if (conceptID === "") {
    try { const value = await api.topLevel(state.sessionID); if (!isCurrent(state, request)) return; applySession(state, value); clearSelection(state, elements); await renderState(state, elements, state.workerURL, services, [], request); }
    catch (error) { if (isCurrent(state, request)) setStatus(elements, error.message, error.diagnostics || [], "error"); }
    return;
  }
  try { const value = await api.focus(state.sessionID, conceptID); if (!isCurrent(state, request)) return; applySession(state, value); clearSelection(state, elements); await renderState(state, elements, state.workerURL, services, [], request); if (isCurrent(state, request)) await services.select(conceptID); }
  catch (error) { if (isCurrent(state, request)) setStatus(elements, error.message, error.diagnostics || [], "error"); }
}

async function back(state, api, elements, services) {
  const request = beginRequest(state);
  try { const value = await api.back(state.sessionID); if (!isCurrent(state, request)) return; applySession(state, value); clearSelection(state, elements); await renderState(state, elements, state.workerURL, services, [], request); }
  catch (error) { if (isCurrent(state, request)) setStatus(elements, error.message, error.diagnostics || [], "error"); }
}

function applySession(state, value) {
  const snapshot = value && value.projection ? value.projection : (value && value.status && value.nodes ? value : null);
  if (snapshot) {
    state.snapshot = snapshot;
    state.bundleID = snapshot.source?.bundle_id || state.bundleID;
    state.profileID = snapshot.profile?.profile_id || state.profileID;
    state.depth = snapshot.navigation?.depth || state.depth;
    state.full = Boolean(snapshot.navigation?.full);
    state.layoutProfile = state.layoutOverride ? cloneOKFLayoutProfile(state.layoutOverride) : cloneOKFLayoutProfile(snapshot.profile?.layout);
    if (state.layoutOverride && state.snapshot.profile) state.snapshot.profile.layout = cloneOKFLayoutProfile(state.layoutOverride);
  }
  if (value && value.bundle_id) state.bundleID = value.bundle_id;
  if (value && value.profile_id) state.profileID = value.profile_id;
}

function clearSelection(state, elements) {
  state.selectedID = "";
  state.detail = null;
  renderDetail(elements.details, null);
  if (elements.detailsTitle) elements.detailsTitle.textContent = "Select a concept";
}

async function renderState(state, elements, workerURL, services, extraDiagnostics, requestID) {
  const snapshot = state.snapshot;
  if (!snapshot) return;
  if (requestID != null && !isCurrent(state, requestID)) return;
  ensureOKFViewport(state, snapshot);
  renderSelectors(state, elements);
  const diagnostics = catalogDiagnostics(state.catalog).concat(extraDiagnostics || [], snapshot.diagnostics || [],
    sharedFeatureRegistry(state.layoutCatalog).negotiate(state.layoutProfile || {}).diagnostics);
  const tone = diagnostics.some((item) => item.severity === "error") ? "error" : diagnostics.length ? "warning" : "info";
  setStatus(elements, snapshot.status === "truncated" ? "Projection is visible but truncated by safety limits." : "Showing " + (snapshot.source?.bundle_id || "selected bundle"), diagnostics, tone);
  const counts = snapshot.counts || {};
  elements.counts.textContent = (counts.visible_nodes || 0) + " visible concepts · " + (counts.hidden_nodes || 0) + " hidden · " + (counts.visible_relationships || 0) + " relationships";
  elements.summary.innerHTML = "<article class=\"summary-card\"><span class=\"eyebrow\">OKF SOURCE</span><strong>" + escapeOKF(snapshot.source?.bundle_id || "—") + "</strong><span class=\"muted\">" + escapeOKF(snapshot.profile?.profile_id || "builtin:neutral") + " · " + escapeOKF(snapshot.status || "ready") + "</span></article>";
  const layout = await renderOKFGraph(elements.graph, snapshot, state.selectedID, { onSelect: services.select, onFocus: services.focus, onZoom: (delta) => changeOKFZoom(state, elements, delta), layoutProfile: state.layoutProfile, layoutCatalog: state.layoutCatalog, getViewport: () => state.viewport, onViewportChange: () => persistOKFViewport(state), onViewportRender: () => renderOKFViewportControls(state, elements), onLayoutReady: (value) => { state.layout = value; }, isCurrent: () => requestID == null || isCurrent(state, requestID), onLayoutError: (error, recovery) => { if (requestID == null || isCurrent(state, requestID)) setStatus(elements, recovery.retained ? "Layout failed; the previous layout is retained." : "Layout failed; deterministic fallback is active.", diagnostics.concat({ code: "okf_layout_failed", severity: "warning", message: error.message }), tone === "error" ? "error" : "warning"); } }, workerURL || state.workerURL);
  if (requestID != null && !isCurrent(state, requestID)) return;
  if (layout) state.layout = layout;
  updateOKFSelection(elements.graph, state.selectedID);
  updateOKFSemanticLinks(elements.graph, snapshot, state.selectedID, state.layout, state.viewport);
  if (!state.viewportInitialized) fitOKFViewport(state, elements);
  renderOKFViewportControls(state, elements);
  renderAccessibleItems(elements.accessible, snapshot, services.select);
  services.renderHiddenConcepts();
  if (elements.accessibleCount) elements.accessibleCount.textContent = (snapshot.counts?.visible_nodes || snapshot.nodes?.length || 0) + " visible";
  renderBreadcrumbs(elements.breadcrumbs, snapshot.navigation?.breadcrumbs || [], services.focus);
  if (elements.back) elements.back.disabled = !snapshot.navigation?.can_go_back;
}

function renderBreadcrumbs(element, values, focus) {
  if (!element) return;
  if (!values.length) {
    element.innerHTML = '<span class="breadcrumb current" aria-current="page">Top level</span>';
    return;
  }
  const crumbs = ['<button class="breadcrumb" type="button" data-okf-focus="">Top level</button><span class="breadcrumb-separator" aria-hidden="true">/</span>'];
  values.forEach((value, index) => {
    const current = index === values.length - 1;
    if (index) crumbs.push('<span class="breadcrumb-separator" aria-hidden="true">/</span>');
    crumbs.push(current ? '<span class="breadcrumb current" aria-current="page">' + escapeOKF(value) + '</span>' : '<button class="breadcrumb" type="button" data-okf-focus="' + escapeOKF(value) + '">' + escapeOKF(value) + '</button>');
  });
  element.innerHTML = crumbs.join("");
  element.querySelectorAll("[data-okf-focus]").forEach((button) => button.addEventListener("click", () => focus(button.dataset.okfFocus)));
}

function setStatus(elements, message, diagnostics, tone) {
  elements.status.dataset.tone = tone || "info";
  const list = Array.isArray(diagnostics) ? diagnostics : [];
  elements.diagnostics.innerHTML = diagnosticMarkup(list);
  elements.diagnostics.hidden = list.length === 0;
  const heading = elements.status.querySelector("#okf-view-title");
  if (heading && message) heading.textContent = message;
}

function selectedProfile(state) {
  return profileByID(state.profiles, state.profileID) || { profile_id: state.profileID || "project:custom", name: "Custom", origin: "project_local", bases: ["builtin:neutral"], layout: { algorithm: OKF_DEFAULT_LAYOUT_ALGORITHM, options: {} } };
}

function copyProfile(value) {
  return JSON.parse(JSON.stringify(value || {}));
}

function renderEditorForm(state, elements) {
  if (!elements.editorForm || !state.editorProfile) return;
  renderProfileForm(elements.editorForm, state.editorProfile);
}

function syncEditorProfileFromForm(state, elements) {
  if (!elements.editorForm || !state.editorProfile) return;
  try {
    state.editorProfile = readProfileForm(elements.editorForm, state.editorProfile);
    state.editorProfile.layout = Object.assign({}, state.editorProfile.layout, cloneOKFLayoutProfile(state.editorProfile.layout));
    elements.editorJSON.value = JSON.stringify(state.editorProfile, null, 2);
    return true;
  } catch (error) {
    elements.editorStatus.textContent = error.message;
    return false;
  }
}

function openEditor(state, elements) {
  state.editorProfile = copyProfile(selectedProfile(state));
  state.editorProfile.layout = cloneOKFLayoutProfile(state.layoutProfile);
  state.editorAdvancedDirty = false;
  renderEditorForm(state, elements);
  elements.editorJSON.value = JSON.stringify(state.editorProfile, null, 2);
  elements.editorStatus.textContent = state.editorProfile.immutable ? "This built-in is immutable; use Save As for a project-local copy." : "Validate before saving; failed writes leave the previous configuration intact.";
  elements.editorSave.disabled = Boolean(state.editorProfile.immutable);
  elements.editorRename.disabled = Boolean(state.editorProfile.immutable);
  elements.editorDelete.disabled = Boolean(state.editorProfile.immutable);
  if (typeof elements.editor.showModal === "function") elements.editor.showModal(); else elements.editor.setAttribute("open", "");
}

function editorProfileValue(state, elements) {
  if (state.editorPreviewPending) throw new Error("Inherited settings have not finished resolving.");
  if (state.editorAdvancedDirty) return JSON.parse(elements.editorJSON.value);
  if (!syncEditorProfileFromForm(state, elements)) throw new Error("Invalid profile form");
  return copyProfile(state.editorProfile);
}

async function validateEditorProfile(state, api, elements, value) {
  const result = await api.validateProfile(value);
  if (!result || result.valid !== true) {
    const error = new Error("Profile validation failed.");
    error.diagnostics = result && result.diagnostics || [];
    throw error;
  }
  return result.profile || value;
}

function saveProfile(state, api, elements, services, saveAs) {
  return saveEditorProfile(state, api, elements, services, saveAs, {
    read: () => editorProfileValue(state, elements),
    validate: (value) => validateEditorProfile(state, api, elements, value),
    renderSelectors: () => renderSelectors(state, elements)
  });
}

function okfLayoutContext(state, elements) {
  return { state, elements, layoutFormOptions: { showPersistence: false, originLabel: "Selected OKF profile · session draft" } };
}

function renderOKFLayoutForm(state, elements) {
  const context = okfLayoutContext(state, elements);
  renderLayoutForm(context, { origin: "session", status: "valid", can_save: false, can_save_as: false, diagnostics: [] }, context.layoutFormOptions);
}

function openOKFLayoutSettings(state, elements) {
  const profile = state.editorProfile || selectedProfile(state);
  state.layoutProfile = cloneOKFLayoutProfile(state.editorProfile ? profile.layout : state.layoutProfile || profile.layout || state.snapshot?.profile?.layout);
  state.layoutDraft = cloneOKFLayoutProfile(state.layoutProfile);
  state.layoutSettingsOpen = true;
  state.layoutMessage = "";
  state.layoutMessageError = false;
  renderOKFLayoutForm(state, elements);
  if (typeof elements.layoutSettingsDialog?.showModal === "function") elements.layoutSettingsDialog.showModal();
  else elements.layoutSettingsDialog?.setAttribute("open", "");
}

function closeOKFLayoutSettings(state, elements) {
  state.layoutSettingsOpen = false;
  state.layoutMessage = "";
  state.layoutMessageError = false;
  if (typeof elements.layoutSettingsDialog?.close === "function") elements.layoutSettingsDialog.close();
  else elements.layoutSettingsDialog?.removeAttribute("open");
}

function updateOKFLayoutDraft(state, elements, updater, event) {
  updater(okfLayoutContext(state, elements), event);
}

function resetOKFLayoutDraft(state, elements) {
  state.layoutDraft = { algorithm: OKF_DEFAULT_LAYOUT_ALGORITHM, options: {} };
  state.layoutMessage = "Built-in defaults restored for this session";
  state.layoutMessageError = false;
  renderOKFLayoutForm(state, elements);
}

async function applyOKFLayout(state, elements, services) {
  if (!state.layoutDraft || !state.snapshot) return;
  if (layoutDraftProblems(state.layoutDraft, state.layoutCatalog).length) { renderOKFLayoutForm(state, elements); return; }
  const request = beginRequest(state);
  state.layoutProfile = cloneOKFLayoutProfile(state.layoutDraft);
  state.layoutOverride = cloneOKFLayoutProfile(state.layoutProfile);
  state.snapshot.profile.layout = cloneOKFLayoutProfile(state.layoutProfile);
  if (state.editorProfile) state.editorProfile.layout = cloneOKFLayoutProfile(state.layoutProfile);
  if (state.viewport) state.viewport.positions = {};
  state.viewportInitialized = false;
  persistOKFViewport(state);
  closeOKFLayoutSettings(state, elements);
  await renderState(state, elements, state.workerURL, services, [], request);
}

function openEditorForSaveAs(state, elements) {
  closeOKFLayoutSettings(state, elements);
  if (!state.editorProfile) openEditor(state, elements);
  if (state.layoutDraft) {
    state.editorProfile.layout = cloneOKFLayoutProfile(state.layoutDraft);
    renderEditorForm(state, elements);
    elements.editorJSON.value = JSON.stringify(state.editorProfile, null, 2);
  }
}

async function saveOKFLayout(state, api, elements, services) {
  if (!state.layoutDraft) return;
  closeOKFLayoutSettings(state, elements);
  if (!state.editorProfile) openEditor(state, elements);
  state.editorProfile.layout = cloneOKFLayoutProfile(state.layoutDraft);
  renderEditorForm(state, elements);
  elements.editorJSON.value = JSON.stringify(state.editorProfile, null, 2);
  const draft = state.editorProfile;
  await services.previewProfile();
  if (state.editorProfile !== draft) return;
  await saveProfile(state, api, elements, services, Boolean(draft.immutable));
}

function resetOKFLayout(state, elements, services) {
  if (!state.viewport) return;
  state.viewport.positions = {};
  persistOKFViewport(state);
  renderOKFViewportControls(state, elements);
  if (state.snapshot) void renderState(state, elements, state.workerURL, services, [], beginRequest(state));
}

function closeEditor(state, elements) {
  elements.editor.close();
  state.editorProfile = null;
  state.editorAdvancedDirty = false;
}

function bindProfile(state, api, elements) {
  return changeProfileConfiguration(state, elements, {
    write: ({ bundleID, profileID, revision }) => api.bind(bundleID, profileID, revision, "okf-bind-" + globalThis.crypto.randomUUID()),
    prepare: () => loadConfigurationCatalogs(api),
    publish: (catalogs) => {
      Object.assign(state, catalogs);
      elements.editorStatus.textContent = "Bundle binding saved.";
    }
  });
}

function renameProfile(state, api, elements, services) {
  return renameEditorProfile(state, api, elements, services, {
    read: () => editorProfileValue(state, elements),
    renderSelectors: () => renderSelectors(state, elements),
    close: () => closeEditor(state, elements)
  });
}

function deleteProfile(state, api, elements, services) {
  return deleteEditorProfile(state, api, elements, services, {
    confirm: confirmAction,
    renderSelectors: () => renderSelectors(state, elements),
    close: () => closeEditor(state, elements)
  });
}
