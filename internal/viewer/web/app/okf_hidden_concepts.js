import { escapeOKF } from "./okf_markup.js";

export function renderHiddenConcepts(container, state, api, focus) {
  if (!container) return;
  const snapshot = state.snapshot;
  container.innerHTML = "";
  if (!snapshot || !snapshot.counts?.hidden_nodes) return;
  container.innerHTML = '<details><summary>Browse hidden concepts</summary><p class="muted" role="status">Open to load concepts outside this projection.</p><label class="okf-editor-field">Hidden concept<select aria-label="Hidden concept" disabled></select></label><button class="button secondary" type="button" disabled>Focus subtree</button></details>';
  const disclosure = container.querySelector("details");
  const status = container.querySelector("p");
  const select = container.querySelector("select");
  const button = container.querySelector("button");
  const current = () => state.snapshot === snapshot && container.querySelector("details") === disclosure;
  let loading = false;
  let loaded = false;
  disclosure.addEventListener("toggle", async () => {
    if (!disclosure.open || loading || loaded || !current()) return;
    loading = true;
    status.textContent = "Loading hidden concepts…";
    try {
      const summary = await api.getSummary(snapshot.source.bundle_id);
      if (!current()) return;
      if (summary.source_revision !== snapshot.source.source_revision) throw new Error("Source changed. Refresh the bundle before browsing hidden concepts.");
      const visible = new Set(snapshot.nodes.map((node) => node.concept_id || node.id));
      const hidden = (summary.files || []).filter((id) => !visible.has(id));
      select.innerHTML = hidden.map((id) => '<option value="' + escapeOKF(id) + '">' + escapeOKF(id) + '</option>').join("");
      select.disabled = button.disabled = hidden.length === 0;
      status.textContent = hidden.length + " concepts outside this projection. Focusing keeps the current profile and safety limits.";
      loaded = true;
    } catch (error) {
      if (current()) status.textContent = error.message + " Close and reopen this section to retry.";
    } finally {
      loading = false;
    }
  });
  button.addEventListener("click", () => {
    if (current() && !button.disabled && select.value) void focus(select.value);
  });
}
