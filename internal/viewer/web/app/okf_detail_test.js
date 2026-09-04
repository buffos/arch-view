const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "okf_detail.js")).href).then(async ({ selectOKFConcept }) => {
  const state = { sessionID: "default", detailRequest: 0, detail: { concept_id: "old" }, layout: { positions: {} }, viewport: { zoom: 2 } };
  const layout = state.layout, viewport = state.viewport;
  const elements = { details: { innerHTML: "OLD BODY" }, detailsTitle: { textContent: "Old" } };
  let reject;
  const failure = selectOKFConcept(state, { getDetail: () => new Promise((_, no) => { reject = no; }) }, elements, "<new>");
  assert.equal(state.detail, null);
  assert.doesNotMatch(elements.details.innerHTML, /OLD BODY/);
  assert.match(elements.details.innerHTML, /&lt;new&gt;/);
  reject(new Error("<failed>"));
  await failure;
  assert.equal(elements.detailsTitle.textContent, "Unable to load concept");
  assert.match(elements.details.innerHTML, /&lt;failed&gt;/);
  for (const fails of [false, true]) {
    let resolveOld, rejectOld;
    const old = selectOKFConcept(state, { getDetail: () => new Promise((yes, no) => { resolveOld = yes; rejectOld = no; }) }, elements, "old", () => assert.fail("stale error published"));
    await selectOKFConcept(state, { getDetail: async () => ({ concept_id: "new", overview: { title: "New" } }) }, elements, "new");
    const body = elements.details.innerHTML;
    if (fails) rejectOld(new Error("old failed")); else resolveOld({ concept_id: "old" });
    await old;
    assert.equal(state.detail.concept_id, "new");
    assert.equal(elements.detailsTitle.textContent, "New");
    assert.equal(elements.details.innerHTML, body);
  }
  assert.equal(state.layout, layout);
  assert.equal(state.viewport, viewport);
}).catch((error) => { console.error(error); process.exitCode = 1; });
