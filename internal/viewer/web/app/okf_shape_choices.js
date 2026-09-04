import { escapeOKF } from "./okf_markup.js";

const choicesByRoot = new WeakMap();
const detailsByRoot = new WeakMap();

export function shapeChoicesMarkup(root) {
  const choices = choicesByRoot.get(root) || [];
  return choiceList("okf-shape-choices", choices) + choiceList("okf-detail-renderer-choices", detailsByRoot.get(root) || []);
}

function choiceList(id, choices) {
  return '<datalist id="' + id + '">' + choices.map((item) => '<option value="' + escapeOKF(item.value) + '">' + escapeOKF(item.label) + '</option>').join("") + '</datalist>';
}

export async function loadShapeChoices(root, api) {
  if (!root) return;
  try {
    const catalog = await api.getExtensions();
    const choices = (catalog.extensions || []).filter((item) => item.kind === "shape" && item.id && item.version)
      .map((item) => ({ value: item.id + "@" + item.version, label: item.description || item.id }));
    choicesByRoot.set(root, choices);
    const details = (catalog.extensions || []).filter((item) => item.kind === "detail_renderer" && item.id && item.version)
      .map((item) => ({ value: item.id, label: (item.description || item.id) + " (version " + item.version + ")" }));
    detailsByRoot.set(root, details);
    const list = root.querySelector('#okf-shape-choices');
    if (list) list.outerHTML = choiceList("okf-shape-choices", choices);
    const detailList = root.querySelector('#okf-detail-renderer-choices');
    if (detailList) detailList.outerHTML = choiceList("okf-detail-renderer-choices", details);
  } catch (_error) {
    // Suggestions are optional; server validation still governs typed values.
  }
}
