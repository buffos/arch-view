import { validatedShapeDefinition } from "./node_shape.js";

function capsuleScale(x, y, box, radiusFraction = 0.5) {
  const bound = Math.max(Math.abs(x), Math.abs(y));
  if (bound === 0) return 0;
  const hx = box.width / 2, hy = box.height / 2, radius = Math.min(box.width, box.height) * radiusFraction;
  let low = 0, high = 1 / bound;
  for (let i = 0; i < 48; i++) {
    const t = (low + high) / 2;
    const dx = Math.max(0, Math.abs(x * hx * t) - hx + radius);
    const dy = Math.max(0, Math.abs(y * hy * t) - hy + radius);
    if (Math.hypot(dx, dy) <= radius) low = t;
    else high = t;
  }
  return 1 / low;
}

const radialScales = new Map([
  ["pill", (x, y, box) => capsuleScale(x, y, box)],
  ["diamond", (x, y) => Math.abs(x) + Math.abs(y)],
  ["hexagon", (x, y) => Math.max(Math.abs(y), Math.abs(x) + Math.abs(y) / 2)],
  ["ellipse", (x, y) => Math.hypot(x, y)]
]);

function polygonScale(x, y, _box, definition) {
  const direction = { x: x / 2, y: y / 2 };
  const cross = (a, b) => a.x * b.y - a.y * b.x;
  let nearest = Infinity;
  definition.points.forEach((point, index) => {
    const next = definition.points[(index + 1) % definition.points.length];
    const edge = { x: next.x - point.x, y: next.y - point.y };
    const offset = { x: point.x - 0.5, y: point.y - 0.5 };
    const denominator = cross(direction, edge);
    if (Math.abs(denominator) < 1e-12) return;
    const t = cross(offset, edge) / denominator, u = cross(offset, direction) / denominator;
    if (t >= 0 && u >= -1e-10 && u <= 1 + 1e-10) nearest = Math.min(nearest, t);
  });
  return Number.isFinite(nearest) && nearest > 0 ? 1 / nearest : Math.max(Math.abs(x), Math.abs(y));
}

const definitionScales = new Map([
  ["polygon", polygonScale],
  ["ellipse", (x, y) => Math.hypot(x, y)],
  ["rectangle", (x, y, box, definition) => capsuleScale(x, y, box, definition.corner_radius || 0)]
]);

export function shapeBoundaryPoint(box, deltaX, deltaY) {
  const cx = box.x + box.width / 2, cy = box.y + box.height / 2;
  const hx = Math.max(0.01, box.width / 2), hy = Math.max(0.01, box.height / 2);
  const definition = validatedShapeDefinition(box.shape);
  const scaleFor = (definition && definitionScales.get(definition.geometry)) || radialScales.get(String(box.shape || "").toLowerCase()) || ((x, y) => Math.max(Math.abs(x), Math.abs(y)));
  const scale = scaleFor(deltaX / hx, deltaY / hy, box, definition);
  if (!Number.isFinite(scale) || scale === 0) return { x: cx + hx, y: cy };
  return { x: cx + deltaX / scale, y: cy + deltaY / scale };
}

// Copy only changed route structure; cached layout routes remain immutable.
export function attachShapeEndpoints(route, from, to) {
  if (!Array.isArray(route?.sections) || !route.sections.length) return route;
  if (route.sections.some((section) => !section || !Array.isArray(section.segments))) return route;
  const adjustFrom = validatedShapeDefinition(from.shape) || radialScales.has(String(from.shape || "").toLowerCase());
  const adjustTo = validatedShapeDefinition(to.shape) || radialScales.has(String(to.shape || "").toLowerCase());
  if (!adjustFrom && !adjustTo) return route;
  const sections = route.sections.map((section) => ({ ...section, segments: section.segments?.map((segment) => ({ ...segment })) }));
  const first = sections[0], last = sections[sections.length - 1];
  const endpoint = last.segments?.[last.segments.length - 1];
  const project = (box, point) => shapeBoundaryPoint(box, point.x - box.x - box.width / 2, point.y - box.y - box.height / 2);
  if (adjustFrom && Number.isFinite(first.start?.x) && Number.isFinite(first.start?.y)) first.start = project(from, first.start);
  if (adjustTo && Number.isFinite(endpoint?.to?.x) && Number.isFinite(endpoint?.to?.y)) endpoint.to = project(to, endpoint.to);
  return { ...route, sections };
}
