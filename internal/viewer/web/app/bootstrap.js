import { createContext } from "./context.js";
import { createAPI } from "./api.js";
import { renderGraph } from "./graph.js";
import { downloadCurrentSVG } from "./export.js";
import { applyLayoutProfile, closeLayoutSettings, loadLayoutConfig, openLayoutSettings, prepareLayout, renderLayoutSettings, resetLayoutProfile, saveLayoutProfile, saveLayoutProfileAs, updateLayoutDraftAlgorithm, updateLayoutDraftOption } from "./layout.js";
import { createNavigation } from "./navigation.js";
import { changeZoom, fitViewport, persistViewport, renderViewportControls, resetLayout, resetZoom, syncFocusButton, toggleFocusMode } from "./viewport.js";
import { renderAccessibleList, renderDetails, renderSupportLists, openSource, openSourceFact, selectEntity } from "./details.js";
import { createInspectionController, parseInspectionRoute } from "./inspection.js";
import { createQualityBaseline as persistQualityBaseline, evaluateQualityProfile as runQualityProfile, initializeQualityRuleDraft, loadQualityProfiles as fetchQualityProfiles, loadQualityReport as fetchQualityReport, loadQualityRuleCatalog as fetchQualityRuleCatalog, loadQualitySource as fetchQualitySource, openQualityEvidence as fetchQualityEvidence, qualityBaselineCandidates, qualityRuleBindings, renderQualityBaselineSelection, renderQualityProfileControl, renderQualityRuleCatalog, saveQualityProfile as persistQualityProfile, saveQualityProfileAs as persistQualityProfileAs, setQualityRuleEnabled } from "./quality.js";
import { hideError, renderAll, renderBreadcrumbs, renderSceneState, renderScopeSelector, sceneLayoutKey, showError, showNotice } from "./view.js";
import { createLiveController } from "./live_status.js";

export function loadInitialState(navigation, context, initialPath) {
  const modelLoad = navigation.loadModel();
  const scopeLoad = navigation.loadScopes();
  return Promise.all([modelLoad, scopeLoad]).then(function () {
    const pathValue = initialPath || (context.embeddedExport && Array.isArray(context.embeddedExport.initial_path) ? context.embeddedExport.initial_path : []);
    return navigation.loadScene(pathValue);
  });
}

function renderQualityRuleEditor(context) {
  const elements = context.elements;
  if (!elements.qualityRuleCatalog) return;
  elements.qualityRuleCatalog.innerHTML = renderQualityRuleCatalog(context);
  const catalog = Array.isArray(context.state.qualityRuleCatalog) ? context.state.qualityRuleCatalog : [];
  const bindings = qualityRuleBindings(context);
  const enabled = bindings.filter(function (binding) { return binding.enabled; }).length;
  if (elements.qualityRuleSummary) elements.qualityRuleSummary.textContent = catalog.length ? enabled + " of " + catalog.length + " rules enabled for this session" : "";
  updateQualityProfileSaveAsAvailability(context);
}

function updateQualityProfileSaveAsAvailability(context) {
  const elements = context.elements;
  const state = context.state;
  const busy = state.qualityProfileSaveStatus === "loading";
  const readOnly = Boolean(context.liveEnabled);
  if (elements.qualityProfileSave) elements.qualityProfileSave.disabled = readOnly || busy || !state.activeQualityProfile || !Array.isArray(state.qualityRuleCatalog) || !state.qualityRuleCatalog.length;
  if (elements.qualityProfileSaveAs) elements.qualityProfileSaveAs.disabled = readOnly || busy || !state.activeQualityProfile || !Array.isArray(state.qualityRuleCatalog) || !state.qualityRuleCatalog.length;
  if (elements.qualityProfileSaveAsSubmit) {
    const complete = elements.qualityProfileSaveAsFile && elements.qualityProfileSaveAsFile.value.trim() && elements.qualityProfileSaveAsID && elements.qualityProfileSaveAsID.value.trim() && elements.qualityProfileSaveAsVersion && elements.qualityProfileSaveAsVersion.value.trim() && elements.qualityProfileSaveAsConfirm && elements.qualityProfileSaveAsConfirm.checked;
    elements.qualityProfileSaveAsSubmit.disabled = readOnly || busy || !complete;
  }
}

function selectedQualityRuleBindings(context) {
  return Array.isArray(context.state.qualityRuleDraft) ? qualityRuleBindings(context) : undefined;
}

export function bootstrap() {
  const context = createContext();
  const api = createAPI(context);
  const live = createLiveController(context, api);
  const initialRoute = parseInspectionRoute(window.location.search);
  if (initialRoute && !initialRoute.invalid) {
    if (initialRoute.scope && context.aggregateEnabled) context.state.activeScope = initialRoute.scope;
    if (initialRoute.referenceVisibility) context.state.referenceVisibility = initialRoute.referenceVisibility;
  }
  const services = {
    changeZoom: function (delta) { changeZoom(context, delta, services); },
    navigationTo: function () {},
    openSource: function (evidenceID) { return openSource(context, evidenceID, api, services); },
    openSourceFact: function (entityID) { return openSourceFact(context, entityID, api, services); },
    openInspection: function () {},
    openQualityBaseline: function () {},
    loadQualityProfiles: function () { return fetchQualityProfiles(context, api); },
    loadQualityRules: function (profileID, profileVersion) {
      return fetchQualityRuleCatalog(context, api, profileID, profileVersion).then(function (value) {
        renderQualityRuleEditor(context);
        return value;
      });
    },
    saveQualityProfile: function (ruleBindings) { return persistQualityProfile(context, api, ruleBindings); },
    saveQualityProfileAs: function (ruleBindings, fileName, profileID, profileVersion) { return persistQualityProfileAs(context, api, ruleBindings, fileName, profileID, profileVersion); },
    createQualityBaseline: function (options) { return persistQualityBaseline(context, api, options); },
    loadQualityReport: function () {
      return fetchQualityReport(context, api).then(function (value) {
        services.renderDetails();
        if (context.state.viewMode === "inspection") services.renderInspection();
        return value;
      });
    },
    evaluateQualityProfile: function (profileID, profileVersion, scope, ruleBindings) {
      return runQualityProfile(context, api, profileID, profileVersion, scope, ruleBindings).then(function (value) {
        services.renderDetails();
        if (context.state.viewMode === "inspection") services.renderInspection();
        return value;
      });
    },
    openQualityEvidence: function (findingID) {
      return fetchQualityEvidence(context, api, findingID, false).then(function () { services.renderInspection(); });
    },
    openQualitySource: function (findingID) {
      return fetchQualitySource(context, api, findingID).then(function () { services.renderInspection(); });
    },
    persistViewport: function () { persistViewport(context); },
    prepareLayout: function (scene, profile) { return prepareLayout(context, scene, profile, services); },
    renderAccessibleList: function () { renderAccessibleList(context, services); },
    renderAll: function () { renderAll(context, services); },
    renderBreadcrumbs: function () { renderBreadcrumbs(context, services.navigationTo); },
    renderDetails: function () { renderDetails(context, services); },
    renderGraph: function () { renderGraph(context, services); },
    renderInspection: function () {},
    renderLayoutSettings: function () { renderLayoutSettings(context); },
    renderSceneState: function () { renderSceneState(context); },
    renderScopeSelector: function () { renderScopeSelector(context); },
    renderSupportLists: function () { renderSupportLists(context, services); },
    renderViewportControls: function () { renderViewportControls(context); },
    sceneLayoutKey: sceneLayoutKey,
    selectEntity: function (kind, id, preserveDoubleClick) { selectEntity(context, kind, id, preserveDoubleClick, services); }
  };
  const inspection = createInspectionController(context, api, services);
  services.openInspection = inspection.openInspection;
  services.renderInspection = inspection.render;
  const navigation = createNavigation(context, api, services);
  services.navigationTo = navigation.navigationTo;
  inspection.bindNavigation(navigation);
  live.bindRevisionHandler(async function () {
    const pathValue = context.state.scene ? context.state.scene.hierarchy_path.slice() : [];
    await navigation.loadModel();
    const loaded = await navigation.loadScene(pathValue);
    if (!loaded && pathValue.length) await navigation.loadScene([]);
    context.state.qualityReportCache = {};
    context.state.qualityFindingsCache = {};
    context.state.qualityReport = null;
    context.state.qualityReportStatus = "loading";
    if (context.state.viewMode === "inspection") services.renderInspection();
    await services.loadQualityReport();
  });

  function closeQualityProfileDialog(restoreDraft) {
    if (restoreDraft && Array.isArray(context.state.qualityRuleDialogOriginal)) context.state.qualityRuleDraft = context.state.qualityRuleDialogOriginal;
    context.state.qualityRuleDialogOriginal = null;
    context.state.qualityProfileSaveStatus = "idle";
    context.state.qualityProfileSaveError = "";
    context.state.qualityProfileSaveAsOpen = false;
    if (context.elements.qualityProfileSaveAsPanel) context.elements.qualityProfileSaveAsPanel.hidden = true;
    if (context.elements.qualityProfileDialog && context.elements.qualityProfileDialog.open) context.elements.qualityProfileDialog.close();
    renderQualityRuleEditor(context);
  }

  function activeQualityProfileDescriptor() {
    const active = context.state.activeQualityProfile;
    return (context.state.qualityProfiles || []).find(function (profile) {
      return active && profile && profile.profile_id === active.profile_id && profile.profile_version === active.profile_version;
    }) || null;
  }

  function baselineSlug() {
    const active = context.state.activeQualityProfile || {};
    const descriptor = activeQualityProfileDescriptor();
    return String(descriptor && descriptor.name || active.profile_id || "profile")
      .replace(/^profile:/, "")
      .replace(/[^A-Za-z0-9_-]+/g, "-")
      .replace(/^-+|-+$/g, "") || "profile";
  }

  function updateQualityBaselineAvailability() {
    const elements = context.elements;
    const state = context.state;
    const profile = state.activeQualityProfile;
    const allActive = state.qualityBaselineAllActive !== false;
    const selected = Array.isArray(state.qualityBaselineSelectedFindings) ? state.qualityBaselineSelectedFindings : [];
    const complete = Boolean(profile && elements.qualityBaselineFile && elements.qualityBaselineFile.value.trim() && elements.qualityBaselineID && elements.qualityBaselineID.value.trim() && elements.qualityBaselineRevision && elements.qualityBaselineRevision.value.trim() && elements.qualityBaselineReason && elements.qualityBaselineReason.value.trim() && (allActive || selected.length));
    if (elements.qualityBaselineSubmit) elements.qualityBaselineSubmit.disabled = context.liveEnabled || state.qualityBaselineStatus === "loading" || !complete;
    if (elements.qualityBaselineStatus) {
      elements.qualityBaselineStatus.textContent = state.qualityBaselineStatus === "error" ? state.qualityBaselineError : state.qualityBaselineStatus === "loading" ? "Creating baseline…" : "";
    }
  }

  function renderQualityBaselineDialog() {
    const elements = context.elements;
    if (!elements.qualityBaselineFindingSelection) return;
    elements.qualityBaselineFindingSelection.innerHTML = renderQualityBaselineSelection(context);
    if (elements.qualityBaselineAllActive) elements.qualityBaselineAllActive.checked = context.state.qualityBaselineAllActive !== false;
    updateQualityBaselineAvailability();
  }

  function openQualityBaselineDialog(selection) {
    const elements = context.elements;
    const profile = context.state.activeQualityProfile;
    if (!profile || !elements.qualityBaselineDialog || context.embeddedExport) return;
    if (context.liveEnabled) {
      showError(context, "Baseline writes are disabled in the live viewer. Use an explicitly authorized CLI or MCP policy operation.");
      return;
    }
    const candidates = qualityBaselineCandidates(context);
    if (!candidates.length) {
      showError(context, "There are no active quality findings to add to a baseline.");
      return;
    }
    const requested = selection && selection !== "all" ? String(selection) : "";
    const selected = requested && candidates.some(function (finding) { return String(finding.id || finding.finding_key || "") === requested; }) ? [requested] : [];
    context.state.qualityBaselineAllActive = !selected.length;
    context.state.qualityBaselineSelectedFindings = selected;
    context.state.qualityBaselineStatus = "idle";
    context.state.qualityBaselineError = "";
    context.state.qualityBaselineMessage = "";
    const slug = baselineSlug();
    if (elements.qualityBaselineFile) elements.qualityBaselineFile.value = slug + "-baseline.json";
    if (elements.qualityBaselineID) elements.qualityBaselineID.value = "baseline:" + slug;
    if (elements.qualityBaselineRevision) elements.qualityBaselineRevision.value = "1.0.0";
    if (elements.qualityBaselineReason) elements.qualityBaselineReason.value = "";
    if (elements.qualityBaselineOwner) elements.qualityBaselineOwner.value = "";
    if (elements.qualityBaselineAttach) elements.qualityBaselineAttach.checked = true;
    renderQualityBaselineDialog();
    context.state.qualityBaselineDialogOpen = true;
    if (typeof elements.qualityBaselineDialog.showModal === "function") elements.qualityBaselineDialog.showModal();
    else elements.qualityBaselineDialog.hidden = false;
    if (elements.qualityBaselineReason && elements.qualityBaselineReason.focus) elements.qualityBaselineReason.focus();
  }

  function closeQualityBaselineDialog() {
    context.state.qualityBaselineDialogOpen = false;
    context.state.qualityBaselineStatus = "idle";
    context.state.qualityBaselineError = "";
    if (context.elements.qualityBaselineDialog && context.elements.qualityBaselineDialog.open) context.elements.qualityBaselineDialog.close();
    updateQualityBaselineAvailability();
  }

  function submitQualityBaseline() {
    const elements = context.elements;
    const profile = context.state.activeQualityProfile;
    if (!profile || context.state.qualityBaselineStatus === "loading") return;
    const allActive = context.state.qualityBaselineAllActive !== false;
    const options = {
      baselineID: elements.qualityBaselineID && elements.qualityBaselineID.value,
      revision: elements.qualityBaselineRevision && elements.qualityBaselineRevision.value,
      fileName: elements.qualityBaselineFile && elements.qualityBaselineFile.value,
      reason: elements.qualityBaselineReason && elements.qualityBaselineReason.value,
      owner: elements.qualityBaselineOwner && elements.qualityBaselineOwner.value,
      allActive: allActive,
      findingIDs: context.state.qualityBaselineSelectedFindings,
      attachToProfile: elements.qualityBaselineAttach ? elements.qualityBaselineAttach.checked : true
    };
    updateQualityBaselineAvailability();
    if (elements.qualityBaselineSubmit && elements.qualityBaselineSubmit.disabled) return;
    void services.createQualityBaseline(options).then(async function (value) {
      if (!value || value.status !== "created") throw new Error("The quality baseline was not created.");
      if (value.attached) {
        await services.loadQualityProfiles();
        await services.evaluateQualityProfile(profile.profile_id, profile.profile_version, context.state.activeScope || "all");
        if (context.state.qualityEvaluationStatus !== "available") throw new Error(context.state.qualityEvaluationError || "The baseline could not be applied.");
      } else {
        showNotice(context, value.message || "Quality baseline created.");
      }
      closeQualityBaselineDialog();
      services.renderDetails();
      if (context.state.viewMode === "inspection") services.renderInspection();
    }).catch(function (error) {
      context.state.qualityBaselineStatus = "error";
      context.state.qualityBaselineError = error && error.message || "The quality baseline could not be created.";
      renderQualityBaselineDialog();
    });
  }

  services.openQualityBaseline = openQualityBaselineDialog;

  function openQualityProfileSaveAs() {
    const elements = context.elements;
    const active = context.state.activeQualityProfile;
    if (!active || !elements.qualityProfileSaveAsPanel) return;
    const descriptor = activeQualityProfileDescriptor();
    const name = String(descriptor && descriptor.name || "profile").replace(/[^A-Za-z0-9_-]+/g, "-").replace(/^-+|-+$/g, "") || "profile";
    const sourceName = String(active.profile_id || "").replace(/^profile:/, "").replace(/[^A-Za-z0-9_-]+/g, "-").replace(/^-+|-+$/g, "") || name;
    if (elements.qualityProfileSaveAsFile) elements.qualityProfileSaveAsFile.value = name + "-custom.json";
    if (elements.qualityProfileSaveAsID) elements.qualityProfileSaveAsID.value = "profile:" + sourceName + "-custom";
    if (elements.qualityProfileSaveAsVersion) elements.qualityProfileSaveAsVersion.value = active.profile_version || "1.0.0";
    if (elements.qualityProfileSaveAsConfirm) elements.qualityProfileSaveAsConfirm.checked = false;
    context.state.qualityProfileSaveAsOpen = true;
    context.state.qualityProfileSaveError = "";
    elements.qualityProfileSaveAsPanel.hidden = false;
    updateQualityProfileSaveAsAvailability(context);
    if (elements.qualityProfileSaveAsFile && elements.qualityProfileSaveAsFile.focus) elements.qualityProfileSaveAsFile.focus();
  }

  function closeQualityProfileSaveAs() {
    context.state.qualityProfileSaveAsOpen = false;
    if (context.elements.qualityProfileSaveAsPanel) context.elements.qualityProfileSaveAsPanel.hidden = true;
    updateQualityProfileSaveAsAvailability(context);
  }

  function openQualityProfileDialog() {
    const profile = context.state.activeQualityProfile;
    if (!profile || !context.elements.qualityProfileDialog) return;
    const begin = function () {
      context.state.qualityRuleDialogOriginal = qualityRuleBindings(context);
      initializeQualityRuleDraft(context);
      context.state.qualityProfileSaveStatus = "idle";
      context.state.qualityProfileSaveError = "";
      context.state.qualityProfileSaveAsOpen = false;
      if (context.elements.qualityProfileSaveAsPanel) context.elements.qualityProfileSaveAsPanel.hidden = true;
      renderQualityRuleEditor(context);
      if (context.elements.qualityProfileEditorStatus) context.elements.qualityProfileEditorStatus.textContent = context.liveEnabled
        ? "Switch rules on or off, then apply the temporary selection. Live viewer policy files are read-only."
        : "Switch rules on or off, then apply the temporary selection.";
      if (typeof context.elements.qualityProfileDialog.showModal === "function") context.elements.qualityProfileDialog.showModal();
      else context.elements.qualityProfileDialog.hidden = false;
    };
    if (context.state.qualityRuleCatalogStatus === "available") {
      begin();
      return;
    }
    if (context.elements.qualityProfileEditorStatus) context.elements.qualityProfileEditorStatus.textContent = "Loading the available rules…";
    void services.loadQualityRules(profile.profile_id, profile.profile_version).then(begin);
  }

  function applyQualityProfileDraft() {
    const profile = context.state.activeQualityProfile;
    if (!profile) return;
    const bindings = qualityRuleBindings(context);
    setQualityProfileActionsDisabled(true);
    if (context.elements.qualityProfileEditorStatus) context.elements.qualityProfileEditorStatus.textContent = "Running the selected checks…";
    void services.evaluateQualityProfile(profile.profile_id, profile.profile_version, context.state.activeScope || "all", bindings).then(function (value) {
      if (value && context.state.qualityEvaluationStatus === "available") {
        context.state.qualityRuleDialogOriginal = null;
        if (context.elements.qualityProfileDialog && context.elements.qualityProfileDialog.open) context.elements.qualityProfileDialog.close();
      } else if (context.elements.qualityProfileEditorStatus) {
        context.elements.qualityProfileEditorStatus.textContent = context.state.qualityEvaluationError || "The selected checks could not be evaluated.";
      }
    }).finally(function () {
      setQualityProfileActionsDisabled(false);
      renderQualityRuleEditor(context);
    });
  }

  function setQualityProfileActionsDisabled(disabled) {
    [context.elements.qualityProfileSave, context.elements.qualityProfileSaveAs, context.elements.qualityProfileApply, context.elements.qualityProfileSaveAsCancel, context.elements.qualityProfileClose, context.elements.qualityProfileCancel].forEach(function (element) {
      if (element) element.disabled = disabled;
    });
    updateQualityProfileSaveAsAvailability(context);
  }

  function saveQualityProfileDraft() {
    const profile = context.state.activeQualityProfile;
    if (!profile) return;
    const bindings = qualityRuleBindings(context);
    closeQualityProfileSaveAs();
    context.state.qualityProfileSaveStatus = "loading";
    context.state.qualityProfileSaveError = "";
    setQualityProfileActionsDisabled(true);
    if (context.elements.qualityProfileEditorStatus) context.elements.qualityProfileEditorStatus.textContent = "Saving the selected rules to this profile…";
    void services.saveQualityProfile(bindings).then(async function (value) {
      if (!value || value.status !== "saved") throw new Error("The quality profile was not saved.");
      await services.evaluateQualityProfile(profile.profile_id, profile.profile_version, context.state.activeScope || "all");
      if (context.state.qualityEvaluationStatus !== "available") throw new Error(context.state.qualityEvaluationError || "The saved profile could not be evaluated.");
      await services.loadQualityProfiles();
      await services.loadQualityRules(profile.profile_id, profile.profile_version);
      context.state.qualityRuleDraft = bindings;
      context.state.qualityEvaluationTemporary = false;
      context.state.qualityRuleDialogOriginal = null;
      closeQualityProfileDialog(false);
    }).catch(function (error) {
      context.state.qualityProfileSaveStatus = "error";
      context.state.qualityProfileSaveError = error && error.message || "The quality profile could not be saved.";
      if (context.elements.qualityProfileEditorStatus) context.elements.qualityProfileEditorStatus.textContent = context.state.qualityProfileSaveError;
    }).finally(function () {
      if (context.state.qualityProfileSaveStatus !== "error") context.state.qualityProfileSaveStatus = "idle";
      setQualityProfileActionsDisabled(false);
      renderQualityRuleEditor(context);
    });
  }

  function saveQualityProfileAsDraft() {
    const profile = context.state.activeQualityProfile;
    const elements = context.elements;
    if (!profile || !elements.qualityProfileSaveAsFile || !elements.qualityProfileSaveAsID || !elements.qualityProfileSaveAsVersion || !elements.qualityProfileSaveAsConfirm || !elements.qualityProfileSaveAsConfirm.checked) return;
    const bindings = qualityRuleBindings(context);
    const fileName = elements.qualityProfileSaveAsFile.value.trim();
    const profileID = elements.qualityProfileSaveAsID.value.trim();
    const profileVersion = elements.qualityProfileSaveAsVersion.value.trim();
    context.state.qualityProfileSaveStatus = "loading";
    context.state.qualityProfileSaveError = "";
    setQualityProfileActionsDisabled(true);
    if (elements.qualityProfileEditorStatus) elements.qualityProfileEditorStatus.textContent = "Saving the new quality profile…";
    void services.saveQualityProfileAs(bindings, fileName, profileID, profileVersion).then(async function (value) {
      if (!value || value.status !== "created" || !value.profile) throw new Error("The new quality profile was not saved.");
      const created = value.profile;
      context.state.activeQualityProfile = { profile_id: created.profile_id, profile_version: created.profile_version };
      context.state.activeQualityProfileKey = created.profile_id + "\u0000" + created.profile_version;
      await services.evaluateQualityProfile(created.profile_id, created.profile_version, context.state.activeScope || "all");
      if (context.state.qualityEvaluationStatus !== "available") throw new Error(context.state.qualityEvaluationError || "The new profile could not be evaluated.");
      await services.loadQualityProfiles();
      await services.loadQualityRules(created.profile_id, created.profile_version);
      context.state.qualityRuleDraft = bindings;
      context.state.qualityEvaluationTemporary = false;
      context.state.qualityRuleDialogOriginal = null;
      closeQualityProfileDialog(false);
    }).catch(function (error) {
      context.state.qualityProfileSaveStatus = "error";
      context.state.qualityProfileSaveError = error && error.message || "The new quality profile could not be saved.";
      if (elements.qualityProfileEditorStatus) elements.qualityProfileEditorStatus.textContent = context.state.qualityProfileSaveError;
    }).finally(function () {
      if (context.state.qualityProfileSaveStatus !== "error") context.state.qualityProfileSaveStatus = "idle";
      setQualityProfileActionsDisabled(false);
      renderQualityRuleEditor(context);
    });
  }

  context.elements.footerModelID.textContent = context.modelID;
  context.elements.reanalysisButton.hidden = !context.reanalysisEnabled;
  renderQualityProfileControl(context);
  renderQualityRuleEditor(context);
  context.elements.listToggle.addEventListener("click", function () {
    const pressed = context.elements.listToggle.getAttribute("aria-pressed") === "true";
    context.elements.listToggle.setAttribute("aria-pressed", String(!pressed));
    document.body.classList.toggle("list-mode", !pressed);
  });
  context.elements.search.addEventListener("input", function (event) {
    context.state.query = event.target.value.trim().toLowerCase();
    services.renderGraph();
    services.renderAccessibleList();
  });
  context.elements.referenceVisibility.addEventListener("change", function (event) {
    context.state.referenceVisibility = event.target.value;
    context.state.selected = null;
    navigation.loadScene(context.state.scene ? context.state.scene.hierarchy_path : []);
  });
  context.elements.scopeSelector.addEventListener("change", function (event) {
    context.state.activeScope = event.target.value || "all";
    context.state.history = [];
    context.state.selected = null;
    void navigation.loadScene([]).then(function () {
      const profile = context.state.activeQualityProfile;
      return profile ? services.evaluateQualityProfile(profile.profile_id, profile.profile_version, context.state.activeScope, selectedQualityRuleBindings(context)) : services.loadQualityReport();
    });
  });
  context.elements.qualityProfile.addEventListener("change", function (event) {
    const option = event.target.selectedOptions && event.target.selectedOptions[0];
    const profileID = option && option.dataset.profileId;
    const profileVersion = option && option.dataset.profileVersion;
    if (!profileID || !profileVersion) {
      context.state.activeQualityProfile = null;
      context.state.activeQualityProfileKey = "";
      context.state.qualityEvaluationTemporary = false;
      context.state.qualityRuleCatalog = [];
      context.state.qualityRuleCatalogStatus = "idle";
      context.state.qualityRuleCatalogError = "";
      context.state.qualityRuleDraft = null;
      context.state.qualityReport = null;
      context.state.qualityReportStatus = "missing";
      context.state.qualityReportError = "Choose a quality profile to evaluate the loaded model.";
      context.state.qualityFindingsPage = null;
      context.state.qualityFindingsCache = {};
      context.state.qualityEvidence = null;
      renderQualityProfileControl(context);
      services.renderDetails();
      if (context.state.viewMode === "inspection") services.renderInspection();
      renderQualityRuleEditor(context);
      return;
    }
    context.state.activeQualityProfile = { profile_id: profileID, profile_version: profileVersion };
    context.state.activeQualityProfileKey = profileID + "\u0000" + profileVersion;
    context.state.qualityEvaluationTemporary = false;
    context.state.qualityRuleDraft = null;
    void services.loadQualityRules(profileID, profileVersion);
    void services.evaluateQualityProfile(profileID, profileVersion, context.state.activeScope || "all");
  });
  if (context.elements.qualityProfileConfigure) context.elements.qualityProfileConfigure.addEventListener("click", openQualityProfileDialog);
  if (context.elements.qualityProfileClose) context.elements.qualityProfileClose.addEventListener("click", function () { closeQualityProfileDialog(true); });
  if (context.elements.qualityProfileCancel) context.elements.qualityProfileCancel.addEventListener("click", function () { closeQualityProfileDialog(true); });
  if (context.elements.qualityProfileDialog) context.elements.qualityProfileDialog.addEventListener("cancel", function (event) { event.preventDefault(); closeQualityProfileDialog(true); });
  if (context.elements.qualityRuleSearch) context.elements.qualityRuleSearch.addEventListener("input", function (event) {
    context.state.qualityRuleSearch = event.target.value.trim().toLowerCase();
    renderQualityRuleEditor(context);
    context.elements.qualityRuleSearch.focus();
  });
  if (context.elements.qualityRuleCatalog) context.elements.qualityRuleCatalog.addEventListener("change", function (event) {
    const input = event.target;
    if (!input || !input.matches("[data-quality-rule-toggle]")) return;
    setQualityRuleEnabled(context, input.dataset.ruleId, input.dataset.ruleVersion, input.checked);
    renderQualityRuleEditor(context);
    const next = Array.from(context.elements.qualityRuleCatalog.querySelectorAll("[data-quality-rule-toggle]")).find(function (candidate) {
      return candidate.dataset.ruleId === input.dataset.ruleId && candidate.dataset.ruleVersion === input.dataset.ruleVersion;
    });
    if (next) next.focus();
  });
  if (context.elements.qualityProfileApply) context.elements.qualityProfileApply.addEventListener("click", applyQualityProfileDraft);
  if (context.elements.qualityProfileSave) context.elements.qualityProfileSave.addEventListener("click", saveQualityProfileDraft);
  if (context.elements.qualityProfileSaveAs) context.elements.qualityProfileSaveAs.addEventListener("click", openQualityProfileSaveAs);
  if (context.elements.qualityProfileSaveAsCancel) context.elements.qualityProfileSaveAsCancel.addEventListener("click", closeQualityProfileSaveAs);
  [context.elements.qualityProfileSaveAsFile, context.elements.qualityProfileSaveAsID, context.elements.qualityProfileSaveAsVersion].forEach(function (element) {
    if (element) element.addEventListener("input", function () { updateQualityProfileSaveAsAvailability(context); });
  });
  if (context.elements.qualityProfileSaveAsConfirm) context.elements.qualityProfileSaveAsConfirm.addEventListener("change", function () { updateQualityProfileSaveAsAvailability(context); });
  if (context.elements.qualityProfileSaveAsSubmit) context.elements.qualityProfileSaveAsSubmit.addEventListener("click", saveQualityProfileAsDraft);
  if (context.elements.qualityBaselineClose) context.elements.qualityBaselineClose.addEventListener("click", closeQualityBaselineDialog);
  if (context.elements.qualityBaselineCancel) context.elements.qualityBaselineCancel.addEventListener("click", closeQualityBaselineDialog);
  if (context.elements.qualityBaselineDialog) context.elements.qualityBaselineDialog.addEventListener("cancel", function (event) { event.preventDefault(); closeQualityBaselineDialog(); });
  if (context.elements.qualityBaselineAllActive) context.elements.qualityBaselineAllActive.addEventListener("change", function (event) {
    context.state.qualityBaselineAllActive = event.target.checked;
    renderQualityBaselineDialog();
  });
  if (context.elements.qualityBaselineFindingSelection) context.elements.qualityBaselineFindingSelection.addEventListener("change", function (event) {
    const input = event.target;
    if (!input || !input.matches("[data-quality-baseline-finding]")) return;
    const selected = new Set(Array.isArray(context.state.qualityBaselineSelectedFindings) ? context.state.qualityBaselineSelectedFindings : []);
    if (input.checked) selected.add(input.value);
    else selected.delete(input.value);
    context.state.qualityBaselineSelectedFindings = Array.from(selected);
    updateQualityBaselineAvailability();
  });
  [context.elements.qualityBaselineFile, context.elements.qualityBaselineID, context.elements.qualityBaselineRevision, context.elements.qualityBaselineReason, context.elements.qualityBaselineOwner, context.elements.qualityBaselineAttach].forEach(function (element) {
    if (element) element.addEventListener("input", updateQualityBaselineAvailability);
    if (element && element.type === "checkbox") element.addEventListener("change", updateQualityBaselineAvailability);
  });
  if (context.elements.qualityBaselineSubmit) context.elements.qualityBaselineSubmit.addEventListener("click", submitQualityBaseline);
  context.elements.backButton.addEventListener("click", navigation.goBack);
  context.elements.zoomOut.addEventListener("click", function () { services.changeZoom(-0.12); });
  context.elements.zoomIn.addEventListener("click", function () { services.changeZoom(0.12); });
  context.elements.resetZoom.addEventListener("click", function () { resetZoom(context, services); });
  context.elements.fitViewport.addEventListener("click", function () { fitViewport(context, services); });
  context.elements.resetLayout.addEventListener("click", function () { resetLayout(context, services); });
  context.elements.layoutSettingsButton.hidden = Boolean(context.embeddedExport);
  context.elements.layoutSettingsButton.addEventListener("click", function () { openLayoutSettings(context); });
  context.elements.layoutSettingsClose.addEventListener("click", function () { closeLayoutSettings(context); });
  context.elements.layoutSettingsDialog.addEventListener("cancel", function (event) { event.preventDefault(); closeLayoutSettings(context); });
  context.elements.layoutAlgorithm.addEventListener("change", function (event) { updateLayoutDraftAlgorithm(context, event); });
  context.elements.layoutOptionSearch.addEventListener("input", function (event) {
    context.state.layoutOptionSearch = event.target.value.trim().toLowerCase();
    renderLayoutSettings(context);
  });
  context.elements.layoutOptionsList.addEventListener("input", function (event) { updateLayoutDraftOption(context, event); });
  context.elements.layoutOptionsList.addEventListener("change", function (event) { updateLayoutDraftOption(context, event); });
  context.elements.layoutResetDefaults.addEventListener("click", function () { resetLayoutProfile(context, api, services); });
  context.elements.layoutSettingsApply.addEventListener("click", function () { applyLayoutProfile(context, api, services); });
  context.elements.layoutSettingsSave.addEventListener("click", function () { saveLayoutProfile(context, api, services); });
  context.elements.layoutSettingsSaveAs.addEventListener("click", function () { saveLayoutProfileAs(context, api, services); });
  context.elements.focusToggle.addEventListener("click", function () { toggleFocusMode(context); });
  context.elements.downloadSVG.addEventListener("click", function () { downloadCurrentSVG(context); });
  document.addEventListener("fullscreenchange", function () { syncFocusButton(context); });
  context.elements.reanalysisButton.addEventListener("click", navigation.reanalyze);
  if (context.embeddedExport && context.embeddedExport.initial_reference_visibility && !initialRoute) context.state.referenceVisibility = context.embeddedExport.initial_reference_visibility;
  const initialLoad = context.liveEnabled ? live.waitForReady() : Promise.resolve();
  void initialLoad.then(function () {
    return loadInitialState(navigation, context, initialRoute && !initialRoute.invalid ? initialRoute.path : null);
  }).then(function () {
    inspection.initialize();
    live.start();
    return services.loadQualityProfiles();
  }).then(function () {
    void services.loadQualityReport();
  }).catch(function (error) {
    showError(context, error && error.message ? error.message : "The viewer could not load the live session.");
  });
  void loadLayoutConfig(context, api, services);
}
