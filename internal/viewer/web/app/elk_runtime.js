import { buildELKGraph } from "../layout_request.js";
import { sharedFeatureRegistry } from "./layout_features.js";

// Both scene adapters use the same feature pipeline and worker boundary.
export async function runFeatureLayout(scene, profile, catalog, workerURL, Engine = globalThis.ELK, source = null) {
  const effectiveProfile = catalog ? profile : { ...(profile || {}), features: [] };
  const registry = sharedFeatureRegistry(catalog);
  const negotiated = registry.negotiate(effectiveProfile);
  let context = { scene, profile: effectiveProfile, source, diagnostics: negotiated.diagnostics };
  context = await registry.run("negotiate", context, negotiated.effective);
  context.graph = buildELKGraph(scene, effectiveProfile, catalog);
  context = await registry.run("prepare", context, negotiated.effective);
  context.output = await runELKLayout(context.graph, workerURL, Engine);
  context = await registry.run("normalize", context, negotiated.effective);
  context = await registry.run("validate", context, negotiated.effective);
  context = await registry.run("render", context, negotiated.effective);
  return Object.assign(context.output, {
    featureDiagnostics: context.diagnostics,
    featureGeometry: context.featureGeometry,
    effectiveFeatures: negotiated.effective,
    geometrySource: source
  });
}

// Both scene adapters use the same worker lifetime and failure boundary.
export async function runELKLayout(graph, workerURL, Engine = globalThis.ELK) {
  let engine;
  let worker;
  let rejectWorker;
  const failure = new Promise((_resolve, reject) => { rejectWorker = reject; });
  const onError = (event) => {
    event.preventDefault?.();
    rejectWorker(new Error(event.message || "ELK worker could not process the layout."));
  };
  const options = workerURL ? { workerUrl: workerURL } : undefined;
  if (options && typeof globalThis.Worker === "function") {
    options.workerFactory = (url) => {
      worker = new globalThis.Worker(url);
      worker.addEventListener("error", onError);
      worker.addEventListener("messageerror", onError);
      return worker;
    };
  }
  try {
    engine = new Engine(options);
    return await Promise.race([engine.layout(graph), failure]);
  } finally {
    worker?.removeEventListener("error", onError);
    worker?.removeEventListener("messageerror", onError);
    // Native workers can be terminated directly, including constructor failure.
    if (worker) worker.terminate();
    else if (typeof engine?.terminateWorker === "function") {
      try {
        const termination = engine.terminateWorker();
        termination?.catch?.(() => {});
      } catch {
        // Some bundled non-worker adapters do not implement termination.
      }
    }
  }
}
