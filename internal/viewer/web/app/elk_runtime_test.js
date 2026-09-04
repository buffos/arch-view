const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

import(pathToFileURL(path.join(__dirname, "elk_runtime.js")).href).then(async ({ runELKLayout }) => {
  const previous = globalThis.Worker;
  let worker;
  class FakeWorker {
    constructor(url) { this.url = url; this.listeners = new Map(); worker = this; }
    addEventListener(type, callback) { this.listeners.set(type, callback); }
    removeEventListener(type) { this.listeners.delete(type); }
    terminate() { this.terminated = true; }
  }
  class Engine {
    constructor(options) { options.workerFactory(options.workerUrl); }
    layout() { return new Promise(() => {}); }
  }
  globalThis.Worker = FakeWorker;
  try {
    for (const type of ["error", "messageerror"]) {
      const pending = runELKLayout({}, "/elk-worker.js", Engine);
      assert.equal(worker.url, "/elk-worker.js");
      let prevented = false;
      worker.listeners.get(type)({ message: "worker failed", preventDefault() { prevented = true; } });
      await assert.rejects(pending, /worker failed/);
      assert.equal(prevented, true);
      assert.equal(worker.terminated, true);
      assert.equal(worker.listeners.size, 0);
    }
    class SuccessfulEngine extends Engine { async layout(graph) { return graph; } }
    const graph = { id: "root" };
    assert.equal(await runELKLayout(graph, "/elk-worker.js", SuccessfulEngine), graph);
    assert.equal(worker.terminated, true);
    class BrokenConstructor extends Engine { constructor(options) { super(options); throw new Error("constructor failed"); } }
    await assert.rejects(runELKLayout({}, "/elk-worker.js", BrokenConstructor), /constructor failed/);
    assert.equal(worker.terminated, true);
  } finally {
    if (previous === undefined) delete globalThis.Worker;
    else globalThis.Worker = previous;
  }
}).catch((error) => { console.error(error); process.exitCode = 1; });
