import { samePath } from "./utils.js";
import { hideError, showError } from "./view.js";
import { loadViewport, restoreScroll, viewportKey } from "./viewport.js";
import { prepareLayout } from "./layout.js";

export function createNavigation(context, api, services) {
  function rememberSceneContext(scene) {
    if (!scene) return;
    const key = viewportKey(scene);
    context.state.scrollContexts[key] = { left: context.elements.graph.scrollLeft, top: context.elements.graph.scrollTop };
  }

  function navigationTo(pathValue) {
    const current = context.state.scene ? context.state.scene.hierarchy_path : [];
    if (samePath(current, pathValue)) return;
    rememberSceneContext(context.state.scene);
    loadScene(pathValue).then(function (loaded) {
      if (loaded) {
        context.state.history.push(current.slice());
        services.renderBreadcrumbs();
      }
    });
  }

  function goBack() {
    if (!context.state.history.length) return;
    const target = context.state.history[context.state.history.length - 1];
    rememberSceneContext(context.state.scene);
    loadScene(target).then(function (loaded) {
      if (loaded) {
        context.state.history.pop();
        services.renderBreadcrumbs();
      }
    });
  }

  async function loadModel() {
    const requestedModelID = api.currentModelID();
    try {
      const value = await api.getJSON("/v1/models/" + encodeURIComponent(requestedModelID));
      if (requestedModelID !== api.currentModelID()) return;
      context.state.model = value;
      if (context.state.scene) services.renderDetails();
    } catch (error) {
      showError(context, error.message || "The canonical model could not be loaded.");
    }
  }

  async function loadScopes() {
    if (!context.aggregateEnabled || !context.analysisRunID || context.embeddedExport) {
      services.renderScopeSelector();
      return true;
    }
    const request = ++context.state.scopeRequest;
    try {
      const response = await api.getJSON("/v1/analyses/" + encodeURIComponent(context.analysisRunID) + "/scopes");
      if (request !== context.state.scopeRequest) return false;
      context.state.scopes = Array.isArray(response.scopes) ? response.scopes : [];
      services.renderScopeSelector();
      return true;
    } catch (error) {
      if (request !== context.state.scopeRequest) return false;
      context.state.scopes = [];
      services.renderScopeSelector();
      showError(context, error.message || "The analysis scopes could not be loaded.");
      return false;
    }
  }

  async function loadScene(pathValue) {
    const selectedPath = (pathValue || []).slice();
    const request = ++context.state.sceneRequest;
    try {
      const query = new URLSearchParams();
      query.set("mode", "overview");
      query.set("reference_visibility", context.state.referenceVisibility);
      selectedPath.forEach(function (segment) { query.append("path", segment); });
      let endpoint = "/v1/models/" + encodeURIComponent(api.currentModelID()) + "/projection?" + query.toString();
      if (context.aggregateEnabled && context.analysisRunID && !context.embeddedExport) {
        query.set("scope", context.state.activeScope || "all");
        endpoint = "/v1/analyses/" + encodeURIComponent(context.analysisRunID) + "/projection?" + query.toString();
      }
      const scene = await api.getJSON(endpoint);
      if (request !== context.state.sceneRequest) return false;
      context.state.lastNodeClick = null;
      context.state.scene = scene;
      if (context.aggregateEnabled && scene.scope_id) context.state.activeScope = scene.scope_id;
      context.state.referenceVisibility = scene.reference_visibility || context.state.referenceVisibility;
      context.state.selected = null;
      context.state.source = null;
      context.state.sourceRequest += 1;
      context.state.layout = null;
      context.state.layoutKey = services.sceneLayoutKey(scene);
      context.state.layoutError = false;
      context.state.layoutRequest += 1;
      context.state.viewport = loadViewport(context, scene);
      context.elements.referenceVisibility.value = context.state.referenceVisibility;
      services.renderScopeSelector();
      context.elements.footerModelID.textContent = scene.model_id;
      hideError(context);
      services.renderAll();
      restoreScroll(context, scene);
      void prepareLayout(context, scene, context.state.layoutProfile, services);
      return true;
    } catch (error) {
      if (request !== context.state.sceneRequest) return false;
      showError(context, error.message || "The local model could not be loaded.");
      return false;
    }
  }

  async function reanalyze() {
    if (!context.reanalysisEnabled || context.elements.reanalysisButton.disabled) return;
    context.elements.reanalysisButton.disabled = true;
    context.elements.reanalysisButton.textContent = "Reanalyzing…";
    const previousPath = context.state.scene ? context.state.scene.hierarchy_path.slice() : [];
    try {
      const response = await api.postJSON("/v1/reanalysis", context.aggregateEnabled
        ? { project_root: "", source_scope_policy: {}, cli_options: {} }
        : { project_root: "", language: context.state.scene ? context.state.scene.project.language : null, options: {} });
      if (context.aggregateEnabled) {
        context.analysisRunID = response.run_id || context.analysisRunID;
        context.state.model = response.model || context.state.model;
        context.state.scopes = Array.isArray(response.scopes) ? response.scopes : context.state.scopes;
        context.state.activeScope = "all";
        services.renderScopeSelector();
      } else {
        context.state.model = response.model || context.state.model;
      }
      context.state.selected = null;
      context.state.source = null;
      context.state.history = [];
      const loaded = await loadScene(previousPath);
      if (!loaded && previousPath.length) await loadScene([]);
    } catch (error) {
      showError(context, error.message || "Reanalysis failed; the previous revision remains active.");
    } finally {
      context.elements.reanalysisButton.disabled = false;
      context.elements.reanalysisButton.textContent = "Reanalyze";
    }
  }

  return { goBack, loadModel, loadScene, loadScopes, navigationTo, reanalyze, rememberSceneContext };
}
