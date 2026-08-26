  function finiteNumber(value, fallback) {
    return typeof value === "number" && Number.isFinite(value) ? value : fallback;
  }

  function normalizedPoint(value, offset) {
    const shift = finiteNumber(offset, 0);
    return {
      x: finiteNumber(value && value.x, 0) + shift,
      y: finiteNumber(value && value.y, 0) + shift
    };
  }

  function samePoint(left, right) {
    return left && right && left.x === right.x && left.y === right.y;
  }

  function appendUnique(points, value) {
    if (!value) return;
    const point = { x: finiteNumber(value.x, 0), y: finiteNumber(value.y, 0) };
    if (!points.length || !samePoint(points[points.length - 1], point)) points.push(point);
  }

  function lineSection(points) {
    if (!points || points.length < 2) return null;
    return {
      start: points[0],
      segments: points.slice(1).map(function (point) {
        return { kind: "line", to: point };
      })
    };
  }

  function midpoint(points) {
    return points[Math.floor(points.length / 2)] || { x: 0, y: 0 };
  }

  function lineRoute(points, labelX, labelY, kind) {
    const normalized = [];
    (points || []).forEach(function (point) { appendUnique(normalized, point); });
    if (normalized.length < 2) return null;
    const middle = midpoint(normalized);
    const label = {
      x: finiteNumber(labelX, middle.x),
      y: finiteNumber(labelY, middle.y - 7)
    };
    return {
      kind: kind || "polyline",
      sections: [lineSection(normalized)],
      label: label,
      // Keep the legacy fields while callers migrate to sections.
      points: normalized,
      labelX: label.x,
      labelY: label.y
    };
  }

  function routeFromSections(sections, offset) {
    const normalizedSections = [];
    const flattened = [];
    (sections || []).forEach(function (section) {
      const points = [];
      appendUnique(points, normalizedPoint(section && section.startPoint, offset));
      (section && section.bendPoints || []).forEach(function (point) {
        appendUnique(points, normalizedPoint(point, offset));
      });
      appendUnique(points, normalizedPoint(section && section.endPoint, offset));
      const line = lineSection(points);
      if (line) {
        normalizedSections.push(line);
        points.forEach(function (point) { appendUnique(flattened, point); });
      }
    });
    if (!normalizedSections.length) return null;
    const middle = midpoint(flattened);
    const route = {
      kind: "polyline",
      sections: normalizedSections,
      label: { x: middle.x, y: middle.y - 7 },
      points: flattened,
      labelX: middle.x,
      labelY: middle.y - 7
    };
    return route;
  }

  function routePoints(route) {
    if (!route) return null;
    if (route.sections && route.sections.length) {
      const points = [];
      for (let sectionIndex = 0; sectionIndex < route.sections.length; sectionIndex += 1) {
        const section = route.sections[sectionIndex];
        if (!section || !section.start || !Array.isArray(section.segments)) return null;
        appendUnique(points, section.start);
        for (let segmentIndex = 0; segmentIndex < section.segments.length; segmentIndex += 1) {
          const segment = section.segments[segmentIndex];
          if (!segment || segment.kind !== "line" || !segment.to) return null;
          appendUnique(points, segment.to);
        }
      }
      return points.length > 1 ? points : null;
    }
    if (route.points && route.points.length > 1) return route.points;
    if (!route.sourcePoint || !route.targetPoint) return null;
    const points = [];
    appendUnique(points, route.sourcePoint);
    (route.bendPoints || []).forEach(function (point) { appendUnique(points, point); });
    appendUnique(points, route.targetPoint);
    return points.length > 1 ? points : null;
  }

  function pathForRoute(route) {
    if (!route || !route.sections || !route.sections.length) return null;
    const parts = [];
    for (let sectionIndex = 0; sectionIndex < route.sections.length; sectionIndex += 1) {
      const section = route.sections[sectionIndex];
      if (!section || !section.start || !Array.isArray(section.segments)) return null;
      parts.push("M", section.start.x, section.start.y);
      for (let segmentIndex = 0; segmentIndex < section.segments.length; segmentIndex += 1) {
        const segment = section.segments[segmentIndex];
        if (!segment || !segment.to) return null;
        if (segment.kind === "line") {
          parts.push("L", segment.to.x, segment.to.y);
          continue;
        }
        if (segment.kind === "cubic" && segment.control1 && segment.control2) {
          parts.push("C", segment.control1.x, segment.control1.y + ",", segment.control2.x, segment.control2.y + ",", segment.to.x, segment.to.y);
          continue;
        }
        return null;
      }
    }
    return parts.join(" ");
  }

  function geometryFromRoute(route, labelX, labelY) {
    const path = pathForRoute(route);
    if (!path) return null;
    const points = routePoints(route);
    const middle = midpoint(points || []);
    const label = route.label || {};
    return {
      path: path,
      labelX: finiteNumber(labelX, finiteNumber(label.x, middle.x)),
      labelY: finiteNumber(labelY, finiteNumber(label.y, middle.y - 7))
    };
  }

  function orthogonalRoute(from, to) {
    const fromCenterX = from.x + from.width / 2;
    const fromCenterY = from.y + from.height / 2;
    const toCenterX = to.x + to.width / 2;
    const toCenterY = to.y + to.height / 2;
    const points = [];
    if (Math.abs(toCenterX - fromCenterX) >= Math.abs(toCenterY - fromCenterY)) {
      const forward = toCenterX >= fromCenterX;
      const sourceX = forward ? from.x + from.width : from.x;
      const targetX = forward ? to.x : to.x + to.width;
      const middleX = (sourceX + targetX) / 2;
      points.push({ x: sourceX, y: fromCenterY }, { x: middleX, y: fromCenterY }, { x: middleX, y: toCenterY }, { x: targetX, y: toCenterY });
    } else {
      const forward = toCenterY >= fromCenterY;
      const sourceY = forward ? from.y + from.height : from.y;
      const targetY = forward ? to.y : to.y + to.height;
      const middleY = (sourceY + targetY) / 2;
      points.push({ x: fromCenterX, y: sourceY }, { x: fromCenterX, y: middleY }, { x: toCenterX, y: middleY }, { x: toCenterX, y: targetY });
    }
    return lineRoute(points, undefined, undefined, "orthogonal");
  }

  function selfLoopRoute(from) {
    const x = from.x + from.width / 2;
    const start = { x: x, y: from.y };
    const end = { x: x, y: from.y + from.height };
    return {
      kind: "self-loop",
      sections: [{
        start: start,
        segments: [{
          kind: "cubic",
          control1: { x: x + 100, y: from.y - 55 },
          control2: { x: x + 100, y: from.y + from.height + 55 },
          to: end
        }]
      }],
      label: { x: x + 50, y: from.y + from.height / 2 }
    };
  }

  function smoothCurveGeometry(from, to) {
    const x1 = from.x + from.width;
    const y1 = from.y + from.height / 2;
    const x2 = to.x;
    const y2 = to.y + to.height / 2;
    const bend = Math.max(32, Math.abs(x2 - x1) * 0.35);
    return {
      path: "M " + x1 + " " + y1 + " C " + (x1 + bend) + " " + y1 + ", " + (x2 - bend) + " " + y2 + ", " + x2 + " " + y2,
      labelX: (x1 + x2) / 2,
      labelY: (y1 + y2) / 2 - 7
    };
  }

  function edgeGeometry(relationship, from, to, route, preferOrthogonal) {
    const routed = geometryFromRoute(route, route && route.labelX, route && route.labelY);
    if (routed) return routed;
    if (relationship.from_visible_id === relationship.to_visible_id) return geometryFromRoute(selfLoopRoute(from));
    if (preferOrthogonal) return geometryFromRoute(orthogonalRoute(from, to));
    return smoothCurveGeometry(from, to);
  }

export {
  edgeGeometry,
  geometryFromRoute,
  orthogonalRoute,
  pathForRoute,
  routeFromSections as fromELKSections,
  routePoints,
  selfLoopRoute
};
