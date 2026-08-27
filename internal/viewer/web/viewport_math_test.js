const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "app", "viewport_math.js"), "utf8");
import("data:text/javascript;charset=utf-8," + encodeURIComponent(source)).then(function (viewport) {
  assert.deepEqual(
    viewport.visibleGraphArea(710, 496),
    { width: 698, height: 484 },
    "fit must use the visible clipped graph viewport instead of an oversized SVG child"
  );
  assert.deepEqual(
    viewport.visibleGraphArea(300, 200),
    { width: 288, height: 188 },
    "fit must not overstate a narrow but measurable viewport"
  );
  assert.deepEqual(
    viewport.visibleGraphArea(0, 0),
    { width: 360, height: 260 },
    "fit must retain safe fallback dimensions before the viewport is measurable"
  );

  const input = {
    availableWidth: 698,
    availableHeight: 484,
    renderedWidth: 960,
    renderedHeight: 480,
    layoutWidth: 902,
    layoutHeight: 451,
    bounds: { minX: 30, minY: 30, maxX: 870, maxY: 420, centerX: 450, centerY: 225 },
    minimumZoom: 0.35,
    maximumZoom: 1.45,
    panLimit: 100000
  };
  const fitted = viewport.fitViewportTransform(input);
  const baseScale = Math.min(input.renderedWidth / input.layoutWidth, input.renderedHeight / input.layoutHeight);
  const offsetX = (input.renderedWidth - input.layoutWidth * baseScale) / 2;
  const left = offsetX + baseScale * (fitted.panX + input.bounds.minX * fitted.zoom);
  const right = offsetX + baseScale * (fitted.panX + input.bounds.maxX * fitted.zoom);
  assert.ok(fitted.zoom < 1, "fit must be able to zoom out when the rendered SVG is wider than the clipped viewport");
  assert.ok(left >= -0.001 && right <= input.availableWidth + 0.001, "fit must keep horizontal content inside the visible viewport");
}).catch(function (error) {
  console.error(error);
  process.exitCode = 1;
});
