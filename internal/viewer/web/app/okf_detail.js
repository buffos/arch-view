import { escapeOKF, renderDetail } from "./okf_markup.js";
import { updateOKFSelection, updateOKFSemanticLinks } from "./okf_graph.js";

export async function selectOKFConcept(state, api, elements, conceptID, reportError) {
  const request = ++state.detailRequest;
  const current = () => state.detailRequest === request;
  state.selectedID = conceptID;
  state.detail = null;
  updateOKFSelection(elements.graph, conceptID);
  updateOKFSemanticLinks(elements.graph, state.snapshot, conceptID, state.layout, state.viewport);
  showNotice(elements, "Loading concept…", "Loading details for " + conceptID + "…");
  try {
    const detail = await api.getDetail(state.sessionID, conceptID);
    if (!current()) return;
    state.detail = detail;
    renderDetail(elements.details, detail);
    if (elements.detailsTitle) elements.detailsTitle.textContent = detail.overview?.title || conceptID;
  } catch (error) {
    if (!current()) return;
    showNotice(elements, "Unable to load concept", "Could not load " + conceptID + ": " + error.message);
    if (reportError) reportError(error);
  }
}

function showNotice(elements, title, message) {
  if (elements.detailsTitle) elements.detailsTitle.textContent = title;
  if (elements.details) elements.details.innerHTML = '<p class="muted" role="status">' + escapeOKF(message) + "</p>";
}
