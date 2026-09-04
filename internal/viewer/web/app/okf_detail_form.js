import { escapeOKF } from "./okf_markup.js";

export function detailRendererMarkup(details = {}) {
  const selection = details.renderer || {};
  return '<section class="okf-form-section"><h4>Detail presentation</h4><p class="muted">Leave the renderer empty to inherit the base profile. Without an inherited renderer, formatted CommonMark is used. Select okf.detail.commonmark version 1 to explicitly use the default.</p><div class="okf-form-grid">' +
    '<label><span>Renderer ID</span><input data-profile="details.renderer.id" list="okf-detail-renderer-choices" value="' + escapeOKF(selection.id || "") + '" placeholder="okf.detail.commonmark"></label>' +
    '<label><span>Version</span><input data-profile="details.renderer.version" value="' + escapeOKF(selection.version || "1") + '"></label>' +
    '<label class="full"><span>Extension parameters (JSON object)</span><textarea data-profile="details.renderer.parameters" rows="2" spellcheck="false">' + escapeOKF(JSON.stringify(selection.parameters || {}, null, 2)) + '</textarea></label></div></section>';
}

export function readDetailRenderer(root, details) {
  const input = (name) => root.querySelector('[data-profile="details.renderer.' + name + '"]');
  if (!input("id")) return;
  const id = input("id").value.trim();
  if (!id) { delete details.renderer; return; }
  const version = input("version").value.trim();
  if (!version) throw new Error("Detail renderer version is required.");
  const parameters = JSON.parse(input("parameters").value.trim() || "{}");
  if (!parameters || typeof parameters !== "object" || Array.isArray(parameters)) throw new Error("Detail renderer parameters must be a JSON object.");
  details.renderer = { id, version, parameters };
}
