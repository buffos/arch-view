export function createOKFAPI(fetchImplementation = globalThis.fetch, basePath = "/v1/okf") {
  async function request(path, options = {}) {
    if (typeof fetchImplementation !== "function") throw new Error("The browser fetch API is unavailable.");
    const requestOptions = Object.assign({}, options, { headers: Object.assign({ "Accept": "application/json" }, options.headers || {}) });
    const response = await fetchImplementation(basePath + path, requestOptions);
    let payload = null;
    try { payload = await response.json(); } catch (error) { payload = null; }
    if (!response.ok || (payload && payload.error)) {
      const failure = payload && payload.error ? payload.error : {};
      const error = new Error(failure.message || "The OKF request failed.");
      error.code = failure.code || "okf_request_failed";
      error.details = failure.details || {};
      error.diagnostics = failure.diagnostics || [];
      error.status = response.status;
      throw error;
    }
    return payload && Object.prototype.hasOwnProperty.call(payload, "data") ? payload.data : payload;
  }

  async function rootRequest(path, options = {}) {
    if (typeof fetchImplementation !== "function") throw new Error("The browser fetch API is unavailable.");
    const requestOptions = Object.assign({}, options, { headers: Object.assign({ "Accept": "application/json" }, options.headers || {}) });
    const response = await fetchImplementation(path, requestOptions);
    let payload = null;
    try { payload = await response.json(); } catch (error) { payload = null; }
    if (!response.ok || (payload && payload.error)) {
      const failure = payload && payload.error ? payload.error : {};
      const error = new Error(failure.message || "The viewer request failed.");
      error.code = failure.code || "viewer_request_failed";
      error.diagnostics = failure.diagnostics || [];
      throw error;
    }
    return payload && Object.prototype.hasOwnProperty.call(payload, "data") ? payload.data : payload;
  }

  function json(path, method, value, headers = {}) {
    return request(path, { method, headers: Object.assign({ "Content-Type": "application/json" }, headers), body: JSON.stringify(value) });
  }

  return Object.freeze({
    refreshCatalog: () => json("/catalog/refresh", "POST", {}),
    getCatalog: () => request("/catalog"),
    getSummary: (bundleID) => request("/bundles/summary?bundle_id=" + encodeURIComponent(bundleID)),
    getProfiles: () => request("/profiles"),
    getLayoutOptions: () => rootRequest("/v1/layout/options"),
    getExtensions: () => request("/extensions"),
    validateProfile: (profile) => json("/profiles/validate", "POST", profile),
    selectBundle: (sessionID, bundleID) => json("/sessions/" + encodeURIComponent(sessionID) + "/bundle", "PUT", { bundle_id: bundleID }),
    selectProfile: (sessionID, profileID) => json("/sessions/" + encodeURIComponent(sessionID) + "/profile", "PUT", { profile_id: profileID }),
    setDepth: (sessionID, depth, full) => json("/sessions/" + encodeURIComponent(sessionID) + "/navigation/depth", "PUT", { depth, full }),
    focus: (sessionID, conceptID) => json("/sessions/" + encodeURIComponent(sessionID) + "/navigation/focus", "POST", { concept_id: conceptID }),
    back: (sessionID) => json("/sessions/" + encodeURIComponent(sessionID) + "/navigation/back", "POST", {}),
    topLevel: (sessionID) => json("/sessions/" + encodeURIComponent(sessionID) + "/navigation/top", "POST", {}),
    getProjection: (sessionID) => request("/sessions/" + encodeURIComponent(sessionID) + "/projection"),
    getNavigation: (sessionID) => request("/sessions/" + encodeURIComponent(sessionID) + "/navigation"),
    getDetail: (sessionID, conceptID) => request("/sessions/" + encodeURIComponent(sessionID) + "/concept-detail?concept_id=" + encodeURIComponent(conceptID)),
    bind: (bundleID, profileID, expectedRevision, operationID) => json("/bindings?bundle_id=" + encodeURIComponent(bundleID), "PUT", { profile_id: profileID, expected_revision: expectedRevision, operation_id: operationID }, operationID ? { "Idempotency-Key": operationID } : {}),
    saveProfile: (profile, expectedRevision, operationID) => json("/profiles/" + encodeURIComponent(profile.profile_id), "PUT", profile, Object.assign({}, expectedRevision ? { "If-Match": expectedRevision } : {}, operationID ? { "Idempotency-Key": operationID } : {})),
    saveProfileAs: (profile, newProfileID, sourceProfileID, expectedRevision, operationID) => json("/profiles/save-as", "POST", { profile, new_profile_id: newProfileID, source_profile_id: sourceProfileID, expected_revision: expectedRevision, operation_id: operationID }, operationID ? { "Idempotency-Key": operationID } : {}),
    renameProfile: (profileID, newProfileID, newName, expectedRevision, operationID) => json("/profiles/" + encodeURIComponent(profileID) + "/rename", "POST", { new_profile_id: newProfileID, new_name: newName, expected_revision: expectedRevision, operation_id: operationID }, operationID ? { "Idempotency-Key": operationID } : {}),
    deleteProfile: (profileID, replacementProfileID, neutralFallback, expectedRevision, operationID) => json("/profiles/" + encodeURIComponent(profileID), "DELETE", { replacement_profile_id: replacementProfileID, neutral_fallback: neutralFallback, expected_revision: expectedRevision, operation_id: operationID }, operationID ? { "Idempotency-Key": operationID } : {})
  });
}
