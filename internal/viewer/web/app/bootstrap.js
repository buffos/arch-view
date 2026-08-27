import { createContext } from "./context.js";
import { createAPI } from "./api.js";
import { renderGraph } from "./graph.js";
import { downloadCurrentSVG } from "./export.js";
import { applyLayoutProfile, closeLayoutSettings, loadLayoutConfig, openLayoutSettings, prepareLayout, renderLayoutSettings, resetLayoutProfile, saveLayoutProfile, saveLayoutProfileAs, updateLayoutDraftAlgorithm, updateLayoutDraftOption } from "./layout.js";
import { createNavigation } from "./navigation.js";
import { changeZoom, fitViewport, persistViewport, renderViewportControls, resetLayout, resetZoom, syncFocusButton, toggleFocusMode } from "./viewport.js";
import { renderAccessibleList, renderDetails, renderSupportLists, openSource, selectEntity } from "./details.js";
import { hideError, renderAll, renderBreadcrumbs, renderSceneState, sceneLayoutKey, showError } from "./view.js";

export function bootstrap() {
  const context = createContext();
  const api = createAPI(context);
  const services = {
    changeZoom: function (delta) { changeZoom(context, delta, services); },
    navigationTo: function () {},
    openSource: function (evidenceID) { return openSource(context, evidenceID, api, services); },
    persistViewport: function () { persistViewport(context); },
    prepareLayout: function (scene, profile) { return prepareLayout(context, scene, profile, services); },
    renderAccessibleList: function () { renderAccessibleList(context, services); },
    renderAll: function () { renderAll(context, services); },
    renderBreadcrumbs: function () { renderBreadcrumbs(context, services.navigationTo); },
    renderDetails: function () { renderDetails(context, services); },
    renderGraph: function () { renderGraph(context, services); },
    renderLayoutSettings: function () { renderLayoutSettings(context); },
    renderSceneState: function () { renderSceneState(context); },
    renderSupportLists: function () { renderSupportLists(context, services); },
    renderViewportControls: function () { renderViewportControls(context); },
    sceneLayoutKey: sceneLayoutKey,
    selectEntity: function (kind, id, preserveDoubleClick) { selectEntity(context, kind, id, preserveDoubleClick, services); }
  };
  const navigation = createNavigation(context, api, services);
  services.navigationTo = navigation.navigationTo;

  context.elements.footerModelID.textContent = context.modelID;
  context.elements.reanalysisButton.hidden = !context.reanalysisEnabled;
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
  if (context.embeddedExport && context.embeddedExport.initial_reference_visibility) context.state.referenceVisibility = context.embeddedExport.initial_reference_visibility;
  void navigation.loadModel();
  void loadLayoutConfig(context, api, services);
  void navigation.loadScene(context.embeddedExport && Array.isArray(context.embeddedExport.initial_path) ? context.embeddedExport.initial_path : []);
}
