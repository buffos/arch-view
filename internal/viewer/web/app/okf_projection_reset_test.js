const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

Promise.all(["okf_projection_reset.js", "okf_detail.js", "okf_state.js"].map((file) =>
  import(pathToFileURL(path.join(__dirname, file)).href)
)).then(async ([{ clearOKFProjection }, { selectOKFConcept }, { beginRequest }]) => {
  const state = {
    request: 0, detailRequest: 0,
    bundleID: "broken", snapshot: { source: { bundle_id: "broken" } },
    layout: {}, viewport: {}, viewportKey: "old", viewportInitialized: true,
    selectedID: "old", detail: {}, layoutOverride: { algorithm: "mrtree" }
  };
  const elements = {};
  for (const key of ["graph", "summary", "breadcrumbs", "accessible", "hiddenConcepts"]) {
    elements[key] = { children: ["stale"], replaceChildren() { this.children = []; } };
  }
  for (const key of ["details", "detailsTitle", "counts", "accessibleCount", "back", "resetLayout"]) {
    elements[key] = { innerHTML: "stale", textContent: "stale", disabled: false };
  }
  clearOKFProjection(state, elements);
  for (const key of ["snapshot", "layout", "viewport", "detail"]) assert.equal(state[key], null);
  assert.equal(state.selectedID, "");
  assert.equal(state.viewportKey, "");
  assert.equal(state.viewportInitialized, false);
  assert.equal(state.bundleID, "broken", "keep the unavailable selection for recovery");
  assert.equal(state.layoutOverride.algorithm, "mrtree", "do not discard the user's layout draft");
  for (const key of ["graph", "summary", "breadcrumbs", "accessible", "hiddenConcepts"]) {
    assert.deepEqual(elements[key].children, []);
  }
  assert.doesNotMatch(elements.details.innerHTML, /stale/);
  assert.equal(elements.detailsTitle.textContent, "Select a concept");
  assert.equal(elements.counts.textContent, "0 visible concepts");
  assert.equal(elements.accessibleCount.textContent, "0 visible");
  assert.equal(elements.back.disabled, true);
  assert.equal(elements.resetLayout.disabled, true);
  // The old graph remains clickable while catalog discovery is in flight.
  for (const fails of [false, true]) {
    beginRequest(state);
    let resolveDetail, rejectDetail;
    const pending = selectOKFConcept(state, {
      getDetail: () => new Promise((resolve, reject) => { resolveDetail = resolve; rejectDetail = reject; })
    }, { details: elements.details, detailsTitle: elements.detailsTitle }, "old", () => {
      assert.fail("late detail error replaced the unavailable-bundle warning");
    });
    clearOKFProjection(state, elements);
    const emptyBody = elements.details.innerHTML;
    if (fails) rejectDetail(new Error("source disappeared"));
    else resolveDetail({ concept_id: "old", overview: { title: "Old source" } });
    await pending;
    assert.equal(state.detail, null, "late detail must not restore invalidated source content");
    assert.equal(elements.detailsTitle.textContent, "Select a concept");
    assert.equal(elements.details.innerHTML, emptyBody);
  }
}).catch((error) => { console.error(error); process.exitCode = 1; });
