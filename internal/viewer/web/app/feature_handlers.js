import { edgeLabelFeature } from "./edge_label_feature.js";
import { junctionFeature } from "./junction_feature.js";
import { portFeature } from "./port_feature.js";
import { splineFeature } from "./spline_feature.js";

// This is the only browser composition root for implemented layout features.
// Future stages add handlers here without adding feature branches to a scene.
export function implementedFeatureHandlers() {
  return [edgeLabelFeature, junctionFeature, splineFeature, portFeature];
}
