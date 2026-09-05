const assert = require("node:assert/strict");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

global.document = {
  fullscreenElement: {},
  body: { classList: { contains: function () { return true; } } }
};
global.sessionStorage = { getItem: function () { return null; }, setItem: function () {} };

import(pathToFileURL(path.join(__dirname, "viewport.js")).href).then(function (viewport) {
  const context = {
    embeddedExport: null,
    constants: { minZoom: 0.35, maxZoom: 40000, panLimit: 100000 },
    state: {
      scene: {
        model_revision: "revision-1",
        model_id: "model-1",
        hierarchy_path: [],
        reference_visibility: "hidden",
        visible_nodes: [{ id: "root", label: "Root", layer: 0 }],
        visible_relationships: []
      },
      layout: null,
      viewport: null,
      scrollContexts: {}
    },
    elements: {
      graph: {
        clientWidth: 1000,
        clientHeight: 800,
        querySelector: function () { return { clientWidth: 1000, clientHeight: 800 }; }
      },
      zoomValue: { textContent: "" },
      resetLayout: { disabled: false }
    }
  };

  for (const expanded of [false, true]) {
    document.fullscreenElement = expanded ? {} : null;
    document.body.classList.contains = () => expanded;
    viewport.fitViewport(context, { renderGraph: function () {} });
    assert.ok(context.state.viewport.zoom > 1.45, "Fit must enlarge content beyond the former 145% ceiling in either mode");
  }
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});

import(pathToFileURL(path.join(__dirname, "okf_viewport.js")).href).then(function (viewport) {
  const state = {
    viewport: { zoom: 1, panX: 0, panY: 0, positions: { leaf: { x: 750, y: 300 } } },
    layout: {
      width: 1000, height: 800,
      positions: {
        group: { x: 400, y: 300, width: 200, height: 100 },
        leaf: { x: 450, y: 330, width: 100, height: 50 }
      },
      geometry: {
        nodes: [
          { id: "group", bounds: { x: 400, y: 300, width: 200, height: 100 }, children_ids: ["leaf"] },
          { id: "leaf", bounds: { x: 450, y: 330, width: 100, height: 50 }, children_ids: [] }
        ],
        containers: [{ id: "arch-view-container::group", owner_node_id: "group", bounds: { x: 388, y: 288, width: 224, height: 124 } }]
      }
    }
  };
  const elements = { graph: { clientWidth: 1000, clientHeight: 800, querySelector: (selector) => selector === "svg" ? { clientWidth: 1000, clientHeight: 800 } : null } };
  viewport.fitOKFViewport(state, elements);
  const fittedFrameRight = state.viewport.panX + (750 + 100 + 12) * state.viewport.zoom;
  assert.ok(fittedFrameRight <= 988, "Fit includes a container expanded by manual child movement");
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});

import(pathToFileURL(path.join(__dirname, "okf_viewport.js")).href).then(function (viewport) {
  const state = {
    viewport: { zoom: 1, panX: 0, panY: 0, positions: {} },
    layout: {
      width: 1000, height: 800,
      positions: { root: { x: 400, y: 300, width: 200, height: 100 } }
    }
  };
  const elements = {
    graph: {
      clientWidth: 1000, clientHeight: 800,
      querySelector: function (selector) {
        return selector === "svg" ? { clientWidth: 1000, clientHeight: 800 } : null;
      }
    }
  };
  for (const expanded of [false, true]) {
    document.fullscreenElement = expanded ? {} : null;
    document.body.classList.contains = () => expanded;
    viewport.fitOKFViewport(state, elements);
    assert.equal(state.viewport.zoom, 988 / (200 + 56), "OKF Fit must fill the available width with its existing margin in either mode");
    assert.equal(state.viewport.panX + 500 * state.viewport.zoom, 494, "Fit centers the graph horizontally");
    assert.equal(state.viewport.panY + 350 * state.viewport.zoom, 394, "Fit centers the graph vertically");
  }
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
