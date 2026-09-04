import { shapeBoundaryPoint, attachShapeEndpoints } from "./app/shape_boundary.js";

  function finiteNumber(value, fallback) {
    return typeof value === "number" && Number.isFinite(value) ? value : fallback;
  }

  function pointValue(value) {
    if (!value || !Number.isFinite(value.x) || !Number.isFinite(value.y)) return null;
    return { x: value.x, y: value.y };
  }

  function normalizedPoint(value, offset) {
    const point = pointValue(value);
    if (!point) return null;
    const shift = finiteNumber(offset, 0);
    return {
      x: point.x + shift,
      y: point.y + shift
    };
  }

  function samePoint(left, right) {
    return left && right && left.x === right.x && left.y === right.y;
  }

  function sectionsConnect(previous, current) {
    if (!previous) return true;
    const last = previous.segments[previous.segments.length - 1];
    return Boolean(last && samePoint(last.to, current.start));
  }

  function appendUnique(points, value) {
    const point = pointValue(value);
    if (!point) return false;
    if (!points.length || !samePoint(points[points.length - 1], point)) points.push(point);
    return true;
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
    if (!Array.isArray(points)) return null;
    const normalized = [];
    for (let index = 0; index < points.length; index += 1) {
      if (!appendUnique(normalized, points[index])) return null;
    }
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
    if (!Array.isArray(sections) || sections.length === 0) return null;
    const normalizedSections = [];
    const flattened = [];
    for (let sectionIndex = 0; sectionIndex < sections.length; sectionIndex += 1) {
      const section = sections[sectionIndex];
      if (!section) return null;
      const bendPoints = section.bendPoints == null ? [] : section.bendPoints;
      if (!Array.isArray(bendPoints)) return null;
      const points = [];
      if (!appendUnique(points, normalizedPoint(section.startPoint, offset))) return null;
      for (let pointIndex = 0; pointIndex < bendPoints.length; pointIndex += 1) {
        if (!appendUnique(points, normalizedPoint(bendPoints[pointIndex], offset))) return null;
      }
      if (!appendUnique(points, normalizedPoint(section.endPoint, offset))) return null;
      const line = lineSection(points);
      if (!line || !sectionsConnect(normalizedSections[normalizedSections.length - 1], line)) return null;
      normalizedSections.push(line);
      for (let pointIndex = 0; pointIndex < points.length; pointIndex += 1) {
        if (!appendUnique(flattened, points[pointIndex])) return null;
      }
    }
    if (!normalizedSections.length || flattened.length < 2) return null;
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

  function cubicPoint(start, control1, control2, to, ratio) {
    const inverse = 1 - ratio;
    return {
      x: (inverse * inverse * inverse * start.x)
        + (3 * inverse * inverse * ratio * control1.x)
        + (3 * inverse * ratio * ratio * control2.x)
        + (ratio * ratio * ratio * to.x),
      y: (inverse * inverse * inverse * start.y)
        + (3 * inverse * inverse * ratio * control1.y)
        + (3 * inverse * ratio * ratio * control2.y)
        + (ratio * ratio * ratio * to.y)
    };
  }

  function splineLabel(sections) {
    let segmentCount = 0;
    sections.forEach(function (section) { segmentCount += section.segments.length; });
    const target = Math.floor(segmentCount / 2);
    let current = 0;
    for (let sectionIndex = 0; sectionIndex < sections.length; sectionIndex += 1) {
      const section = sections[sectionIndex];
      for (let segmentIndex = 0; segmentIndex < section.segments.length; segmentIndex += 1) {
        if (current === target) {
          const segment = section.segments[segmentIndex];
          const start = segmentIndex === 0 ? section.start : section.segments[segmentIndex - 1].to;
          const point = cubicPoint(start, segment.control1, segment.control2, segment.to, 0.5);
          return { x: point.x, y: point.y - 7 };
        }
        current += 1;
      }
    }
    return { x: 0, y: -7 };
  }

  function splineSection(section, offset) {
    if (!section) return null;
    const start = normalizedPoint(section.startPoint, offset);
    const end = normalizedPoint(section.endPoint, offset);
    const bendPoints = section.bendPoints == null ? [] : section.bendPoints;
    if (!start || !end || !Array.isArray(bendPoints) || bendPoints.length < 2 || (bendPoints.length + 1) % 3 !== 0) return null;
    const segments = [];
    const anchors = [start];
    for (let bendIndex = 0; bendIndex < bendPoints.length; bendIndex += 3) {
      const control1 = normalizedPoint(bendPoints[bendIndex], offset);
      const control2 = normalizedPoint(bendPoints[bendIndex + 1], offset);
      const to = bendIndex + 2 < bendPoints.length
        ? normalizedPoint(bendPoints[bendIndex + 2], offset)
        : end;
      if (!control1 || !control2 || !to) return null;
      segments.push({ kind: "cubic", control1: control1, control2: control2, to: to });
      anchors.push(to);
    }
    return { start: start, segments: segments, anchors: anchors };
  }

  function routeFromSplineSections(sections, offset) {
    if (!Array.isArray(sections) || sections.length === 0) return null;
    const normalizedSections = [];
    const flattened = [];
    for (let sectionIndex = 0; sectionIndex < sections.length; sectionIndex += 1) {
      const spline = splineSection(sections[sectionIndex], offset);
      if (!spline) return null;
      const normalized = { start: spline.start, segments: spline.segments };
      if (!sectionsConnect(normalizedSections[normalizedSections.length - 1], normalized)) return null;
      normalizedSections.push(normalized);
      spline.anchors.forEach(function (point) { appendUnique(flattened, point); });
    }
    if (!normalizedSections.length) return null;
    const label = splineLabel(normalizedSections);
    return {
      kind: "spline",
      sections: normalizedSections,
      label: label,
      // Keep the legacy fields while callers migrate to sections. For a
      // spline, points contains route anchors, not the control-point stream.
      points: flattened,
      labelX: label.x,
      labelY: label.y
    };
  }

  function routePoints(route) {
    if (!route) return null;
    if (route.kind === "self-loop") return null;
    if (Object.prototype.hasOwnProperty.call(route, "sections")) {
      if (!Array.isArray(route.sections) || route.sections.length === 0) return null;
      const points = [];
      for (let sectionIndex = 0; sectionIndex < route.sections.length; sectionIndex += 1) {
        const section = route.sections[sectionIndex];
        const start = pointValue(section && section.start);
        if (!start || !Array.isArray(section.segments)) return null;
        if (!appendUnique(points, start)) return null;
        for (let segmentIndex = 0; segmentIndex < section.segments.length; segmentIndex += 1) {
          const segment = section.segments[segmentIndex];
          const to = pointValue(segment && segment.to);
          if (!segment || !to) return null;
          if (segment.kind === "cubic") {
            const control1 = pointValue(segment.control1);
            const control2 = pointValue(segment.control2);
            if (!control1 || !control2) return null;
            if (!appendUnique(points, control1) || !appendUnique(points, control2)) return null;
          }
          if (segment.kind !== "line" && segment.kind !== "cubic") return null;
          if (!appendUnique(points, to)) return null;
        }
      }
      return points.length > 1 ? points : null;
    }
    if (Object.prototype.hasOwnProperty.call(route, "points")) {
      if (!Array.isArray(route.points)) return null;
      const points = [];
      for (let index = 0; index < route.points.length; index += 1) {
        if (!appendUnique(points, route.points[index])) return null;
      }
      return points.length > 1 ? points : null;
    }
    if (!pointValue(route.sourcePoint) || !pointValue(route.targetPoint)) return null;
    const points = [];
    if (!appendUnique(points, route.sourcePoint)) return null;
    const bendPoints = route.bendPoints == null ? [] : route.bendPoints;
    if (!Array.isArray(bendPoints)) return null;
    for (let index = 0; index < bendPoints.length; index += 1) {
      if (!appendUnique(points, bendPoints[index])) return null;
    }
    if (!appendUnique(points, route.targetPoint)) return null;
    return points.length > 1 ? points : null;
  }

  function routeFromLegacyLayoutEdge(route) {
    if (!route || Object.prototype.hasOwnProperty.call(route, "sections")) return route;
    if (!Object.prototype.hasOwnProperty.call(route, "points")) return route;
    return lineRoute(route.points, route.label_x, route.label_y, "orthogonal");
  }

  function pathForRoute(route) {
    if (!route || !Array.isArray(route.sections) || !route.sections.length) return null;
    const parts = [];
    for (let sectionIndex = 0; sectionIndex < route.sections.length; sectionIndex += 1) {
      const section = route.sections[sectionIndex];
      const start = pointValue(section && section.start);
      if (!start || !Array.isArray(section.segments)) return null;
      parts.push("M", start.x, start.y);
      for (let segmentIndex = 0; segmentIndex < section.segments.length; segmentIndex += 1) {
        const segment = section.segments[segmentIndex];
        const to = pointValue(segment && segment.to);
        if (!segment || !to) return null;
        if (segment.kind === "line") {
          parts.push("L", to.x, to.y);
          continue;
        }
        const control1 = pointValue(segment.control1);
        const control2 = pointValue(segment.control2);
        if (segment.kind === "cubic" && control1 && control2) {
          parts.push("C", control1.x, control1.y + ",", control2.x, control2.y + ",", to.x, to.y);
          continue;
        }
        return null;
      }
    }
    return parts.join(" ");
  }

  function geometryFromRoute(route, labelX, labelY) {
    const normalizedRoute = routeFromLegacyLayoutEdge(route);
    const path = pathForRoute(normalizedRoute);
    if (!path) return null;
    const points = routePoints(normalizedRoute);
    const middle = midpoint(points || []);
    const label = normalizedRoute.label || {};
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
    return attachShapeEndpoints(lineRoute(points, undefined, undefined, "orthogonal"), from, to);
  }

  function straightRoute(from, to) {
    const fromCenterX = from.x + from.width / 2;
    const fromCenterY = from.y + from.height / 2;
    const toCenterX = to.x + to.width / 2;
    const toCenterY = to.y + to.height / 2;
    const deltaX = toCenterX - fromCenterX;
    const deltaY = toCenterY - fromCenterY;
    if (deltaX === 0 && deltaY === 0) return null;
    return lineRoute([
      shapeBoundaryPoint(from, deltaX, deltaY),
      shapeBoundaryPoint(to, -deltaX, -deltaY)
    ], undefined, undefined, "polyline");
  }

  function selfLoopRoute(from) {
    const x = from.x + from.width / 2;
    const start = shapeBoundaryPoint(from, 0, -from.height / 2);
    const end = shapeBoundaryPoint(from, 0, from.height / 2);
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

  function edgeGeometry(relationship, from, to, route) {
    if (relationship.from_visible_id === relationship.to_visible_id) return geometryFromRoute(selfLoopRoute(from));
    const attached = attachShapeEndpoints(routeFromLegacyLayoutEdge(route), from, to);
    const routed = geometryFromRoute(attached, route && route.labelX, route && route.labelY);
    if (routed) return routed;
    // Any missing or malformed route is deterministic orthogonal fallback.
    // This keeps layout failures and manual-position routing on the same
    // renderer-neutral path representation.
    return geometryFromRoute(orthogonalRoute(from, to));
  }

export {
  edgeGeometry,
  geometryFromRoute,
  orthogonalRoute,
  straightRoute,
  pathForRoute,
  routeFromSections as fromELKSections,
  routeFromSplineSections as fromELKSplineSections,
  routePoints,
  selfLoopRoute
};
