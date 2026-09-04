import { configurationFailureMessage, loadConfigurationCatalogs } from "./okf_profile_lifecycle.js";

export async function saveEditorProfile(state, api, elements, services, saveAs, editor) {
  if (state.editorSavePending) return;
  let value;
  try { value = editor.read(); } catch (error) { elements.editorStatus.textContent = error.message; return; }
  const draft = state.editorProfile;
  const declaration = JSON.stringify(draft);
  const advanced = elements.editorJSON.value;
  const profileID = state.profileID;
  const bundleID = state.bundleID;
  const revision = state.configurationRevision;
  const current = () => state.editorProfile === draft && JSON.stringify(draft) === declaration
    && elements.editorJSON.value === advanced && state.profileID === profileID && state.bundleID === bundleID;
  const operationID = "okf-profile-" + globalThis.crypto.randomUUID();
  state.editorSavePending = true;
  let saved = false;
  try {
    value = await editor.validate(value);
    if (!current()) return;
    let result;
    let savedID = profileID;
    if (saveAs || value.immutable || String(value.profile_id || "").startsWith("builtin:")) {
      const newID = String(value.profile_id || "").trim();
      const normalizedID = newID.startsWith("project:") ? newID : "project:" + newID;
      if (!newID || newID.startsWith("builtin:") || normalizedID === profileID) {
        elements.editorStatus.textContent = "Enter a new project Profile ID in the form, then choose Save As.";
        return;
      }
      result = await api.saveProfileAs(value, newID, profileID, revision, operationID);
      savedID = newID.startsWith("project:") ? newID : "project:" + newID;
    } else {
      result = await api.saveProfile(value, revision, operationID);
    }
    saved = true;
    if (state.configurationRevision === revision) state.configurationRevision = result.revision || revision;
    if (!current()) {
      if (state.editorProfile === draft) elements.editorStatus.textContent = "The earlier draft was saved. Your newer edits remain unsaved.";
      return;
    }
    const catalogs = await loadConfigurationCatalogs(api);
    if (!current()) return;
    state.profileID = savedID;
    state.layoutOverride = null;
    elements.editor.close();
    state.editorProfile = null;
    state.editorAdvancedDirty = false;
    Object.assign(state, catalogs);
    editor.renderSelectors();
    elements.profile.value = savedID;
    await services.selectProfile();
  } catch (error) {
    if (!current()) return;
    elements.editorStatus.textContent = saved
      ? "Profile was saved, but the viewer could not refresh: " + error.message
      : configurationFailureMessage(error);
    if (error.diagnostics && error.diagnostics.length) elements.editorStatus.textContent += " " + error.diagnostics.map((item) => item.message || item.code).join(" ");
  } finally {
    state.editorSavePending = false;
  }
}
