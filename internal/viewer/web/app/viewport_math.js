export function visibleGraphArea(clientWidth, clientHeight) {
  return {
    width: Number.isFinite(clientWidth) && clientWidth > 12 ? clientWidth - 12 : 360,
    height: Number.isFinite(clientHeight) && clientHeight > 12 ? clientHeight - 12 : 260
  };
}

export function fitViewportTransform(input) {
  const baseScale = Math.min(input.renderedWidth / input.layoutWidth, input.renderedHeight / input.layoutHeight);
  const contentWidth = Math.max(1, input.bounds.maxX - input.bounds.minX);
  const contentHeight = Math.max(1, input.bounds.maxY - input.bounds.minY);
  const contentFitZoom = Math.min(input.availableWidth / (contentWidth * baseScale), input.availableHeight / (contentHeight * baseScale));
  const zoom = clamp(contentFitZoom, input.minimumZoom, input.maximumZoom, 1);
  const offsetX = (input.renderedWidth - input.layoutWidth * baseScale) / 2;
  const offsetY = (input.renderedHeight - input.layoutHeight * baseScale) / 2;
  return {
    zoom: zoom,
    panX: clamp((input.availableWidth / 2 - offsetX) / baseScale - input.bounds.centerX * zoom, -input.panLimit, input.panLimit, 0),
    panY: clamp((input.availableHeight / 2 - offsetY) / baseScale - input.bounds.centerY * zoom, -input.panLimit, input.panLimit, 0)
  };
}

function clamp(value, minimum, maximum, fallback) {
  if (!Number.isFinite(value)) return fallback;
  return Math.min(maximum, Math.max(minimum, value));
}
