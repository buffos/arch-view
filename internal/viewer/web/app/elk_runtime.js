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
