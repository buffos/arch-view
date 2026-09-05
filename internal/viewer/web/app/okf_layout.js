import { cloneLayoutValue } from "./layout_value.js";
export const OKF_DEFAULT_LAYOUT_ALGORITHM = "layered";
export const OKF_DEFAULT_LAYOUT_FEATURES = Object.freeze(["junctions", "ports"]);

export function cloneOKFLayoutProfile(profile) {
  const result = cloneLayoutValue(profile, OKF_DEFAULT_LAYOUT_ALGORITHM);
  if (profile == null) result.features = [...OKF_DEFAULT_LAYOUT_FEATURES];
  return result;
}
