export function createAPI(context) {
  async function getJSON(path) {
    if (context.embeddedExport) return embeddedJSON(path);
    const response = await fetch(path, { headers: { Accept: "application/json" } });
    const body = await response.json();
    if (!response.ok) throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    return body;
  }

  function embeddedJSON(path) {
    const url = new URL(path, window.location.href);
    const modelPath = "/v1/models/" + encodeURIComponent(context.modelID);
    if (url.pathname === modelPath) return context.embeddedExport.model;
    if (url.pathname === modelPath + "/projection") {
      const pathValue = url.searchParams.getAll("path");
      const visibility = url.searchParams.get("reference_visibility") || "hidden";
      const key = JSON.stringify(pathValue) + "|" + visibility;
      if (context.embeddedExport.scenes && context.embeddedExport.scenes[key]) return context.embeddedExport.scenes[key];
      throw new Error("The exported hierarchy path or reference view is unavailable.");
    }
    if (url.pathname === "/v1/layout/config") return { schema_version: "arch-view.config/v1", layout: { algorithm: "layered", options: {} }, origin: "session", status: "valid", can_save: false, can_save_as: false, diagnostics: [] };
    if (url.pathname === "/v1/layout/options") return { schema_version: "arch-view.config/v1", adapter: { id: "export", version: "embedded", source: "export" }, algorithms: [], categories: [], options: [] };
    throw new Error("The self-contained export does not require network access.");
  }

  async function postJSON(path, value) {
    const response = await fetch(path, { method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(value) });
    const body = await response.json();
    if (!response.ok) throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    return body;
  }

  async function putJSON(path, value) {
    const response = await fetch(path, { method: "PUT", headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(value) });
    const body = await response.json();
    if (!response.ok) throw new Error(body && body.error && body.error.message ? body.error.message : "The viewer request failed.");
    return body;
  }

  function currentModelID() {
    const state = context.state;
    if (state.model && state.model.model_id) return state.model.model_id;
    if (state.scene && state.scene.model_id) return state.scene.model_id;
    return context.modelID;
  }

  return { currentModelID, getJSON, postJSON, putJSON };
}
