import { cloneOKFLayoutProfile } from "./okf_layout.js";

export function createOKFState(sessionID) {
  return {
    sessionID: sessionID || "default",
    catalog: null,
    profiles: null,
    profile: null,
    snapshot: null,
    detail: null,
    selectedID: "",
    viewport: null,
    viewportKey: "",
    viewportInitialized: false,
    layout: null,
    layoutProfile: cloneOKFLayoutProfile(),
    layoutOverride: null,
    layoutCatalog: null,
    layoutSettingsOpen: false,
    layoutOptionSearch: "",
    layoutMessage: "",
    layoutMessageError: false,
    editorProfile: null,
    editorAdvancedDirty: false,
    depth: 2,
    full: false,
    request: 0,
    detailRequest: 0,
    configurationRevision: "",
    busy: false,
    diagnostics: [],
    error: ""
  };
}

export function beginRequest(state) {
  state.request += 1;
  state.detailRequest = (state.detailRequest || 0) + 1;
  return state.request;
}

export function isCurrent(state, request) {
  return state.request === request;
}

export async function refreshCatalogState(state, api, request) {
  const catalog = await api.refreshCatalog();
  if (!isCurrent(state, request)) return null;
  const [profiles, layoutCatalog] = await Promise.all([
    api.getProfiles(), api.getLayoutOptions().catch(() => null)
  ]);
  if (!isCurrent(state, request)) return null;
  Object.assign(state, { catalog, profiles, layoutCatalog });
  return catalog;
}

export function profileByID(catalog, profileID) {
  const values = catalog && Array.isArray(catalog.profiles) ? catalog.profiles : [];
  return values.find((value) => value.profile_id === profileID) || null;
}

export function catalogDiagnostics(catalog) {
  const diagnostics = [...(catalog?.diagnostics || [])];
  for (const bundle of catalog?.bundles || []) {
    if (bundle.selectable) continue; // Active projection reports valid-bundle diagnostics.
    for (const diagnostic of bundle.diagnostics || []) {
      diagnostics.push({ ...diagnostic, bundle_id: diagnostic.bundle_id || bundle.bundle_id });
    }
  }
  return diagnostics;
}
