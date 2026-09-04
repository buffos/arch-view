// Read the entire input, preserving browser range/step validation rather than
// silently truncating numeric prefixes or replacing invalid input with defaults.
export function readNumericInput(element, fallback, integer = false) {
  if (!element) return fallback;
  if (element.validity && !element.validity.valid) {
    throw new Error(element.validationMessage || "Enter a valid number within the allowed range.");
  }
  const raw = String(element.value).trim();
  if (!raw) return fallback;
  const value = Number(raw);
  if (!Number.isFinite(value) || (integer && !Number.isSafeInteger(value))) {
    throw new Error(integer ? "Enter a whole number." : "Enter a finite number.");
  }
  if (element.min != null && element.min !== "" && value < Number(element.min)) throw new Error("Value must be at least " + element.min + ".");
  if (element.max != null && element.max !== "" && value > Number(element.max)) throw new Error("Value must be at most " + element.max + ".");
  return value;
}
