export function selectedView(search) {
  const params = new URLSearchParams(search == null ? globalThis.location?.search || "" : search);
  return params.get("view") === "okf" ? "okf" : "architecture";
}

export function configureModeNavigation(mode, hasOKF) {
  const architecture = document.getElementById("architecture-mode-link");
  const okf = document.getElementById("okf-mode-link");
  if (architecture) {
    architecture.textContent = mode === "okf" ? "Back to architecture viewer" : "Architecture viewer";
    architecture.hidden = mode === "architecture";
    architecture.setAttribute("aria-current", mode === "architecture" ? "page" : "false");
  }
  if (okf) {
    okf.hidden = mode === "okf" || !hasOKF;
    okf.setAttribute("aria-current", mode === "okf" ? "page" : "false");
  }
}

export async function discoverSelectableOKF(fetchImplementation = globalThis.fetch) {
  if (typeof fetchImplementation !== "function") return false;
  try {
    const response = await fetchImplementation("/v1/okf/catalog", { headers: { Accept: "application/json" } });
    if (!response.ok) return false;
    const payload = await response.json();
    const catalog = payload && Object.prototype.hasOwnProperty.call(payload, "data") ? payload.data : payload;
    return Boolean(catalog && Array.isArray(catalog.bundles) && catalog.bundles.some((bundle) => bundle && bundle.selectable === true));
  } catch (error) {
    return false;
  }
}
