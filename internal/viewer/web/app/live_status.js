function statusText(status, freshness, revision) {
  const state = String(status || "unknown").replaceAll("_", " ");
  const fresh = String(freshness || "unknown").replaceAll("_", " ");
  return revision ? "Live · " + state + " · " + fresh + " · revision " + revision : "Live · " + state + " · waiting for a ready revision";
}

function diagnosticText(diagnostics) {
  if (!Array.isArray(diagnostics) || !diagnostics.length) return "";
  return diagnostics.map(function (diagnostic) {
    return diagnostic && diagnostic.message ? diagnostic.message : "Live analysis reported a diagnostic.";
  }).join(" ");
}

export function createLiveController(context, api) {
  let timer = null;
  let stopped = false;
  let lastRevision = 0;
  let onRevision = null;

  function endpoint(suffix) {
    return "/v1/live/" + encodeURIComponent(context.liveSessionID) + "/" + suffix;
  }

  function render(envelope) {
    const result = envelope && envelope.result || {};
    const snapshot = result.snapshot || null;
    const state = result.state || snapshot && snapshot.state || "unknown";
    const freshness = envelope && envelope.freshness && envelope.freshness.status || snapshot && snapshot.freshness && snapshot.freshness.status || "unknown";
    const revision = Number(envelope && envelope.revision || snapshot && snapshot.revision || 0);
    const diagnostics = Array.isArray(result.diagnostics) && result.diagnostics.length ? result.diagnostics : envelope && Array.isArray(envelope.diagnostics) ? envelope.diagnostics : [];
    context.state.liveStatus = { state: state, freshness: freshness, revision: revision, diagnostics: diagnostics };
    if (revision > 0) context.state.liveRevision = revision;
    if (context.elements.liveStatus) {
      context.elements.liveStatus.hidden = false;
      const summary = statusText(state, freshness, revision);
      context.elements.liveStatus.textContent = diagnostics.length ? summary + " · " + diagnostics.length + " diagnostic" + (diagnostics.length === 1 ? "" : "s") : summary;
      context.elements.liveStatus.setAttribute("aria-label", diagnostics.length ? summary + ". " + diagnosticText(diagnostics) : summary);
      context.elements.liveStatus.title = diagnostics.length ? diagnosticText(diagnostics) : "";
      context.elements.liveStatus.dataset.status = String(freshness || state).toLowerCase();
    }
    return { state: state, freshness: freshness, revision: revision, snapshot: snapshot };
  }

  async function refresh() {
    if (!context.liveEnabled || stopped) return null;
    const envelope = await api.getJSON(endpoint("status"));
    const value = render(envelope);
    if (value.revision && value.revision !== lastRevision) {
      const previous = lastRevision;
      lastRevision = value.revision;
      if (previous && typeof onRevision === "function") await onRevision(value.revision);
    }
    return value;
  }

  async function waitForReady(maxAttempts) {
    if (!context.liveEnabled) return null;
    const attempts = maxAttempts || 120;
    for (let attempt = 0; attempt < attempts; attempt += 1) {
      try {
        const value = await refresh();
        if (value && value.snapshot && value.revision) {
          lastRevision = value.revision;
          return value;
        }
        if (value && value.state === "failed") throw new Error("The live session failed before publishing a ready revision.");
      } catch (error) {
        if (attempt === attempts - 1) throw error;
      }
      await new Promise(function (resolve) { setTimeout(resolve, 250); });
    }
    throw new Error("The live session did not publish a ready revision in time.");
  }

  function start() {
    if (!context.liveEnabled || timer) return;
    stopped = false;
    timer = setInterval(function () {
      void refresh().catch(function (error) {
        if (context.elements.liveStatus) {
          context.elements.liveStatus.hidden = false;
          context.elements.liveStatus.textContent = "Live · status unavailable · " + (error.message || "retrying");
          context.elements.liveStatus.setAttribute("aria-label", context.elements.liveStatus.textContent);
          context.elements.liveStatus.title = error.message || "Live status is unavailable.";
          context.elements.liveStatus.dataset.status = "failed";
        }
      });
    }, 1000);
  }

  function stop() {
    stopped = true;
    if (timer) clearInterval(timer);
    timer = null;
  }

  function bindRevisionHandler(handler) {
    onRevision = handler;
  }

  return { bindRevisionHandler, refresh, start, stop, waitForReady };
}
