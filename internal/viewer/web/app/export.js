const SVG_NAMESPACE = "http://www.w3.org/2000/svg";
const EXPORT_PADDING = 32;

const SVG_EXPORT_STYLES = `
svg {
  background: #090e1d;
  color: #f3f6ff;
  font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}
.edge-line { fill: none; stroke: #7483a9; stroke-width: 2; opacity: 0.72; }
.edge-line.cycle { stroke: #fb7185; stroke-width: 3; opacity: 0.95; }
.edge-line.feedback { stroke: #fbbf74; stroke-dasharray: 7 5; }
.edge-label { fill: #c2cce3; font-size: 11px; font-weight: 800; paint-order: stroke; stroke: #11182c; stroke-width: 4px; }
.edge-label.cycle { fill: #fb7185; }
.node-shape { fill: #1b2a4d; stroke: #4d6292; stroke-width: 1.5; }
.node-shape.group { fill: #252248; stroke: #7961bf; }
.node-shape.reference { fill: #1b3030; stroke: #4c8c8a; stroke-dasharray: 4 4; }
.node-shape.cycle { stroke: #fb7185; stroke-width: 3; }
.node-shape.warning { stroke: #fbbf74; }
.node-shape.error { stroke: #fb7185; }
.node-shape.dimmed, .node-label.dimmed { opacity: 0.18; }
.node-shape.selected { fill: #28456a; stroke: #6ee7f9; stroke-width: 3; }
.node-label { fill: #f3f6ff; font-size: 12px; font-weight: 800; }
.node-subtitle { fill: #9eaccb; font-size: 10px; }
`;

export function downloadCurrentSVG(context) {
  const source = context && context.elements && context.elements.graph
    ? context.elements.graph.querySelector("svg")
    : null;
  if (!source) return false;

  const bounds = exportBounds(source);
  const data = serializeSVG(source, bounds);
  const blob = new Blob([data], { type: "image/svg+xml;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = downloadFilename(context);
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(function () { URL.revokeObjectURL(url); }, 0);
  return true;
}

export function serializeSVG(source, bounds) {
  const copy = source.cloneNode(true);
  const viewportContent = copy.querySelector(".viewport-content");
  if (viewportContent) viewportContent.removeAttribute("transform");
  copy.querySelectorAll(".edge-hit, .node-hitzone").forEach(function (element) { element.remove(); });
  copy.setAttribute("xmlns", SVG_NAMESPACE);
  copy.setAttribute("version", "1.1");
  copy.setAttribute("viewBox", [bounds.x, bounds.y, bounds.width, bounds.height].join(" "));
  copy.setAttribute("width", String(bounds.width));
  copy.setAttribute("height", String(bounds.height));
  copy.setAttribute("preserveAspectRatio", "xMidYMid meet");

  const style = copy.ownerDocument.createElementNS(SVG_NAMESPACE, "style");
  style.textContent = SVG_EXPORT_STYLES;
  copy.insertBefore(style, copy.firstChild);
  return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" + new XMLSerializer().serializeToString(copy);
}

function exportBounds(source) {
  const content = source.querySelector(".viewport-content");
  if (content && typeof content.getBBox === "function") {
    try {
      const box = content.getBBox();
      if (finiteBox(box) && box.width > 0 && box.height > 0) return paddedBox(box);
    } catch (error) {
      // Use the established viewBox when SVG geometry is not measurable yet.
    }
  }
  const values = String(source.getAttribute("viewBox") || "").trim().split(/[\s,]+/).map(Number);
  if (values.length === 4 && values.every(Number.isFinite) && values[2] > 0 && values[3] > 0) {
    return { x: values[0], y: values[1], width: values[2], height: values[3] };
  }
  return { x: 0, y: 0, width: 1, height: 1 };
}

function finiteBox(box) {
  return box && [box.x, box.y, box.width, box.height].every(Number.isFinite);
}

function paddedBox(box) {
  return {
    x: box.x - EXPORT_PADDING,
    y: box.y - EXPORT_PADDING,
    width: box.width + EXPORT_PADDING * 2,
    height: box.height + EXPORT_PADDING * 2
  };
}

function downloadFilename(context) {
  const scene = context && context.state ? context.state.scene : null;
  const label = scene && scene.project ? scene.project.root_label : "architecture";
  const safe = String(label || "architecture")
    .trim()
    .replace(/[^a-z0-9._-]+/gi, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80);
  return "arch-view-" + (safe || "architecture") + ".svg";
}
