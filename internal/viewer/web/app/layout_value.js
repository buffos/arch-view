// Layout controls and request encoding share this structured value contract.
export const PADDING_SIDES = Object.freeze(["top", "right", "bottom", "left"]);

export function validPadding(value) {
  return value && typeof value === "object" && Object.keys(value).length === 4
    && PADDING_SIDES.every((side) => Number.isFinite(value[side]) && value[side] >= 0 && value[side] <= 10000);
}

const serializers = {
  padding(value) {
    if (!validPadding(value)) throw new Error("Graph padding requires four finite values between 0 and 10000.");
    return "[" + PADDING_SIDES.map((side) => side + "=" + value[side]).join(",") + "]";
  }
};

export function serializeLayoutValue(option, value) {
  return (serializers[option.control] || String)(value);
}

export function cloneLayoutValue(profile, algorithm) {
  const value = profile || {};
  const result = { algorithm: value.algorithm || algorithm, options: structuredClone(value.options || {}) };
  if (value.features != null) result.features = [...value.features];
  return result;
}
