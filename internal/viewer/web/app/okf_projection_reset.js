import { renderDetail } from "./okf_markup.js";

// A refreshed catalog can invalidate the source of every visible projection item.
export function clearOKFProjection(state, elements) {
  state.detailRequest = (state.detailRequest || 0) + 1;
  state.snapshot = null;
  state.layout = null;
  state.viewport = null;
  state.viewportKey = "";
  state.viewportInitialized = false;
  state.selectedID = "";
  state.detail = null;
  renderDetail(elements.details, null);
  if (elements.detailsTitle) elements.detailsTitle.textContent = "Select a concept";
  for (const key of ["graph", "summary", "breadcrumbs", "accessible", "hiddenConcepts"]) {
    elements[key]?.replaceChildren();
  }
  elements.counts.textContent = "0 visible concepts";
  if (elements.accessibleCount) elements.accessibleCount.textContent = "0 visible";
  if (elements.back) elements.back.disabled = true;
  if (elements.resetLayout) elements.resetLayout.disabled = true;
}
