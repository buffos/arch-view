import { renderProfileForm } from "./okf_profile_form.js";

export function updateEditorFromJSON(state, elements, preview) {
  state.editorPreviewRequest = (state.editorPreviewRequest || 0) + 1;
  state.editorAdvancedDirty = true;
  state.editorPreviewPending = true;
  elements.editorForm.inert = true;
  let value;
  try {
    value = JSON.parse(elements.editorJSON.value);
    if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("Profile must be an object");
  } catch (error) {
    elements.editorStatus.textContent = "Advanced JSON is not a valid profile object yet.";
    return;
  }
  state.editorProfile = value;
  return preview();
}

export async function refreshEditorPreview(state, api, elements) {
  if (!state.editorProfile) return;
  const request = state.editorPreviewRequest = (state.editorPreviewRequest || 0) + 1;
  const declaration = JSON.stringify(state.editorProfile);
  const current = () => state.editorPreviewRequest === request && state.editorProfile && JSON.stringify(state.editorProfile) === declaration;
  state.editorPreviewPending = true;
  elements.editorForm.inert = true;
  elements.editorStatus.textContent = "Resolving inherited profile settings…";
  try {
    const result = await api.validateProfile(JSON.parse(declaration));
    if (!current()) return;
    if (!result.effective_profile) throw new Error("Profile preview did not include effective settings.");
    renderProfileForm(elements.editorForm, state.editorProfile, result.effective_profile);
    elements.editorStatus.textContent = result.valid
      ? "Showing effective settings. Unchanged inherited values stay inherited."
      : (result.diagnostics || []).map((item) => item.message || item.code).join(" ") || "Profile validation failed.";
    state.editorPreviewPending = false;
    elements.editorForm.inert = false;
  } catch (error) {
    if (current()) elements.editorStatus.textContent = "Could not resolve inherited settings: " + error.message + ". Reopen the editor to retry.";
  }
}
