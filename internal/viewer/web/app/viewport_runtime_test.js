const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

const documentListeners = new Map();
global.document = {
  addEventListener: (type, handler) => documentListeners.set(type, handler),
  removeEventListener: (type, handler) => {
    if (documentListeners.get(type) === handler) documentListeners.delete(type);
  }
};

import(pathToFileURL(path.join(__dirname, "viewport_runtime.js")).href).then((runtime) => {
  const surfaceListeners = new Map();
  const surface = {
    classList: { add() {}, remove() {} },
    addEventListener: (type, handler) => surfaceListeners.set(type, handler),
    removeEventListener: (type, handler) => {
      if (surfaceListeners.get(type) === handler) surfaceListeners.delete(type);
    }
  };
  const content = {
    transform: "",
    setAttribute(name, value) { if (name === "transform") this.transform = value; }
  };
  const container = { querySelector: (selector) => selector === ".viewport-content" ? content : null };
  const viewport = runtime.createViewportState();
  let renderCount = 0;
  runtime.bindViewportGestures(surface, {
    getViewport: () => viewport,
    getBaseScale: () => 0.5,
    onRender: () => {
      renderCount += 1;
      runtime.applyViewportTransform(container, ".viewport-content", viewport);
    }
  });

  surfaceListeners.get("pointerdown")({ button: 0, target: {}, clientX: 100, clientY: 100, preventDefault() {} });
  documentListeners.get("pointermove")({ clientX: 110, clientY: 115 });
  documentListeners.get("pointermove")({ clientX: 120, clientY: 130 });

  assert.equal(renderCount, 2);
  assert.equal(content.transform, "translate(40 60) scale(1)");
  assert.deepEqual(viewport, { zoom: 1, panX: 40, panY: 60, positions: {} });
  assert.equal(container.querySelector(".viewport-content"), content, "panning must keep the live SVG content mounted");
}).catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
