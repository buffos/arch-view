import { clampNumber } from "./utils.js";

export function createViewportState() {
  return { zoom: 1, panX: 0, panY: 0, positions: {} };
}

export function viewportTransform(viewport) {
  const value = viewport || createViewportState();
  return "translate(" + value.panX + " " + value.panY + ") scale(" + value.zoom + ")";
}

export function updateViewportZoom(viewport, delta, minimum, maximum) {
  const value = viewport || createViewportState();
  value.zoom = clampNumber(value.zoom + delta, minimum, maximum, 1);
  return value;
}

export function resetViewportZoom(viewport) {
  const value = viewport || createViewportState();
  value.zoom = 1;
  return value;
}

export function applyViewportFit(viewport, fitted) {
  const value = viewport || createViewportState();
  value.zoom = fitted.zoom;
  value.panX = fitted.panX;
  value.panY = fitted.panY;
  return value;
}

export function bindViewportGestures(surface, options) {
  if (!surface) return function () {};
  const settings = options || {};
  const getViewport = settings.getViewport || function () { return createViewportState(); };
  const panLimit = Number.isFinite(settings.panLimit) ? settings.panLimit : 100000;
  const panSpeed = Number.isFinite(settings.panSpeed) ? settings.panSpeed : 1;
  const zoomStep = Number.isFinite(settings.zoomStep) ? settings.zoomStep : 0.08;
  const canPan = settings.isPanTarget || function () { return true; };
  const onPointerDown = function (event) {
    if (event.button !== 0 || !canPan(event.target, event)) return;
    const viewport = getViewport();
    if (!viewport) return;
    const start = { x: event.clientX, y: event.clientY, panX: viewport.panX, panY: viewport.panY };
    event.preventDefault();
    surface.classList.add("is-panning");
    const move = function (pointerEvent) {
      const current = getViewport();
      const baseScale = Math.max(0.01, Number(settings.getBaseScale ? settings.getBaseScale() : 1) || 1);
      current.panX = clampNumber(start.panX + ((pointerEvent.clientX - start.x) / baseScale) * panSpeed, -panLimit, panLimit, start.panX);
      current.panY = clampNumber(start.panY + ((pointerEvent.clientY - start.y) / baseScale) * panSpeed, -panLimit, panLimit, start.panY);
      if (settings.onChange) settings.onChange(current);
      if (settings.onRender) settings.onRender(current);
    };
    const stop = function () {
      document.removeEventListener("pointermove", move);
      document.removeEventListener("pointerup", stop);
      document.removeEventListener("pointercancel", stop);
      surface.classList.remove("is-panning");
    };
    document.addEventListener("pointermove", move);
    document.addEventListener("pointerup", stop);
    document.addEventListener("pointercancel", stop);
  };
  const onWheel = function (event) {
    event.preventDefault();
    if (settings.onZoom) settings.onZoom(event.deltaY < 0 ? zoomStep : -zoomStep);
  };
  surface.addEventListener("pointerdown", onPointerDown);
  surface.addEventListener("wheel", onWheel, { passive: false });
  return function () {
    surface.removeEventListener("pointerdown", onPointerDown);
    surface.removeEventListener("wheel", onWheel);
  };
}
