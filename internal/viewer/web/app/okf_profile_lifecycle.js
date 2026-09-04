export async function loadConfigurationCatalogs(api) {
  const [profiles, catalog] = await Promise.all([api.getProfiles(), api.getCatalog()]);
  return { profiles, catalog };
}

export function configurationFailureMessage(error) {
  const message = error.message || "Configuration could not be updated.";
  if (error.code !== "okf_revision_conflict") return message;
  return message + " Your draft has not been saved. Copy Advanced JSON before choosing Cancel. "
    + "Then use Refresh, reopen Edit profile, and merge your changes into the latest profile before saving.";
}

export function renameEditorProfile(state, api, elements, services, editor) {
  if (state.editorSavePending) return;
  let value;
  try { value = editor.read(); }
  catch (error) { elements.editorStatus.textContent = error.message; return; }
  const newID = String(value.profile_id || "").trim();
  if (!newID || newID.startsWith("builtin:")) {
    elements.editorStatus.textContent = "Enter a project Profile ID and Name in the form, then choose Rename.";
    return;
  }
  const newName = String(value.name || newID).trim();
  return changeProfileConfiguration(state, elements, {
    write: ({ profileID, revision }) => api.renameProfile(profileID, newID, newName, revision, "okf-rename-" + globalThis.crypto.randomUUID()),
    prepare: () => loadConfigurationCatalogs(api),
    publish: async (catalogs) => {
      state.profileID = newID.startsWith("project:") ? newID : "project:" + newID;
      Object.assign(state, catalogs);
      editor.renderSelectors();
      editor.close();
      await services.selectProfile();
    }
  });
}

export async function deleteEditorProfile(state, api, elements, services, editor) {
  if (state.editorSavePending || state.editorConfirmationPending) return;
  const draft = state.editorProfile;
  const profileID = state.profileID;
  if (!draft || draft.immutable || profileID.startsWith("builtin:")) return;
  const bundleID = state.bundleID;
  const revision = state.configurationRevision;
  const declaration = JSON.stringify(draft);
  const input = elements.editorJSON.value;
  const current = () => state.editorProfile === draft && JSON.stringify(draft) === declaration
    && elements.editorJSON.value === input && state.profileID === profileID
    && state.bundleID === bundleID && state.configurationRevision === revision;
  state.editorConfirmationPending = true;
  try {
    const confirmed = await editor.confirm({
      title: "Delete project profile?",
      message: "Delete " + profileID + " and use Neutral for affected bindings? This cannot be undone.",
      confirmLabel: "Delete profile"
    });
    if (!confirmed || !current()) return;
  } catch (error) {
    if (current()) elements.editorStatus.textContent = "Profile was not deleted: " + error.message;
    return;
  } finally { state.editorConfirmationPending = false; }
  return changeProfileConfiguration(state, elements, {
    write: ({ profileID, revision }) => api.deleteProfile(profileID, "builtin:neutral", true, revision, "okf-delete-" + globalThis.crypto.randomUUID()),
    prepare: () => loadConfigurationCatalogs(api),
    publish: async (catalogs) => {
      state.profileID = "builtin:neutral";
      Object.assign(state, catalogs);
      editor.renderSelectors();
      editor.close();
      await services.selectProfile();
    }
  });
}

// Configuration writes complete on the server even if the editor is closed.
// Only publish their UI effects into the editor that initiated the operation.
export async function changeProfileConfiguration(state, elements, operation) {
  if (state.editorSavePending) return;
  const draft = state.editorProfile;
  const declaration = JSON.stringify(draft);
  const input = elements.editorJSON.value;
  const profileID = state.profileID;
  const bundleID = state.bundleID;
  const revision = state.configurationRevision;
  const current = () => state.editorProfile === draft && JSON.stringify(draft) === declaration
    && elements.editorJSON.value === input && state.profileID === profileID && state.bundleID === bundleID;
  state.editorSavePending = true;
  let saved = false;
  try {
    const result = await operation.write({ profileID, bundleID, revision });
    saved = true;
    if (state.configurationRevision === revision) state.configurationRevision = result.revision || revision;
    if (!current()) return;
    const prepared = operation.prepare ? await operation.prepare() : null;
    if (current()) await operation.publish(prepared);
  } catch (error) {
    if (current()) elements.editorStatus.textContent = saved
      ? "Configuration was saved, but the viewer could not refresh: " + error.message
      : configurationFailureMessage(error);
  } finally {
    state.editorSavePending = false;
  }
}
