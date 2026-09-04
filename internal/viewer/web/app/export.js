const SVG_NAMESPACE = "http://www.w3.org/2000/svg";
const EXPORT_PADDING = 32;

const VIEWPORT_SELECTOR = ".viewport-content, .okf-viewport-content";
const PRESENTATION_PROPERTIES = [
  "fill", "fill-opacity", "fill-rule", "stroke", "stroke-width", "stroke-opacity",
  "stroke-dasharray", "stroke-dashoffset", "stroke-linecap", "stroke-linejoin",
  "opacity", "color", "font-family", "font-size", "font-weight", "font-style",
  "letter-spacing", "text-anchor", "dominant-baseline", "paint-order", "visibility", "filter"
];

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
  capturePresentation(source, copy);
  const viewportContent = copy.querySelector(VIEWPORT_SELECTOR);
  if (viewportContent) viewportContent.removeAttribute("transform");
  copy.querySelectorAll(".edge-hit, .node-hitzone").forEach(function (element) { element.remove(); });
  copy.setAttribute("xmlns", SVG_NAMESPACE);
  copy.setAttribute("version", "1.1");
  copy.setAttribute("viewBox", [bounds.x, bounds.y, bounds.width, bounds.height].join(" "));
  copy.setAttribute("width", String(bounds.width));
  copy.setAttribute("height", String(bounds.height));
  copy.setAttribute("preserveAspectRatio", "xMidYMid meet");

  return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" + new XMLSerializer().serializeToString(copy);
}

function capturePresentation(source, copy) {
  const originals = [source, ...source.querySelectorAll("*")];
  const clones = [copy, ...copy.querySelectorAll("*")];
  originals.forEach((element, index) => {
    const computed = source.ownerDocument.defaultView.getComputedStyle(element);
    PRESENTATION_PROPERTIES.forEach((property) => {
      const value = computed.getPropertyValue(property);
      if (value) clones[index].style.setProperty(property, value);
    });
  });
  const canvasStyle = source.ownerDocument.defaultView.getComputedStyle(source);
  copy.style.setProperty("background", canvasStyle.background);
  copy.style.setProperty("background-color", canvasStyle.getPropertyValue("--bg") || "#090e1d");
}

export function exportBounds(source) {
  const content = source.querySelector(VIEWPORT_SELECTOR);
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
  const label = context.exportLabel || (scene && scene.project ? scene.project.root_label : "architecture");
  const safe = String(label || "architecture")
    .trim()
    .replace(/[^a-z0-9._-]+/gi, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80);
  return "arch-view-" + (safe || "architecture") + ".svg";
}
