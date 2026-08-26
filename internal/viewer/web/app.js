(function () {
  "use strict";

  const modelID = document.querySelector('meta[name="model-id"]').content;
  const embeddedExport = window.__ARCH_VIEW_EXPORT__ || null;
  const sourceEnabled = document.querySelector('meta[name="source-enabled"]').content === "true";
  const reanalysisEnabled = document.querySelector('meta[name="reanalysis-enabled"]').content === "true";
  const MIN_ZOOM = 0.35;
  // Interaction guardrail only; Fit uses a separate, conservative ceiling.
  // Keep this finite because extreme SVG transform values lose precision, but
  // make it effectively unlimited for normal interactive use.
  const MAX_ZOOM = 40000;
  // One screen pixel of pointer movement should represent one screen pixel of
  // canvas movement, regardless of the SVG viewBox scale.
  const PAN_SPEED = 1;
  const PAN_LIMIT = 100000;
  const NODE_DOUBLE_CLICK_WINDOW = 450;
  const NODE_DOUBLE_CLICK_DISTANCE = 12;
  const WINDOWED_FIT_ZOOM = 1.45;
  const EXPANDED_FIT_ZOOM = 1;
  const state = {
    model: null,
    scene: null,
    selected: null,
    query: "",
    referenceVisibility: "hidden",
    importScope: "all",
    layout: null,
    layoutProfile: { algorithm: "layered", options: {} },
    layoutCatalog: null,
    layoutConfig: null,
    layoutDraft: null,
    layoutOptionSearch: "",
    layoutMessage: "",
    layoutMessageError: false,
    layoutSettingsOpen: false,
    layoutConfigRequest: 0,
    layoutKey: "",
    layoutRequest: 0,
    layoutError: false,
    viewport: null,
    draggingNode: null,
    dragFrame: 0,
    history: [],
    scrollContexts: {},
    sceneRequest: 0,
    source: null,
    sourceRequest: 0,
    suppressClickUntil: 0,
    lastNodeClick: null,
    nativeFullscreen: false
  };

  const elements = {
    projectLabel: document.getElementById("project-label"),
    modelStatus: document.getElementById("model-status"),
    viewDescription: document.getElementById("view-description"),
    summary: document.getElementById("summary"),
    errorBanner: document.getElementById("error-banner"),
    sceneState: document.getElementById("scene-state"),
    layerLegend: document.getElementById("layer-legend"),
    referenceSummary: document.getElementById("reference-summary"),
    graph: document.getElementById("graph-wrap"),
    detailsTitle: document.getElementById("details-title"),
    detailsKind: document.getElementById("details-kind"),
    detailsContent: document.getElementById("details-content"),
    accessibleList: document.getElementById("accessible-list"),
    listCount: document.getElementById("list-count"),
    listToggle: document.getElementById("list-toggle"),
    cycles: document.getElementById("cycles"),
    cycleCount: document.getElementById("cycle-count"),
    diagnostics: document.getElementById("diagnostics"),
    diagnosticCount: document.getElementById("diagnostic-count"),
    footerModelID: document.getElementById("footer-model-id"),
    search: document.getElementById("search"),
    referenceVisibility: document.getElementById("reference-visibility"),
    reanalysisButton: document.getElementById("reanalysis-button"),
    backButton: document.getElementById("back-button"),
    breadcrumbs: document.getElementById("breadcrumbs"),
    focusToggle: document.getElementById("focus-toggle"),
    zoomOut: document.getElementById("zoom-out"),
    zoomIn: document.getElementById("zoom-in"),
    zoomValue: document.getElementById("zoom-value"),
    resetZoom: document.getElementById("reset-zoom"),
    fitViewport: document.getElementById("fit-viewport"),
    resetLayout: document.getElementById("reset-layout"),
    layoutSettingsButton: document.getElementById("layout-settings-button"),
    layoutSettingsDialog: document.getElementById("layout-settings-dialog"),
    layoutSettingsClose: document.getElementById("layout-settings-close"),
    layoutSettingsOrigin: document.getElementById("layout-settings-origin"),
    layoutSettingsDiagnostic: document.getElementById("layout-settings-diagnostic"),
    layoutAlgorithm: document.getElementById("layout-algorithm"),
    layoutAlgorithmHelp: document.getElementById("layout-algorithm-help"),
    layoutOptionSearch: document.getElementById("layout-option-search"),
    layoutOptionsList: document.getElementById("layout-options-list"),
    layoutSaveAsDirectory: document.getElementById("layout-save-as-directory"),
    layoutSaveAsConfirm: document.getElementById("layout-save-as-confirm"),
    layoutSettingsStatus: document.getElementById("layout-settings-status"),
    layoutResetDefaults: document.getElementById("layout-reset-defaults"),
    layoutSettingsApply: document.getElementById("layout-settings-apply"),
    layoutSettingsSave: document.getElementById("layout-settings-save"),
    layoutSettingsSaveAs: document.getElementById("layout-settings-save-as")
  };

  function escapeHTML(value) {
    return String(value == null ? "" : value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function truncate(value, length) {
    const text = String(value || "");
    return text.length > length ? text.slice(0, length - 1) + "…" : text;
  }

  function classForState(value) {
    return String(value || "none").replace(/[^a-z0-9_-]/gi, "-");
  }

  function formatList(values, empty) {
    if (!values || values.length === 0) return empty || "None";
    return values.join(", ");
  }

  function showError(message) {
    elements.errorBanner.textContent = message;
    elements.errorBanner.hidden = false;
  }

  function hideError() {
    elements.errorBanner.hidden = true;
    elements.errorBanner.textContent = "";
  }

  async function getJSON(path) {
    if (embeddedExport) return embeddedJSON(path);
    const response = await fetch(path, { headers: { Accept: "application/json" } });
    const body = await response.json();
    if (!response.ok) {
      throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    }
    return body;
  }

  function embeddedJSON(path) {
    const url = new URL(path, window.location.href);
    const modelPath = "/v1/models/" + encodeURIComponent(modelID);
    if (url.pathname === modelPath) return embeddedExport.model;
    if (url.pathname === modelPath + "/projection") {
      const pathValue = url.searchParams.getAll("path");
      const visibility = url.searchParams.get("reference_visibility") || "hidden";
      const key = JSON.stringify(pathValue) + "|" + visibility;
      if (embeddedExport.scenes && embeddedExport.scenes[key]) return embeddedExport.scenes[key];
      throw new Error("The exported hierarchy path or reference view is unavailable.");
    }
    if (url.pathname === "/v1/layout/config") return { schema_version: "arch-view.config/v1", layout: { algorithm: "layered", options: {} }, origin: "session", status: "valid", can_save: false, can_save_as: false, diagnostics: [] };
    if (url.pathname === "/v1/layout/options") return { schema_version: "arch-view.config/v1", adapter: { id: "export", version: "embedded", source: "export" }, algorithms: [], categories: [], options: [] };
    throw new Error("The self-contained export does not require network access.");
  }

  async function postJSON(path, value) {
    const response = await fetch(path, {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(value)
    });
    const body = await response.json();
    if (!response.ok) {
      throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    }
    return body;
  }

  async function putJSON(path, value) {
    const response = await fetch(path, {
      method: "PUT",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(value)
    });
    const body = await response.json();
    if (!response.ok) {
      throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    }
    return body;
  }

  function currentModelID() {
    if (state.model && state.model.model_id) return state.model.model_id;
    if (state.scene && state.scene.model_id) return state.scene.model_id;
    return modelID;
  }

  function init() {
    elements.footerModelID.textContent = modelID;
    elements.reanalysisButton.hidden = !reanalysisEnabled;
    elements.listToggle.addEventListener("click", function () {
      const pressed = elements.listToggle.getAttribute("aria-pressed") === "true";
      elements.listToggle.setAttribute("aria-pressed", String(!pressed));
      document.body.classList.toggle("list-mode", !pressed);
    });
    elements.search.addEventListener("input", function (event) {
      state.query = event.target.value.trim().toLowerCase();
      renderGraph();
      renderAccessibleList();
    });
    elements.referenceVisibility.addEventListener("change", function (event) {
      state.referenceVisibility = event.target.value;
      state.selected = null;
      loadScene(state.scene ? state.scene.hierarchy_path : []);
    });
    elements.backButton.addEventListener("click", goBack);
    elements.zoomOut.addEventListener("click", function () { changeZoom(-0.12); });
    elements.zoomIn.addEventListener("click", function () { changeZoom(0.12); });
    elements.resetZoom.addEventListener("click", resetZoom);
    elements.fitViewport.addEventListener("click", fitViewport);
    elements.resetLayout.addEventListener("click", resetLayout);
    elements.layoutSettingsButton.hidden = Boolean(embeddedExport);
    elements.layoutSettingsButton.addEventListener("click", openLayoutSettings);
    elements.layoutSettingsClose.addEventListener("click", closeLayoutSettings);
    elements.layoutSettingsDialog.addEventListener("cancel", function (event) { event.preventDefault(); closeLayoutSettings(); });
    elements.layoutAlgorithm.addEventListener("change", function (event) {
      if (!state.layoutDraft) state.layoutDraft = cloneLayoutProfile(state.layoutProfile);
      state.layoutDraft.algorithm = event.target.value;
      if (state.layoutCatalog && state.layoutDraft.options) {
        Object.keys(state.layoutDraft.options).forEach(function (optionID) {
          const option = state.layoutCatalog.options.find(function (item) { return item.id === optionID; });
          if (option && !optionAppliesToAlgorithm(option, state.layoutDraft.algorithm)) delete state.layoutDraft.options[optionID];
        });
      }
      state.layoutMessage = "Unsaved settings changes";
      state.layoutMessageError = false;
      renderLayoutSettings();
    });
    elements.layoutOptionSearch.addEventListener("input", function (event) {
      state.layoutOptionSearch = event.target.value.trim().toLowerCase();
      renderLayoutSettings();
    });
    elements.layoutOptionsList.addEventListener("input", updateLayoutDraftOption);
    elements.layoutOptionsList.addEventListener("change", updateLayoutDraftOption);
    elements.layoutResetDefaults.addEventListener("click", resetLayoutProfile);
    elements.layoutSettingsApply.addEventListener("click", applyLayoutProfile);
    elements.layoutSettingsSave.addEventListener("click", saveLayoutProfile);
    elements.layoutSettingsSaveAs.addEventListener("click", saveLayoutProfileAs);
    elements.focusToggle.addEventListener("click", toggleFocusMode);
    document.addEventListener("fullscreenchange", syncFocusButton);
    elements.reanalysisButton.addEventListener("click", reanalyze);
    if (embeddedExport && embeddedExport.initial_reference_visibility) {
      state.referenceVisibility = embeddedExport.initial_reference_visibility;
    }
    loadModel();
    loadLayoutConfig();
    loadScene(embeddedExport && Array.isArray(embeddedExport.initial_path) ? embeddedExport.initial_path : []);
  }

  async function loadModel() {
    const requestedModelID = currentModelID();
    try {
      const value = await getJSON("/v1/models/" + encodeURIComponent(requestedModelID));
      if (requestedModelID !== currentModelID()) return;
      state.model = value;
      if (state.scene) renderDetails();
    } catch (error) {
      showError(error.message || "The canonical model could not be loaded.");
    }
  }

  function cloneLayoutProfile(profile) {
    const value = profile || {};
    return {
      algorithm: value.algorithm || "layered",
      options: Object.assign({}, value.options || {})
    };
  }

  function layoutRequestPayload(profile) {
    return { schema_version: "arch-view.config/v1", layout: cloneLayoutProfile(profile) };
  }

  async function loadLayoutConfig() {
    if (embeddedExport) return;
    const request = ++state.layoutConfigRequest;
    try {
      const values = await Promise.all([getJSON("/v1/layout/options"), getJSON("/v1/layout/config")]);
      if (request !== state.layoutConfigRequest) return;
      state.layoutCatalog = values[0];
      state.layoutConfig = values[1];
      state.layoutProfile = cloneLayoutProfile(values[1].layout);
      if (!state.layoutSettingsOpen) state.layoutDraft = cloneLayoutProfile(state.layoutProfile);
      if (state.layoutSettingsOpen) renderLayoutSettings();
      if (state.scene) {
        state.layout = null;
        state.layoutRequest += 1;
        renderSceneState();
        renderGraph();
        void prepareLayout(state.scene, state.layoutProfile);
      }
    } catch (error) {
      if (request !== state.layoutConfigRequest) return;
      state.layoutConfig = { layout: cloneLayoutProfile(state.layoutProfile), origin: "session", status: "invalid", can_save: false, can_save_as: false, diagnostics: [{ code: "config_unavailable", severity: "error", message: error.message || "Layout settings could not be loaded." }] };
      state.layoutMessage = error.message || "Layout settings could not be loaded.";
      state.layoutMessageError = true;
      renderLayoutSettings();
    }
  }

  function layoutOriginLabel(config) {
    if (!config) return "Loading layout profile…";
    const origin = config.origin || "default";
    const labels = { default: "Built-in defaults", project: "Project .archview.json", ancestor: "Ancestor .archview.json", custom: "Custom .archview.json", session: "Current session (unsaved)" };
    const label = labels[origin] || origin;
    return config.active_path ? label + " · " + config.active_path : label;
  }

  function renderLayoutSettingsDiagnostic(config) {
    const diagnostics = config && config.diagnostics ? config.diagnostics : [];
    if (!diagnostics.length) {
      elements.layoutSettingsDiagnostic.hidden = true;
      elements.layoutSettingsDiagnostic.textContent = "";
      return;
    }
    elements.layoutSettingsDiagnostic.hidden = false;
    elements.layoutSettingsDiagnostic.className = "layout-settings-diagnostic " + (diagnostics.some(function (item) { return item.severity === "error"; }) ? "error" : "info");
    elements.layoutSettingsDiagnostic.innerHTML = diagnostics.map(function (item) {
      return "<strong>" + escapeHTML(item.code || "diagnostic") + "</strong> " + escapeHTML(item.message || "") + (item.path ? " <code>" + escapeHTML(item.path) + "</code>" : "");
    }).join("<br>");
  }

  function optionAppliesToAlgorithm(option, algorithm) {
    const algorithms = option.algorithms || [];
    return !algorithms.length || algorithms.includes("all") || algorithms.includes(algorithm);
  }

  function optionDisplayValue(option, draft) {
    if (draft && draft.options && Object.prototype.hasOwnProperty.call(draft.options, option.id)) {
      return String(draft.options[option.id]);
    }
    return "";
  }

  function optionDefaultText(option) {
    if (option.default === "engine default") return "engine default";
    return formatOptionValue(option.default);
  }

  function formatOptionValue(value) {
    if (value == null) return "—";
    if (typeof value === "string") return value;
    return JSON.stringify(value);
  }

  function optionBoundsText(option) {
    const bounds = [];
    if (option.minimum != null) bounds.push((option.minimum_exclusive ? "> " : "≥ ") + formatOptionValue(option.minimum));
    if (option.maximum != null) bounds.push((option.maximum_exclusive ? "< " : "≤ ") + formatOptionValue(option.maximum));
    return bounds.length ? " · bounds: " + bounds.join(", ") : "";
  }

  function layoutOptionInput(option, draft, applicable) {
    if (!applicable) return '<span class="layout-option-state">Not applicable to this algorithm</span>';
    if (!option.editable || option.renderer_support !== "supported") return '<span class="layout-option-state">Cataloged · unsupported by this renderer</span>';
    const value = optionDisplayValue(option, draft);
    const id = "layout-option-" + option.id.replace(/[^a-z0-9_-]/gi, "-");
    const attributes = 'data-layout-option-id="' + escapeHTML(option.id) + '" aria-label="' + escapeHTML(option.name) + '"';
    if (option.allowed_values && option.allowed_values.length) {
      const choices = ['<option value="">Engine default</option>'].concat(option.allowed_values.map(function (choice) {
        const text = String(choice);
        return '<option value="' + escapeHTML(text) + '"' + (value === text ? " selected" : "") + '>' + escapeHTML(text) + '</option>';
      }));
      return '<select id="' + id + '" class="layout-option-input" ' + attributes + '>' + choices.join("") + "</select>";
    }
    if (option.type === "BOOLEAN") {
      return '<select id="' + id + '" class="layout-option-input" ' + attributes + '><option value="">Engine default</option><option value="true"' + (value === "true" ? " selected" : "") + '>true</option><option value="false"' + (value === "false" ? " selected" : "") + '>false</option></select>';
    }
    if (option.type === "INT" || option.type === "DOUBLE") {
      const step = option.type === "INT" ? "1" : "any";
      const min = option.minimum == null ? "" : ' min="' + escapeHTML(option.minimum) + '"';
      const max = option.maximum == null ? "" : ' max="' + escapeHTML(option.maximum) + '"';
      return '<input id="' + id + '" class="layout-option-input" type="number" step="' + step + '" value="' + escapeHTML(value) + '" placeholder="Engine default"' + min + max + " " + attributes + ">";
    }
    return '<input id="' + id + '" class="layout-option-input" type="text" value="' + escapeHTML(value) + '" placeholder="Engine default" ' + attributes + ">";
  }

  function renderLayoutSettings() {
    if (!elements.layoutSettingsDialog) return;
    const config = state.layoutConfig;
    const catalog = state.layoutCatalog;
    elements.layoutSettingsOrigin.textContent = layoutOriginLabel(config);
    renderLayoutSettingsDiagnostic(config);
    elements.layoutSettingsStatus.textContent = state.layoutMessage || (config && config.status === "invalid" ? "Safe defaults are active until the profile is corrected." : "");
    elements.layoutSettingsStatus.className = "muted" + (state.layoutMessageError || (config && config.status === "invalid") ? " layout-settings-status-error" : "");
    if (!catalog || !catalog.algorithms || !catalog.options || !state.layoutDraft) {
      elements.layoutAlgorithm.innerHTML = '<option>Loading…</option>';
      elements.layoutAlgorithm.disabled = true;
      elements.layoutOptionsList.innerHTML = '<p class="muted">Loading the pinned ELK catalog…</p>';
      elements.layoutSettingsApply.disabled = true;
      elements.layoutSettingsSave.disabled = true;
      elements.layoutSettingsSaveAs.disabled = true;
      return;
    }
    elements.layoutAlgorithm.disabled = false;
    elements.layoutAlgorithm.innerHTML = catalog.algorithms.map(function (algorithm) {
      return '<option value="' + escapeHTML(algorithm.id) + '"' + (state.layoutDraft.algorithm === algorithm.id ? " selected" : "") + '>' + escapeHTML(algorithm.name) + "</option>";
    }).join("");
    const selectedAlgorithm = catalog.algorithms.find(function (algorithm) { return algorithm.id === state.layoutDraft.algorithm; });
    elements.layoutAlgorithmHelp.textContent = selectedAlgorithm
      ? selectedAlgorithm.description + (selectedAlgorithm.category ? " Category: " + selectedAlgorithm.category + "." : "")
      : "The selected algorithm is not available in the pinned catalog.";
    const query = state.layoutOptionSearch || "";
    const options = catalog.options.slice().sort(function (left, right) {
      const group = String(left.group || "").localeCompare(String(right.group || ""));
      return group || String(left.name).localeCompare(String(right.name));
    }).filter(function (option) {
      return !query || [option.id, option.name, option.group, option.description].join(" ").toLowerCase().includes(query);
    });
    if (!options.length) {
      elements.layoutOptionsList.innerHTML = '<p class="muted">No ELK options match this search.</p>';
    } else {
      let lastGroup = null;
      const markup = [];
      options.forEach(function (option) {
        const group = option.group || "General";
        const applicable = optionAppliesToAlgorithm(option, state.layoutDraft.algorithm);
        const current = optionDisplayValue(option, state.layoutDraft);
        const currentLabel = current || optionDefaultText(option);
        const supportClass = !applicable ? "not-applicable" : option.editable && option.renderer_support === "supported" ? "editable" : "catalog-only";
        if (group !== lastGroup) {
          markup.push('<h4 class="layout-option-group">' + escapeHTML(group) + '</h4>');
          lastGroup = group;
        }
        markup.push('<article class="layout-option ' + supportClass + '"><div class="layout-option-copy"><div class="layout-option-title"><strong>' + escapeHTML(option.name) + '</strong><code>' + escapeHTML(option.id) + '</code></div><div class="layout-option-meta">' + escapeHTML(group) + " · " + escapeHTML(option.type) + " · default: " + escapeHTML(optionDefaultText(option)) + " · current: " + escapeHTML(currentLabel) + escapeHTML(optionBoundsText(option)) + '</div><p>' + escapeHTML(option.description) + '</p></div><div class="layout-option-control">' + layoutOptionInput(option, state.layoutDraft, applicable) + '</div></article>');
      });
      elements.layoutOptionsList.innerHTML = markup.join("");
    }
    elements.layoutSettingsApply.disabled = false;
    elements.layoutSettingsSave.disabled = !config || !config.can_save;
    elements.layoutSettingsSaveAs.disabled = !config || !config.can_save_as;
  }

  function openLayoutSettings() {
    if (embeddedExport) return;
    state.layoutSettingsOpen = true;
    state.layoutMessage = "";
    state.layoutMessageError = false;
    state.layoutDraft = cloneLayoutProfile(state.layoutProfile);
    renderLayoutSettings();
    if (elements.layoutSettingsDialog.showModal) elements.layoutSettingsDialog.showModal();
    else elements.layoutSettingsDialog.setAttribute("open", "");
  }

  function closeLayoutSettings() {
    state.layoutSettingsOpen = false;
    state.layoutMessage = "";
    state.layoutMessageError = false;
    if (elements.layoutSettingsDialog.close) elements.layoutSettingsDialog.close();
    else elements.layoutSettingsDialog.removeAttribute("open");
  }

  function updateLayoutDraftOption(event) {
    const input = event.target.closest("[data-layout-option-id]");
    if (!input || !state.layoutDraft) return;
    const option = (state.layoutCatalog.options || []).find(function (item) { return item.id === input.dataset.layoutOptionId; });
    if (!option) return;
    const raw = input.value;
    if (raw === "") {
      delete state.layoutDraft.options[option.id];
    } else if (option.type === "BOOLEAN") {
      state.layoutDraft.options[option.id] = raw === "true";
    } else if (option.type === "INT") {
      state.layoutDraft.options[option.id] = Number.parseInt(raw, 10);
    } else if (option.type === "DOUBLE") {
      state.layoutDraft.options[option.id] = Number.parseFloat(raw);
    } else {
      state.layoutDraft.options[option.id] = raw;
    }
    state.layoutMessage = "Unsaved settings changes";
    state.layoutMessageError = false;
    elements.layoutSettingsStatus.textContent = state.layoutMessage;
    elements.layoutSettingsStatus.className = "muted";
  }

  function clearManualLayoutPositions() {
    if (!state.viewport) state.viewport = defaultViewport();
    state.viewport.positions = {};
    persistViewport();
  }

  async function applyLayoutProfile() {
    if (!state.layoutDraft || embeddedExport) return;
    state.layoutMessage = "Applying layout…";
    state.layoutMessageError = false;
    renderLayoutSettings();
    try {
      const response = await postJSON("/v1/layout/apply", layoutRequestPayload(state.layoutDraft));
      state.layoutConfig = response;
      state.layoutProfile = cloneLayoutProfile(response.layout);
      state.layoutDraft = cloneLayoutProfile(state.layoutProfile);
      clearManualLayoutPositions();
      state.layout = null;
      state.layoutError = false;
      state.layoutRequest += 1;
      state.layoutMessage = "Applied to the current session";
      state.layoutMessageError = false;
      renderAll();
      renderLayoutSettings();
      void prepareLayout(state.scene, state.layoutProfile);
    } catch (error) {
      state.layoutMessage = error.message || "Layout settings could not be applied.";
      state.layoutMessageError = true;
      renderLayoutSettings();
    }
  }

  async function resetLayoutProfile() {
    if (embeddedExport) return;
    state.layoutMessage = "Restoring built-in defaults…";
    state.layoutMessageError = false;
    renderLayoutSettings();
    try {
      const response = await postJSON("/v1/layout/reset", {});
      state.layoutConfig = response;
      state.layoutProfile = cloneLayoutProfile(response.layout);
      state.layoutDraft = cloneLayoutProfile(state.layoutProfile);
      clearManualLayoutPositions();
      state.layout = null;
      state.layoutError = false;
      state.layoutRequest += 1;
      state.layoutMessage = "Built-in defaults restored for this session";
      state.layoutMessageError = false;
      renderAll();
      renderLayoutSettings();
      void prepareLayout(state.scene, state.layoutProfile);
    } catch (error) {
      state.layoutMessage = error.message || "Layout defaults could not be restored.";
      state.layoutMessageError = true;
      renderLayoutSettings();
    }
  }

  async function saveLayoutProfile() {
    if (!state.layoutDraft || embeddedExport) return;
    state.layoutMessage = "Saving active profile…";
    state.layoutMessageError = false;
    renderLayoutSettings();
    try {
      const response = await putJSON("/v1/layout/config", layoutRequestPayload(state.layoutDraft));
      state.layoutConfig = response;
      state.layoutProfile = cloneLayoutProfile(response.layout);
      state.layoutDraft = cloneLayoutProfile(state.layoutProfile);
      clearManualLayoutPositions();
      state.layout = null;
      state.layoutRequest += 1;
      state.layoutMessage = "Saved active .archview.json";
      state.layoutMessageError = false;
      renderAll();
      renderLayoutSettings();
      void prepareLayout(state.scene, state.layoutProfile);
    } catch (error) {
      state.layoutMessage = error.message || "The active layout profile could not be saved.";
      state.layoutMessageError = true;
      renderLayoutSettings();
    }
  }

  async function saveLayoutProfileAs() {
    if (!state.layoutDraft || embeddedExport) return;
    const destination = elements.layoutSaveAsDirectory.value.trim();
    const confirm = elements.layoutSaveAsConfirm.checked;
    if (!destination || !confirm) {
      state.layoutMessage = "Enter a directory and confirm Save As before writing .archview.json.";
      state.layoutMessageError = true;
      renderLayoutSettings();
      return;
    }
    state.layoutMessage = "Saving a custom profile…";
    state.layoutMessageError = false;
    renderLayoutSettings();
    try {
      const response = await putJSON("/v1/layout/config/save-as", { schema_version: "arch-view.config/v1", layout: cloneLayoutProfile(state.layoutDraft), destination_dir: destination, confirm: confirm });
      state.layoutConfig = response;
      state.layoutProfile = cloneLayoutProfile(response.layout);
      state.layoutDraft = cloneLayoutProfile(state.layoutProfile);
      clearManualLayoutPositions();
      state.layout = null;
      state.layoutRequest += 1;
      state.layoutMessage = "Saved custom .archview.json";
      state.layoutMessageError = false;
      renderAll();
      renderLayoutSettings();
      void prepareLayout(state.scene, state.layoutProfile);
    } catch (error) {
      state.layoutMessage = error.message || "The custom layout profile could not be saved.";
      state.layoutMessageError = true;
      renderLayoutSettings();
    }
  }

  function samePath(left, right) {
    const a = left || [];
    const b = right || [];
    return a.length === b.length && a.every(function (segment, index) { return segment === b[index]; });
  }

  function pathKey(pathValue) {
    return (pathValue || []).join("/");
  }

  function rememberSceneContext(scene) {
    if (!scene) return;
    const key = viewportKey(scene);
    state.scrollContexts[key] = { left: elements.graph.scrollLeft, top: elements.graph.scrollTop };
  }

  function navigationTo(pathValue) {
    const current = state.scene ? state.scene.hierarchy_path : [];
    if (samePath(current, pathValue)) return;
    rememberSceneContext(state.scene);
    loadScene(pathValue).then(function (loaded) {
      if (loaded) {
        state.history.push(current.slice());
        renderBreadcrumbs();
      }
    });
  }

  function goBack() {
    if (!state.history.length) return;
    const target = state.history[state.history.length - 1];
    rememberSceneContext(state.scene);
    loadScene(target).then(function (loaded) {
      if (loaded) {
        state.history.pop();
        renderBreadcrumbs();
      }
    });
  }

  async function loadScene(pathValue) {
    const selectedPath = (pathValue || []).slice();
    const request = ++state.sceneRequest;
    try {
      const query = new URLSearchParams();
      query.set("mode", "overview");
      query.set("reference_visibility", state.referenceVisibility);
      selectedPath.forEach(function (segment) { query.append("path", segment); });
      const scene = await getJSON("/v1/models/" + encodeURIComponent(currentModelID()) + "/projection?" + query.toString());
      if (request !== state.sceneRequest) return false;
      state.lastNodeClick = null;
      state.scene = scene;
      state.referenceVisibility = scene.reference_visibility || state.referenceVisibility;
      state.selected = null;
      state.source = null;
      state.sourceRequest += 1;
      state.layout = null;
      state.layoutKey = sceneLayoutKey(scene);
      state.layoutError = false;
      state.layoutRequest += 1;
      state.viewport = loadViewport(scene);
      elements.referenceVisibility.value = state.referenceVisibility;
      elements.footerModelID.textContent = scene.model_id;
      hideError();
      renderAll();
      restoreScroll(scene);
      void prepareLayout(scene, state.layoutProfile);
      return true;
    } catch (error) {
      if (request !== state.sceneRequest) return false;
      showError(error.message || "The local model could not be loaded.");
      return false;
    }
  }

  function renderAll() {
    const scene = state.scene;
    elements.projectLabel.textContent = scene.project.root_label + " · " + scene.project.language;
    elements.modelStatus.textContent = scene.status;
    elements.viewDescription.textContent = scene.hierarchy_path.length === 0
      ? "A local-first semantic overview. Non-local references remain available through the boundary summary and imports list."
      : "Viewing hierarchy: " + scene.hierarchy_path.join(" / ");
    renderSummary();
    renderSceneState();
    renderBreadcrumbs();
    renderLayerLegend();
    renderReferenceSummary();
    renderViewportControls();
    renderGraph();
    renderAccessibleList();
    renderSupportLists();
    renderDetails();
  }

  function renderSummary() {
    const summary = state.scene.summary;
    const values = [
      [summary.visible_node_count, "visible nodes"],
      [summary.visible_relationship_count, "directed relations"],
      [summary.module_count, "canonical modules"],
      [summary.cycle_count, "cycles"],
      [summary.diagnostic_count, "diagnostics"],
      [summary.evidence_count, "evidence links"]
    ];
    elements.summary.innerHTML = values.map(function (item) {
      return '<div class="summary-item"><div class="summary-value">' + escapeHTML(item[0]) + '</div><p class="summary-label">' + escapeHTML(item[1]) + "</p></div>";
    }).join("");
  }

  function renderSceneState() {
    const scene = state.scene;
    const pills = [];
    pills.push('<span class="state-pill ' + (scene.status === "complete" ? "ok" : "warning") + '">' + escapeHTML(scene.status) + " model</span>");
    if (scene.cycle_indicators.length) pills.push('<span class="state-pill error">' + escapeHTML(scene.cycle_indicators.length) + " cycle(s)</span>");
    if (scene.diagnostic_indicators.length) pills.push('<span class="state-pill warning">' + escapeHTML(scene.diagnostic_indicators.length) + " diagnostic(s)</span>");
    pills.push('<span class="state-pill">' + escapeHTML(referenceVisibilityLabel(scene.reference_visibility)) + " references</span>");
    if (embeddedExport) {
      pills.push('<span class="state-pill ok">deterministic export layout</span>');
    } else if (state.layout && state.layout.key === sceneLayoutKey(scene)) {
      pills.push('<span class="state-pill ok">ELK ' + escapeHTML((state.layoutProfile && state.layoutProfile.algorithm) || "layered") + " layout</span>");
    } else if (state.layoutError) {
      pills.push('<span class="state-pill warning">deterministic fallback layout</span>');
    }
    elements.sceneState.innerHTML = pills.join("");
  }

  function renderBreadcrumbs() {
    const scene = state.scene;
    elements.backButton.disabled = state.history.length === 0;
    const top = scene.hierarchy_path.length === 0
      ? '<span class="breadcrumb current" aria-current="page">Top level</span>'
      : '<button class="breadcrumb" type="button" data-breadcrumb-path="">Top level</button>';
    const crumbs = [top];
    scene.hierarchy_path.forEach(function (segment, index) {
      const current = index === scene.hierarchy_path.length - 1;
      crumbs.push('<span class="breadcrumb-separator" aria-hidden="true">/</span>');
      crumbs.push(current
        ? '<span class="breadcrumb current" aria-current="page">' + escapeHTML(segment) + '</span>'
        : '<button class="breadcrumb" type="button" data-breadcrumb-path="' + escapeHTML(scene.hierarchy_path.slice(0, index + 1).join("/")) + '">' + escapeHTML(segment) + '</button>');
    });
    elements.breadcrumbs.innerHTML = crumbs.join("");
    elements.breadcrumbs.querySelectorAll("[data-breadcrumb-path]").forEach(function (element) {
      element.addEventListener("click", function () {
        const pathValue = element.dataset.breadcrumbPath ? element.dataset.breadcrumbPath.split("/") : [];
        navigationTo(pathValue);
      });
    });
  }

  function renderLayerLegend() {
    elements.layerLegend.innerHTML = state.scene.layer_labels.map(function (layer) {
      return '<span class="layer-pill">' + escapeHTML(layer.label) + " · " + escapeHTML(layer.module_ids.length) + " module(s)</span>";
    }).join("");
  }

  function referenceVisibilityLabel(value) {
    return value === "expanded" ? "Expanded" : value === "aggregated" ? "Aggregated" : "Local-first";
  }

  function referenceScopeLabel(value) {
    return String(value || "reference").replaceAll("_", " ");
  }

  function renderReferenceSummary() {
    const summary = state.scene.reference_summary;
    if (!summary || !summary.total) {
      elements.referenceSummary.innerHTML = '<span class="reference-pill">No non-local references in this projection</span>';
      return;
    }
    const scopePills = (summary.by_scope || []).map(function (scope) {
      const visible = scope.visible_node_count ? " · " + scope.visible_node_count + " shown" : "";
      return '<span class="reference-pill"><strong>' + escapeHTML(scope.reference_count) + '</strong> ' + escapeHTML(referenceScopeLabel(scope.scope)) + escapeHTML(visible) + '</span>';
    }).join("");
    const policyText = summary.hidden_count
      ? summary.hidden_count + " hidden in overview"
      : summary.aggregated_count
        ? summary.aggregated_count + " boundary node(s)"
        : summary.expanded_count + " shown individually";
    elements.referenceSummary.innerHTML = '<span class="reference-pill"><strong>' + escapeHTML(summary.relationship_count) + '</strong> import relation(s) · ' + escapeHTML(policyText) + '</span>' + scopePills;
  }

  function sceneLayoutKey(scene) {
    const nodes = scene.visible_nodes.map(function (node) { return node.id; }).sort().join(",");
    const relationships = scene.visible_relationships.map(function (relationship) {
      return relationship.id + ":" + relationship.from_visible_id + ":" + relationship.to_visible_id;
    }).sort().join(",");
    return [scene.model_revision || scene.model_id, (scene.hierarchy_path || []).join("/"), scene.reference_visibility, nodes, relationships].join("|");
  }

  function viewportKey(scene) {
    return "arch-view:viewport:" + [scene.model_revision || scene.model_id, (scene.hierarchy_path || []).join("/"), scene.reference_visibility].join("|");
  }

  function defaultViewport() {
    return { zoom: 1, panX: 0, panY: 0, positions: {} };
  }

  function loadViewport(scene) {
    const fallback = defaultViewport();
    try {
      const value = JSON.parse(sessionStorage.getItem(viewportKey(scene)) || "null");
      if (!value || typeof value !== "object") return fallback;
      return {
        zoom: clampNumber(value.zoom, MIN_ZOOM, MAX_ZOOM, 1),
        panX: clampNumber(value.panX, -PAN_LIMIT, PAN_LIMIT, 0),
        panY: clampNumber(value.panY, -PAN_LIMIT, PAN_LIMIT, 0),
        positions: manualPositionsForScene(value.positions, scene)
      };
    } catch (error) {
      return fallback;
    }
  }

  function persistViewport() {
    if (!state.scene || !state.viewport) return;
    try {
      sessionStorage.setItem(viewportKey(state.scene), JSON.stringify(state.viewport));
    } catch (error) {
      // Session persistence is an enhancement; the active view remains usable.
    }
  }

  function restoreScroll(scene) {
    const context = state.scrollContexts[viewportKey(scene)];
    window.requestAnimationFrame(function () {
      elements.graph.scrollLeft = context ? context.left : 0;
      elements.graph.scrollTop = context ? context.top : 0;
    });
  }

  function clampNumber(value, minimum, maximum, fallback) {
    return typeof value === "number" && Number.isFinite(value) ? Math.max(minimum, Math.min(maximum, value)) : fallback;
  }

  function manualPositionsForScene(value, scene) {
    const positions = Object.create(null);
    if (!value || typeof value !== "object" || Array.isArray(value)) return positions;
    const nodeIDs = new Set((scene.visible_nodes || []).map(function (node) { return node.id; }));
    Object.keys(value).forEach(function (nodeID) {
      if (!nodeIDs.has(nodeID)) return;
      const position = value[nodeID];
      if (!position || typeof position !== "object" || Array.isArray(position)) return;
      if (!Number.isFinite(position.x) || !Number.isFinite(position.y)) return;
      positions[nodeID] = { x: position.x, y: position.y };
    });
    return positions;
  }

  function currentGraphPositions(scene) {
    const fallback = fallbackLayout(scene);
    const active = state.layout && state.layout.key === sceneLayoutKey(scene) ? state.layout : fallback;
    const positions = Object.assign({}, fallback.positions, active.positions);
    const manualPositions = state.viewport && state.viewport.positions ? state.viewport.positions : {};
    Object.keys(manualPositions).forEach(function (nodeID) {
      if (positions[nodeID]) positions[nodeID] = Object.assign({}, positions[nodeID], manualPositions[nodeID]);
    });
    return positions;
  }

  function layoutBounds(scene, layout, positions) {
    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    function includePoint(x, y) {
      if (!Number.isFinite(x) || !Number.isFinite(y)) return;
      minX = Math.min(minX, x);
      minY = Math.min(minY, y);
      maxX = Math.max(maxX, x);
      maxY = Math.max(maxY, y);
    }
    function includeBox(box) {
      if (!box) return;
      includePoint(box.x, box.y);
      includePoint(box.x + box.width, box.y + box.height);
    }
    scene.visible_nodes.forEach(function (node) { includeBox(positions[node.id]); });
    const manualPositions = state.viewport && state.viewport.positions ? state.viewport.positions : {};
    const relationshipsByID = {};
    scene.visible_relationships.forEach(function (relationship) { relationshipsByID[relationship.id] = relationship; });
    Object.keys(layout.edges || {}).forEach(function (edgeID) {
      const relationship = relationshipsByID[edgeID];
      if (relationship && (manualPositions[relationship.from_visible_id] || manualPositions[relationship.to_visible_id])) return;
      const points = routePoints(layout.edges[edgeID]);
      if (points) points.forEach(function (point) { includePoint(point.x, point.y); });
    });
    if (!Number.isFinite(minX)) return { minX: 0, minY: 0, maxX: layout.width, maxY: layout.height };
    const padding = 28;
    return {
      minX: minX - padding,
      minY: minY - padding,
      maxX: maxX + padding,
      maxY: maxY + padding,
      centerX: (minX + maxX) / 2,
      centerY: (minY + maxY) / 2
    };
  }

  function renderViewportControls() {
    const viewport = state.viewport || defaultViewport();
    elements.zoomValue.textContent = Math.round(viewport.zoom * 100) + "%";
    elements.resetLayout.disabled = Object.keys(viewport.positions || {}).length === 0;
  }

  function changeZoom(delta) {
    if (!state.viewport) state.viewport = defaultViewport();
    state.viewport.zoom = clampNumber(state.viewport.zoom + delta, MIN_ZOOM, MAX_ZOOM, 1);
    persistViewport();
    renderViewportControls();
    renderGraph();
  }

  function resetZoom() {
    if (!state.viewport) state.viewport = defaultViewport();
    state.viewport.zoom = 1;
    persistViewport();
    renderViewportControls();
    renderGraph();
  }

  function resetLayout() {
    if (!state.viewport) state.viewport = defaultViewport();
    if (!Object.keys(state.viewport.positions || {}).length) return;
    state.viewport.positions = {};
    persistViewport();
    renderViewportControls();
    renderSceneState();
    renderGraph();
  }

  function isExpandedCanvas() {
    return Boolean(document.fullscreenElement || document.body.classList.contains("canvas-focus") || document.body.classList.contains("canvas-fullscreen"));
  }

  function syncCanvasOverflow() {
    document.documentElement.classList.toggle("canvas-focus", document.body.classList.contains("canvas-focus"));
    document.documentElement.classList.toggle("canvas-fullscreen", document.body.classList.contains("canvas-fullscreen"));
  }

  function fitViewport() {
    if (!state.scene) return;
    const fallback = fallbackLayout(state.scene);
    const active = state.layout && state.layout.key === sceneLayoutKey(state.scene) ? state.layout : fallback;
    const svg = elements.graph.querySelector("svg");
    const availableWidth = Math.max(360, svg ? svg.clientWidth : elements.graph.clientWidth - 12);
    const availableHeight = Math.max(260, svg ? svg.clientHeight : elements.graph.clientHeight - 12);
    const positions = currentGraphPositions(state.scene);
    const bounds = layoutBounds(state.scene, active, positions);
    const baseScale = Math.min(availableWidth / active.width, availableHeight / active.height);
    const contentWidth = Math.max(1, bounds.maxX - bounds.minX);
    const contentHeight = Math.max(1, bounds.maxY - bounds.minY);
    const contentFitZoom = Math.min(availableWidth / (contentWidth * baseScale), availableHeight / (contentHeight * baseScale));
    if (!state.viewport) state.viewport = defaultViewport();
    const maximumFitZoom = isExpandedCanvas() ? EXPANDED_FIT_ZOOM : WINDOWED_FIT_ZOOM;
    state.viewport.zoom = clampNumber(Math.max(1, contentFitZoom), 1, maximumFitZoom, 1);
    const offsetX = (availableWidth - active.width * baseScale) / 2;
    const offsetY = (availableHeight - active.height * baseScale) / 2;
    state.viewport.panX = clampNumber((availableWidth / 2 - offsetX) / baseScale - bounds.centerX * state.viewport.zoom, -PAN_LIMIT, PAN_LIMIT, 0);
    state.viewport.panY = clampNumber((availableHeight / 2 - offsetY) / baseScale - bounds.centerY * state.viewport.zoom, -PAN_LIMIT, PAN_LIMIT, 0);
    persistViewport();
    renderViewportControls();
    renderGraph();
  }

  function toggleFocusMode() {
    const active = isExpandedCanvas() || elements.focusToggle.textContent === "Exit canvas";
    if (active) {
      state.nativeFullscreen = false;
      if (document.fullscreenElement) document.exitFullscreen().catch(function () {});
      document.body.classList.remove("canvas-focus", "canvas-fullscreen");
      syncCanvasOverflow();
      elements.focusToggle.textContent = "Full canvas";
      return;
    }
    document.body.classList.add("canvas-focus");
    syncCanvasOverflow();
    elements.focusToggle.textContent = "Exit canvas";
    const sceneCard = document.querySelector(".scene-card");
    if (sceneCard && sceneCard.requestFullscreen) {
      state.nativeFullscreen = true;
      sceneCard.requestFullscreen().catch(function () {
        state.nativeFullscreen = false;
      });
    }
  }

  function syncFocusButton() {
    if (document.fullscreenElement) {
      document.body.classList.add("canvas-fullscreen");
      syncCanvasOverflow();
      elements.focusToggle.textContent = "Exit canvas";
      return;
    }
    document.body.classList.remove("canvas-fullscreen");
    if (state.nativeFullscreen) {
      state.nativeFullscreen = false;
      document.body.classList.remove("canvas-focus");
    }
    syncCanvasOverflow();
    elements.focusToggle.textContent = document.body.classList.contains("canvas-focus") ? "Exit canvas" : "Full canvas";
  }

  function fallbackLayout(scene) {
    if (embeddedExport && embeddedExport.layouts) {
      const exported = embeddedExport.layouts[sceneLayoutKey(scene)];
      if (exported) return exported;
    }
    const nodes = scene.visible_nodes;
    const byLayer = {};
    nodes.forEach(function (node) {
      const layer = node.layer == null ? ((node.layers && node.layers[0]) || 0) : node.layer;
      if (!byLayer[layer]) byLayer[layer] = [];
      byLayer[layer].push(node);
    });
    const layers = Object.keys(byLayer).sort(function (left, right) { return Number(left) - Number(right); });
    const nodeWidth = 190;
    const nodeHeight = 82;
    const columnGap = 84;
    const rowGap = 35;
    const maxRows = Math.max.apply(null, layers.map(function (layer) { return byLayer[layer].length; }).concat([1]));
    const width = Math.max(760, layers.length * (nodeWidth + columnGap) + 80);
    const height = Math.max(430, maxRows * (nodeHeight + rowGap) + 100);
    const positions = {};
    layers.forEach(function (layer, layerIndex) {
      byLayer[layer].sort(function (left, right) {
        if (left.label === right.label) return left.id < right.id ? -1 : left.id > right.id ? 1 : 0;
        return left.label < right.label ? -1 : 1;
      });
      byLayer[layer].forEach(function (node, rowIndex) {
        positions[node.id] = {
          x: 40 + layerIndex * (nodeWidth + columnGap),
          y: 42 + rowIndex * (nodeHeight + rowGap),
          width: nodeWidth,
          height: nodeHeight
        };
      });
    });
    return { engine: "fallback", key: sceneLayoutKey(scene), width: width, height: height, positions: positions, edges: {} };
  }

  function numberOrZero(value) {
    return typeof value === "number" && Number.isFinite(value) ? value : 0;
  }

  function pointWithOffset(point, offset) {
    return { x: numberOrZero(point && point.x) + offset, y: numberOrZero(point && point.y) + offset };
  }

  function appendUniquePoint(points, point) {
    if (!point) return;
    const next = { x: numberOrZero(point.x), y: numberOrZero(point.y) };
    const previous = points[points.length - 1];
    if (!previous || previous.x !== next.x || previous.y !== next.y) points.push(next);
  }

  function adaptELKLayout(scene, result, key) {
    const offset = 24;
    const positions = {};
    (result.children || []).forEach(function (child) {
      positions[child.id] = {
        x: numberOrZero(child.x) + offset,
        y: numberOrZero(child.y) + offset,
        width: numberOrZero(child.width) || 190,
        height: numberOrZero(child.height) || 82
      };
    });
    const edges = {};
    (result.edges || []).forEach(function (edge) {
      const points = [];
      (edge.sections || []).forEach(function (section) {
        appendUniquePoint(points, pointWithOffset(section.startPoint, offset));
        (section.bendPoints || []).forEach(function (point) { appendUniquePoint(points, pointWithOffset(point, offset)); });
        appendUniquePoint(points, pointWithOffset(section.endPoint, offset));
      });
      if (points.length < 2) return;
      const midpoint = points[Math.floor(points.length / 2)];
      edges[edge.id] = { points: points, labelX: midpoint.x, labelY: midpoint.y - 7 };
    });
    const fallback = fallbackLayout(scene);
    return {
      engine: "elk." + ((state.layoutProfile && state.layoutProfile.algorithm) || "layered"),
      key: key,
      width: Math.max(760, numberOrZero(result.width) + offset * 2, fallback.width),
      height: Math.max(430, numberOrZero(result.height) + offset * 2, fallback.height),
      positions: positions,
      edges: edges
    };
  }

  async function prepareLayout(scene, profile) {
    if (embeddedExport || !scene || !scene.visible_nodes.length || typeof window.ELK !== "function") return;
    const key = sceneLayoutKey(scene);
    state.layoutKey = key;
    const request = ++state.layoutRequest;
    let elk;
    try {
      const workerURL = new URL("/assets/vendor/elk-worker.min.js", window.location.href).toString();
      elk = new window.ELK({ workerUrl: workerURL });
      const result = await elk.layout(window.ArchViewELKRequest.buildELKGraph(scene, profile || state.layoutProfile, state.layoutCatalog));
      if (request !== state.layoutRequest || state.scene !== scene) return;
      state.layout = adaptELKLayout(scene, result, key);
      state.layoutError = false;
      renderSceneState();
      renderViewportControls();
      renderGraph();
    } catch (error) {
      if (request !== state.layoutRequest || state.scene !== scene) return;
      state.layout = null;
      state.layoutError = true;
      state.layoutMessage = "ELK layout failed; deterministic fallback is active.";
      state.layoutMessageError = true;
      renderSceneState();
      renderGraph();
      if (state.layoutSettingsOpen) renderLayoutSettings();
    } finally {
      if (elk && typeof elk.terminateWorker === "function") {
        try {
          const termination = elk.terminateWorker();
          if (termination && typeof termination.catch === "function") termination.catch(function () {});
        } catch (error) {
          // The bundled ELK build can fall back to a non-worker adapter.
        }
      }
    }
  }

  function routePoints(route) {
    if (route && route.points && route.points.length > 1) return route.points;
    if (!route || !route.sourcePoint || !route.targetPoint) return null;
    const points = [];
    appendUniquePoint(points, route.sourcePoint);
    (route.bendPoints || []).forEach(function (point) { appendUniquePoint(points, point); });
    appendUniquePoint(points, route.targetPoint);
    return points.length > 1 ? points : null;
  }

  function geometryFromPoints(points, labelX, labelY) {
    if (!points || points.length < 2) return null;
    const path = "M " + points.map(function (point, index) {
      return (index === 0 ? "" : "L ") + point.x + " " + point.y;
    }).join(" ");
    const midpoint = points[Math.floor(points.length / 2)];
    return {
      path: path,
      labelX: Number.isFinite(labelX) ? labelX : midpoint.x,
      labelY: Number.isFinite(labelY) ? labelY : midpoint.y - 7
    };
  }

  function orthogonalFallbackGeometry(from, to) {
    const fromCenterX = from.x + from.width / 2;
    const fromCenterY = from.y + from.height / 2;
    const toCenterX = to.x + to.width / 2;
    const toCenterY = to.y + to.height / 2;
    const points = [];
    if (Math.abs(toCenterX - fromCenterX) >= Math.abs(toCenterY - fromCenterY)) {
      const forward = toCenterX >= fromCenterX;
      const sourceX = forward ? from.x + from.width : from.x;
      const targetX = forward ? to.x : to.x + to.width;
      const middleX = (sourceX + targetX) / 2;
      points.push({ x: sourceX, y: fromCenterY });
      points.push({ x: middleX, y: fromCenterY });
      points.push({ x: middleX, y: toCenterY });
      points.push({ x: targetX, y: toCenterY });
    } else {
      const forward = toCenterY >= fromCenterY;
      const sourceY = forward ? from.y + from.height : from.y;
      const targetY = forward ? to.y : to.y + to.height;
      const middleY = (sourceY + targetY) / 2;
      points.push({ x: fromCenterX, y: sourceY });
      points.push({ x: fromCenterX, y: middleY });
      points.push({ x: toCenterX, y: middleY });
      points.push({ x: toCenterX, y: targetY });
    }
    return geometryFromPoints(points);
  }

  function edgeGeometry(relationship, from, to, route, preferOrthogonal) {
    const routedPoints = routePoints(route);
    if (routedPoints) {
      return geometryFromPoints(routedPoints, route.labelX, route.labelY);
    }
    if (relationship.from_visible_id === relationship.to_visible_id) {
      const x = from.x + from.width / 2;
      return {
        path: "M " + x + " " + from.y + " C " + (x + 100) + " " + (from.y - 55) + ", " + (x + 100) + " " + (from.y + from.height + 55) + ", " + x + " " + (from.y + from.height),
        labelX: x + 50,
        labelY: from.y + from.height / 2
      };
    }
    if (preferOrthogonal) return orthogonalFallbackGeometry(from, to);
    const x1 = from.x + from.width;
    const y1 = from.y + from.height / 2;
    const x2 = to.x;
    const y2 = to.y + to.height / 2;
    const bend = Math.max(32, Math.abs(x2 - x1) * 0.35);
    return {
      path: "M " + x1 + " " + y1 + " C " + (x1 + bend) + " " + y1 + ", " + (x2 - bend) + " " + y2 + ", " + x2 + " " + y2,
      labelX: (x1 + x2) / 2,
      labelY: (y1 + y2) / 2 - 7
    };
  }

  function renderGraph() {
    if (!state.scene) return;
    const scene = state.scene;
    const nodes = scene.visible_nodes;
    const nodesByID = {};
    nodes.forEach(function (node) { nodesByID[node.id] = node; });
    const fallback = fallbackLayout(scene);
    const activeLayout = state.layout && state.layout.key === sceneLayoutKey(scene) ? state.layout : fallback;
    const positions = Object.assign({}, fallback.positions, activeLayout.positions);
    const manualPositions = state.viewport && state.viewport.positions ? state.viewport.positions : {};
    Object.keys(manualPositions).forEach(function (nodeID) {
      if (!positions[nodeID]) return;
      positions[nodeID] = Object.assign({}, positions[nodeID], manualPositions[nodeID]);
    });
    const width = activeLayout.width;
    const height = activeLayout.height;
    const viewport = state.viewport || defaultViewport();
    const transform = "translate(" + viewport.panX + " " + viewport.panY + ") scale(" + viewport.zoom + ")";

    const edgeMarkup = scene.visible_relationships.map(function (relationship) {
      const from = positions[relationship.from_visible_id];
      const to = positions[relationship.to_visible_id];
      if (!from || !to) return "";
      const selected = state.selected && state.selected.kind === "relationship" && state.selected.id === relationship.id;
      const matches = relationshipMatches(relationship, nodesByID);
      const className = "edge-line " + classForState(relationship.cycle_state) + (selected ? " selected" : "") + (!matches ? " dimmed" : "");
      const hasManualEndpoint = manualPositions[relationship.from_visible_id] || manualPositions[relationship.to_visible_id];
      const geometry = edgeGeometry(relationship, from, to, hasManualEndpoint ? null : activeLayout.edges[relationship.id], Boolean(hasManualEndpoint));
      return '<g class="edge-group" data-edge-id="' + escapeHTML(relationship.id) + '" tabindex="0" role="button" aria-label="' + escapeHTML(relationship.accessible_label) + '">' +
        '<path class="edge-hit" d="' + geometry.path + '"></path><path class="' + className + '" d="' + geometry.path + '" marker-end="url(#arrow)"></path>' +
        '<text class="edge-label ' + classForState(relationship.cycle_state) + (!matches ? " dimmed" : "") + '" x="' + geometry.labelX + '" y="' + geometry.labelY + '" text-anchor="middle">' + escapeHTML(relationship.count) + '</text></g>';
    }).join("");

    const nodeMarkup = nodes.map(function (node) {
      const position = positions[node.id];
      if (!position) return "";
      const selected = state.selected && state.selected.kind === "node" && state.selected.id === node.id;
      const matches = nodeMatches(node);
      const diagnosticClass = node.diagnostic_state === "none" ? "" : " " + classForState(node.diagnostic_state);
      const className = "node-shape " + classForState(node.kind) + " " + classForState(node.cycle_state) + diagnosticClass + (selected ? " selected" : "") + (!matches ? " dimmed" : "");
      const layer = node.layer == null ? (node.layers && node.layers.length ? "L" + node.layers.join(", L") : "—") : "L" + node.layer;
      const scope = node.reference_scope ? " · " + referenceScopeLabel(node.reference_scope) : "";
      const subtitle = node.kind + scope + " · " + layer + (node.counts.module_count > 1 ? " · " + node.counts.module_count + " modules" : "");
      const status = nodeStatusText(node);
      return '<g data-node-id="' + escapeHTML(node.id) + '" data-node-kind="' + escapeHTML(node.kind) + '" tabindex="0" role="button" aria-label="' + escapeHTML(node.accessible_label) + '">' +
        '<title>' + escapeHTML(node.accessible_label) + '</title>' +
        '<rect class="' + className + '" x="' + position.x + '" y="' + position.y + '" width="' + position.width + '" height="' + position.height + '" rx="12"></rect>' +
        '<rect class="node-hitzone" x="' + position.x + '" y="' + position.y + '" width="' + position.width + '" height="' + position.height + '" rx="12"></rect>' +
        '<text class="node-label ' + (!matches ? "dimmed" : "") + '" x="' + (position.x + 14) + '" y="' + (position.y + 30) + '">' + escapeHTML(truncate(node.label, 25)) + '</text>' +
        '<text class="node-subtitle" x="' + (position.x + 14) + '" y="' + (position.y + 51) + '">' + escapeHTML(truncate(subtitle, 29)) + '</text>' +
        '<text class="node-subtitle" x="' + (position.x + 14) + '" y="' + (position.y + 68) + '">' + escapeHTML(status) + "</text></g>";
    }).join("");

    const empty = nodes.length === 0 ? '<p class="muted">No visible nodes in this projection.</p>' : "";
    elements.graph.innerHTML = empty + '<svg viewBox="0 0 ' + width + ' ' + height + '" role="img" aria-labelledby="graph-title graph-desc" xmlns="http://www.w3.org/2000/svg"><title id="graph-title">Architecture graph</title><desc id="graph-desc">' + escapeHTML(scene.accessibility.reading_order.length + " semantic items in the current scene") + '</desc><defs><marker id="arrow" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto"><path d="M 0 0 L 9 4.5 L 0 9 z" fill="#7483a9"></path></marker></defs><g class="viewport-content" transform="' + transform + '"><g class="edges">' + edgeMarkup + '</g><g class="nodes">' + nodeMarkup + '</g></g></svg>';
    const svg = elements.graph.querySelector("svg");
    if (!svg) return;
    elements.graph.insertAdjacentHTML("beforeend", '<span class="viewport-hint graph-viewport-hint">Drag the canvas to pan; Shift-drag a node to adjust this session.</span>');
    bindGraphInteractions(svg, positions);
  }

  function bindGraphInteractions(svg, positions) {
    elements.graph.querySelectorAll("[data-node-id]").forEach(function (element) {
      const nodeID = element.dataset.nodeId;
      const select = function (preserveDoubleClick) {
        if (Date.now() < state.suppressClickUntil) return;
        selectEntity("node", nodeID, preserveDoubleClick);
      };
      const navigate = function () {
        const node = state.scene.visible_nodes.find(function (item) { return item.id === nodeID; });
        if (node && node.kind === "group") navigationTo(node.hierarchy_path);
      };
      const handleClick = function (event) {
        const now = Date.now();
        if (now < state.suppressClickUntil) return;
        const previous = state.lastNodeClick;
        const dx = previous && event ? event.clientX - previous.x : Infinity;
        const dy = previous && event ? event.clientY - previous.y : Infinity;
        const isDoubleClick = previous && previous.nodeID === nodeID && (event.detail >= 2 || (now - previous.time <= NODE_DOUBLE_CLICK_WINDOW && Math.sqrt(dx * dx + dy * dy) <= NODE_DOUBLE_CLICK_DISTANCE));
        if (isDoubleClick) {
          state.lastNodeClick = null;
          navigate();
          return;
        }
        state.lastNodeClick = { nodeID: nodeID, time: now, x: event.clientX, y: event.clientY };
        select(true);
      };
      element.addEventListener("click", handleClick);
      element.addEventListener("keydown", function (event) {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          select(false);
        }
        if (event.key === "ArrowRight") {
          const node = state.scene.visible_nodes.find(function (item) { return item.id === nodeID; });
          if (node && node.kind === "group") navigationTo(node.hierarchy_path);
        }
      });
      element.addEventListener("pointerdown", function (event) {
        if (event.shiftKey) beginNodeDrag(event, svg, nodeID, positions[nodeID]);
      });
    });
    elements.graph.querySelectorAll("[data-edge-id]").forEach(function (element) {
      const select = function () {
        if (Date.now() < state.suppressClickUntil) return;
        selectEntity("relationship", element.dataset.edgeId);
      };
      element.addEventListener("click", select);
      element.addEventListener("keydown", function (event) {
        if (event.key === "Enter" || event.key === " ") { event.preventDefault(); select(); }
      });
    });
    svg.addEventListener("pointerdown", function (event) {
      if (event.button !== 0 || event.target.closest("[data-node-id], [data-edge-id]")) return;
      beginPan(event, svg);
    });
    svg.addEventListener("wheel", function (event) {
      event.preventDefault();
      changeZoom(event.deltaY < 0 ? 0.08 : -0.08);
    }, { passive: false });
  }

  function graphBaseScale(svg) {
    if (!state.scene || !svg) return 1;
    const fallback = fallbackLayout(state.scene);
    const active = state.layout && state.layout.key === sceneLayoutKey(state.scene) ? state.layout : fallback;
    const width = Math.max(1, svg.clientWidth);
    const height = Math.max(1, svg.clientHeight);
    return Math.max(0.01, Math.min(width / Math.max(1, active.width), height / Math.max(1, active.height)));
  }

  function beginPan(event, svg) {
    if (!state.viewport) state.viewport = defaultViewport();
    const start = { x: event.clientX, y: event.clientY, panX: state.viewport.panX, panY: state.viewport.panY };
    const baseScale = graphBaseScale(svg);
    event.preventDefault();
    function move(pointerEvent) {
      state.viewport.panX = clampNumber(start.panX + ((pointerEvent.clientX - start.x) / baseScale) * PAN_SPEED, -PAN_LIMIT, PAN_LIMIT, start.panX);
      state.viewport.panY = clampNumber(start.panY + ((pointerEvent.clientY - start.y) / baseScale) * PAN_SPEED, -PAN_LIMIT, PAN_LIMIT, start.panY);
      persistViewport();
      renderGraph();
    }
    function stop() {
      document.removeEventListener("pointermove", move);
      document.removeEventListener("pointerup", stop);
      document.removeEventListener("pointercancel", stop);
    }
    document.addEventListener("pointermove", move);
    document.addEventListener("pointerup", stop);
    document.addEventListener("pointercancel", stop);
  }

  function beginNodeDrag(event, svg, nodeID, position) {
    if (event.button !== 0 || !position) return;
    if (!state.viewport) state.viewport = defaultViewport();
    const start = { x: event.clientX, y: event.clientY, nodeX: position.x, nodeY: position.y };
    const baseScale = graphBaseScale(svg);
    const drag = { nodeID: nodeID };
    state.draggingNode = drag;
    let moved = false;
    event.preventDefault();
    event.stopPropagation();
    function move(pointerEvent) {
      const dx = (pointerEvent.clientX - start.x) / (baseScale * state.viewport.zoom);
      const dy = (pointerEvent.clientY - start.y) / (baseScale * state.viewport.zoom);
      if (Math.abs(dx) + Math.abs(dy) > 3) moved = true;
      state.viewport.positions[nodeID] = { x: start.nodeX + dx, y: start.nodeY + dy };
      persistViewport();
      if (!state.dragFrame) {
        state.dragFrame = window.requestAnimationFrame(function () {
          state.dragFrame = 0;
          if (state.draggingNode === drag) {
            renderViewportControls();
            renderGraph();
          }
        });
      }
    }
    function stop() {
      document.removeEventListener("pointermove", move);
      document.removeEventListener("pointerup", stop);
      document.removeEventListener("pointercancel", stop);
      if (state.dragFrame) window.cancelAnimationFrame(state.dragFrame);
      state.dragFrame = 0;
      if (state.draggingNode === drag) state.draggingNode = null;
      renderViewportControls();
      renderGraph();
      if (moved) {
        state.lastNodeClick = null;
        state.suppressClickUntil = Date.now() + 180;
      }
    }
    document.addEventListener("pointermove", move);
    document.addEventListener("pointerup", stop);
    document.addEventListener("pointercancel", stop);
  }

  function selectEntity(kind, id, preserveDoubleClick) {
    if (!preserveDoubleClick) state.lastNodeClick = null;
    state.selected = { kind: kind, id: id };
    state.source = null;
    state.sourceRequest += 1;
    renderGraph();
    renderAccessibleList();
    renderDetails();
  }

  function nodeMatches(node) {
    if (!state.query) return true;
    return [node.label, node.id, node.kind, node.reference_scope, (node.hierarchy_path || []).join(" ")].join(" ").toLowerCase().includes(state.query);
  }

  function relationshipMatches(relationship, nodesByID) {
    if (!state.query) return true;
    const from = nodesByID[relationship.from_visible_id];
    const to = nodesByID[relationship.to_visible_id];
    return [relationship.id, relationship.type, relationship.target_scope, from && from.label, to && to.label].join(" ").toLowerCase().includes(state.query);
  }

  function nodeStatusText(node) {
    if (node.counts.internal_relationship_count > 0) {
      return node.counts.internal_relationship_count + " internal relationship" + (node.counts.internal_relationship_count === 1 ? "" : "s");
    }
    const identity = node.identity_state || "stable";
    if (node.confidence_state === "not_applicable") return (node.cycle_state !== "none" ? node.cycle_state + " · " : "") + "identity " + identity;
    return (node.cycle_state !== "none" ? node.cycle_state + " · " : "") + node.confidence_state + " confidence";
  }

  function nodeListMeta(node) {
    const parts = [node.kind, "identity " + (node.identity_state || "stable")];
    if (node.confidence_state !== "not_applicable") parts.push(node.confidence_state + " confidence");
    if (node.counts.internal_relationship_count > 0) parts.push(node.counts.internal_relationship_count + " internal relationship(s)");
    return parts.join(" · ");
  }

  function relationshipListMeta(relationship) {
    const nodesByID = {};
    state.scene.visible_nodes.forEach(function (node) { nodesByID[node.id] = node; });
    const from = nodesByID[relationship.from_visible_id];
    const to = nodesByID[relationship.to_visible_id];
    return (from ? from.label : relationship.from_visible_id) + " → " + (to ? to.label : relationship.to_visible_id) + " · " + relationship.count + " contributor(s) · " + relationship.confidence_state + " confidence";
  }

  function allListItems() {
    const scene = state.scene;
    return scene.visible_nodes.map(function (node) { return { kind: "node", id: node.id, title: node.label, meta: nodeListMeta(node), value: node }; })
      .concat(scene.visible_relationships.map(function (relationship) { return { kind: "relationship", id: relationship.id, title: relationship.type, meta: relationshipListMeta(relationship), value: relationship }; }))
      .concat((scene.reference_details || []).map(function (reference) { return { kind: "reference-detail", id: reference.id, title: reference.name, meta: referenceScopeLabel(reference.scope) + " · " + reference.count + " import(s) · " + reference.confidence_state + " confidence", value: reference }; }))
      .concat(scene.cycle_indicators.map(function (cycle) { return { kind: "cycle", id: cycle.id, title: cycle.label, meta: cycle.module_ids.length + " module(s) · " + cycle.relationship_ids.length + " relationship(s)", value: cycle }; }))
      .concat(scene.diagnostic_indicators.map(function (diagnostic) { return { kind: "diagnostic", id: diagnostic.id, title: diagnostic.code, meta: diagnostic.severity + " · " + diagnostic.message, value: diagnostic }; }));
  }

  function renderAccessibleList() {
    const items = allListItems().filter(function (item) {
      if (!state.query) return true;
      return [item.title, item.meta, item.id].join(" ").toLowerCase().includes(state.query);
    });
    elements.listCount.textContent = items.length + " item(s)";
    elements.accessibleList.innerHTML = items.length ? items.map(function (item) {
      const selected = state.selected && state.selected.kind === item.kind && state.selected.id === item.id;
      return '<button class="list-item ' + (selected ? "selected" : "") + '" type="button" data-list-kind="' + escapeHTML(item.kind) + '" data-list-id="' + escapeHTML(item.id) + '" aria-label="' + escapeHTML(item.title + ". " + item.meta) + '"><span class="list-item-title">' + escapeHTML(item.title) + '</span><span class="list-item-meta">' + escapeHTML(item.meta) + '</span></button>';
    }).join("") : '<p class="list-empty">No items match the current search.</p>';
    elements.accessibleList.querySelectorAll("[data-list-id]").forEach(function (element) {
      element.addEventListener("click", function () { selectEntity(element.dataset.listKind, element.dataset.listId); });
    });
  }

  function detailRow(key, value, extraClass) {
    return '<div class="detail-row"><span class="detail-key">' + escapeHTML(key) + '</span><span class="detail-value ' + (extraClass || "") + '">' + escapeHTML(value) + "</span></div>";
  }

  function detailSection(title, content) {
    return '<section class="detail-section"><h4>' + escapeHTML(title) + '</h4>' + content + "</section>";
  }

  function modelModule(moduleID) {
    if (!state.model) return null;
    return (state.model.modules || []).find(function (module) { return module.id === moduleID; }) || null;
  }

  function moduleNames(moduleIDs) {
    return (moduleIDs || []).map(function (moduleID) {
      const module = modelModule(moduleID);
      return module ? module.display_name : moduleID;
    });
  }

  function locationText(link) {
    if (!link) return "";
    const start = link.start ? "L" + link.start.line + ":" + link.start.column : "line unavailable";
    const end = link.end ? "–L" + endPosition(link.end) : "";
    return link.path + " · " + start + end;
  }

  function endPosition(position) {
    return position.line + ":" + position.column;
  }

  function evidenceLinks(ids) {
    const linksByID = {};
    (state.scene.evidence_links || []).forEach(function (link) { linksByID[link.id] = link; });
    const links = (ids || []).map(function (id) { return linksByID[id]; }).filter(Boolean);
    if (!links.length) return '<p class="muted">No source evidence is attached to this item.</p>';
    if (!sourceEnabled && !embeddedExport) return '<p class="muted">Source inspection is unavailable for this model-only session.</p>';
    if (!sourceEnabled) {
      return '<div class="evidence-list">' + links.map(function (link) {
        return '<div class="evidence-item" aria-label="' + escapeHTML((link.symbol || link.kind || "Source evidence") + ". " + locationText(link)) + '"><span>' + escapeHTML(link.symbol || link.kind || "Source evidence") + '</span><small>' + escapeHTML(locationText(link)) + ' · source not embedded</small></div>';
      }).join("") + "</div>";
    }
    return '<div class="evidence-list">' + links.map(function (link) {
      return '<button type="button" class="evidence-item" data-evidence-id="' + escapeHTML(link.id) + '"><span>' + escapeHTML(link.symbol || link.kind || "Source evidence") + '</span><small>' + escapeHTML(locationText(link)) + '</small></button>';
    }).join("") + "</div>";
  }

  function relationshipTargetLabel(relationship) {
    const node = state.scene.visible_nodes.find(function (item) { return item.id === relationship.to_visible_id; });
    return node ? node.label : relationship.to_visible_id;
  }

  function importsForNode(node) {
    const imports = [];
    state.scene.visible_relationships.filter(function (relationship) {
      return relationship.from_visible_id === node.id && !relationship.target_scope;
    }).forEach(function (relationship) {
      imports.push({
        kind: "relationship",
        id: relationship.id,
        title: relationshipTargetLabel(relationship),
        scope: relationship.target_scope || "local",
        count: relationship.count,
        confidence: relationship.confidence_state,
        evidenceIDs: relationship.evidence_ids || [],
        meta: relationship.type + " · " + relationshipListMeta(relationship),
        value: relationship
      });
    });
    (state.scene.reference_details || []).filter(function (reference) {
      return (reference.from_visible_ids || []).includes(node.id);
    }).forEach(function (reference) {
      imports.push({
        kind: "reference-detail",
        id: reference.id,
        title: reference.name,
        scope: reference.scope,
        count: reference.count,
        confidence: reference.confidence_state,
        evidenceIDs: reference.evidence_ids || [],
        meta: "import · " + referenceScopeLabel(reference.scope) + " · " + reference.count + " occurrence(s)",
        value: reference
      });
    });
    if (state.model && node.internal_relationship_ids) {
      node.internal_relationship_ids.forEach(function (relationshipID) {
        const relationship = (state.model.relationships || []).find(function (item) { return item.id === relationshipID; });
        if (!relationship) return;
        imports.push({
          kind: "internal-relationship",
          id: relationship.id,
          title: moduleNames([relationship.to_module_id])[0] || relationship.to_module_id,
          scope: "internal",
          count: 1,
          confidence: relationship.confidence ? confidenceState(relationship.confidence.score) : "unknown",
          evidenceIDs: relationship.source_reference_ids || [],
          meta: relationship.type + " · collapsed internal relationship",
          value: relationship
        });
      });
    }
    return imports.filter(function (item) { return state.importScope === "all" || item.scope === state.importScope; });
  }

  function confidenceState(score) {
    if (score == null) return "unknown";
    if (score >= 0.85) return "high";
    if (score >= 0.55) return "medium";
    return "low";
  }

  function importSection(node) {
    const imports = importsForNode(node);
    const scopes = ["all", "local", "standard_library", "external", "unresolved", "dynamic", "internal"];
    const options = scopes.map(function (scope) {
      return '<option value="' + scope + '"' + (scope === state.importScope ? " selected" : "") + '>' + escapeHTML(scope === "all" ? "All scopes" : referenceScopeLabel(scope)) + "</option>";
    }).join("");
    const list = imports.length ? '<div class="import-list">' + imports.map(function (item) {
      return '<button type="button" class="import-item" data-import-kind="' + escapeHTML(item.kind) + '" data-import-id="' + escapeHTML(item.id) + '"><span class="import-item-title">' + escapeHTML(item.title) + '</span><span class="import-item-meta">' + escapeHTML(referenceScopeLabel(item.scope)) + " · " + escapeHTML(item.confidence) + " confidence · " + escapeHTML(item.count) + " occurrence(s)</span><small>" + escapeHTML(item.meta) + "</small></button>";
    }).join("") + "</div>" : '<p class="muted">No imports match this scope in the current projection.</p>';
    return detailSection("Imports & evidence", '<div class="detail-filter"><label for="detail-import-scope">Scope</label><select id="detail-import-scope">' + options + "</select></div>" + list);
  }

  function moduleSection(node) {
    if (!node.module_ids || !node.module_ids.length) return "";
    const modules = node.module_ids.map(function (moduleID) {
      const module = modelModule(moduleID);
      if (!module) return '<li><code>' + escapeHTML(moduleID) + "</code></li>";
      return '<li><strong>' + escapeHTML(module.display_name) + '</strong><span>' + escapeHTML(module.kind + " · " + module.language + " · " + (module.tags && module.tags.length ? module.tags.join(", ") : "no tags")) + "</span></li>";
    }).join("");
    return detailSection("Canonical modules", '<ul class="module-list">' + modules + "</ul>");
  }

  function bindDetailActions() {
    const drill = elements.detailsContent.querySelector("[data-drill-path]");
    if (drill) drill.addEventListener("click", function () { navigationTo(drill.dataset.drillPath ? drill.dataset.drillPath.split("/") : []); });
    const filter = elements.detailsContent.querySelector("#detail-import-scope");
    if (filter) filter.addEventListener("change", function () { state.importScope = filter.value; renderDetails(); });
    elements.detailsContent.querySelectorAll("[data-evidence-id]").forEach(function (element) {
      element.addEventListener("click", function () { openSource(element.dataset.evidenceId); });
    });
    elements.detailsContent.querySelectorAll("[data-import-kind]").forEach(function (element) {
      element.addEventListener("click", function () { selectEntity(element.dataset.importKind, element.dataset.importId); });
    });
  }

  function sourcePanel() {
    if (!state.source) return "";
    if (state.source.loading) return detailSection("Read-only source", '<p class="muted">Loading source excerpt…</p>');
    if (state.source.error) return detailSection("Read-only source", '<div class="notice error">' + escapeHTML(state.source.error) + "</div>");
    const excerpt = state.source.data;
    const lines = (excerpt.lines || []).map(function (line) {
      return '<span class="source-line"><span class="source-number" aria-hidden="true">' + escapeHTML(line.number) + '</span><span class="source-text">' + escapeHTML(line.text) + "</span></span>";
    }).join("");
    const range = "L" + excerpt.start.line + ":" + excerpt.start.column + "–L" + excerpt.end.line + ":" + excerpt.end.column;
    return detailSection("Read-only source", '<div class="source-meta"><strong>' + escapeHTML(excerpt.path) + '</strong><span>' + escapeHTML(range) + ' · read-only</span></div><pre class="source-excerpt"><code>' + lines + "</code></pre>");
  }

  function renderDetails() {
    const scene = state.scene;
    if (!scene) return;
    if (!state.selected) {
      elements.detailsTitle.textContent = "Select an item";
      elements.detailsKind.textContent = "Overview";
      elements.detailsContent.innerHTML = '<p class="muted">The graphic and list use the same renderer-neutral scene. Select a node or directed relationship to inspect stable IDs, aggregation counts, layers, uncertainty, and evidence.</p>' + '<div class="detail-table">' + detailRow("Hierarchy", scene.hierarchy_path.length ? scene.hierarchy_path.join(" / ") : "Top level") + detailRow("Model status", scene.status, scene.status === "complete" ? "high" : "warning") + detailRow("Language", scene.project.language) + detailRow("Boundary", scene.project.boundary) + detailRow("Revision", scene.model_revision) + "</div>" + sourcePanel();
      bindDetailActions();
      return;
    }
    if (state.selected.kind === "node") {
      const node = scene.visible_nodes.find(function (item) { return item.id === state.selected.id; });
      if (!node) { state.selected = null; renderDetails(); return; }
      elements.detailsTitle.textContent = node.label;
      elements.detailsKind.textContent = node.kind;
      const drill = node.kind === "group" ? '<button class="button secondary detail-action" type="button" data-drill-path="' + escapeHTML(node.hierarchy_path.join("/")) + '">Open group</button>' : "";
      elements.detailsContent.innerHTML = drill + '<div class="detail-table">' +
        detailRow("Stable ID", node.id, "emphasis") +
        detailRow("Hierarchy", formatList(node.hierarchy_path, "Top level")) +
        detailRow("Modules", node.counts.module_count) +
        detailRow("Visible relationships", node.counts.relationship_count) +
        (node.counts.internal_relationship_count ? detailRow("Internal relationships", node.counts.internal_relationship_count + " (collapsed in overview)", "emphasis") : "") +
        (node.internal_relationship_ids && node.internal_relationship_ids.length ? detailRow("Internal IDs", formatList(node.internal_relationship_ids)) : "") +
        detailRow("Layer", node.layer == null ? formatList((node.layers || []).map(function (item) { return "Layer " + item; }), "Unassigned") : "Layer " + node.layer) +
        detailRow("Cycle state", node.cycle_state, node.cycle_state !== "none" ? "cycle" : "") +
        detailRow("Diagnostics", node.diagnostic_state, node.diagnostic_state !== "none" ? "warning" : "") +
        detailRow("Identity", node.identity_state || "stable") +
        detailRow("Relationship confidence", node.confidence_state === "not_applicable" ? "Not aggregated for local node" : node.confidence_state, node.confidence_state === "not_applicable" ? "" : node.confidence_state) +
        (node.reference_scope ? detailRow("Reference scope", referenceScopeLabel(node.reference_scope)) : "") +
        detailRow("Evidence", node.counts.evidence_count) +
        detailRow("Tags", formatList(node.tags)) +
        "</div>" + moduleSection(node) + importSection(node) + detailSection("Source evidence", evidenceLinks(node.evidence_ids)) + sourcePanel();
      bindDetailActions();
      return;
    }
    if (state.selected.kind === "reference-detail") {
      const reference = (scene.reference_details || []).find(function (item) { return item.id === state.selected.id; });
      if (!reference) { state.selected = null; renderDetails(); return; }
      elements.detailsTitle.textContent = reference.name;
      elements.detailsKind.textContent = "Import detail";
      elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Stable ID", reference.id, "emphasis") + detailRow("Reference scope", referenceScopeLabel(reference.scope)) + detailRow("Imported by", formatList(reference.from_visible_ids)) + detailRow("Import count", reference.count) + detailRow("Confidence", reference.confidence_state, reference.confidence_state) + detailRow("Basis", reference.confidence_basis || "Not supplied") + detailRow("Canonical IDs", formatList(reference.relationship_ids)) + "</div>" + detailSection("Source evidence", evidenceLinks(reference.evidence_ids)) + sourcePanel();
      bindDetailActions();
      return;
    }
    if (state.selected.kind === "cycle") {
      const cycle = scene.cycle_indicators.find(function (item) { return item.id === state.selected.id; });
      if (!cycle) { state.selected = null; renderDetails(); return; }
      const ids = [];
      scene.visible_relationships.forEach(function (relationship) {
        if (relationship.contributor_relationship_ids.some(function (id) { return cycle.relationship_ids.includes(id); })) ids.push.apply(ids, relationship.evidence_ids || []);
      });
      elements.detailsTitle.textContent = "Cycle";
      elements.detailsKind.textContent = "Graph health";
      elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Stable ID", cycle.id, "emphasis") + detailRow("Modules", moduleNames(cycle.module_ids).join(", ")) + detailRow("Relationship IDs", formatList(cycle.relationship_ids)) + detailRow("State", cycle.state, "cycle") + "</div>" + detailSection("Source evidence", evidenceLinks(ids)) + sourcePanel();
      bindDetailActions();
      return;
    }
    if (state.selected.kind === "diagnostic") {
      const diagnostic = scene.diagnostic_indicators.find(function (item) { return item.id === state.selected.id; });
      if (!diagnostic) { state.selected = null; renderDetails(); return; }
      elements.detailsTitle.textContent = diagnostic.code;
      elements.detailsKind.textContent = "Diagnostic";
      elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Severity", diagnostic.severity, diagnostic.severity) + detailRow("Message", diagnostic.message, "emphasis") + detailRow("Subject", diagnostic.subject || "Not supplied") + detailRow("Path", diagnostic.path || "Not supplied") + detailRow("Recoverable", diagnostic.recoverable ? "Yes" : "No") + "</div>" + detailSection("Source evidence", evidenceLinks(diagnostic.evidence_ids)) + sourcePanel();
      bindDetailActions();
      return;
    }
    if (state.selected.kind === "internal-relationship") {
      const relationship = state.model && (state.model.relationships || []).find(function (item) { return item.id === state.selected.id; });
      if (!relationship) { state.selected = null; renderDetails(); return; }
      elements.detailsTitle.textContent = relationship.type;
      elements.detailsKind.textContent = "Internal relation";
      elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Canonical ID", relationship.id, "emphasis") + detailRow("Direction", moduleNames([relationship.from_module_id])[0] + " → " + moduleNames([relationship.to_module_id])[0], "emphasis") + detailRow("Confidence", relationship.confidence ? confidenceState(relationship.confidence.score) : "unknown") + detailRow("Evidence", relationship.source_reference_ids.length) + "</div>" + detailSection("Source evidence", evidenceLinks(relationship.source_reference_ids)) + sourcePanel();
      bindDetailActions();
      return;
    }
    const relationship = scene.visible_relationships.find(function (item) { return item.id === state.selected.id; });
    if (!relationship) { state.selected = null; renderDetails(); return; }
    const nodesByID = {};
    scene.visible_nodes.forEach(function (node) { nodesByID[node.id] = node; });
    elements.detailsTitle.textContent = relationship.type;
    elements.detailsKind.textContent = "Directed relation";
    elements.detailsContent.innerHTML = '<div class="detail-table">' + detailRow("Stable ID", relationship.id, "emphasis") + detailRow("Direction", (nodesByID[relationship.from_visible_id] || {}).label + " → " + (nodesByID[relationship.to_visible_id] || {}).label, "emphasis") + detailRow("Type", relationship.type) + detailRow("Contributors", relationship.count) + detailRow("Canonical IDs", formatList(relationship.contributor_relationship_ids)) + detailRow("Cycle state", relationship.cycle_state, relationship.cycle_state !== "none" ? "cycle" : "") + detailRow("Confidence", relationship.confidence_state, relationship.confidence_state) + detailRow("Basis", relationship.confidence_basis || "Not supplied") + (relationship.target_scope ? detailRow("Target scope", referenceScopeLabel(relationship.target_scope)) : "") + detailRow("Evidence", relationship.evidence_ids.length) + "</div>" + detailSection("Source evidence", evidenceLinks(relationship.evidence_ids)) + sourcePanel();
    bindDetailActions();
  }

  async function openSource(evidenceID) {
    const link = (state.scene.evidence_links || []).find(function (item) { return item.id === evidenceID; });
    if (!link) return;
    const request = ++state.sourceRequest;
    state.source = { loading: true, link: link };
    renderDetails();
    const query = new URLSearchParams({ model_id: currentModelID(), evidence_id: evidenceID, path: link.path });
    if (link.start) query.set("start_line", String(link.start.line));
    if (link.end) query.set("end_line", String(link.end.line));
    try {
      const data = await getJSON("/v1/source?" + query.toString());
      if (request !== state.sourceRequest || !state.scene || data.model_revision !== state.scene.model_revision) return;
      state.source = { data: data, link: link };
      renderDetails();
    } catch (error) {
      if (request !== state.sourceRequest) return;
      state.source = { error: error.message || "The source excerpt could not be loaded.", link: link };
      renderDetails();
    }
  }

  function renderSupportLists() {
    const scene = state.scene;
    elements.cycleCount.textContent = scene.cycle_indicators.length;
    elements.diagnosticCount.textContent = scene.diagnostic_indicators.length;
    elements.cycles.innerHTML = scene.cycle_indicators.length ? scene.cycle_indicators.map(function (cycle) {
      return '<button type="button" class="support-item cycle" data-support-kind="cycle" data-support-id="' + escapeHTML(cycle.id) + '"><strong>' + escapeHTML(cycle.label) + '</strong><br><span>' + escapeHTML(cycle.module_ids.length) + ' module(s) · ' + escapeHTML(cycle.relationship_ids.length) + ' relationship(s)</span></button>';
    }).join("") : '<p class="muted">No cycles are present in this model.</p>';
    elements.diagnostics.innerHTML = scene.diagnostic_indicators.length ? scene.diagnostic_indicators.map(function (diagnostic) {
      return '<button type="button" class="support-item ' + classForState(diagnostic.severity) + '" data-support-kind="diagnostic" data-support-id="' + escapeHTML(diagnostic.id) + '"><strong>' + escapeHTML(diagnostic.code) + '</strong><br><span>' + escapeHTML(diagnostic.message) + '</span>' + (diagnostic.path ? '<br><code>' + escapeHTML(diagnostic.path) + '</code>' : "") + '</button>';
    }).join("") : '<p class="muted">No diagnostics are attached to this model.</p>';
    document.querySelectorAll("[data-support-id]").forEach(function (element) {
      element.addEventListener("click", function () { selectEntity(element.dataset.supportKind, element.dataset.supportId); });
    });
  }

  async function reanalyze() {
    if (!reanalysisEnabled || elements.reanalysisButton.disabled) return;
    elements.reanalysisButton.disabled = true;
    elements.reanalysisButton.textContent = "Reanalyzing…";
    const previousPath = state.scene ? state.scene.hierarchy_path.slice() : [];
    try {
      const response = await postJSON("/v1/reanalysis", { project_root: "", language: state.scene ? state.scene.project.language : null, options: {} });
      state.model = response.model || state.model;
      state.selected = null;
      state.source = null;
      state.history = [];
      const loaded = await loadScene(previousPath);
      if (!loaded && previousPath.length) await loadScene([]);
    } catch (error) {
      showError(error.message || "Reanalysis failed; the previous revision remains active.");
    } finally {
      elements.reanalysisButton.disabled = false;
      elements.reanalysisButton.textContent = "Reanalyze";
    }
  }

  document.addEventListener("DOMContentLoaded", init);
}());
