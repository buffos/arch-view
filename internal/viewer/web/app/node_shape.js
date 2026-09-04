// Callers supply escaped, trusted SVG attributes; geometry stays shared.
function rectangle(box, attributes, radius) {
  return '<rect' + attributes + ' x="' + box.x + '" y="' + box.y + '" width="' + box.width + '" height="' + box.height + '" rx="' + radius + '"></rect>';
}

function polygon(box, attributes, points) {
  return '<polygon' + attributes + ' points="' + points.map(([x, y]) => (box.x + x * box.width) + ',' + (box.y + y * box.height)).join(' ') + '"></polygon>';
}

const shapes = new Map([
  ["rectangle", (box, attributes) => rectangle(box, attributes, 0)],
  ["rounded_rectangle", (box, attributes) => rectangle(box, attributes, 12)],
  ["pill", (box, attributes) => rectangle(box, attributes, Math.min(box.width, box.height) / 2)],
  ["diamond", (box, attributes) => polygon(box, attributes, [[0.5, 0], [1, 0.5], [0.5, 1], [0, 0.5]])],
  ["hexagon", (box, attributes) => polygon(box, attributes, [[0.25, 0], [0.75, 0], [1, 0.5], [0.75, 1], [0.25, 1], [0, 0.5]])],
  ["ellipse", (box, attributes) => '<ellipse' + attributes + ' cx="' + (box.x + box.width / 2) + '" cy="' + (box.y + box.height / 2) + '" rx="' + box.width / 2 + '" ry="' + box.height / 2 + '"></ellipse>']
]);

const contentFractions = new Map([
  ["diamond", { width: 0.5, height: 0.5 }],
  ["hexagon", { width: 0.5, height: 1 }],
  ["ellipse", { width: Math.SQRT1_2, height: Math.SQRT1_2 }]
]);

function contentFraction(shape) {
  const definition = validatedShapeDefinition(shape);
  if (definition) return definition.content;
  return contentFractions.get(String(shape || "").toLowerCase()) || { width: 1, height: 1 };
}

export function shapeDimensions(shape, content) {
  const fraction = contentFraction(shape);
  return { width: content.width / fraction.width, height: content.height / fraction.height };
}

export function shapeContentBox(shape, box) {
  const fraction = contentFraction(shape);
  const width = box.width * fraction.width, height = box.height * fraction.height;
  return { x: box.x + (fraction.x === undefined ? (box.width - width) / 2 : fraction.x * box.width), y: box.y + (fraction.y === undefined ? (box.height - height) / 2 : fraction.y * box.height), width, height };
}

const geometryRenderers = new Map([
  ["rectangle", (value, box, attributes) => rectangle(box, attributes, (value.corner_radius || 0) * Math.min(box.width, box.height))],
  ["ellipse", (_value, box, attributes) => shapes.get("ellipse")(box, attributes)],
  ["polygon", (value, box, attributes) => polygon(box, attributes, value.points.map((point) => [point.x, point.y]))]
]);

export function validatedShapeDefinition(value) {
  if (!value || typeof value !== "object" || !geometryRenderers.has(value.geometry)) return null;
  const unit = (number) => Number.isFinite(number) && number >= 0 && number <= 1;
  const content = value.content;
  // Match registration's minimum extent at unit-box floating-point precision.
  if (!content || ![content.x, content.y, content.width, content.height].every(unit) || content.width < Number.EPSILON || content.height < Number.EPSILON || content.x + content.width > 1 || content.y + content.height > 1) return null;
  if (value.corner_radius !== undefined && (!unit(value.corner_radius) || value.corner_radius > 0.5)) return null;
  if (value.geometry === "polygon" && (!Array.isArray(value.points) || value.points.length < 3 || value.points.length > 64 || value.points.some((point) => !point || !unit(point.x) || !unit(point.y)))) return null;
  return value;
}

export function nodeShapeMarkup(shape, box, attributes = "") {
  const definition = validatedShapeDefinition(shape);
  if (definition) return geometryRenderers.get(definition.geometry)(definition, box, attributes);
  const render = shapes.get(String(shape || "").toLowerCase()) || shapes.get("rounded_rectangle");
  return render(box, attributes);
}
