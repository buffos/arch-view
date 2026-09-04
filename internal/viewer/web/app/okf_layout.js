import { cloneLayoutValue } from "./layout_value.js";
export const OKF_DEFAULT_LAYOUT_ALGORITHM = "mrtree";

export function cloneOKFLayoutProfile(profile) {
  return cloneLayoutValue(profile, OKF_DEFAULT_LAYOUT_ALGORITHM);
}
