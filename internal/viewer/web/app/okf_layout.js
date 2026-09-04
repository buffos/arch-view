export const OKF_DEFAULT_LAYOUT_ALGORITHM = "mrtree";

export function cloneOKFLayoutProfile(profile) {
  const value = profile || {};
  return { algorithm: value.algorithm || OKF_DEFAULT_LAYOUT_ALGORITHM, options: Object.assign({}, value.options || {}) };
}
